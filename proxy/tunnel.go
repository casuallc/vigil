/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/casuallc/vigil/poll"
	"github.com/gorilla/websocket"
)

// tunnelFrameChunk bounds a single binary body frame on the tunnel wire.
const tunnelFrameChunk = 256 * 1024

// TunnelRequestMeta precedes one request's body frames (text frame, JSON).
type TunnelRequestMeta struct {
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	URL     string      `json:"url"` // path or absolute URI; absolute hosts must match the target
	Headers http.Header `json:"headers"`
	BodyLen int64       `json:"body_len"`
}

// TunnelResponseMeta precedes one response's body frames (text frame, JSON).
type TunnelResponseMeta struct {
	ID      string      `json:"id"`
	Status  int         `json:"status"`
	Headers http.Header `json:"headers"`
	BodyLen int64       `json:"body_len"`
	Error   string      `json:"error,omitempty"`
}

// TunnelCore executes poll-mode proxy_session tunnel sessions.
//
// Wire protocol (serial per connection): the upstream sends a text frame
// with a JSON TunnelRequestMeta followed by ceil(body_len/chunk) binary
// body frames; bbx answers with a TunnelResponseMeta text frame plus body
// frames. Requests are executed serially on the connection; parallelism
// comes from multiple concurrent proxy_session tasks. The meta id field
// leaves room for a future multiplexed revision.
type TunnelCore struct {
	cfg  TunnelConfig
	wl   *Whitelist
	hook AccessHook
	sem  chan struct{}
}

// newTunnelCore builds the tunnel core from config. An empty
// AllowedTargets list (or a disabled tunnel) yields a whitelist that
// rejects everything, so no session can be established.
func newTunnelCore(cfg TunnelConfig, hook AccessHook) *TunnelCore {
	cfg.applyDefaults()
	wl, err := ParseWhitelist(cfg.AllowedTargets, true)
	if err != nil {
		// Invalid entries must not silently widen access: reject all.
		wl, _ = ParseWhitelist(nil, false)
	}
	return &TunnelCore{
		cfg:  cfg,
		wl:   wl,
		hook: hook,
		sem:  make(chan struct{}, cfg.MaxSessions),
	}
}

// Enabled reports whether tunnel sessions are allowed at all.
func (tc *TunnelCore) Enabled() bool {
	return tc.cfg.Enabled && len(tc.cfg.AllowedTargets) > 0
}

// TargetAllowed reports whether a tunnel session may proxy to target.
func (tc *TunnelCore) TargetAllowed(target string) bool {
	return tc.Enabled() && tc.wl.Allowed(target)
}

// acquire takes a session slot without blocking; the upstream is expected
// to retry when the agent is saturated.
func (tc *TunnelCore) acquire() (release func(), ok bool) {
	select {
	case tc.sem <- struct{}{}:
		return func() { <-tc.sem }, true
	default:
		return nil, false
	}
}

// Close releases tunnel resources. Sessions are owned by the poll agent's
// task contexts and end with it; there is nothing persistent to close.
func (tc *TunnelCore) Close() {}

// RunProxySession implements poll.ProxyRunner: it serves HTTP requests
// arriving over conn against target until either side closes, ctx ends, or
// the duration cap hits.
func (tc *TunnelCore) RunProxySession(ctx context.Context, conn *websocket.Conn, target string, limits poll.SessionLimits) (poll.SessionStats, error) {
	start := time.Now()
	stats := poll.SessionStats{EndReason: "closed"}
	finish := func(reason string, err error) (poll.SessionStats, error) {
		stats.EndReason = reason
		stats.DurationMs = time.Since(start).Milliseconds()
		return stats, err
	}

	if !tc.Enabled() {
		return finish("error", fmt.Errorf("proxy tunnel is disabled (proxy.tunnel.enabled / allowed_targets)"))
	}
	targetURL, err := url.Parse(target)
	if err != nil || targetURL.Host == "" || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		return finish("error", fmt.Errorf("invalid tunnel target %q", target))
	}
	// The local policy decides: the upstream cannot broaden allowed_targets.
	if !tc.wl.Allowed(targetURL.Host) && !tc.wl.Allowed(targetURL.Hostname()) {
		tc.record(AccessRecord{
			Instance: "tunnel", ClientIP: upstreamAddr(conn), Path: target,
			Status: http.StatusForbidden, Via: "tunnel", Denied: true,
		})
		return finish("error", fmt.Errorf("tunnel target %q rejected by local allowed_targets policy", target))
	}
	release, ok := tc.acquire()
	if !ok {
		return finish("error", fmt.Errorf("too many tunnel sessions (max %d)", tc.cfg.MaxSessions))
	}
	defer release()

	// Local caps win: the upstream can only narrow them.
	maxDuration := time.Duration(tc.cfg.MaxDurationSec) * time.Second
	if limits.MaxDuration > 0 && limits.MaxDuration < maxDuration {
		maxDuration = limits.MaxDuration
	}
	maxBody := tc.cfg.MaxBodyMB
	if limits.MaxBodyMB > 0 && (maxBody == 0 || limits.MaxBodyMB < maxBody) {
		maxBody = limits.MaxBodyMB
	}

	sessionCtx, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()
	// Unblock ReadMessage when the session context ends.
	go func() {
		<-sessionCtx.Done()
		conn.Close()
	}()

	// The session target is fixed and already authorized; the transport's
	// checking dialer re-validates the resolved IP of every connection.
	sessionWL, _ := ParseWhitelist([]string{targetURL.Hostname()}, true)
	client := &http.Client{Transport: newCheckingTransport(sessionWL)}

	for {
		req, meta, rerr := tc.readRequest(conn, maxBody)
		if rerr != nil {
			return finish(sessionEndReason(sessionCtx, ctx), nil)
		}

		rec := AccessRecord{
			Instance: "tunnel",
			ClientIP: upstreamAddr(conn),
			Method:   meta.Method,
			Path:     meta.URL,
			Via:      "tunnel",
		}
		reqStart := time.Now()
		respMeta, body, xerr := tc.executeRequest(sessionCtx, client, targetURL, req, meta, maxBody, &rec)
		rec.DurationMs = time.Since(reqStart).Milliseconds()
		if xerr != nil {
			rec.Status = respMeta.Status
			// Only policy rejections count as denials; upstream 502s don't.
			rec.Denied = respMeta.Status == http.StatusForbidden
		}
		rec.BytesOut = int(respMeta.BodyLen)
		stats.BytesIn += meta.BodyLen
		stats.BytesOut += respMeta.BodyLen

		if werr := tc.writeResponse(conn, respMeta, body); werr != nil {
			tc.record(rec)
			return finish("closed", nil)
		}
		stats.Requests++
		tc.record(rec)
		_ = xerr // per-request failures are reported in the response meta
	}
}

// readRequest reads one request meta frame plus its body frames.
func (tc *TunnelCore) readRequest(conn *websocket.Conn, maxBodyMB int64) (*http.Request, *TunnelRequestMeta, error) {
	msgType, payload, err := conn.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	if msgType != websocket.TextMessage {
		return nil, nil, fmt.Errorf("expected text meta frame, got type %d", msgType)
	}
	var meta TunnelRequestMeta
	if err := json.Unmarshal(payload, &meta); err != nil {
		return nil, nil, fmt.Errorf("invalid request meta: %w", err)
	}
	if meta.Method == "" || meta.URL == "" {
		return nil, nil, fmt.Errorf("request meta requires method and url")
	}
	if maxBodyMB > 0 && meta.BodyLen > maxBodyMB<<20 {
		return nil, nil, fmt.Errorf("request body %d bytes exceeds %d MB cap", meta.BodyLen, maxBodyMB)
	}

	var body []byte
	if meta.BodyLen > 0 {
		body = make([]byte, 0, meta.BodyLen)
		for int64(len(body)) < meta.BodyLen {
			mt, chunk, err := conn.ReadMessage()
			if err != nil {
				return nil, nil, err
			}
			if mt != websocket.BinaryMessage {
				return nil, nil, fmt.Errorf("expected binary body frame, got type %d", mt)
			}
			body = append(body, chunk...)
		}
		if int64(len(body)) != meta.BodyLen {
			return nil, nil, fmt.Errorf("body length mismatch: got %d, want %d", len(body), meta.BodyLen)
		}
	}

	req, err := http.NewRequest(meta.Method, meta.URL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid request: %w", err)
	}
	if meta.Headers != nil {
		req.Header = meta.Headers
	}
	return req, &meta, nil
}

// executeRequest runs one tunneled request against the session target.
// Per-request failures are reported in the response meta, not by failing
// the session.
func (tc *TunnelCore) executeRequest(ctx context.Context, client *http.Client, targetURL *url.URL, req *http.Request, meta *TunnelRequestMeta, maxBodyMB int64, rec *AccessRecord) (*TunnelResponseMeta, []byte, error) {
	respMeta := &TunnelResponseMeta{ID: meta.ID}

	// WebSocket-over-tunnel is not supported by the serial frame protocol.
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		respMeta.Status = http.StatusNotImplemented
		respMeta.Error = "websocket upgrade over tunnel is not supported"
		return respMeta, nil, errors.New(respMeta.Error)
	}

	// Rebase onto the session target; absolute URIs pointing elsewhere are
	// rejected so the tunnel can never be steered past its target.
	req.URL.Scheme = targetURL.Scheme
	if req.URL.Host != "" && !strings.EqualFold(req.URL.Host, targetURL.Host) {
		respMeta.Status = http.StatusForbidden
		respMeta.Error = "absolute URI host does not match the session target"
		rec.Status = respMeta.Status
		return respMeta, nil, errors.New(respMeta.Error)
	}
	req.URL.Host = targetURL.Host
	req.Host = targetURL.Host
	req.RequestURI = ""
	req.Header = stripHopByHop(req.Header)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		respMeta.Status = http.StatusBadGateway
		respMeta.Error = err.Error()
		rec.Status = respMeta.Status
		return respMeta, nil, err
	}
	defer resp.Body.Close()

	limit := int64(-1)
	if maxBodyMB > 0 {
		limit = maxBodyMB << 20
	}
	var reader io.Reader = resp.Body
	if limit >= 0 {
		reader = io.LimitReader(resp.Body, limit+1)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		respMeta.Status = http.StatusBadGateway
		respMeta.Error = "read upstream body: " + err.Error()
		rec.Status = respMeta.Status
		return respMeta, nil, err
	}
	if limit >= 0 && int64(len(body)) > limit {
		respMeta.Status = http.StatusBadGateway
		respMeta.Error = fmt.Sprintf("upstream body exceeds %d MB cap", maxBodyMB)
		rec.Status = respMeta.Status
		return respMeta, nil, errors.New(respMeta.Error)
	}

	respMeta.Status = resp.StatusCode
	respMeta.Headers = resp.Header
	respMeta.BodyLen = int64(len(body))
	rec.Status = resp.StatusCode
	return respMeta, body, nil
}

// writeResponse sends one response meta frame plus chunked body frames.
func (tc *TunnelCore) writeResponse(conn *websocket.Conn, meta *TunnelResponseMeta, body []byte) error {
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return err
	}
	for offset := 0; offset < len(body); offset += tunnelFrameChunk {
		end := offset + tunnelFrameChunk
		if end > len(body) {
			end = len(body)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, body[offset:end]); err != nil {
			return err
		}
	}
	return nil
}

// record fires the access hook for one tunneled request.
func (tc *TunnelCore) record(rec AccessRecord) {
	if tc.hook != nil {
		tc.hook(rec)
	}
}

// sessionEndReason maps context state to a SessionStats end reason.
func sessionEndReason(sessionCtx, taskCtx context.Context) string {
	if taskCtx.Err() != nil {
		return "shutdown" // task timeout / agent stop
	}
	if sessionCtx.Err() == context.DeadlineExceeded {
		return "max_duration"
	}
	return "closed"
}

// upstreamAddr identifies the tunnel peer for audit purposes.
func upstreamAddr(conn *websocket.Conn) string {
	if addr := conn.RemoteAddr(); addr != nil {
		return addr.String()
	}
	return "upstream"
}

// hopHeaders are connection-scoped and must not be forwarded.
var hopHeaders = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
	"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// stripHopByHop returns a copy of h without connection-scoped headers,
// including the extra tokens named by the Connection header.
func stripHopByHop(h http.Header) http.Header {
	out := make(http.Header, len(h))
	extra := h.Values("Connection")
	for k, vv := range h {
		skip := false
		for _, hb := range hopHeaders {
			if strings.EqualFold(k, hb) {
				skip = true
				break
			}
		}
		if !skip {
			for _, connVal := range extra {
				for _, token := range strings.Split(connVal, ",") {
					if strings.EqualFold(strings.TrimSpace(token), k) {
						skip = true
						break
					}
				}
				if skip {
					break
				}
			}
		}
		if skip {
			continue
		}
		out[k] = append([]string(nil), vv...)
	}
	return out
}

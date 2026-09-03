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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/casuallc/vigil/poll"
	"github.com/gorilla/websocket"
)

// tunnelHarness wires a fake upstream WS server to a TunnelCore session.
type tunnelHarness struct {
	core     *TunnelCore
	conn     *websocket.Conn // upstream side
	statsCh  chan poll.SessionStats
	backend  *httptest.Server
	upstream *httptest.Server
}

// newTunnelHarness starts a backend and a fake upstream; the core dials
// the upstream like the poll executor would.
func newTunnelHarness(t *testing.T, cfg TunnelConfig, handler http.HandlerFunc) *tunnelHarness {
	t.Helper()
	cfg.Enabled = true
	if len(cfg.AllowedTargets) == 0 {
		cfg.AllowedTargets = []string{"127.0.0.1"} // httptest backends bind loopback
	}

	h := &tunnelHarness{statsCh: make(chan poll.SessionStats, 1)}
	h.backend = httptest.NewServer(handler)
	t.Cleanup(h.backend.Close)

	h.core = newTunnelCore(cfg, nil)

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	h.upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.conn = conn
	}))
	t.Cleanup(h.upstream.Close)

	// Dial like executor.execProxySession does.
	wsURL := "ws" + strings.TrimPrefix(h.upstream.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial upstream: %v", err)
	}
	go func() {
		stats, err := h.core.RunProxySession(context.Background(), conn, h.backend.URL, poll.SessionLimits{})
		if err != nil {
			stats.EndReason = "error: " + err.Error()
		}
		h.statsCh <- stats
	}()
	// Wait for the upstream side to be attached.
	deadline := time.Now().Add(2 * time.Second)
	for h.conn == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if h.conn == nil {
		t.Fatal("upstream side never connected")
	}
	return h
}

// roundTrip sends one request over the tunnel and reads the response.
func (h *tunnelHarness) roundTrip(t *testing.T, meta TunnelRequestMeta, body []byte) (*TunnelResponseMeta, []byte) {
	t.Helper()
	meta.BodyLen = int64(len(body))
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if err := h.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	for offset := 0; offset < len(body); offset += tunnelFrameChunk {
		end := offset + tunnelFrameChunk
		if end > len(body) {
			end = len(body)
		}
		if err := h.conn.WriteMessage(websocket.BinaryMessage, body[offset:end]); err != nil {
			t.Fatalf("write body frame: %v", err)
		}
	}

	_ = h.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	mt, respRaw, err := h.conn.ReadMessage()
	if err != nil {
		t.Fatalf("read response meta: %v", err)
	}
	if mt != websocket.TextMessage {
		t.Fatalf("response meta frame type = %d, want text", mt)
	}
	var respMeta TunnelResponseMeta
	if err := json.Unmarshal(respRaw, &respMeta); err != nil {
		t.Fatalf("unmarshal response meta: %v", err)
	}
	var respBody []byte
	for int64(len(respBody)) < respMeta.BodyLen {
		_, chunk, err := h.conn.ReadMessage()
		if err != nil {
			t.Fatalf("read body frame: %v", err)
		}
		respBody = append(respBody, chunk...)
	}
	return &respMeta, respBody
}

func (h *tunnelHarness) close(t *testing.T) poll.SessionStats {
	t.Helper()
	h.conn.Close()
	select {
	case stats := <-h.statsCh:
		return stats
	case <-time.After(3 * time.Second):
		t.Fatal("session did not end after close")
		return poll.SessionStats{}
	}
}

func TestTunnelRequestResponse(t *testing.T) {
	h := newTunnelHarness(t, TunnelConfig{}, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "echo:%s %s body=%d", r.Method, r.URL.Path, len(body))
	})

	meta, body := h.roundTrip(t, TunnelRequestMeta{
		ID: "1", Method: "POST", URL: "/api/data",
		Headers: http.Header{"Content-Type": {"text/plain"}},
	}, []byte("payload"))
	if meta.Status != 200 || string(body) != "echo:POST /api/data body=7" {
		t.Fatalf("got status=%d body=%q", meta.Status, body)
	}

	stats := h.close(t)
	if stats.Requests != 1 || stats.EndReason != "closed" {
		t.Fatalf("stats = %+v", stats)
	}
	if stats.BytesOut == 0 || stats.BytesIn != 7 {
		t.Fatalf("byte counters = %+v", stats)
	}
}

func TestTunnelLargeBodyFraming(t *testing.T) {
	size := tunnelFrameChunk*2 + 12345 // forces 3 frames each way
	h := newTunnelHarness(t, TunnelConfig{}, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Len", fmt.Sprintf("%d", len(body)))
		w.Write(make([]byte, size))
	})

	meta, body := h.roundTrip(t, TunnelRequestMeta{
		ID: "big", Method: "POST", URL: "/big",
	}, make([]byte, size))
	if meta.Status != 200 {
		t.Fatalf("status = %d err=%q", meta.Status, meta.Error)
	}
	if len(body) != size {
		t.Fatalf("response body = %d bytes, want %d", len(body), size)
	}
	if meta.Headers.Get("X-Len") != fmt.Sprintf("%d", size) {
		t.Fatalf("backend saw %s request bytes, want %d", meta.Headers.Get("X-Len"), size)
	}
	h.close(t)
}

func TestTunnelRejectsAbsoluteURIHostMismatch(t *testing.T) {
	h := newTunnelHarness(t, TunnelConfig{}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend must not be reached by a foreign absolute URI")
	})

	meta, _ := h.roundTrip(t, TunnelRequestMeta{
		ID: "evil", Method: "GET", URL: "http://169.254.169.254/latest/meta-data",
	}, nil)
	if meta.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", meta.Status)
	}
	h.close(t)
}

func TestTunnelDisabledByDefault(t *testing.T) {
	// TunnelConfig zero value: enabled=false / empty allowed_targets.
	core := newTunnelCore(TunnelConfig{}, nil)
	if core.Enabled() {
		t.Fatal("tunnel must be disabled without explicit configuration")
	}
	cfg := TunnelConfig{Enabled: true} // enabled but no allowed targets
	core = newTunnelCore(cfg, nil)
	if core.Enabled() {
		t.Fatal("tunnel must stay disabled with empty allowed_targets")
	}
}

func TestTunnelTargetPolicy(t *testing.T) {
	core := newTunnelCore(TunnelConfig{
		Enabled:        true,
		AllowedTargets: []string{"127.0.0.1", "10.0.0.0/8"},
	}, nil)
	cases := map[string]bool{
		"http://127.0.0.1:9000": true,
		"http://10.1.2.3:80":    true,
		"http://192.168.1.1":    false,
		"http://169.254.169.254": false, // always denied
	}
	for target, want := range cases {
		if got := core.TargetAllowed(strings.TrimPrefix(target, "http://")); got != want {
			t.Errorf("TargetAllowed(%q) = %v, want %v", target, got, want)
		}
	}
}

func TestTunnelTargetRejected(t *testing.T) {
	var denied atomic.Int32
	cfg := TunnelConfig{Enabled: true, AllowedTargets: []string{"127.0.0.1"}}
	core := newTunnelCore(cfg, func(rec AccessRecord) {
		if rec.Denied {
			denied.Add(1)
		}
	})
	if !core.Enabled() {
		t.Fatal("tunnel should be enabled")
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer backend.Close()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
	}))
	defer upstream.Close()

	wsURL := "ws" + strings.TrimPrefix(upstream.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Backend listens on 127.0.0.1 but the policy only allows exact
	// host "127.0.0.1"... use a disallowed target instead.
	_, err = core.RunProxySession(context.Background(), conn, "http://192.0.2.1:9999", poll.SessionLimits{})
	if err == nil {
		t.Fatal("session to a disallowed target must fail")
	}
	if denied.Load() != 1 {
		t.Fatalf("denied hook fired %d times, want 1", denied.Load())
	}
}

func TestTunnelMaxDurationCap(t *testing.T) {
	h := newTunnelHarness(t, TunnelConfig{MaxDurationSec: 1}, func(w http.ResponseWriter, r *http.Request) {})
	h.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	select {
	case stats := <-h.statsCh:
		if stats.EndReason != "max_duration" {
			t.Fatalf("end_reason = %q, want max_duration", stats.EndReason)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session did not end at the duration cap")
	}
}

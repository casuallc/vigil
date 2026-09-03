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
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// AuthFunc validates proxy client credentials (from the Proxy-Authorization
// header). The api layer wires it to the super admin account from
// config.yaml; a nil AuthFunc fails closed.
type AuthFunc func(username, password string) bool

// serveForward handles one request on a forward-mode instance: classic
// proxy semantics where the client names the destination (absolute URI for
// plain HTTP, CONNECT for anything else, typically HTTPS).
func (i *Instance) serveForward(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := AccessRecord{
		Instance: i.cfg.Name,
		ClientIP: clientIP(r),
		Method:   r.Method,
		Path:     r.URL.String(),
		Via:      "forward",
	}

	if !i.checkProxyAuth(w, r, &rec, start) {
		return
	}

	if r.Method == http.MethodConnect {
		rec.Path = r.Host
		i.serveConnect(w, r, &rec, start)
		return
	}

	// Plain HTTP forward: the request line must be absolute-form.
	if !r.URL.IsAbs() || r.URL.Host == "" {
		rec.Status = http.StatusBadRequest
		http.Error(w, "Bad Request: forward proxy requires an absolute URI", http.StatusBadRequest)
		i.record(rec, start)
		return
	}
	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		rec.Status = http.StatusBadRequest
		http.Error(w, "Bad Request: unsupported scheme "+r.URL.Scheme, http.StatusBadRequest)
		i.record(rec, start)
		return
	}

	if !i.wl.Allowed(r.URL.Host) && !i.wl.Allowed(r.URL.Hostname()) {
		rec.Denied = true
		rec.Status = http.StatusForbidden
		http.Error(w, "Forbidden: destination rejected by whitelist policy", http.StatusForbidden)
		i.record(rec, start)
		return
	}

	if i.cfg.MaxBodyMB > 0 && r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, i.cfg.MaxBodyMB<<20)
	}

	i.stats.requests.Add(1)
	i.stats.activeConn.Add(1)
	defer i.stats.activeConn.Add(-1)
	if r.ContentLength > 0 {
		i.stats.bytesIn.Add(r.ContentLength)
	}

	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	outReq.Header = stripHopByHop(outReq.Header)
	outReq.Host = "" // derived from URL by the transport

	resp, err := i.transport.RoundTrip(outReq)
	if err != nil {
		i.stats.upstreamErr.Add(1)
		rec.Status = http.StatusBadGateway
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
		i.record(rec, start)
		return
	}
	defer resp.Body.Close()

	for k, vv := range stripHopByHop(resp.Header) {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	cw := &countingWriter{ResponseWriter: w, status: http.StatusOK}
	cw.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(cw, resp.Body)

	rec.Status = cw.status
	rec.BytesOut = int(cw.n.Load())
	i.record(rec, start)
}

// serveConnect establishes a raw TCP tunnel: dial the client-named
// destination (whitelist-gated, post-dial IP re-checked), answer 200, then
// relay bytes in both directions until either side closes.
func (i *Instance) serveConnect(w http.ResponseWriter, r *http.Request, rec *AccessRecord, start time.Time) {
	target := r.Host
	if _, _, err := net.SplitHostPort(target); err != nil {
		target = net.JoinHostPort(target, "443")
	}

	if !i.wl.Allowed(target) {
		rec.Denied = true
		rec.Status = http.StatusForbidden
		http.Error(w, "Forbidden: destination rejected by whitelist policy", http.StatusForbidden)
		i.record(*rec, start)
		return
	}

	dial := i.wl.CheckingDialer(nil)
	upstream, err := dial(r.Context(), "tcp", target)
	if err != nil {
		i.stats.upstreamErr.Add(1)
		rec.Status = http.StatusBadGateway
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
		i.record(*rec, start)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		upstream.Close()
		rec.Status = http.StatusInternalServerError
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		i.record(*rec, start)
		return
	}
	client, rw, err := hj.Hijack()
	if err != nil {
		upstream.Close()
		rec.Status = http.StatusInternalServerError
		i.record(*rec, start)
		return
	}
	if _, err := rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err == nil {
		err = rw.Flush()
	}
	if err != nil {
		client.Close()
		upstream.Close()
		rec.Status = http.StatusBadGateway
		i.record(*rec, start)
		return
	}

	i.stats.requests.Add(1)
	i.stats.activeConn.Add(1)
	defer i.stats.activeConn.Add(-1)

	rec.Status = http.StatusOK
	var in, out atomic.Int64
	done := make(chan struct{}, 2)
	relay := func(dst, src net.Conn, counter *atomic.Int64) {
		n, _ := io.Copy(dst, src)
		counter.Add(n)
		// Unblock the opposite direction.
		_ = dst.Close()
		_ = src.Close()
		done <- struct{}{}
	}
	go relay(upstream, client, &in)
	go relay(client, upstream, &out)
	<-done
	<-done

	i.stats.bytesIn.Add(in.Load())
	rec.BytesOut = int(out.Load())
	i.record(*rec, start)
}

// checkProxyAuth validates the Proxy-Authorization header against the
// instance AuthFunc. On failure it answers 407 and records the request.
func (i *Instance) checkProxyAuth(w http.ResponseWriter, r *http.Request, rec *AccessRecord, start time.Time) bool {
	user, pass, ok := parseProxyAuth(r)
	if ok && i.auth != nil && i.auth(user, pass) {
		return true
	}
	rec.Status = http.StatusProxyAuthRequired
	w.Header().Set("Proxy-Authenticate", `Basic realm="vigil"`)
	http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
	i.record(*rec, start)
	return false
}

// parseProxyAuth extracts Basic credentials from the Proxy-Authorization
// header (the proxy variant of r.BasicAuth, which reads Authorization).
func parseProxyAuth(r *http.Request) (username, password string, ok bool) {
	h := r.Header.Get("Proxy-Authorization")
	scheme, encoded, found := strings.Cut(h, " ")
	if !found || !strings.EqualFold(scheme, "Basic") {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", "", false
	}
	user, pass, found := strings.Cut(string(raw), ":")
	if !found {
		return "", "", false
	}
	return user, pass, true
}

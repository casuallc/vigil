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
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Instance lifecycle states.
const (
	StateStopped  = "stopped"
	StateStarting = "starting"
	StateRunning  = "running"
	StateError    = "error"
)

// state codes stored in the atomic state field.
const (
	stateStopped int32 = iota
	stateStarting
	stateRunning
	stateError
)

func stateString(code int32) string {
	switch code {
	case stateStarting:
		return StateStarting
	case stateRunning:
		return StateRunning
	case stateError:
		return StateError
	default:
		return StateStopped
	}
}

// Stats is a point-in-time snapshot of an instance's counters.
type Stats struct {
	Requests    int64     `json:"requests"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	ActiveConn  int64     `json:"active_conn"`
	UpstreamErr int64     `json:"upstream_err"`
	StartedAt   time.Time `json:"started_at,omitempty"`
}

// counters holds the live atomic counterparts of Stats.
type counters struct {
	requests    atomic.Int64
	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	activeConn  atomic.Int64
	upstreamErr atomic.Int64
}

// InstanceStatus describes an instance for the API.
type InstanceStatus struct {
	Name      string         `json:"name"`
	Origin    string         `json:"origin"` // "config" | "api"
	State     string         `json:"state"`
	Config    InstanceConfig `json:"config"`
	Stats     Stats          `json:"stats"`
	LastError string         `json:"last_error,omitempty"`
}

// Instance is one reverse proxy listener. It owns a dedicated http.Server
// and never passes through the API server's logging middleware, which would
// buffer whole request/response bodies.
type Instance struct {
	cfg    InstanceConfig
	origin string

	wl            *Whitelist
	targetURL     *url.URL
	targetAllowed bool
	rp            *httputil.ReverseProxy
	hook          AccessHook

	mu   sync.Mutex // guards srv/ln across Start/Stop
	srv  *http.Server
	ln   net.Listener
	done chan struct{}

	state     atomic.Int32
	lastErr   atomic.Value // string
	startedAt atomic.Value // time.Time
	stats     counters
}

// NewInstance builds an instance from its config. The whitelist always
// implicitly contains the target host itself; cfg.Whitelist can broaden it.
func NewInstance(cfg InstanceConfig, origin string, hook AccessHook) (*Instance, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("proxy: instance name is required")
	}
	if cfg.Listen == "" {
		return nil, fmt.Errorf("proxy: instance %q: listen address is required", cfg.Name)
	}
	targetURL, err := url.Parse(cfg.Target)
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		return nil, fmt.Errorf("proxy: instance %q: invalid target %q (want http(s)://host[:port])", cfg.Name, cfg.Target)
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, fmt.Errorf("proxy: instance %q: unsupported target scheme %q", cfg.Name, targetURL.Scheme)
	}
	if cfg.TLS.Enabled && (cfg.TLS.CertPath == "" || cfg.TLS.KeyPath == "") {
		return nil, fmt.Errorf("proxy: instance %q: tls.enabled requires cert_path and key_path", cfg.Name)
	}

	entries := append([]string{}, cfg.Whitelist...)
	entries = append(entries, targetURL.Hostname())
	wl, err := ParseWhitelist(entries, cfg.AllowPrivate)
	if err != nil {
		return nil, err
	}

	i := &Instance{
		cfg:       cfg,
		origin:    origin,
		wl:        wl,
		targetURL: targetURL,
		hook:      hook,
	}
	// The target is static, so the whitelist verdict is computed once.
	i.targetAllowed = wl.Allowed(targetURL.Host) || wl.Allowed(targetURL.Hostname())

	i.rp = httputil.NewSingleHostReverseProxy(targetURL)
	baseDirector := i.rp.Director
	i.rp.Director = func(r *http.Request) {
		baseDirector(r)
		for k, v := range cfg.HeaderSet {
			r.Header.Set(k, v)
		}
	}
	i.rp.Transport = newCheckingTransport(wl)
	i.rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		i.stats.upstreamErr.Add(1)
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
	}
	return i, nil
}

// newCheckingTransport returns a Transport whose dialer re-validates the
// connected IP against the whitelist (DNS rebinding mitigation).
func newCheckingTransport(wl *Whitelist) *http.Transport {
	return &http.Transport{
		Proxy:                 nil, // never honor proxy env vars
		DialContext:           wl.CheckingDialer(nil),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// ServeHTTP handles one proxied request: whitelist precheck, body limit,
// then the reverse proxy, recording an AccessRecord on the way out.
func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := AccessRecord{
		Instance: i.cfg.Name,
		ClientIP: clientIP(r),
		Method:   r.Method,
		Path:     r.URL.Path,
		Via:      "listen",
	}

	if !i.targetAllowed {
		rec.Denied = true
		rec.Status = http.StatusForbidden
		http.Error(w, "Forbidden: target rejected by whitelist policy", http.StatusForbidden)
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

	cw := &countingWriter{ResponseWriter: w, status: http.StatusOK}
	i.rp.ServeHTTP(cw, r)

	rec.Status = cw.status
	rec.BytesOut = int(cw.n.Load())
	i.record(rec, start)
}

// record finishes an AccessRecord: updates byte counters and fires the hook.
func (i *Instance) record(rec AccessRecord, start time.Time) {
	rec.DurationMs = time.Since(start).Milliseconds()
	i.stats.bytesOut.Add(int64(rec.BytesOut))
	if i.hook != nil {
		i.hook(rec)
	}
}

// Start binds the listener and serves in the background.
func (i *Instance) Start(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.state.Load() == stateRunning || i.state.Load() == stateStarting {
		return fmt.Errorf("proxy: instance %q is already running", i.cfg.Name)
	}

	ln, err := net.Listen("tcp", i.cfg.Listen)
	if err != nil {
		i.setError(err)
		return fmt.Errorf("proxy: instance %q: listen %s: %w", i.cfg.Name, i.cfg.Listen, err)
	}

	i.srv = &http.Server{
		Handler:           i,
		ReadHeaderTimeout: 30 * time.Second,
	}
	i.ln = ln
	i.done = make(chan struct{})
	i.state.Store(stateStarting)

	serve := func() error {
		if i.cfg.TLS.Enabled {
			return i.srv.ServeTLS(ln, i.cfg.TLS.CertPath, i.cfg.TLS.KeyPath)
		}
		return i.srv.Serve(ln)
	}
	go func() {
		defer close(i.done)
		if err := serve(); err != nil && err != http.ErrServerClosed {
			i.setError(fmt.Errorf("serve: %w", err))
		}
	}()

	i.state.Store(stateRunning)
	i.startedAt.Store(time.Now())
	return nil
}

// Stop gracefully shuts the listener down with a 5s grace period.
func (i *Instance) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.state.Load() == stateStopped {
		return nil
	}
	if i.srv == nil {
		i.state.Store(stateStopped)
		return nil
	}
	grace, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := i.srv.Shutdown(grace)
	if i.done != nil {
		<-i.done
	}
	i.srv = nil
	i.ln = nil
	i.state.Store(stateStopped)
	return err
}

// Status returns the current snapshot.
func (i *Instance) Status() InstanceStatus {
	st := InstanceStatus{
		Name:   i.cfg.Name,
		Origin: i.origin,
		State:  stateString(i.state.Load()),
		Config: i.cfg,
		Stats: Stats{
			Requests:    i.stats.requests.Load(),
			BytesIn:     i.stats.bytesIn.Load(),
			BytesOut:    i.stats.bytesOut.Load(),
			ActiveConn:  i.stats.activeConn.Load(),
			UpstreamErr: i.stats.upstreamErr.Load(),
		},
	}
	if t, ok := i.startedAt.Load().(time.Time); ok && i.state.Load() == stateRunning {
		st.Stats.StartedAt = t
	}
	if s, ok := i.lastErr.Load().(string); ok {
		st.LastError = s
	}
	return st
}

// Name returns the instance name.
func (i *Instance) Name() string { return i.cfg.Name }

// Origin returns where the instance was defined ("config" | "api").
func (i *Instance) Origin() string { return i.origin }

func (i *Instance) setError(err error) {
	i.state.Store(stateError)
	i.lastErr.Store(err.Error())
}

// clientIP prefers X-Forwarded-For when present.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// countingWriter counts response bytes and captures the status code while
// preserving Hijacker/Flusher so WebSocket upgrades and streaming survive.
type countingWriter struct {
	http.ResponseWriter
	status int
	n      atomic.Int64
}

func (w *countingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *countingWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.n.Add(int64(n))
	return n, err
}

// Hijack implements http.Hijacker for WebSocket upgrades. Bytes exchanged
// after the hijack are not counted.
func (w *countingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("proxy: response writer does not implement http.Hijacker")
	}
	return hj.Hijack()
}

// Flush implements http.Flusher for streaming responses.
func (w *countingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

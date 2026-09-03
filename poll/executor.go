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

package poll

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// InternalTokenHeader authenticates loopback requests towards bbx's own
// API server. The server generates the token at startup; it is never sent
// to upstreams.
const InternalTokenHeader = "X-Vigil-Internal-Token"

// maxAckResultBytes caps the result embedded in an ack payload; larger
// results are truncated and flagged.
const maxAckResultBytes = 64 * 1024

// Executor executes the four poll task types. The poll/ack control plane
// only ever carries small JSON; bulk data and long connections are always
// dialed outbound by bbx itself using addresses carried inside the task.
type Executor struct {
	opts        Options
	taskTimeout time.Duration

	// loopbackBase is the local API base URL, e.g. http://127.0.0.1:57575.
	loopbackBase string
	httpClient   *http.Client // loopback api calls
	pushClient   *http.Client // push_file / tail_file (ctx-driven, no timeout)
	wsDialer     *websocket.Dialer
	localDialer  *websocket.Dialer
}

func newExecutor(opts Options, taskTimeout time.Duration) *Executor {
	scheme := "http"
	if opts.LoopbackTLS {
		scheme = "https"
	}
	e := &Executor{
		opts:         opts,
		taskTimeout:  taskTimeout,
		loopbackBase: scheme + "://" + opts.LoopbackAddr,
		httpClient:   &http.Client{},
		pushClient:   &http.Client{},
		wsDialer:     &websocket.Dialer{HandshakeTimeout: 15 * time.Second},
		localDialer:  &websocket.Dialer{HandshakeTimeout: 10 * time.Second},
	}
	if opts.LoopbackTLS {
		// Loopback-only: the local cert is typically self-signed.
		insecure := &tls.Config{InsecureSkipVerify: true}
		e.httpClient.Transport = &http.Transport{TLSClientConfig: insecure}
		e.localDialer.TLSClientConfig = insecure
	}
	return e
}

// Execute dispatches on the action type (default "api").
func (e *Executor) Execute(ctx context.Context, t *Task) (json.RawMessage, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(t.Action, &head); err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}
	switch head.Type {
	case "", "api":
		return e.execAPI(ctx, t)
	case "push_file":
		return e.execPushFile(ctx, t)
	case "tail_file":
		return e.execTailFile(ctx, t)
	case "ws_bridge":
		return e.execWSBridge(ctx, t)
	default:
		return nil, fmt.Errorf("unknown action type %q", head.Type)
	}
}

// --- api: local API loopback call -----------------------------------------

type apiAction struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
}

func (e *Executor) execAPI(ctx context.Context, t *Task) (json.RawMessage, error) {
	var a apiAction
	if err := json.Unmarshal(t.Action, &a); err != nil {
		return nil, fmt.Errorf("invalid api action: %w", err)
	}
	if a.Path == "" || !strings.HasPrefix(a.Path, "/") {
		return nil, fmt.Errorf("api action path must be absolute, got %q", a.Path)
	}
	method := a.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if len(a.Body) > 0 {
		body = bytes.NewReader(a.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.loopbackBase+a.Path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set(InternalTokenHeader, e.opts.InternalToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loopback call failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAckResultBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read loopback response: %w", err)
	}
	truncated := len(data) > maxAckResultBytes
	if truncated {
		data = data[:maxAckResultBytes]
	}
	return json.Marshal(map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(data),
		"truncated":   truncated,
	})
}

// --- push_file: stream a local file to a task-provided URL -----------------

type pushFileAction struct {
	Path    string            `json:"path"`
	PushURL string            `json:"push_url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

func (e *Executor) execPushFile(ctx context.Context, t *Task) (json.RawMessage, error) {
	var a pushFileAction
	if err := json.Unmarshal(t.Action, &a); err != nil {
		return nil, fmt.Errorf("invalid push_file action: %w", err)
	}
	if a.Path == "" || a.PushURL == "" {
		return nil, fmt.Errorf("push_file requires path and push_url")
	}

	f, err := os.Open(a.Path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("%s is a directory", a.Path)
	}

	method := a.Method
	if method == "" {
		method = http.MethodPost
	}
	hash := sha256.New()
	req, err := http.NewRequestWithContext(ctx, method, a.PushURL, io.TeeReader(f, hash))
	if err != nil {
		return nil, err
	}
	req.ContentLength = fi.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	for k, v := range a.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.pushClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("push failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("push returned status %d", resp.StatusCode)
	}

	return json.Marshal(map[string]interface{}{
		"size":   fi.Size(),
		"sha256": hex.EncodeToString(hash.Sum(nil)),
	})
}

// --- tail_file: stream log lines to a task-provided URL --------------------

type tailFileAction struct {
	Path    string            `json:"path"`
	PushURL string            `json:"push_url"`
	Headers map[string]string `json:"headers"`
	Follow  bool              `json:"follow"`
	Lines   int               `json:"lines"`
}

// tailReadChunk bounds how many bytes are read when collecting the
// trailing lines and how much is forwarded per follow tick.
const tailReadChunk = 8 << 20

// pushResult carries the outcome of a streaming POST round trip.
type pushResult struct {
	status int
	err    error
}

func (e *Executor) execTailFile(ctx context.Context, t *Task) (json.RawMessage, error) {
	var a tailFileAction
	if err := json.Unmarshal(t.Action, &a); err != nil {
		return nil, fmt.Errorf("invalid tail_file action: %w", err)
	}
	if a.Path == "" || a.PushURL == "" {
		return nil, fmt.Errorf("tail_file requires path and push_url")
	}
	if a.Lines < 0 {
		a.Lines = 0
	}

	f, err := os.Open(a.Path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Collect the trailing `lines` lines as the initial burst.
	initial, offset, err := readLastLines(f, a.Lines)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.PushURL, pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	for k, v := range a.Headers {
		req.Header.Set(k, v)
	}

	respCh := make(chan pushResult, 1)
	go func() {
		resp, err := e.pushClient.Do(req)
		if err != nil {
			respCh <- pushResult{err: err}
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		respCh <- pushResult{status: resp.StatusCode}
	}()

	var lines, bytesSent int64
	endReason := "completed"

	write := func(p []byte) error {
		n, err := pw.Write(p)
		bytesSent += int64(n)
		lines += int64(bytes.Count(p, []byte("\n")))
		return err
	}

	if len(initial) > 0 {
		if err := write(initial); err != nil {
			endReason = "peer_closed"
		}
	}

	if endReason != "peer_closed" && a.Follow {
		endReason = e.followFile(ctx, f, offset, write, respCh)
	} else if !a.Follow {
		// Close the body so the upstream can finish the request.
		pw.Close()
	}

	// Wait for the HTTP round trip to settle.
	select {
	case r := <-respCh:
		if r.err != nil && endReason == "completed" {
			endReason = "peer_closed"
		}
		pw.Close()
	case <-time.After(10 * time.Second):
		pw.Close()
	}

	return json.Marshal(map[string]interface{}{
		"lines":      lines,
		"bytes":      bytesSent,
		"end_reason": endReason,
	})
}

// followFile forwards file growth to write until the context ends, the
// peer closes, or a read/write fails. It polls because no fsnotify-style
// dependency is available.
func (e *Executor) followFile(ctx context.Context, f *os.File, offset int64, write func([]byte) error, respCh <-chan pushResult) string {
	buf := make([]byte, 64*1024)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "timeout"
		case r := <-respCh:
			_ = r
			return "peer_closed"
		case <-ticker.C:
		}

		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return "read_error"
		}
		for {
			n, err := f.Read(buf)
			if n > 0 {
				offset += int64(n)
				if werr := write(buf[:n]); werr != nil {
					return "peer_closed"
				}
			}
			if err != nil {
				break // io.EOF: nothing new right now
			}
		}
	}
}

// readLastLines returns up to n trailing lines of f and the offset just
// after them. n <= 0 starts following from the current end of file.
func readLastLines(f *os.File, n int) ([]byte, int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	size := fi.Size()
	if n <= 0 || size == 0 {
		return nil, size, nil
	}

	start := int64(0)
	if size > tailReadChunk {
		start = size - tailReadChunk
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	if start > 0 {
		// Drop the first (likely partial) line.
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	}

	lines := bytes.Split(data, []byte("\n"))
	// A trailing newline produces an empty final element; drop it.
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := bytes.Join(lines, []byte("\n"))
	if len(out) > 0 {
		out = append(out, '\n')
	}
	return out, size, nil
}

// --- ws_bridge: outbound WS bridging to a local WS handler ------------------

type wsBridgeAction struct {
	ConnectURL string            `json:"connect_url"`
	Headers    map[string]string `json:"headers"`
	Local      struct {
		Path string `json:"path"`
	} `json:"local"`
}

func (e *Executor) execWSBridge(ctx context.Context, t *Task) (json.RawMessage, error) {
	var a wsBridgeAction
	if err := json.Unmarshal(t.Action, &a); err != nil {
		return nil, fmt.Errorf("invalid ws_bridge action: %w", err)
	}
	if a.ConnectURL == "" || a.Local.Path == "" {
		return nil, fmt.Errorf("ws_bridge requires connect_url and local.path")
	}

	// Dial the upstream (outbound, allowed by the network constraint).
	upHeader := http.Header{}
	for k, v := range a.Headers {
		upHeader.Set(k, v)
	}
	upstream, _, err := e.wsDialer.DialContext(ctx, a.ConnectURL, upHeader)
	if err != nil {
		return nil, fmt.Errorf("dial upstream ws: %w", err)
	}
	defer upstream.Close()

	// Dial the local handler over loopback.
	localURL := e.loopbackBase + a.Local.Path
	if strings.HasPrefix(localURL, "http") {
		localURL = "ws" + strings.TrimPrefix(localURL, "http")
	}
	localHeader := http.Header{}
	localHeader.Set(InternalTokenHeader, e.opts.InternalToken)
	local, _, err := e.localDialer.DialContext(ctx, localURL, localHeader)
	if err != nil {
		return nil, fmt.Errorf("dial local ws: %w", err)
	}
	defer local.Close()

	start := time.Now()
	endReason := e.bridgeWS(ctx, upstream, local)

	return json.Marshal(map[string]interface{}{
		"duration_ms": time.Since(start).Milliseconds(),
		"end_reason":  endReason,
	})
}

// bridgeWS copies frames in both directions until either side closes or
// the context ends.
func (e *Executor) bridgeWS(ctx context.Context, a, b *websocket.Conn) string {
	endReason := "closed"
	var once sync.Once
	done := make(chan struct{})
	stop := func() {
		once.Do(func() {
			a.Close()
			b.Close()
			close(done)
		})
	}

	copyMsgs := func(dst, src *websocket.Conn) {
		defer stop()
		for {
			msgType, payload, err := src.ReadMessage()
			if err != nil {
				return
			}
			if err := dst.WriteMessage(msgType, payload); err != nil {
				return
			}
		}
	}
	go copyMsgs(a, b)
	go copyMsgs(b, a)

	select {
	case <-ctx.Done():
		endReason = "timeout"
		stop()
	case <-done:
	}
	return endReason
}

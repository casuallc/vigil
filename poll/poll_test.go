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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// --- fakes -----------------------------------------------------------------

type recordedAck struct {
	mu       sync.Mutex
	payloads []AckPayload
}

func (r *recordedAck) add(p AckPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payloads = append(r.payloads, p)
}

func (r *recordedAck) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.payloads)
}

type fakeAcker struct{ rec *recordedAck }

func (a *fakeAcker) Ack(ctx context.Context, p AckPayload) error {
	a.rec.add(p)
	return nil
}

// fakeExec tracks per-topic concurrency to verify the serial/parallel model.
type fakeExec struct {
	delay   time.Duration
	mu      sync.Mutex
	running map[string]int
	maxPer  map[string]int
	total   int
	maxAll  int
	calls   int
}

func newFakeExec(delay time.Duration) *fakeExec {
	return &fakeExec{delay: delay, running: map[string]int{}, maxPer: map[string]int{}}
}

func (f *fakeExec) Execute(ctx context.Context, t *Task) (json.RawMessage, error) {
	f.mu.Lock()
	f.calls++
	f.running[t.Topic]++
	f.total++
	if f.running[t.Topic] > f.maxPer[t.Topic] {
		f.maxPer[t.Topic] = f.running[t.Topic]
	}
	if f.total > f.maxAll {
		f.maxAll = f.total
	}
	f.mu.Unlock()

	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
	}

	f.mu.Lock()
	f.running[t.Topic]--
	f.total--
	f.mu.Unlock()
	return json.RawMessage(`{"ok":true}`), nil
}

func testDefaults() *Defaults {
	return &Defaults{
		TaskTimeout:  Duration(5 * time.Second),
		MaxTopics:    8,
		QueueBuffer:  16,
		TopicIdleTTL: Duration(time.Minute),
	}
}

func dispatchNTasks(d *Dispatcher, ack *recordedAck, topic string, n int, prefix string) {
	for i := 0; i < n; i++ {
		t := &Task{
			ID:    fmt.Sprintf("%s-%d", prefix, i),
			Topic: topic,
			acker: &fakeAcker{rec: ack},
		}
		if err := d.Dispatch(t); err != nil {
			panic(err)
		}
	}
}

// --- dispatcher tests --------------------------------------------------------

func TestDispatcherTopicSerialAndParallel(t *testing.T) {
	exec := newFakeExec(30 * time.Millisecond)
	d := newDispatcher(exec, testDefaults())
	defer d.Shutdown()

	ack := &recordedAck{}
	dispatchNTasks(d, ack, "docker", 4, "d")
	dispatchNTasks(d, ack, "file", 4, "f")

	deadline := time.Now().Add(5 * time.Second)
	for ack.count() < 8 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if ack.count() != 8 {
		t.Fatalf("expected 8 acks, got %d", ack.count())
	}
	for topic, max := range exec.maxPer {
		if max > 1 {
			t.Errorf("topic %s ran %d tasks concurrently, want serial", topic, max)
		}
	}
	if exec.maxAll < 2 {
		t.Errorf("expected cross-topic parallelism, max concurrent = %d", exec.maxAll)
	}
	for _, p := range ack.payloads {
		if p.Status != StatusSuccess || p.ExitCode != 0 {
			t.Errorf("task %s unexpected payload: %+v", p.ID, p)
		}
	}
}

func TestDispatcherMaxTopicsFallback(t *testing.T) {
	exec := newFakeExec(time.Millisecond)
	defs := testDefaults()
	defs.MaxTopics = 1
	d := newDispatcher(exec, defs)
	defer d.Shutdown()

	ack := &recordedAck{}
	dispatchNTasks(d, ack, "a", 1, "a")
	dispatchNTasks(d, ack, "b", 1, "b")
	dispatchNTasks(d, ack, "c", 1, "c")

	deadline := time.Now().Add(5 * time.Second)
	for ack.count() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if ack.count() != 3 {
		t.Fatalf("expected all 3 tasks to run, got %d acks", ack.count())
	}

	d.mu.Lock()
	n := len(d.queues)
	_, hasDefault := d.queues["default"]
	d.mu.Unlock()
	if n != 2 || !hasDefault {
		t.Errorf("expected queues {a, default}, got %d queues (default=%v)", n, hasDefault)
	}
}

func TestDispatcherShutdownNacksQueuedTasks(t *testing.T) {
	exec := newFakeExec(300 * time.Millisecond) // first task blocks the worker
	defs := testDefaults()
	defs.QueueBuffer = 4
	d := newDispatcher(exec, defs)

	ack := &recordedAck{}
	dispatchNTasks(d, ack, "t", 3, "t")
	time.Sleep(20 * time.Millisecond) // let the worker pick up task 0

	done := make(chan struct{})
	go func() { d.Shutdown(); close(done) }()

	// drainTimeout is 10s; the blocked task finishes in 300ms, then the
	// remaining queued tasks are executed within the drain window.
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("shutdown took too long")
	}
	if ack.count() != 3 {
		t.Fatalf("expected 3 acks after drain, got %d", ack.count())
	}
}

// --- executor tests ----------------------------------------------------------

func stripHTTP(u string) string { return strings.TrimPrefix(u, "http://") }

func TestExecAPI(t *testing.T) {
	var gotToken string
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get(InternalTokenHeader)
		w.Write([]byte(`{"hello":"world"}`))
	}))
	defer local.Close()

	e := newExecutor(Options{InternalToken: "tok", LoopbackAddr: stripHTTP(local.URL)}, 5*time.Second)
	task := &Task{Action: json.RawMessage(`{"type":"api","method":"GET","path":"/x"}`)}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("execAPI failed: %v", err)
	}
	if gotToken != "tok" {
		t.Errorf("loopback request missing internal token, got %q", gotToken)
	}
	var out struct {
		StatusCode int    `json:"status_code"`
		Body       string `json:"body"`
		Truncated  bool   `json:"truncated"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatalf("bad result: %v", err)
	}
	if out.StatusCode != 200 || out.Body != `{"hello":"world"}` || out.Truncated {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestExecPushFile(t *testing.T) {
	content := []byte("push me to the upstream service\n")
	dir := t.TempDir()
	path := filepath.Join(dir, "pkg.bin")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	var got []byte
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
	}))
	defer receiver.Close()

	e := newExecutor(Options{LoopbackAddr: "127.0.0.1:1"}, 5*time.Second)
	task := &Task{Action: json.RawMessage(fmt.Sprintf(
		`{"type":"push_file","path":%q,"push_url":%q}`, path, receiver.URL))}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("execPushFile failed: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("receiver got %q, want %q", got, content)
	}
	var out struct {
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if out.Size != int64(len(content)) || out.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestExecPullFile(t *testing.T) {
	content := []byte("deployment package payload\n")
	sum := sha256.Sum256(content)

	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer src.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "pkg.tar.gz") // parent dir does not exist yet

	e := newExecutor(Options{LoopbackAddr: "127.0.0.1:1"}, 5*time.Second)
	task := &Task{Action: json.RawMessage(fmt.Sprintf(
		`{"type":"pull_file","url":%q,"path":%q,"sha256":%q}`,
		src.URL, path, hex.EncodeToString(sum[:])))}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("execPullFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("target file missing: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("target file got %q, want %q", got, content)
	}
	var out struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatal(err)
	}
	if out.Path != path || out.Size != int64(len(content)) || out.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestExecPullFileSHA256Mismatch(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("tampered"))
	}))
	defer src.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "pkg.tar.gz")

	e := newExecutor(Options{LoopbackAddr: "127.0.0.1:1"}, 5*time.Second)
	task := &Task{Action: json.RawMessage(fmt.Sprintf(
		`{"type":"pull_file","url":%q,"path":%q,"sha256":%q}`,
		src.URL, path, strings.Repeat("0", 64)))}
	if _, err := e.Execute(context.Background(), task); err == nil ||
		!strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected sha256 mismatch error, got %v", err)
	}

	// Neither the target nor the temp file may be left behind.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("target file should not exist after failed verification")
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".pull-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left behind: %v", leftovers)
	}
}

func TestExecTailFileNoFollow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	lines := []string{"l1", "l2", "l3", "l4", "l5"}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var got []byte
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
	}))
	defer receiver.Close()

	e := newExecutor(Options{LoopbackAddr: "127.0.0.1:1"}, 5*time.Second)
	task := &Task{Action: json.RawMessage(fmt.Sprintf(
		`{"type":"tail_file","path":%q,"push_url":%q,"follow":false,"lines":2}`, path, receiver.URL))}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("execTailFile failed: %v", err)
	}
	if string(got) != "l4\nl5\n" {
		t.Errorf("receiver got %q, want %q", got, "l4\nl5\n")
	}
	var out struct {
		Lines     int64  `json:"lines"`
		EndReason string `json:"end_reason"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatal(err)
	}
	if out.Lines != 2 || out.EndReason != "completed" {
		t.Errorf("unexpected result: %+v", out)
	}
}

var testWSUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func TestExecWSBridge(t *testing.T) {
	// Local WS handler: sends one message, then waits for the echo.
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		if err := ws.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
			return
		}
		// Keep reading until the bridge closes us.
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer local.Close()

	// Upstream WS endpoint: echoes whatever it receives.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			mt, payload, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if err := ws.WriteMessage(mt, payload); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()

	e := newExecutor(Options{InternalToken: "tok", LoopbackAddr: stripHTTP(local.URL)}, 5*time.Second)
	action := fmt.Sprintf(`{"type":"ws_bridge","connect_url":%q,"local":{"path":"/ws"}}`,
		"ws"+strings.TrimPrefix(upstream.URL, "http"))
	task := &Task{Action: json.RawMessage(action)}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := e.Execute(ctx, task)
	if err != nil {
		t.Fatalf("execWSBridge failed: %v", err)
	}
	var out struct {
		EndReason string `json:"end_reason"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		t.Fatal(err)
	}
	if out.EndReason != "timeout" {
		t.Errorf("unexpected end_reason: %q", out.EndReason)
	}
}

// --- end-to-end poller test with a fake upstream -----------------------------

// fakeUpstream serves /poll from a task channel and records /ack payloads.
type fakeUpstream struct {
	tasks chan []Task
	acks  *recordedAck
	srv   *httptest.Server
}

func newFakeUpstream() *fakeUpstream {
	f := &fakeUpstream{tasks: make(chan []Task, 8), acks: &recordedAck{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/poll", func(w http.ResponseWriter, r *http.Request) {
		select {
		case tasks := <-f.tasks:
			json.NewEncoder(w).Encode(pollResponse{Tasks: tasks})
		default:
			json.NewEncoder(w).Encode(pollResponse{})
		}
	})
	mux.HandleFunc("/ack", func(w http.ResponseWriter, r *http.Request) {
		var p AckPayload
		json.NewDecoder(r.Body).Decode(&p)
		f.acks.add(p)
	})
	f.srv = httptest.NewServer(mux)
	return f
}

func waitForAcks(rec *recordedAck, n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if rec.count() >= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestAgentEndToEnd(t *testing.T) {
	upstream := newFakeUpstream()
	defer upstream.srv.Close()

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	}))
	defer local.Close()

	cfg := &PollConfig{
		Enabled: true,
		AgentID: "test-agent",
		Defaults: Defaults{
			LongPollWait:         Duration(50 * time.Millisecond),
			BusyInterval:         Duration(10 * time.Millisecond),
			IdleBackoffMax:       Duration(50 * time.Millisecond),
			BusyToIdleEmptyPolls: 2,
			TaskTimeout:          Duration(5 * time.Second),
			MaxTopics:            4,
			QueueBuffer:          8,
			TopicIdleTTL:         Duration(time.Minute),
		},
		Upstreams: []UpstreamConfig{{
			Name:      "fake",
			Endpoint:  upstream.srv.URL,
			AllowHTTP: true,
		}},
	}

	agent, err := NewAgent(cfg, Options{InternalToken: "tok", LoopbackAddr: stripHTTP(local.URL)})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("agent should not be nil when enabled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agent.Start(ctx)
	defer agent.Stop()

	// Feed two tasks on different topics and one duplicate of the first.
	upstream.tasks <- []Task{
		{ID: "t1", Topic: "a", Action: json.RawMessage(`{"type":"api","path":"/"}`)},
		{ID: "t2", Topic: "b", Action: json.RawMessage(`{"type":"api","path":"/"}`)},
	}
	if !waitForAcks(upstream.acks, 2, 5*time.Second) {
		t.Fatalf("expected 2 acks, got %d", upstream.acks.count())
	}

	// Re-dispatch t1: the dedup cache must answer it without re-executing.
	upstream.tasks <- []Task{
		{ID: "t1", Topic: "a", Action: json.RawMessage(`{"type":"api","path":"/"}`)},
	}
	if !waitForAcks(upstream.acks, 3, 5*time.Second) {
		t.Fatalf("expected dedup re-ack, got %d acks", upstream.acks.count())
	}

	upstream.acks.mu.Lock()
	defer upstream.acks.mu.Unlock()
	statuses := map[string]int{}
	for _, p := range upstream.acks.payloads {
		statuses[p.ID+":"+p.Status]++
	}
	if statuses["t1:success"] != 2 || statuses["t2:success"] != 1 {
		t.Errorf("unexpected acks: %v", statuses)
	}
	for _, p := range upstream.acks.payloads {
		if !strings.Contains(string(p.Result), "pong") {
			t.Errorf("task %s result missing loopback body: %s", p.ID, p.Result)
		}
	}
}

// --- config tests ------------------------------------------------------------

func TestConfigValidation(t *testing.T) {
	// Plain HTTP must be explicitly allowed.
	cfg := &PollConfig{
		Enabled:   true,
		Upstreams: []UpstreamConfig{{Name: "a", Endpoint: "http://10.0.0.1:9000"}},
	}
	if _, err := NewAgent(cfg, Options{}); err == nil || !strings.Contains(err.Error(), "allow_http") {
		t.Errorf("expected allow_http error, got %v", err)
	}

	cfg.Upstreams[0].AllowHTTP = true
	if _, err := NewAgent(cfg, Options{}); err != nil {
		t.Errorf("unexpected error after allow_http: %v", err)
	}

	// No upstreams at all is an error.
	empty := &PollConfig{Enabled: true}
	if _, err := NewAgent(empty, Options{}); err == nil {
		t.Error("expected error for empty upstreams")
	}

	// Disabled config yields a nil agent.
	if a, err := NewAgent(&PollConfig{}, Options{}); err != nil || a != nil {
		t.Errorf("disabled config should yield nil agent, got %v, %v", a, err)
	}
}

func TestDurationUnmarshal(t *testing.T) {
	cfgStr := `
enabled: true
defaults:
  long_poll_wait: 25s
  busy_interval: 500
upstreams:
  - name: a
    endpoint: http://x
    allow_http: true
`
	var c PollConfig
	if err := yaml.Unmarshal([]byte(cfgStr), &c); err != nil {
		t.Fatal(err)
	}
	if c.Defaults.LongPollWait.Std() != 25*time.Second {
		t.Errorf("long_poll_wait = %v", c.Defaults.LongPollWait.Std())
	}
	if c.Defaults.BusyInterval.Std() != 500*time.Second {
		t.Errorf("busy_interval = %v", c.Defaults.BusyInterval.Std())
	}
}

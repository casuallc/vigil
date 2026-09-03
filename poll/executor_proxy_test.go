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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// fakeProxyRunner records the invocation and returns canned stats.
type fakeProxyRunner struct {
	gotTarget string
	gotLimits SessionLimits
	stats     SessionStats
	err       error
}

func (f *fakeProxyRunner) RunProxySession(ctx context.Context, conn *websocket.Conn, target string, limits SessionLimits) (SessionStats, error) {
	f.gotTarget = target
	f.gotLimits = limits
	conn.Close()
	return f.stats, f.err
}

func newWSEndpoint(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			defer conn.Close()
			// Hold briefly so the runner sees a live connection.
			_, _, _ = conn.ReadMessage()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestExecProxySessionDisabled(t *testing.T) {
	e := newExecutor(Options{}, 0) // no ProxyRunner injected
	task := &Task{Action: json.RawMessage(`{
		"type": "proxy_session",
		"connect_url": "ws://127.0.0.1:1/tunnel",
		"target": "http://127.0.0.1:9000"
	}`)}
	_, err := e.Execute(context.Background(), task)
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected explicit disabled error, got %v", err)
	}
}

func TestExecProxySessionRuns(t *testing.T) {
	_, wsURL := newWSEndpoint(t)

	runner := &fakeProxyRunner{stats: SessionStats{
		Requests: 3, BytesIn: 10, BytesOut: 42, EndReason: "closed",
	}}
	e := newExecutor(Options{ProxyRunner: runner}, 0)

	action, _ := json.Marshal(map[string]interface{}{
		"type":             "proxy_session",
		"connect_url":      wsURL,
		"target":           "http://127.0.0.1:9000",
		"max_duration_sec": 600,
		"max_body_mb":      16,
	})
	task := &Task{Action: action}

	result, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if runner.gotTarget != "http://127.0.0.1:9000" {
		t.Fatalf("runner target = %q", runner.gotTarget)
	}
	if runner.gotLimits.MaxDuration.Seconds() != 600 || runner.gotLimits.MaxBodyMB != 16 {
		t.Fatalf("runner limits = %+v", runner.gotLimits)
	}

	var stats SessionStats
	if err := json.Unmarshal(result, &stats); err != nil {
		t.Fatalf("ack result is not SessionStats JSON: %v (%s)", err, result)
	}
	if stats.Requests != 3 || stats.BytesOut != 42 || stats.EndReason != "closed" {
		t.Fatalf("ack stats = %+v", stats)
	}
}

func TestExecProxySessionDialFailure(t *testing.T) {
	runner := &fakeProxyRunner{}
	e := newExecutor(Options{ProxyRunner: runner}, 0)
	task := &Task{Action: json.RawMessage(`{
		"type": "proxy_session",
		"connect_url": "ws://127.0.0.1:1/tunnel",
		"target": "http://127.0.0.1:9000"
	}`)}
	_, err := e.Execute(context.Background(), task)
	if err == nil || !strings.Contains(err.Error(), "dial tunnel ws") {
		t.Fatalf("expected dial failure, got %v", err)
	}
}

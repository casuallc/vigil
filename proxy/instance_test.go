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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startInstanceForTest creates and starts an instance on an ephemeral
// loopback port and returns the instance plus its base URL.
func startInstanceForTest(t *testing.T, cfg InstanceConfig, hook AccessHook) (*Instance, string) {
	t.Helper()
	return startAuthInstanceForTest(t, cfg, hook, nil)
}

// startAuthInstanceForTest is startInstanceForTest with an explicit
// AuthFunc for forward-mode instances.
func startAuthInstanceForTest(t *testing.T, cfg InstanceConfig, hook AccessHook, auth AuthFunc) (*Instance, string) {
	t.Helper()
	cfg.Listen = "127.0.0.1:0"
	inst, err := NewInstance(cfg, OriginAPI, hook, auth)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := inst.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = inst.Stop(context.Background()) })

	// Retrieve the actual bound address.
	inst.mu.Lock()
	addr := inst.ln.Addr().String()
	inst.mu.Unlock()
	return inst, "http://" + addr
}

func TestInstanceProxiesRequests(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "yes")
		fmt.Fprintf(w, "hello %s", r.URL.Path)
	}))
	defer backend.Close()

	var records []AccessRecord
	_, base := startInstanceForTest(t, InstanceConfig{
		Name:         "t1",
		Target:       backend.URL,
		AllowPrivate: true,
	}, func(rec AccessRecord) { records = append(records, rec) })

	resp, err := http.Get(base + "/foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != "hello /foo" {
		t.Fatalf("body = %q, want %q", body, "hello /foo")
	}
	if resp.Header.Get("X-Backend") != "yes" {
		t.Fatal("backend response header must pass through")
	}
	if len(records) != 1 || records[0].Via != "listen" || records[0].Status != 200 {
		t.Fatalf("access hook records = %+v", records)
	}
}

func TestInstanceInjectsHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.Header.Get("X-Injected"))
	}))
	defer backend.Close()

	_, base := startInstanceForTest(t, InstanceConfig{
		Name:         "t2",
		Target:       backend.URL,
		AllowPrivate: true,
		HeaderSet:    map[string]string{"X-Injected": "vigil"},
	}, nil)

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "vigil" {
		t.Fatalf("body = %q, want injected header value", body)
	}
}

func TestInstanceDeniedTarget(t *testing.T) {
	// The metadata IP is always denied: the instance starts but every
	// request gets a 403 without ever dialing out.
	_, base := startInstanceForTest(t, InstanceConfig{
		Name:         "t3",
		Target:       "http://169.254.169.254/latest",
		AllowPrivate: true,
	}, nil)

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestInstanceUpstreamError502(t *testing.T) {
	// Bind a port and close it immediately to get a refused address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	dead := "http://" + ln.Addr().String()
	ln.Close()

	inst, base := startInstanceForTest(t, InstanceConfig{
		Name:         "t4",
		Target:       dead,
		AllowPrivate: true,
	}, nil)

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	if got := inst.Status().Stats.UpstreamErr; got != 1 {
		t.Fatalf("UpstreamErr = %d, want 1", got)
	}
}

func TestInstanceWebSocketPassthrough(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Echo one message.
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(mt, msg)
	}))
	defer backend.Close()

	_, base := startInstanceForTest(t, InstanceConfig{
		Name:         "t5",
		Target:       backend.URL,
		AllowPrivate: true,
	}, nil)

	wsURL := "ws://" + strings.TrimPrefix(base, "http://")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial through proxy: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("ws write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if string(msg) != "ping" {
		t.Fatalf("ws echo = %q, want %q", msg, "ping")
	}
}

func TestInstanceInvalidConfig(t *testing.T) {
	for _, cfg := range []InstanceConfig{
		{Name: "", Listen: ":0", Target: "http://x"},
		{Name: "a", Listen: "", Target: "http://x"},
		{Name: "a", Listen: ":0", Target: "not-a-url"},
		{Name: "a", Listen: ":0", Target: "ftp://x"},
		{Name: "a", Listen: ":0", Target: "http://x", TLS: TLSConfig{Enabled: true}},
		{Name: "a", Listen: ":0", Mode: "bogus", Target: "http://x"},
		{Name: "a", Listen: ":0", Mode: ModeForward, Target: "http://x", Whitelist: []string{"x"}},
		{Name: "a", Listen: ":0", Mode: ModeForward}, // forward without whitelist
	} {
		if _, err := NewInstance(cfg, OriginAPI, nil, nil); err == nil {
			t.Errorf("NewInstance(%+v) should fail", cfg)
		}
	}
}

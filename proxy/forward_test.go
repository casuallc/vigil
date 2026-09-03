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
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testProxyUser = "admin"
	testProxyPass = "secret"
)

// testAuthFunc accepts only the fixed test credentials.
func testAuthFunc(u, p string) bool { return u == testProxyUser && p == testProxyPass }

// startForwardForTest starts a forward-mode instance whose whitelist covers
// loopback, and returns a ready-to-use proxied HTTP client plus the
// instance.
func startForwardForTest(t *testing.T, hook AccessHook, entries []string) (*Instance, *http.Client) {
	t.Helper()
	inst, base := startAuthInstanceForTest(t, InstanceConfig{
		Name:         "fwd",
		Mode:         ModeForward,
		Whitelist:    entries,
		AllowPrivate: true,
	}, hook, testAuthFunc)

	proxyURL, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	proxyURL.User = url.UserPassword(testProxyUser, testProxyPass)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, // test backends only
			DisableKeepAlives: true,                                  // CONNECT tunnels: close when the response is done
		},
		Timeout: 10 * time.Second,
	}
	return inst, client
}

// recorder is a thread-safe AccessHook sink: hooks fire from handler
// goroutines, and CONNECT records only land when the tunnel closes, so
// tests poll with a deadline instead of asserting immediately.
type recorder struct {
	mu   sync.Mutex
	recs []AccessRecord
}

func (r *recorder) hook(rec AccessRecord) {
	r.mu.Lock()
	r.recs = append(r.recs, rec)
	r.mu.Unlock()
}

func (r *recorder) waitFor(want int) []AccessRecord {
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.mu.Lock()
		recs := append([]AccessRecord(nil), r.recs...)
		r.mu.Unlock()
		if len(recs) >= want || time.Now().After(deadline) {
			return recs
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestForwardPlainHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			t.Error("Proxy-Authorization must not leak upstream")
		}
		w.Header().Set("X-Backend", "yes")
		fmt.Fprintf(w, "fwd %s", r.URL.Path)
	}))
	defer backend.Close()
	backendHost := strings.TrimPrefix(backend.URL, "http://")

	rec := &recorder{}
	_, client := startForwardForTest(t, rec.hook, []string{"127.0.0.1"})

	resp, err := client.Get("http://" + backendHost + "/data")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "fwd /data" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
	if resp.Header.Get("X-Backend") != "yes" {
		t.Fatal("response headers must pass through")
	}
	records := rec.waitFor(1)
	if len(records) != 1 || records[0].Via != "forward" || records[0].Status != 200 || records[0].Denied {
		t.Fatalf("records = %+v", records)
	}
}

func TestForwardConnectHTTPS(t *testing.T) {
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "secure hello")
	}))
	defer backend.Close()
	backendURL := strings.TrimPrefix(backend.URL, "https://")

	rec := &recorder{}
	inst, client := startForwardForTest(t, rec.hook, []string{"127.0.0.1"})

	resp, err := client.Get("https://" + backendURL + "/")
	if err != nil {
		t.Fatalf("HTTPS via CONNECT: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "secure hello" {
		t.Fatalf("body = %q", body)
	}
	records := rec.waitFor(1)
	if len(records) != 1 || records[0].Method != http.MethodConnect || records[0].Status != 200 {
		t.Fatalf("records = %+v", records)
	}
	if got := inst.Status().Stats.BytesOut; got <= 0 {
		t.Fatalf("tunnel BytesOut = %d, want > 0", got)
	}
}

func TestForwardRequiresAuth(t *testing.T) {
	inst, base := startAuthInstanceForTest(t, InstanceConfig{
		Name:         "fwd",
		Mode:         ModeForward,
		Whitelist:    []string{"127.0.0.1"},
		AllowPrivate: true,
	}, nil, testAuthFunc)
	_ = inst

	// No credentials at all.
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want 407", resp.StatusCode)
	}
	if resp.Header.Get("Proxy-Authenticate") == "" {
		t.Fatal("407 must carry Proxy-Authenticate")
	}

	// Wrong credentials must also get a 407.
	proxyURL, _ := url.Parse(base)
	proxyURL.User = url.UserPassword(testProxyUser, "wrong")
	badClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 5 * time.Second}
	resp3, err := badClient.Get("http://127.0.0.1/")
	if err != nil {
		t.Fatalf("GET with wrong password: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want 407", resp3.StatusCode)
	}
}

func TestForwardWhitelistDeny(t *testing.T) {
	rec := &recorder{}
	_, client := startForwardForTest(t, rec.hook, []string{"10.0.0.0/8"}) // loopback not whitelisted

	// Plain HTTP to a non-whitelisted destination.
	resp, err := client.Get("http://127.0.0.1:1/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	denied := rec.waitFor(1)
	if len(denied) != 1 || denied[0].Via != "forward" || !denied[0].Denied {
		t.Fatalf("denied records = %+v", denied)
	}
}

func TestForwardConnectWhitelistDeny(t *testing.T) {
	rec := &recorder{}
	inst, _ := startAuthInstanceForTest(t, InstanceConfig{
		Name:         "fwd",
		Mode:         ModeForward,
		Whitelist:    []string{"10.0.0.0/8"},
		AllowPrivate: true,
	}, rec.hook, testAuthFunc)

	inst.mu.Lock()
	addr := inst.ln.Addr().String()
	inst.mu.Unlock()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(testProxyUser+":"+testProxyPass))
	fmt.Fprintf(conn, "CONNECT 169.254.169.254:443 HTTP/1.1\r\nHost: 169.254.169.254:443\r\nProxy-Authorization: %s\r\n\r\n", auth)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read status line: %v", err)
	}
	if !strings.Contains(line, "403") {
		t.Fatalf("status line = %q, want 403", line)
	}
	denied := rec.waitFor(1)
	if len(denied) != 1 || denied[0].Method != http.MethodConnect || !denied[0].Denied {
		t.Fatalf("denied records = %+v", denied)
	}
}

func TestForwardRejectsNonAbsoluteURI(t *testing.T) {
	// A client that talks origin-form to a forward proxy gets a 400.
	_, base := startAuthInstanceForTest(t, InstanceConfig{
		Name:         "fwd2",
		Mode:         ModeForward,
		Whitelist:    []string{"127.0.0.1"},
		AllowPrivate: true,
	}, nil, testAuthFunc)

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(testProxyUser+":"+testProxyPass))
	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /origin-form HTTP/1.1\r\nHost: 127.0.0.1\r\nProxy-Authorization: %s\r\n\r\n", auth)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(line, "400") {
		t.Fatalf("status line = %q, want 400", line)
	}
}

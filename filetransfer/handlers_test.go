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

package filetransfer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

const (
	itAuthUser = "admq"
	itAuthPass = "admq-file-transfer"
)

func newReceiverServer(t *testing.T, targetDir string) (*Manager, *httptest.Server) {
	t.Helper()
	m := NewManager(Options{
		DataDir:       t.TempDir(),
		EncryptionKey: testKey,
		Roots:         []string{targetDir},
	})
	r := mux.NewRouter()
	m.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return m, srv
}

func TestDirectTransferEndToEnd(t *testing.T) {
	targetDir := t.TempDir()
	_, srv := newReceiverServer(t, targetDir)

	// Create + start a RECV task over the API.
	createReq := TaskConfig{
		TaskID:          100,
		Role:            RoleRecv,
		RelayType:       RelayDirect,
		TargetDir:       targetDir,
		OverwritePolicy: Overwrite,
	}
	doAuthedJSON(t, http.MethodPost, srv.URL+"/api/transfer/tasks", createReq)
	doAuthedJSON(t, http.MethodPost, srv.URL+"/api/transfer/tasks/100/start", nil)

	// Prepare a source file larger than one chunk.
	srcDir := t.TempDir()
	content := bytes.Repeat([]byte("vigil-"), 1000) // 6000 bytes
	srcFile := filepath.Join(srcDir, "payload.bin")
	if err := os.WriteFile(srcFile, content, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	host, port := splitHostPort(t, srv.URL)
	target := TargetConfig{
		Host:        host,
		Port:        port,
		AuthUser:    itAuthUser,
		AuthPass:    itAuthPass,
		AgentTaskID: 100,
	}
	file := FileEntry{RelPath: "out/payload.bin", Size: int64(len(content)), Sha256: sha256Hex(content)}

	reader := func(offset int64, length int) ([]byte, error) {
		f, err := os.Open(srcFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buf := make([]byte, length)
		n, _ := f.ReadAt(buf, offset)
		return buf[:n], nil
	}

	dt := newDirectTransport()
	cfg := TaskConfig{RelayType: RelayDirect, ChunkSize: 1024} // forces multiple chunks
	if err := dt.SendFile(context.Background(), cfg, target, file, reader, func(int) {}); err != nil {
		t.Fatalf("SendFile: %v", err)
	}

	// The receiver should have written and finalized the file.
	got, err := os.ReadFile(filepath.Join(targetDir, "out", "payload.bin"))
	if err != nil {
		t.Fatalf("read finalized file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %d bytes want %d", len(got), len(content))
	}
}

func TestProgressEndpointDrivesResume(t *testing.T) {
	targetDir := t.TempDir()
	m, srv := newReceiverServer(t, targetDir)

	_ = m.CreateTask(TaskConfig{TaskID: 101, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite})
	// Receive a partial first chunk directly so progress shows received bytes.
	if err := m.ReceiveChunk(101, ChunkMeta{RelPath: "r.bin", Offset: 0, Length: 4, Eof: false}, []byte("abcd")); err != nil {
		t.Fatalf("ReceiveChunk: %v", err)
	}

	host, port := splitHostPort(t, srv.URL)
	dt := newDirectTransport()
	offset := dt.queryResumeOffset(context.Background(), "http://"+host+":"+strconv.Itoa(port), TargetConfig{AuthUser: itAuthUser, AuthPass: itAuthPass, AgentTaskID: 101}, "r.bin")
	if offset != 4 {
		t.Fatalf("expected resume offset 4, got %d", offset)
	}
}

// ===================== helpers =====================

func doAuthedJSON(t *testing.T, method, urlStr string, body interface{}) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req, err := http.NewRequest(method, urlStr, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.SetBasicAuth(itAuthUser, itAuthPass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, urlStr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s -> %d", method, urlStr, resp.StatusCode)
	}
}

func splitHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return u.Hostname(), port
}

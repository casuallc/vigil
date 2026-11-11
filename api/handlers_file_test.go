package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/casuallc/vigil/file"
)

func TestHandleFileStreamUpload(t *testing.T) {
	server := &Server{}

	// Create a temporary directory for uploads
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "test-upload.txt")

	// Create request with file content in body
	content := []byte("hello, this is a stream upload test")
	req := httptest.NewRequest(http.MethodPost, "/api/files/stream", bytes.NewReader(content))
	req.Header.Set("X-Target-Path", targetPath)

	rr := httptest.NewRecorder()
	server.handleFileStreamUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify file was written
	written, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if !bytes.Equal(written, content) {
		t.Errorf("expected %q, got %q", content, written)
	}
}

func TestHandleFileStreamUploadMissingHeader(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/files/stream", bytes.NewReader([]byte("test")))
	rr := httptest.NewRecorder()
	server.handleFileStreamUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleFileFetch(t *testing.T) {
	server := &Server{}

	// Source HTTP server serving file content
	content := []byte("hello, this is a fetch test")
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer src.Close()

	tempDir := t.TempDir()
	sum := fmt.Sprintf("%x", md5.Sum(content))

	body, _ := json.Marshal(map[string]interface{}{
		"files": []map[string]interface{}{
			{"url": src.URL + "/a.txt", "dir": tempDir, "name": "a.txt", "md5": sum},
			{"url": src.URL + "/b.txt", "dir": tempDir}, // name defaults to URL basename
			{"url": src.URL + "/c.txt", "dir": tempDir, "name": "c.txt", "md5": "deadbeef"}, // checksum mismatch
			{"dir": tempDir}, // missing url
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/files/fetch", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	server.handleFileFetch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Results []file.FetchResult `json:"results"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(resp.Results))
	}

	// First two succeed
	for i, name := range []string{"a.txt", "b.txt"} {
		if !resp.Results[i].Success {
			t.Errorf("result %d expected success, got error: %s", i, resp.Results[i].Error)
			continue
		}
		written, err := os.ReadFile(filepath.Join(tempDir, name))
		if err != nil {
			t.Errorf("failed to read fetched file %s: %v", name, err)
			continue
		}
		if !bytes.Equal(written, content) {
			t.Errorf("file %s: expected %q, got %q", name, content, written)
		}
	}
	if resp.Results[0].Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), resp.Results[0].Size)
	}

	// Last two fail
	if resp.Results[2].Success {
		t.Errorf("expected checksum mismatch failure")
	}
	if resp.Results[3].Success {
		t.Errorf("expected missing url failure")
	}

	// Checksum mismatch must not leave the target file behind
	if _, err := os.Stat(filepath.Join(tempDir, "c.txt")); !os.IsNotExist(err) {
		t.Errorf("expected c.txt to not exist after checksum mismatch")
	}
}

func TestHandleFileFetchBasicAuth(t *testing.T) {
	server := &Server{}

	content := []byte("protected content")
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write(content)
	}))
	defer src.Close()

	tempDir := t.TempDir()
	body, _ := json.Marshal(map[string]interface{}{
		"files": []map[string]interface{}{
			{
				"url":  src.URL + "/secret.txt",
				"dir":  tempDir,
				"name": "secret.txt",
				"auth": map[string]string{"type": "basic", "username": "admin", "password": "secret"},
			},
			{"url": src.URL + "/secret.txt", "dir": tempDir, "name": "noauth.txt"}, // missing auth
			{
				"url":  src.URL + "/secret.txt",
				"dir":  tempDir,
				"name": "badtype.txt",
				"auth": map[string]string{"type": "digest", "username": "admin", "password": "secret"},
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/files/fetch", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	server.handleFileFetch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Results []file.FetchResult `json:"results"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}

	if !resp.Results[0].Success {
		t.Fatalf("expected basic auth success, got error: %s", resp.Results[0].Error)
	}
	written, err := os.ReadFile(filepath.Join(tempDir, "secret.txt"))
	if err != nil {
		t.Fatalf("failed to read fetched file: %v", err)
	}
	if !bytes.Equal(written, content) {
		t.Errorf("expected %q, got %q", content, written)
	}

	if resp.Results[1].Success {
		t.Errorf("expected 401 failure without auth")
	}
	if resp.Results[2].Success {
		t.Errorf("expected unsupported auth type failure")
	}
}

func TestHandleFileFetchEmptyFiles(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/files/fetch", bytes.NewReader([]byte(`{"files":[]}`)))
	rr := httptest.NewRecorder()
	server.handleFileFetch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

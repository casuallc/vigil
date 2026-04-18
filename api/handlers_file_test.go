package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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

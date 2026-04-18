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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type sseEvent struct {
	event string
	data  string
}

func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	lines := strings.Split(body, "\n")
	var currentEvent sseEvent
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			currentEvent.event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			if currentEvent.data != "" {
				currentEvent.data += "\n"
			}
			currentEvent.data += strings.TrimPrefix(line, "data: ")
		} else if line == "" {
			if currentEvent.event != "" || currentEvent.data != "" {
				events = append(events, currentEvent)
				currentEvent = sseEvent{}
			}
		}
	}
	// Handle trailing event without final blank line
	if currentEvent.event != "" || currentEvent.data != "" {
		events = append(events, currentEvent)
	}
	return events
}

func createTestLogFile(t *testing.T, dir string) string {
	t.Helper()
	content := "line1\nline2\nline3\nline4\nline5\n"
	path := filepath.Join(dir, "test.log")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test log file: %v", err)
	}
	return path
}

func TestHandleLogStreamFromBeginning(t *testing.T) {
	server := &Server{}
	tempDir := t.TempDir()
	logPath := createTestLogFile(t, tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream?path="+logPath+"&from_line=0", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	server.handleLogStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	events := parseSSEEvents(t, rr.Body.String())
	if len(events) != 5 {
		t.Fatalf("expected 5 line events, got %d", len(events))
	}

	for i, ev := range events {
		if ev.event != "line" {
			t.Errorf("expected event 'line', got %q", ev.event)
		}
		var line LogLine
		if err := json.Unmarshal([]byte(ev.data), &line); err != nil {
			t.Fatalf("failed to unmarshal line data: %v", err)
		}
		if line.LineNumber != i+1 {
			t.Errorf("expected line_number %d, got %d", i+1, line.LineNumber)
		}
		expectedContent := fmt.Sprintf("line%d", i+1)
		if line.Content != expectedContent {
			t.Errorf("expected content %q, got %q", expectedContent, line.Content)
		}
	}
}

func TestHandleLogStreamFromEndDefault(t *testing.T) {
	server := &Server{}
	tempDir := t.TempDir()
	logPath := createTestLogFile(t, tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream?path="+logPath, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	server.handleLogStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	events := parseSSEEvents(t, rr.Body.String())
	if len(events) != 0 {
		t.Fatalf("expected 0 line events when starting from end, got %d", len(events))
	}
}

func TestHandleLogStreamFromSpecificLine(t *testing.T) {
	server := &Server{}
	tempDir := t.TempDir()
	logPath := createTestLogFile(t, tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream?path="+logPath+"&from_line=3", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	server.handleLogStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	events := parseSSEEvents(t, rr.Body.String())
	if len(events) != 3 {
		t.Fatalf("expected 3 line events, got %d", len(events))
	}

	expectedLines := []struct {
		number  int
		content string
	}{
		{3, "line3"},
		{4, "line4"},
		{5, "line5"},
	}

	for i, ev := range events {
		if ev.event != "line" {
			t.Errorf("expected event 'line', got %q", ev.event)
		}
		var line LogLine
		if err := json.Unmarshal([]byte(ev.data), &line); err != nil {
			t.Fatalf("failed to unmarshal line data: %v", err)
		}
		if line.LineNumber != expectedLines[i].number {
			t.Errorf("expected line_number %d, got %d", expectedLines[i].number, line.LineNumber)
		}
		if line.Content != expectedLines[i].content {
			t.Errorf("expected content %q, got %q", expectedLines[i].content, line.Content)
		}
	}
}

func TestHandleLogStreamFromEndOffset(t *testing.T) {
	server := &Server{}
	tempDir := t.TempDir()
	logPath := createTestLogFile(t, tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream?path="+logPath+"&from_line=-2", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	server.handleLogStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	events := parseSSEEvents(t, rr.Body.String())
	if len(events) != 2 {
		t.Fatalf("expected 2 line events, got %d", len(events))
	}

	expectedLines := []struct {
		number  int
		content string
	}{
		{4, "line4"},
		{5, "line5"},
	}

	for i, ev := range events {
		if ev.event != "line" {
			t.Errorf("expected event 'line', got %q", ev.event)
		}
		var line LogLine
		if err := json.Unmarshal([]byte(ev.data), &line); err != nil {
			t.Fatalf("failed to unmarshal line data: %v", err)
		}
		if line.LineNumber != expectedLines[i].number {
			t.Errorf("expected line_number %d, got %d", expectedLines[i].number, line.LineNumber)
		}
		if line.Content != expectedLines[i].content {
			t.Errorf("expected content %q, got %q", expectedLines[i].content, line.Content)
		}
	}
}

func TestHandleLogStreamMissingPath(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream", nil)
	rr := httptest.NewRecorder()

	server.handleLogStream(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "missing required query parameter: path") {
		t.Errorf("expected error message about missing path, got %q", body)
	}
}

func TestHandleLogStreamFileNotFound(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/files/logs/stream?path=/nonexistent/file.log", nil)
	rr := httptest.NewRecorder()

	server.handleLogStream(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 (SSE stream starts before error), got %d", rr.Code)
	}

	events := parseSSEEvents(t, rr.Body.String())
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}

	ev := events[0]
	if ev.event != "error" {
		t.Errorf("expected event 'error', got %q", ev.event)
	}

	var errData map[string]string
	if err := json.Unmarshal([]byte(ev.data), &errData); err != nil {
		t.Fatalf("failed to unmarshal error data: %v", err)
	}

	msg, ok := errData["message"]
	if !ok {
		t.Fatalf("expected 'message' field in error data")
	}
	if !strings.Contains(msg, "failed to open file") {
		t.Errorf("expected error message to contain 'failed to open file', got %q", msg)
	}
}

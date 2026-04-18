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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// LogLine represents a single log line in SSE events
type LogLine struct {
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing required query parameter: path", http.StatusBadRequest)
		return
	}

	fromLineStr := r.URL.Query().Get("from_line")
	fromLine := -1 // default: tail -f behavior
	if fromLineStr != "" {
		var err error
		fromLine, err = strconv.Atoi(fromLineStr)
		if err != nil {
			http.Error(w, "invalid from_line parameter", http.StatusBadRequest)
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	file, err := os.Open(path)
	if err != nil {
		sseWriteError(w, fmt.Sprintf("failed to open file: %v", err))
		flusher.Flush()
		return
	}
	defer file.Close()

	startLine, err := seekToLine(file, fromLine)
	if err != nil {
		sseWriteError(w, fmt.Sprintf("failed to seek to line: %v", err))
		flusher.Flush()
		return
	}

	const logPollInterval = 500 * time.Millisecond

	reader := bufio.NewReader(file)
	lineNumber := startLine
	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait for new content
				flusher.Flush()
				select {
				case <-r.Context().Done():
					return
				case <-ticker.C:
					continue
				}
			}
			sseWriteError(w, fmt.Sprintf("error reading file: %v", err))
			flusher.Flush()
			return
		}

		// Trim trailing newline (and carriage return) for the content
		content := line
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}
		if len(content) > 0 && content[len(content)-1] == '\r' {
			content = content[:len(content)-1]
		}

		sseWriteLine(w, LogLine{LineNumber: lineNumber, Content: content})
		flusher.Flush()
		lineNumber++
	}
}

// seekToLine positions the file to the correct starting line based on fromLine.
// Returns the starting line number.
func seekToLine(file *os.File, fromLine int) (int, error) {
	if fromLine < 0 {
		return seekFromEnd(file, -fromLine)
	}
	if fromLine == 0 {
		_, err := file.Seek(0, io.SeekStart)
		return 1, err
	}

	// fromLine > 0: start from that line number
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	reader := bufio.NewReader(file)
	currentLine := 1
	for currentLine < fromLine {
		_, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Position at end of file
				return currentLine, nil
			}
			return 0, err
		}
		currentLine++
	}

	// Reposition file to where the reader ended up
	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	// The bufio.Reader may have buffered data, so we need to account for unread bytes
	unread := reader.Buffered()
	_, err = file.Seek(offset-int64(unread), io.SeekStart)
	if err != nil {
		return 0, err
	}

	return fromLine, nil
}

// seekFromEnd positions the file n lines before the end of the file.
// Returns the starting line number.
func seekFromEnd(file *os.File, n int) (int, error) {
	if n <= 0 {
		_, err := file.Seek(0, io.SeekEnd)
		return 1, err
	}

	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}
	size := stat.Size()
	if size == 0 {
		return 1, nil
	}

	// Read the file backwards to count newlines
	const bufSize = 8192
	buf := make([]byte, bufSize)
	newlines := 0
	pos := size

	for pos > 0 && newlines < n {
		readSize := int64(bufSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		_, err := file.Seek(pos, io.SeekStart)
		if err != nil {
			return 0, err
		}

		_, err = io.ReadFull(file, buf[:readSize])
		if err != nil {
			return 0, err
		}

		// Scan backwards through the buffer
		for i := int(readSize) - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				newlines++
				if newlines == n {
					// Position after this newline
					_, err := file.Seek(pos+int64(i)+1, io.SeekStart)
					if err != nil {
						return 0, err
					}
					return countLines(file, pos+int64(i)+1)
				}
			}
		}
	}

	// If we didn't find enough newlines, start from beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return 1, nil
}

// countLines counts the number of lines from the start of the file up to the given offset.
func countLines(file *os.File, offset int64) (int, error) {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	reader := bufio.NewReader(file)
	lineCount := 1
	var read int64
	for read < offset {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return lineCount, nil
			}
			return 0, err
		}
		read += int64(len(line))
		if read <= offset {
			lineCount++
		}
	}
	return lineCount, nil
}

// sseWriteLine writes an SSE event with JSON data for a log line.
func sseWriteLine(w io.Writer, line LogLine) {
	data, _ := json.Marshal(line)
	fmt.Fprintf(w, "event: line\ndata: %s\n\n", data)
}

// sseWriteError writes an SSE error event.
func sseWriteError(w io.Writer, message string) {
	data, _ := json.Marshal(map[string]string{"message": message})
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
}

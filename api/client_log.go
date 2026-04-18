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
	"net/http"
	"net/url"
	"strings"
)

// LogLine represents a single log line in SSE events.
type LogLine struct {
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
}

// StreamLogs opens an SSE connection to stream log lines from the server.
// For each "event: line" received, the JSON data is unmarshaled into a LogLine
// and passed to the handler. The method returns when the connection closes
// or an error occurs.
func (c *Client) StreamLogs(path string, fromLine int, handler func(line LogLine)) error {
	q := url.Values{}
	q.Set("path", path)
	if fromLine != 0 {
		q.Set("from_line", fmt.Sprintf("%d", fromLine))
	}

	reqURL := fmt.Sprintf("%s/api/files/logs/stream?%s", c.host, q.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")

	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024) // 1MB max line
	var eventType string
	var dataLines []string

	for scanner.Scan() {
		text := scanner.Text()

		if text == "" {
			// Empty line means the event is complete
			if eventType == "line" && len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				var line LogLine
				if err := json.Unmarshal([]byte(data), &line); err != nil {
					return fmt.Errorf("failed to unmarshal log line: %w", err)
				}
				handler(line)
			}
			// Reset for next event
			eventType = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(text, "event: ") {
			eventType = strings.TrimPrefix(text, "event: ")
		} else if strings.HasPrefix(text, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(text, "data: "))
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

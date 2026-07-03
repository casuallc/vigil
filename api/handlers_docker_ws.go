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
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/casuallc/vigil/docker"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var dockerWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleDockerExecWebSocket handles interactive exec in a container via WebSocket.
// Endpoint: /api/docker/containers/{id}/exec/ws
// First client message must be JSON: {"command":"...","tty":true,"width":80,"height":24}
// Subsequent text messages are sent to exec stdin. Messages prefixed with "resize:"
// are parsed as {"cols":N,"rows":N} and resize the exec session.
func (s *Server) handleDockerExecWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		http.Error(w, "docker manager not initialized", http.StatusServiceUnavailable)
		return
	}

	id := mux.Vars(r)["id"]

	ws, err := dockerWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Docker exec WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	_, initMsg, err := ws.ReadMessage()
	if err != nil {
		log.Printf("Docker exec WebSocket read init error: %v", err)
		return
	}

	var req docker.WSExecRequest
	if err := json.Unmarshal(initMsg, &req); err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("invalid init message: "+err.Error()))
		return
	}
	if req.Command == "" {
		ws.WriteMessage(websocket.TextMessage, []byte("command is required"))
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	hijackResp, execID, err := s.dockerManager.ExecInteractive(ctx, id, []string{"sh", "-c", req.Command}, req.Tty, req.Width, req.Height)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("exec failed: "+err.Error()))
		return
	}
	defer hijackResp.Close()

	// WS -> exec stdin (and resize handling)
	go func() {
		defer cancel()
		for {
			msgType, payload, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.TextMessage && strings.HasPrefix(string(payload), "resize:") {
				var resize docker.WSResizeMessage
				if err := json.Unmarshal(payload[7:], &resize); err == nil {
					_ = s.dockerManager.ExecResize(ctx, execID, resize.Cols, resize.Rows)
				}
				continue
			}
			if _, err := hijackResp.Conn.Write(payload); err != nil {
				return
			}
		}
	}()

	// exec stdout/stderr -> WS
	go func() {
		defer cancel()
		buf := make([]byte, 4096)
		for {
			n, err := hijackResp.Reader.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	log.Printf("Docker exec WebSocket closed for container %s", id)
}

// handleDockerLogsWebSocket streams container logs via WebSocket.
// Endpoint: /api/docker/containers/{id}/logs/ws?tail=100&since=...
func (s *Server) handleDockerLogsWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		http.Error(w, "docker manager not initialized", http.StatusServiceUnavailable)
		return
	}

	id := mux.Vars(r)["id"]
	tail := r.URL.Query().Get("tail")
	since := r.URL.Query().Get("since")

	ws, err := dockerWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Docker logs WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		defer pw.Close()
		_ = s.dockerManager.StreamLogs(ctx, id, true, tail, since, pw)
	}()

	go func() {
		defer cancel()
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Wait for client close.
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			cancel()
			break
		}
	}
}

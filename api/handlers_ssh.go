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
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// handleSSHWebSocket handles WebSocket SSH connections
func (s *Server) handleSSHWebSocket(w http.ResponseWriter, r *http.Request) {
	vmName := r.URL.Query().Get("vm_name")

	if vmName == "" {
		http.Error(w, "vm_name required", http.StatusBadRequest)
		return
	}

	vmInfo, err := s.vmManager.GetVM(vmName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
		Host:     vmInfo.IP,
		Port:     vmInfo.Port,
		Username: vmInfo.Username,
		Password: vmInfo.Password,
		KeyPath:  vmInfo.KeyPath,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer sshClient.Close()

	// Upgrade to WebSocket
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Allow all cross-origin requests (should restrict in production)
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	session, err := sshClient.CreateSession()
	if err != nil {
		ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
		return
	}
	defer session.Close()

	// Request PTY
	if err := session.RequestPty(
		"xterm-256color",
		40,
		120,
		ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		},
	); err != nil {
		ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
		return
	}

	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()

	if err := session.Shell(); err != nil {
		ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate unique connection ID
	connID := fmt.Sprintf("%s-%d", vmName, time.Now().UnixNano())

	// Get client IP
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}

	// Get username
	username := s.getConnectionUser(r)

	// Register connection
	s.RegisterSSHConnection(connID, vmName, clientIP, username)
	log.Printf("SSH connection registered: ID=%s, VM=%s, ClientIP=%s, User=%s", connID, vmName, clientIP, username)

	// Ensure connection is unregistered when session ends
	defer func() {
		s.UnregisterSSHConnection(connID)
		log.Printf("SSH connection unregistered: ID=%s", connID)
	}()

	// ---------------- Input: WS -> SSH ----------------
	go func() {
		defer cancel()
		for {
			messageType, payload, err := ws.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error for connection %s: %v", connID, err)
				return
			}

			if messageType == websocket.TextMessage {
				// Handle window resize message
				if strings.HasPrefix(string(payload), "resize:") {
					var resizeData struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					if err := json.Unmarshal(payload[7:], &resizeData); err != nil {
						log.Printf("Failed to parse resize data: %v", err)
						continue
					}
					if err := session.WindowChange(resizeData.Rows, resizeData.Cols); err != nil {
						log.Printf("Failed to change window size: %v", err)
					}
					continue
				}

				// Write to SSH session
				if _, err := stdin.Write(payload); err != nil {
					log.Printf("Failed to write to SSH session: %v", err)
					return
				}
			} else {
				_, _ = stdin.Write(payload)
			}
		}
	}()

	// ---------------- Output: SSH -> WS ----------------
	go func() {
		defer cancel()

		reader := io.MultiReader(stdout, stderr)
		buf := make([]byte, 4096)

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					log.Printf("WebSocket write error for connection %s: %v", connID, err)
					return
				}
			}
			if err != nil {
				log.Printf("SSH output error for connection %s: %v", connID, err)
				return
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("WebSocket connection closed: ID=%s", connID)

	// Close session with timeout
	done := make(chan struct{})
	go func() {
		session.Close()
		close(done)
	}()

	select {
	case <-done:
		// Session closed normally
	case <-time.After(2 * time.Second):
		log.Printf("SSH session close timeout for connection %s", connID)
	}
}

// handleListSSHConnections handles listing SSH connections
func (s *Server) handleListSSHConnections(w http.ResponseWriter, r *http.Request) {
	connections := s.GetSSHConnections()

	// Get filter parameters
	vmName := r.URL.Query().Get("vm_name")
	userName := r.URL.Query().Get("user_name")
	clientIP := r.URL.Query().Get("client_ip")

	// Apply filters
	var filteredConnections []*SSHConnectionInfo
	for _, conn := range connections {
		match := true

		if vmName != "" && conn.VMName != vmName {
			match = false
		}
		if userName != "" && conn.Username != userName {
			match = false
		}
		if clientIP != "" && conn.ClientIP != clientIP {
			match = false
		}

		if match {
			filteredConnections = append(filteredConnections, conn)
		}
	}

	writeJSON(w, http.StatusOK, filteredConnections)
}

// handleCloseSSHConnection handles closing a specific SSH connection
func (s *Server) handleCloseSSHConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		writeError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	success := s.CloseSSHConnection(id)
	if !success {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "connection closed successfully"})
}

// handleCloseAllSSHConnections handles closing all SSH connections
func (s *Server) handleCloseAllSSHConnections(w http.ResponseWriter, r *http.Request) {
	count := s.CloseAllSSHConnections()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "all connections closed successfully",
		"count":   count,
	})
}

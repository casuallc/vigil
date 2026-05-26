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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/casuallc/vigil/upgrade"
	"github.com/casuallc/vigil/version"
)

// handleSystemUpgrade handles POST /api/system/upgrade
func (s *Server) handleSystemUpgrade(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeError(w, http.StatusForbidden, "only super admin can perform upgrade")
		return
	}

	var req struct {
		Method   string `json:"method"`   // "upload" or "download"
		URL      string `json:"url"`      // for download method
		Checksum string `json:"checksum"` // optional sha256 checksum
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	u := upgrade.NewUpgrade("data")

	// Check if already upgrading
	isIdle, err := u.StateManager().IsIdle()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read state: "+err.Error())
		return
	}
	if !isIdle {
		state, _ := u.StateManager().Read()
		currentState := "unknown"
		if state != nil {
			currentState = state.Upgrade.State
		}
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("upgrade already in progress: %s", currentState),
		})
		return
	}

	var binaryReader io.Reader
	var binaryPath string

	switch req.Method {
	case "upload":
		// Expect multipart form with file field
		if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB max
			writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing file field: "+err.Error())
			return
		}
		defer file.Close()
		binaryReader = file

	case "download":
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required for download method")
			return
		}
		// Download binary
		bm := upgrade.NewBinaryManager("data")
		path, err := bm.Download(req.URL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "download failed: "+err.Error())
			return
		}
		binaryPath = path

	default:
		writeError(w, http.StatusBadRequest, "invalid method, must be 'upload' or 'download'")
		return
	}

	// Start upgrade (non-blocking, monitors in background)
	go func() {
		if err := u.StartUpgrade(binaryPath, binaryReader, req.Checksum); err != nil {
			// Error already logged in StartUpgrade
			return
		}
		// Trigger server drain and shutdown after successful handoff
		s.handleUpgradeHandoff()
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "upgrade started",
		"state":   upgrade.StateDownloading,
	})
}

// handleSystemStatus handles GET /api/system/status
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	exePath, _ := os.Executable()

	state, _ := upgrade.NewUpgrade("data").StateManager().Read()
	upgradeState := upgrade.StateIdle
	if state != nil {
		upgradeState = state.Upgrade.State
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pid":             os.Getpid(),
		"version":         version.Version,
		"uptime_seconds":  int(time.Since(time.Now()).Seconds()), // placeholder, should track actual start time
		"started_at":      time.Now().UTC().Format(time.RFC3339), // placeholder
		"upgrade_state":   upgradeState,
		"listeners":       []string{s.config.Addr},
		"executable_path": exePath,
		"go_version":      runtime.Version(),
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
	})
}

// handleUpgradeStatus handles GET /api/system/upgrade/status
func (s *Server) handleUpgradeStatus(w http.ResponseWriter, r *http.Request) {
	state, err := upgrade.NewUpgrade("data").StateManager().Read()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read state: "+err.Error())
		return
	}

	if state == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"state": upgrade.StateIdle,
		})
		return
	}

	resp := map[string]interface{}{
		"state":        state.Upgrade.State,
		"new_pid":      state.Upgrade.NewPID,
		"started_at":   nil,
		"completed_at": nil,
		"error":        state.Upgrade.Error,
		"binary_path":  state.Upgrade.BinaryPath,
	}

	if !state.Upgrade.StartedAt.IsZero() {
		resp["started_at"] = state.Upgrade.StartedAt.Format(time.RFC3339)
	}
	if !state.Upgrade.CompletedAt.IsZero() {
		resp["completed_at"] = state.Upgrade.CompletedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleUpgradeHandoff is called by the old process when upgrade reaches ready/completed state
func (s *Server) handleUpgradeHandoff() {
	log.Printf("[upgrade] Handoff initiated, starting drain...")

	// Give a short grace period for the response to be sent
	time.Sleep(500 * time.Millisecond)

	// Close all WebSocket/SSH connections
	count := s.CloseAllSSHConnections()
	if count > 0 {
		log.Printf("[upgrade] Closed %d SSH connections", count)
	}

	// Stop accepting new connections by shutting down the server
	// The actual server shutdown is handled by the main goroutine
	// Here we just signal that draining has started
	log.Printf("[upgrade] Drain started, waiting for connections to close...")

	// Wait up to 30 seconds for connections to drain
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[upgrade] Drain complete")
	case <-time.After(30 * time.Second):
		log.Printf("[upgrade] Drain timeout, forcing exit")
	}

	log.Printf("[upgrade] Old process exiting")
	os.Exit(0)
}

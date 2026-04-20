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
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/models"
	"github.com/casuallc/vigil/version"
)

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetInfo handles GET /api/info endpoint
func (s *Server) handleGetInfo(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	var ifaces []NetworkInterface
	netIfaces, _ := net.Interfaces()
	for _, iface := range netIfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ifaces = append(ifaces, NetworkInterface{
					MAC:     iface.HardwareAddr.String(),
					Network: addr.String(),
					IP:      ipnet.IP.String(),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hostname":    hostname,
		"interfaces":  ifaces,
		"version":     version.GetVersionInfo(),
		"arch":        runtime.GOARCH,
		"os":          runtime.GOOS,
	})
}

// handleGetConfig handles GET /api/config endpoint
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config)
}

// handleUpdateConfig handles PUT /api/config endpoint
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save the new configuration
	if err := config.SaveConfig("./conf/config.yaml", &newConfig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update the in-memory configuration
	s.config = &newConfig

	writeJSON(w, http.StatusOK, map[string]string{"message": "Config updated successfully"})
}

// handleExecuteCommand handles the POST /api/exec endpoint
func (s *Server) handleExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string   `json:"command"`
		Env     []string `json:"env"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate command is not empty
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "Command cannot be empty")
		return
	}

	// Execute command using common package
	output, err := common.ExecuteCommand(req.Command, req.Env)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Command execution failed: %v, output: %s", err, output))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// getCurrentUser gets the current request's username
func (s *Server) getCurrentUser(r *http.Request) string {
	if s.config.BasicAuth.Enabled {
		if username, _, ok := r.BasicAuth(); ok {
			return username
		}
	}
	return "anonymous"
}

// isAdmin checks if the current user is an admin
func (s *Server) isAdmin(r *http.Request) bool {
	if !s.config.BasicAuth.Enabled {
		return true
	}

	username, _, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// Check if super admin
	if username == s.config.BasicAuth.Username {
		return true
	}

	// Check user role
	if s.userDatabase != nil {
		if user, exists := s.userDatabase.GetUser(username); exists && user.Role == "admin" {
			return true
		}
	}

	return false
}

// boolToInt converts bool to int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// recordCommandHistory records command execution history
func (s *Server) recordCommandHistory(vmName, command, executedBy, status string, durationMs int64, output, errorMsg string) {
	if s.commandHistoryDB == nil {
		return
	}
	_, _ = s.commandHistoryDB.Exec(
		"INSERT INTO command_history (id, vm_name, command, executed_by, executed_at, status, duration_ms, output, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		fmt.Sprintf("hist_%d", time.Now().UnixNano()),
		vmName,
		command,
		executedBy,
		time.Now(),
		status,
		durationMs,
		output,
		errorMsg,
	)
}

// dedupMounts deduplicates mount list: prioritize by ID; if no ID, use (Type|Target|Source|Name)
func dedupMounts(mounts []models.Mount) []models.Mount {
	if len(mounts) == 0 {
		return mounts
	}
	seenID := make(map[string]struct{}, len(mounts))
	seenKey := make(map[string]struct{}, len(mounts))
	var uniq []models.Mount
	for _, m := range mounts {
		if m.ID != "" {
			if _, ok := seenID[m.ID]; ok {
				continue
			}
			seenID[m.ID] = struct{}{}
			uniq = append(uniq, m)
			continue
		}
		key := fmt.Sprintf("%s|%s|%s|%s", m.Type, m.Target, m.Source, m.Name)
		if _, ok := seenKey[key]; ok {
			continue
		}
		seenKey[key] = struct{}{}
		uniq = append(uniq, m)
	}
	return uniq
}

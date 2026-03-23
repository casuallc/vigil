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
	"time"

	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
)

// ------------------------- VM Management Handlers -------------------------

// handleAddVM handles adding a VM
func (s *Server) handleAddVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	var newVM vm.VM
	if err := json.NewDecoder(r.Body).Decode(&newVM); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Use name from URL parameter, ignore name in request body
	newVM.Name = vmName

	if err := s.vmManager.AddVM(&newVM); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "VM added successfully"})
}

// handleListVMs handles listing all VMs
func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
	vms := s.vmManager.ListVMs()

	// Create a VM list without passwords
	var vmsWithoutPassword []map[string]interface{}
	for _, vm := range vms {
		vmMap := map[string]interface{}{
			"name":     vm.Name,
			"ip":       vm.IP,
			"port":     vm.Port,
			"username": vm.Username,
			"key_path": vm.KeyPath,
		}
		vmsWithoutPassword = append(vmsWithoutPassword, vmMap)
	}

	writeJSON(w, http.StatusOK, vmsWithoutPassword)
}

// handleGetVM handles getting VM details
func (s *Server) handleGetVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	vmInstance, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Create a VM without password
	vmWithoutPassword := map[string]interface{}{
		"name":     vmInstance.Name,
		"ip":       vmInstance.IP,
		"port":     vmInstance.Port,
		"username": vmInstance.Username,
		"key_path": vmInstance.KeyPath,
	}

	writeJSON(w, http.StatusOK, vmWithoutPassword)
}

// handleDeleteVM handles deleting a VM
func (s *Server) handleDeleteVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	if err := s.vmManager.RemoveVM(vmName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "VM deleted successfully"})
}

// handleUpdateVM handles updating a VM
func (s *Server) handleUpdateVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	// Get VM info
	vmInfo, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Parse request body
	var updateData struct {
		Password string `json:"password"`
		KeyPath  string `json:"key_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update password and key path
	if updateData.Password != "" {
		vmInfo.Password = updateData.Password
	}
	if updateData.KeyPath != "" {
		vmInfo.KeyPath = updateData.KeyPath
	}

	// Save VM info
	if err := s.vmManager.SaveVMs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "VM updated successfully"})
}

// handleVMExec handles VM command execution
func (s *Server) handleVMExec(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	// Parse request body
	var req struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate command is not empty
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command cannot be empty")
		return
	}

	// Set default timeout
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	// Get VM info
	vmInfo, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Create SSH client
	sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
		Host:     vmInfo.IP,
		Port:     vmInfo.Port,
		Username: vmInfo.Username,
		Password: vmInfo.Password,
		KeyPath:  vmInfo.KeyPath,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Connect to SSH server
	if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	// Execute command
	output, err := sshClient.ExecuteCommand(req.Command)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return command output
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// handleVMPing handles VM ping (TCP connection test)
func (s *Server) handleVMPing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	// Get VM info
	vmInfo, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Test TCP connection
	addr := fmt.Sprintf("%s:%d", vmInfo.IP, vmInfo.Port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	latency := time.Since(start)

	if err != nil {
		// Connection failed
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"status":  "TIMEOUT",
			"message": err.Error(),
		})
		return
	}
	defer conn.Close()

	// Connection successful
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "OK",
		"latency": latency.String(),
	})
}

// ------------------------- Group Management Handlers -------------------------

// handleAddGroup handles adding a VM group
func (s *Server) handleAddGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["name"]

	// Parse request body
	var groupData struct {
		Description string   `json:"description"`
		VMs         []string `json:"vms"`
		IsShared    bool     `json:"is_shared"`
		SharedWith  []string `json:"shared_with"`
	}
	if err := json.NewDecoder(r.Body).Decode(&groupData); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get current user as owner
	owner := s.getCurrentUser(r)

	// Add group using name from URL parameter
	if err := s.vmManager.AddGroup(groupName, groupData.Description, owner, groupData.VMs, groupData.IsShared, groupData.SharedWith); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Group added successfully"})
}

// handleListGroups handles listing VM groups
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	// Get all groups
	allGroups := s.vmManager.ListGroups()

	// Get query parameters
	sharedOnly := r.URL.Query().Get("shared_only") == "true"
	mineOnly := r.URL.Query().Get("mine_only") == "true"
	username := s.getCurrentUser(r)
	isAdmin := s.isAdmin(r)

	// Filter groups
	var groups []*vm.Group
	for _, group := range allGroups {
		// Admin can see all groups
		if isAdmin {
			groups = append(groups, group)
			continue
		}

		// Only get shared groups
		if sharedOnly {
			if group.IsShared {
				groups = append(groups, group)
			}
			continue
		}

		// Only get groups created by current user
		if mineOnly {
			if group.Owner == username {
				groups = append(groups, group)
			}
			continue
		}

		// Default: show shared groups and groups created by current user
		if group.IsShared || group.Owner == username {
			groups = append(groups, group)
		}
	}

	writeJSON(w, http.StatusOK, groups)
}

// handleGetGroup handles getting VM group details
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["name"]

	// Get group
	group, err := s.vmManager.GetGroup(groupName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, group)
}

// handleUpdateGroup handles updating a VM group
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["name"]

	// Parse request body
	var updateData struct {
		Description string   `json:"description"`
		VMs         []string `json:"vms"`
		IsShared    bool     `json:"is_shared"`
		SharedWith  []string `json:"shared_with"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check permission
	group, err := s.vmManager.GetGroup(groupName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	username := s.getCurrentUser(r)
	if group.Owner != username && !s.isAdmin(r) {
		writeError(w, http.StatusForbidden, "Permission denied: only owner or admin can update group")
		return
	}

	// Update group
	if err := s.vmManager.UpdateGroup(groupName, updateData.Description, updateData.VMs, updateData.IsShared, updateData.SharedWith); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Group updated successfully"})
}

// handleDeleteGroup handles deleting a VM group
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupName := vars["name"]

	// Delete group
	if err := s.vmManager.RemoveGroup(groupName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

// ------------------------- Permission Handlers -------------------------

// handleAddPermission handles adding permissions
func (s *Server) handleAddPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	type PermissionRequest struct {
		Username    string   `json:"username"`
		Permissions []string `json:"permissions"`
	}

	var req PermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.vmManager.GetPermissionManager().AddPermission(
		vmName, req.Username, req.Permissions); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Permission added successfully"})
}

// handleRemovePermission handles removing permissions
func (s *Server) handleRemovePermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	type PermissionRequest struct {
		Username    string   `json:"username"`
		Permissions []string `json:"permissions"`
	}

	var req PermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.vmManager.GetPermissionManager().RemovePermission(
		vmName, req.Username, req.Permissions); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Permission removed successfully"})
}

// handleListPermissions handles listing permissions
func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	permissions := s.vmManager.GetPermissionManager().ListPermissions(vmName)
	writeJSON(w, http.StatusOK, permissions)
}

// handleCheckPermission handles checking permissions
func (s *Server) handleCheckPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	type CheckPermissionRequest struct {
		Username   string `json:"username"`
		Permission string `json:"permission"`
	}

	var req CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hasPermission := s.vmManager.GetPermissionManager().CheckPermission(
		vmName, req.Username, req.Permission)

	writeJSON(w, http.StatusOK, map[string]bool{"has_permission": hasPermission})
}

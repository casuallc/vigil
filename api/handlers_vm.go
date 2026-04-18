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
  "path/filepath"
  "time"

  "github.com/casuallc/vigil/vm"
  "github.com/gorilla/mux"
)

// handleVmFileList handles listing files in a VM
func (s *Server) handleVmFileList(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type FileListRequest struct {
    Path     string `json:"path"`
    MaxDepth int    `json:"max_depth"`
  }

  var req FileListRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Get file list
  files, err := sshClient.ListFiles(req.Path, req.MaxDepth)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, files)
}

// handleVmFileUpload handles uploading files to a VM
func (s *Server) handleVmFileUpload(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  // Parse multipart/form-data request
  err := r.ParseMultipartForm(10 << 20) // 10MB limit
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // Get target path
  targetPath := r.FormValue("target_path")
  if targetPath == "" {
    writeError(w, http.StatusBadRequest, "target_path is required")
    return
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

  // Get uploaded file
  file, _, err := r.FormFile("file")
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }
  defer file.Close()

  // Upload file
  if err := sshClient.UploadFile(file, targetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleVmFileStreamUpload handles uploading large files to a VM via raw body stream
func (s *Server) handleVmFileStreamUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm name is required in path")
		return
	}

	// Get target path from header
	targetPath := r.Header.Get("X-Target-Path")
	if targetPath == "" {
		writeError(w, http.StatusBadRequest, "X-Target-Path header is required")
		return
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

	// Upload file from request body (streaming, no memory limit)
	if err := sshClient.UploadFile(r.Body, targetPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleVmFileDownload handles downloading files from a VM
func (s *Server) handleVmFileDownload(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type FileDownloadRequest struct {
    SourcePath string `json:"source_path"`
  }

  var req FileDownloadRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Download file
  content, err := sshClient.DownloadFile(req.SourcePath)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  // Set response headers
  w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.SourcePath)))
  w.Header().Set("Content-Type", "application/octet-stream")

  // Write response
  if _, err := w.Write(content); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
}

// handleVmFileDelete handles deleting files in a VM
func (s *Server) handleVmFileDelete(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type FileDeleteRequest struct {
    Path string `json:"path"`
  }

  var req FileDeleteRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Delete file
  if _, err := sshClient.ExecuteCommand("rm " + req.Path); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}

// handleVmFileMkdir handles creating directories in a VM
func (s *Server) handleVmFileMkdir(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type MkdirRequest struct {
    Path    string `json:"path"`
    Parents bool   `json:"parents"`
  }

  var req MkdirRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Create directory
  cmd := "mkdir"
  if req.Parents {
    cmd += " -p"
  }
  cmd += " " + req.Path

  if _, err := sshClient.ExecuteCommand(cmd); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Directory created successfully"})
}

// handleVmFileTouch handles creating files in a VM
func (s *Server) handleVmFileTouch(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type TouchRequest struct {
    Path string `json:"path"`
  }

  var req TouchRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Create file
  if _, err := sshClient.ExecuteCommand("touch " + req.Path); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File created successfully"})
}

// handleVmFileRmdir handles deleting directories in a VM
func (s *Server) handleVmFileRmdir(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  type RmdirRequest struct {
    Path      string `json:"path"`
    Recursive bool   `json:"recursive"`
  }

  var req RmdirRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
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

  // Delete directory
  cmd := "rm"
  if req.Recursive {
    cmd += " -r"
  }
  cmd += " " + req.Path

  if _, err := sshClient.ExecuteCommand(cmd); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Directory deleted successfully"})
}

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

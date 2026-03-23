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
	"net/http"
	"path/filepath"

	"github.com/casuallc/vigil/file"
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

// ------------------------- Local File Management Handlers -------------------------

// handleFileList handles listing local files
func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	type FileListRequest struct {
		Path     string `json:"path"`
		MaxDepth int    `json:"max_depth"`
	}

	var req FileListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create file manager
	fileManager := file.NewManager("")

	// Get file list
	files, err := fileManager.ListFiles(req.Path, req.MaxDepth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, files)
}

// handleFileUpload handles uploading local files
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
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

	// Get uploaded file
	uploadedFile, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer uploadedFile.Close()

	// Create file manager
	fileManager := file.NewManager("")

	// Upload file
	if err := fileManager.UploadFile(uploadedFile, targetPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleFileDownload handles downloading local files
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	type FileDownloadRequest struct {
		SourcePath string `json:"source_path"`
	}

	var req FileDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create file manager
	fileManager := file.NewManager("")

	// Download file
	content, err := fileManager.DownloadFile(req.SourcePath)
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

// handleFileDelete handles deleting local files
func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	type FileDeleteRequest struct {
		Path string `json:"path"`
	}

	var req FileDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create file manager
	fileManager := file.NewManager("")

	// Delete file
	if err := fileManager.DeleteFile(req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}

// handleFileCopy handles copying local files
func (s *Server) handleFileCopy(w http.ResponseWriter, r *http.Request) {
	type FileCopyRequest struct {
		SourcePath string `json:"source_path"`
		TargetPath string `json:"target_path"`
	}

	var req FileCopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create file manager
	fileManager := file.NewManager("")

	// Copy file
	if err := fileManager.CopyFile(req.SourcePath, req.TargetPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File copied successfully"})
}

// handleFileMove handles moving local files
func (s *Server) handleFileMove(w http.ResponseWriter, r *http.Request) {
	type FileMoveRequest struct {
		SourcePath string `json:"source_path"`
		TargetPath string `json:"target_path"`
	}

	var req FileMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create file manager
	fileManager := file.NewManager("")

	// Move file
	if err := fileManager.MoveFile(req.SourcePath, req.TargetPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File moved successfully"})
}

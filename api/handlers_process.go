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
	"strconv"

	"github.com/casuallc/vigil/models"
	"github.com/casuallc/vigil/proc"
	"github.com/gorilla/mux"
)

// handleScanProcesses handles process scanning
func (s *Server) handleScanProcesses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	processes, err := s.manager.ScanProcesses(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, processes)
}

// handleAddProcess handles adding a new managed process
func (s *Server) handleAddProcess(w http.ResponseWriter, r *http.Request) {
	var process models.ManagedProcess
	if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.manager.CreateProcess(process); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Process managed successfully"})
}

// handleStopProcess handles stopping a managed process
func (s *Server) handleStopProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	if err := s.manager.StopProcess(namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process stopped successfully"})
}

// handleRestartProcess handles restarting a managed process
func (s *Server) handleRestartProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	if err := s.manager.RestartProcess(namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process restarted successfully"})
}

// handleGetProcess handles getting process details
func (s *Server) handleGetProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	process, err := s.manager.GetProcessStatus(namespace, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, process)
}

// handleListProcesses handles listing all managed processes
func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)

	// Support legacy API: return all processes if no namespace specified
	processes, err := s.manager.ListManagedProcesses(namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, processes)
}

// handleStartProcess handles starting a managed process
func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	// Support legacy API: use "default" if no namespace specified
	if namespace == "" {
		namespace = "default"
	}

	if err := s.manager.StartProcess(namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleDeleteProcess handles deleting a managed process
func (s *Server) handleDeleteProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	err := s.manager.DeleteProcess(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Process %s deleted successfully", name)))
}

// handleEditProcess handles updating a managed process
func (s *Server) handleEditProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	var updatedProcess models.ManagedProcess
	if err := json.NewDecoder(r.Body).Decode(&updatedProcess); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure namespace and name match URL parameters
	updatedProcess.Metadata.Namespace = namespace
	updatedProcess.Metadata.Name = name

	// Get original process to preserve status
	originalProcess, err := s.manager.GetProcessStatus(namespace, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Preserve original status
	updatedProcess.Status = originalProcess.Status

	// Deduplicate mounts before saving
	updatedProcess.Spec.Mounts = dedupMounts(updatedProcess.Spec.Mounts)

	// Update process
	key := fmt.Sprintf("%s/%s", namespace, name)
	s.manager.GetProcesses()[key] = &updatedProcess

	// Save updated process info
	if err := s.manager.SaveManagedProcesses(proc.ProcessesFilePath); err != nil {
		fmt.Printf("Warning: failed to save managed processes: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process updated successfully"})
}

// handleGetSystemResources handles system resource monitoring
func (s *Server) handleGetSystemResources(w http.ResponseWriter, r *http.Request) {
	// Try to get from cache first
	if resources, found := s.resourceMonitor.GetCachedSystemResources(); found {
		writeJSON(w, http.StatusOK, resources)
		return
	}

	// Fall back to real-time collection
	resources, err := proc.GetSystemResourceUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

// handleGetProcessResources handles process resource monitoring
func (s *Server) handleGetProcessResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid PID")
		return
	}

	// Try to get from cache first
	if resources, found := s.resourceMonitor.GetCachedProcessResources(pid); found {
		writeJSON(w, http.StatusOK, resources)
		return
	}

	// Fall back to real-time collection
	resources, err := proc.GetUnixProcessResourceUsage(pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

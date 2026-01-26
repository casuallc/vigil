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
	"github.com/gorilla/mux"
)

// Router 定义API路由注册函数
func (s *Server) Router() *mux.Router {
	r := mux.NewRouter()

	// Process management endpoints
	r.HandleFunc("/api/processes/scan", s.handleScanProcesses).Methods("GET")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/add", s.handleAddProcess).Methods("POST")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/start", s.handleStartProcess).Methods("POST")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/stop", s.handleStopProcess).Methods("POST")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/restart", s.handleRestartProcess).Methods("POST")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}", s.handleGetProcess).Methods("GET")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}", s.handleEditProcess).Methods("PUT")
	r.HandleFunc("/api/namespaces/{namespace}/processes/{name}", s.handleDeleteProcess).Methods("DELETE")
	r.HandleFunc("/api/namespaces/{namespace}/processes", s.handleListProcesses).Methods("GET")

	// Resource monitoring endpoints
	r.HandleFunc("/api/resources/system", s.handleGetSystemResources).Methods("GET")
	r.HandleFunc("/api/resources/process/{pid}", s.handleGetProcessResources).Methods("GET")

	// Configuration endpoints
	r.HandleFunc("/api/config", s.handleGetConfig).Methods("GET")
	r.HandleFunc("/api/config", s.handleUpdateConfig).Methods("PUT")

	// Health check
	r.HandleFunc("/health", s.handleHealthCheck).Methods("GET")

	// Execute command endpoint
	r.HandleFunc("/api/exec", s.handleExecuteCommand).Methods("POST")

	// Cosmic inspection endpoint
	r.HandleFunc("/api/inspect", s.handleCosmicInspect).Methods("POST")

	// VM Server Management endpoints
	r.HandleFunc("/api/vms/servers/{name}", s.handleAddVM).Methods("POST")
	r.HandleFunc("/api/vms/servers", s.handleListVMs).Methods("GET")
	r.HandleFunc("/api/vms/servers/{name}", s.handleGetVM).Methods("GET")
	r.HandleFunc("/api/vms/servers/{name}", s.handleUpdateVM).Methods("PUT")
	r.HandleFunc("/api/vms/servers/{name}", s.handleDeleteVM).Methods("DELETE")

	// VM Group Management endpoints
	r.HandleFunc("/api/vms/groups/{name}", s.handleAddGroup).Methods("POST")
	r.HandleFunc("/api/vms/groups", s.handleListGroups).Methods("GET")
	r.HandleFunc("/api/vms/groups/{name}", s.handleGetGroup).Methods("GET")
	r.HandleFunc("/api/vms/groups/{name}", s.handleUpdateGroup).Methods("PUT")
	r.HandleFunc("/api/vms/groups/{name}", s.handleDeleteGroup).Methods("DELETE")

	// VM SSH endpoints
	r.HandleFunc("/api/vms/ssh/ws", s.handleSSHWebSocket)

	// VM File Management endpoints
	r.HandleFunc("/api/vms/files/{name}/upload", s.handleVmFileUpload).Methods("POST")
	r.HandleFunc("/api/vms/files/{name}/download", s.handleVmFileDownload).Methods("POST")
	r.HandleFunc("/api/vms/files/{name}/list", s.handleVmFileList).Methods("POST")

	// VM Permission endpoints
	r.HandleFunc("/api/vms/permissions/{name}", s.handleAddPermission).Methods("POST")
	r.HandleFunc("/api/vms/permissions/{name}", s.handleRemovePermission).Methods("DELETE")
	r.HandleFunc("/api/vms/permissions/{name}/check", s.handleCheckPermission).Methods("POST")
	r.HandleFunc("/api/vms/servers/{name}/permissions", s.handleListPermissions).Methods("GET")

	// File Management endpoints
	r.HandleFunc("/api/files/upload", s.handleFileUpload).Methods("POST")
	r.HandleFunc("/api/files/download", s.handleFileDownload).Methods("POST")
	r.HandleFunc("/api/files/list", s.handleFileList).Methods("POST")
	r.HandleFunc("/api/files/delete", s.handleFileDelete).Methods("POST")
	r.HandleFunc("/api/files/copy", s.handleFileCopy).Methods("POST")
	r.HandleFunc("/api/files/move", s.handleFileMove).Methods("POST")
	return r
}

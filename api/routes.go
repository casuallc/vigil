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

  // License endpoint
  r.HandleFunc("/api/license", s.handleGetLicense).Methods("GET")

  // File stream upload endpoint (for large files)
  r.HandleFunc("/api/files/stream", s.handleFileStreamUpload).Methods("POST")

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
  r.HandleFunc("/api/vms/ssh/connections", s.handleListSSHConnections).Methods("GET")
  r.HandleFunc("/api/vms/ssh/connections", s.handleCloseAllSSHConnections).Methods("DELETE")
  r.HandleFunc("/api/vms/ssh/connections/{id}", s.handleCloseSSHConnection).Methods("DELETE")

  // VM File Management endpoints
  r.HandleFunc("/api/vms/files/{name}/upload", s.handleVmFileUpload).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/download", s.handleVmFileDownload).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/list", s.handleVmFileList).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/delete", s.handleVmFileDelete).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/mkdir", s.handleVmFileMkdir).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/touch", s.handleVmFileTouch).Methods("POST")
  r.HandleFunc("/api/vms/files/{name}/rmdir", s.handleVmFileRmdir).Methods("POST")

  // VM File Stream Upload endpoint (for large files)
  r.HandleFunc("/api/vms/files/{name}/stream", s.handleVmFileStreamUpload).Methods("POST")

  // VM Permission endpoints
  r.HandleFunc("/api/vms/permissions/{name}", s.handleAddPermission).Methods("POST")
  r.HandleFunc("/api/vms/permissions/{name}", s.handleRemovePermission).Methods("DELETE")
  r.HandleFunc("/api/vms/permissions/{name}/check", s.handleCheckPermission).Methods("POST")
  r.HandleFunc("/api/vms/servers/{name}/permissions", s.handleListPermissions).Methods("GET")

  // VM Exec and Ping endpoints
  r.HandleFunc("/api/vms/servers/{name}/exec", s.handleVMExec).Methods("POST")
  r.HandleFunc("/api/vms/servers/{name}/ping", s.handleVMPing).Methods("GET")

  // VM Batch Operations endpoints
  r.HandleFunc("/api/vms/batch/exec", s.handleBatchExec).Methods("POST")
  r.HandleFunc("/api/vms/servers/{name}/resources", s.handleGetVMResources).Methods("GET")
  r.HandleFunc("/api/vms/resources/batch", s.handleBatchGetVMResources).Methods("POST")
  r.HandleFunc("/api/vms/files/transfer", s.handleVMFileTransfer).Methods("POST")

  // Command Template endpoints
  r.HandleFunc("/api/commands/templates", s.handleListCommandTemplates).Methods("GET")
  r.HandleFunc("/api/commands/templates", s.handleCreateCommandTemplate).Methods("POST")
  r.HandleFunc("/api/commands/templates/{id}", s.handleGetCommandTemplate).Methods("GET")
  r.HandleFunc("/api/commands/templates/{id}", s.handleUpdateCommandTemplate).Methods("PUT")
  r.HandleFunc("/api/commands/templates/{id}", s.handleDeleteCommandTemplate).Methods("DELETE")

  // Command History endpoints
  r.HandleFunc("/api/commands/history", s.handleListCommandHistory).Methods("GET")
  r.HandleFunc("/api/commands/history", s.handleRecordCommandHistory).Methods("POST")
  r.HandleFunc("/api/commands/history/{id}", s.handleDeleteCommandHistory).Methods("DELETE")

  // Schedule endpoints
  r.HandleFunc("/api/schedules", s.handleListSchedules).Methods("GET")
  r.HandleFunc("/api/schedules", s.handleCreateSchedule).Methods("POST")
  r.HandleFunc("/api/schedules/{id}", s.handleGetSchedule).Methods("GET")
  r.HandleFunc("/api/schedules/{id}", s.handleUpdateSchedule).Methods("PUT")
  r.HandleFunc("/api/schedules/{id}", s.handleDeleteSchedule).Methods("DELETE")
  r.HandleFunc("/api/schedules/{id}/toggle", s.handleToggleSchedule).Methods("POST")
  r.HandleFunc("/api/schedules/{id}/run", s.handleRunSchedule).Methods("POST")
  r.HandleFunc("/api/schedules/{id}/history", s.handleGetScheduleHistory).Methods("GET")

  // AI Assistant endpoints
  r.HandleFunc("/api/ai/generate-command", s.handleAIGenerateCommand).Methods("POST")
  r.HandleFunc("/api/ai/explain-command", s.handleAIExplainCommand).Methods("POST")
  r.HandleFunc("/api/ai/fix-command", s.handleAIFixCommand).Methods("POST")

  // File Management endpoints
  r.HandleFunc("/api/files/upload", s.handleFileUpload).Methods("POST")
  r.HandleFunc("/api/files/download", s.handleFileDownload).Methods("POST")
  r.HandleFunc("/api/files/list", s.handleFileList).Methods("POST")
  r.HandleFunc("/api/files/delete", s.handleFileDelete).Methods("POST")
  r.HandleFunc("/api/files/copy", s.handleFileCopy).Methods("POST")
  r.HandleFunc("/api/files/move", s.handleFileMove).Methods("POST")

  // User management endpoints
  r.HandleFunc("/api/users/register", s.handleRegisterUser).Methods("POST")
  r.HandleFunc("/api/users/login", s.handleUserLogin).Methods("POST")
  r.HandleFunc("/api/users", s.handleListUsers).Methods("GET")
  r.HandleFunc("/api/users/{username}", s.handleGetUser).Methods("GET")
  r.HandleFunc("/api/users/{username}", s.handleUpdateUser).Methods("PUT")
  r.HandleFunc("/api/users/{username}", s.handleDeleteUser).Methods("DELETE")
  r.HandleFunc("/api/users/{username}/configs", s.handleGetUserConfigs).Methods("GET")
  r.HandleFunc("/api/users/{username}/configs", s.handleUpdateUserConfigs).Methods("PUT")

  return r
}

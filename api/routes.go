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

	// Prometheus metrics endpoint
	r.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Configuration endpoints
	r.HandleFunc("/api/config", s.handleGetConfig).Methods("GET")
	r.HandleFunc("/api/config", s.handleUpdateConfig).Methods("PUT")

	// Health check
	r.HandleFunc("/health", s.handleHealthCheck).Methods("GET")

	// Server info
	r.HandleFunc("/api/info", s.handleGetInfo).Methods("GET")

	// System management endpoints
	r.HandleFunc("/api/system/upgrade", s.handleSystemUpgrade).Methods("POST")
	r.HandleFunc("/api/system/status", s.handleSystemStatus).Methods("GET")
	r.HandleFunc("/api/system/upgrade/status", s.handleUpgradeStatus).Methods("GET")

	// Hosts management endpoint
	r.HandleFunc("/api/hosts", s.handleUpdateHosts).Methods("POST")

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
	r.HandleFunc("/api/files/mkdir", s.handleFileMkdir).Methods("POST")

	// File log streaming endpoint
	r.HandleFunc("/api/files/logs/stream", s.handleLogStream).Methods("GET")

	// Network endpoints
	r.HandleFunc("/api/network/probe", s.handleNetworkProbe).Methods("POST")

	// Auth endpoints
	r.HandleFunc("/api/auth/change-password", s.handleChangePassword).Methods("POST")

	// User management endpoints
	r.HandleFunc("/api/users/register", s.handleRegisterUser).Methods("POST")
	r.HandleFunc("/api/users/login", s.handleUserLogin).Methods("POST")
	r.HandleFunc("/api/users", s.handleListUsers).Methods("GET")
	r.HandleFunc("/api/users/{username}", s.handleGetUser).Methods("GET")
	r.HandleFunc("/api/users/{username}", s.handleUpdateUser).Methods("PUT")
	r.HandleFunc("/api/users/{username}", s.handleDeleteUser).Methods("DELETE")
	r.HandleFunc("/api/users/{username}/configs", s.handleGetUserConfigs).Methods("GET")
	r.HandleFunc("/api/users/{username}/configs", s.handleUpdateUserConfigs).Methods("PUT")

	// File-transfer agent endpoints (self-authenticated; registered when enabled)
	if s.filetransfer != nil {
		s.filetransfer.RegisterRoutes(r)
	}

	// Docker Registry HTTP API V2 endpoints (registered when enabled)
	if s.dockerRegistryManager != nil {
		r.HandleFunc("/v2/", s.handleDockerRegistryVersionCheck).Methods("GET")
		r.HandleFunc("/v2/_catalog", s.handleDockerRegistryCatalog).Methods("GET")
		r.HandleFunc("/v2/{name:.*}/tags/list", s.handleDockerRegistryTagsList).Methods("GET")
		r.HandleFunc("/v2/{name:.*}/manifests/{reference}", s.handleDockerRegistryManifestHead).Methods("HEAD")
		r.HandleFunc("/v2/{name:.*}/manifests/{reference}", s.handleDockerRegistryManifestGet).Methods("GET")
		r.HandleFunc("/v2/{name:.*}/manifests/{reference}", s.handleDockerRegistryManifestPut).Methods("PUT")
		r.HandleFunc("/v2/{name:.*}/manifests/{reference}", s.handleDockerRegistryManifestDelete).Methods("DELETE")
		r.HandleFunc("/v2/{name:.*}/blobs/uploads/", s.handleDockerRegistryBlobUploadInit).Methods("POST")
		r.HandleFunc("/v2/{name:.*}/blobs/uploads/{uuid}", s.handleDockerRegistryBlobUploadPatch).Methods("PATCH")
		r.HandleFunc("/v2/{name:.*}/blobs/uploads/{uuid}", s.handleDockerRegistryBlobUploadPut).Methods("PUT")
		r.HandleFunc("/v2/{name:.*}/blobs/uploads/{uuid}", s.handleDockerRegistryBlobUploadDelete).Methods("DELETE")
		r.HandleFunc("/v2/{name:.*}/blobs/{digest}", s.handleDockerRegistryBlobHead).Methods("HEAD")
		r.HandleFunc("/v2/{name:.*}/blobs/{digest}", s.handleDockerRegistryBlobGet).Methods("GET")
		r.HandleFunc("/v2/{name:.*}/blobs/{digest}", s.handleDockerRegistryBlobDelete).Methods("DELETE")
	}

	// Docker management endpoints (registered when daemon is available)
	if s.dockerManager != nil {
		// Docker Compose endpoints
		r.HandleFunc("/api/docker/compose", s.handleDockerComposeDeploy).Methods("POST")
		r.HandleFunc("/api/docker/compose/dir", s.handleDockerComposeDeployFromDir).Methods("POST")
		r.HandleFunc("/api/docker/compose/{project}", s.handleDockerComposeGet).Methods("GET")
		r.HandleFunc("/api/docker/compose/{project}", s.handleDockerComposeRemove).Methods("DELETE")
		r.HandleFunc("/api/docker/compose-version", s.handleDockerComposeVersion).Methods("GET")
		r.HandleFunc("/api/docker/ping", s.handleDockerPing).Methods("GET")
		r.HandleFunc("/api/docker/version", s.handleDockerVersion).Methods("GET")
		r.HandleFunc("/api/docker/images/load", s.handleDockerLoadImage).Methods("POST")
		r.HandleFunc("/api/docker/images/load", s.handleDockerLoadImageList).Methods("GET")
		r.HandleFunc("/api/docker/images/load/{id}", s.handleDockerLoadImageDelete).Methods("DELETE")
		r.HandleFunc("/api/docker/images/load/{id}/status", s.handleDockerLoadImageStatus).Methods("GET")
		r.HandleFunc("/api/docker/images", s.handleDockerListImages).Methods("GET")
		r.HandleFunc("/api/docker/images/{id}", s.handleDockerInspectImage).Methods("GET")
		r.HandleFunc("/api/docker/images/{id}", s.handleDockerRemoveImage).Methods("DELETE")
		r.HandleFunc("/api/docker/images/{id}/history", s.handleDockerImageHistory).Methods("GET")
		r.HandleFunc("/api/docker/images/pull", s.handleDockerPullImage).Methods("POST")
		r.HandleFunc("/api/docker/images/tag", s.handleDockerTagImage).Methods("POST")
		r.HandleFunc("/api/docker/containers", s.handleDockerListContainers).Methods("GET")
		r.HandleFunc("/api/docker/containers", s.handleDockerCreateContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}", s.handleDockerInspectContainer).Methods("GET")
		r.HandleFunc("/api/docker/containers/{id}", s.handleDockerRemoveContainer).Methods("DELETE")
		r.HandleFunc("/api/docker/containers/{id}/start", s.handleDockerStartContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/stop", s.handleDockerStopContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/restart", s.handleDockerRestartContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/pause", s.handleDockerPauseContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/unpause", s.handleDockerUnpauseContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/exec", s.handleDockerExecContainer).Methods("POST")
		r.HandleFunc("/api/docker/containers/{id}/logs", s.handleDockerStreamLogs).Methods("GET")
		r.HandleFunc("/api/docker/containers/{id}/stats", s.handleDockerStreamStats).Methods("GET")
		r.HandleFunc("/api/docker/containers/{id}/exec/ws", s.handleDockerExecWebSocket)
		r.HandleFunc("/api/docker/containers/{id}/logs/ws", s.handleDockerLogsWebSocket)
	}

	return r
}

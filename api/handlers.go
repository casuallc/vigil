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
	"strings"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/inspection"
	"github.com/casuallc/vigil/proc"
	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
)

// 以下是所有的处理函数实现（保持不变）
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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

func (s *Server) handleAddProcess(w http.ResponseWriter, r *http.Request) {
	var process proc.ManagedProcess
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

// 处理函数更新示例
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

func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)

	// 兼容旧版API，没有指定namespace时返回所有进程
	processes, err := s.manager.ListManagedProcesses(namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, processes)
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	// 兼容旧版API，没有指定namespace时使用"default"
	if namespace == "" {
		namespace = "default"
	}

	if err := s.manager.StartProcess(namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleGetSystemResources(w http.ResponseWriter, r *http.Request) {
	resources, err := proc.GetSystemResourceUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetProcessResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid PID")
		return
	}

	resources, err := proc.GetUnixProcessResourceUsage(pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save the new configuration
	if err := config.SaveConfig("config.yaml", &newConfig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update the in-memory configuration
	s.config = &newConfig

	writeJSON(w, http.StatusOK, map[string]string{"message": "Config updated successfully"})
}

// handleDeleteProcess handles the DELETE /api/processes/{namespace}/{name} endpoint
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

	// 验证命令不为空
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "Command cannot be empty")
		return
	}

	// 使用common包中的ExecuteCommand函数执行命令
	output, err := common.ExecuteCommand(req.Command, req.Env)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Command execution failed: %v, output: %s", err, output))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// handleEditProcess handles updating a managed proc
func (s *Server) handleEditProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := getNamespace(vars)
	name := vars["name"]

	var updatedProcess proc.ManagedProcess
	if err := json.NewDecoder(r.Body).Decode(&updatedProcess); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 确保请求中的命名空间和名称与URL中的一致
	updatedProcess.Metadata.Namespace = namespace
	updatedProcess.Metadata.Name = name

	// 获取原始进程以保留状态信息
	originalProcess, err := s.manager.GetProcessStatus(namespace, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 保留原始状态信息
	updatedProcess.Status = originalProcess.Status

	// 在持久化前对挂载列表进行去重（优先按 ID）
	updatedProcess.Spec.Mounts = dedupMounts(updatedProcess.Spec.Mounts)

	// 更新进程配置
	key := fmt.Sprintf("%s/%s", namespace, name)
	s.manager.GetProcesses()[key] = &updatedProcess

	// 保存更新后的进程信息
	if err := s.manager.SaveManagedProcesses(proc.ProcessesFilePath); err != nil {
		fmt.Printf("Warning: failed to save managed processes: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process updated successfully"})
}

// 对挂载列表执行去重：优先按 ID 唯一；若无 ID，则按 (Type|Target|Source|Name) 唯一
func dedupMounts(mounts []proc.Mount) []proc.Mount {
	if len(mounts) == 0 {
		return mounts
	}
	seenID := make(map[string]struct{}, len(mounts))
	seenKey := make(map[string]struct{}, len(mounts))
	var uniq []proc.Mount
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

// handleCosmicInspect 处理cosmic巡检请求
func (s *Server) handleCosmicInspect(w http.ResponseWriter, r *http.Request) {
	var req inspection.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// 验证请求参数
	if len(req.Checks) == 0 {
		writeError(w, http.StatusBadRequest, "No checks specified in request")
		return
	}

	// 执行检查
	var results []inspection.CheckResult
	totalChecks := len(req.Checks)
	passedChecks := 0
	warningChecks := 0
	errorChecks := 0

	for _, check := range req.Checks {
		result := inspection.ExecuteCheck(check, req.Env)
		results = append(results, result)

		// 统计检查结果
		switch strings.ToLower(result.Status) {
		case inspection.StatusOk:
			passedChecks++
		case inspection.StatusError:
			errorChecks++
		}
	}

	// 构建响应
	// 确定整体状态
	var overallStatus string = inspection.StatusOk
	if errorChecks > 0 {
		overallStatus = inspection.StatusError
	}

	response := inspection.Result{
		ID: req.Meta.JobName,
		Meta: inspection.ResultMeta{
			System:  req.Meta.System,
			Host:    req.Meta.Host,
			JobName: req.Meta.JobName,
			Time:    time.Now(),
			Status:  overallStatus,
		},
		Results: results,
		Summary: inspection.SummaryInfo{
			TotalChecks:   totalChecks,
			OK:            passedChecks,
			Warn:          warningChecks,
			Critical:      errorChecks,
			OverallStatus: overallStatus,
			StartedAt:     req.Meta.Time,
			FinishedAt:    time.Now(),
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// ------------------------- VM Management Handlers -------------------------

// handleAddVM 处理添加VM的请求
func (s *Server) handleAddVM(w http.ResponseWriter, r *http.Request) {
	var newVM vm.VM
	if err := json.NewDecoder(r.Body).Decode(&newVM); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.vmManager.AddVM(&newVM); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "VM added successfully"})
}

// handleListVMs 处理列出所有VM的请求
func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
	vms := s.vmManager.ListVMs()
	writeJSON(w, http.StatusOK, vms)
}

// handleGetVM 处理获取VM详情的请求
func (s *Server) handleGetVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	vmInstance, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, vmInstance)
}

// handleDeleteVM 处理删除VM的请求
func (s *Server) handleDeleteVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	if err := s.vmManager.RemoveVM(vmName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "VM deleted successfully"})
}

// ------------------------- SSH Handlers -------------------------

// handleSSHConnect 处理SSH连接请求
func (s *Server) handleSSHConnect(w http.ResponseWriter, r *http.Request) {
	var sshConfig vm.SSHConfig
	if err := json.NewDecoder(r.Body).Decode(&sshConfig); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(&sshConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接
	if err := sshClient.Connect(sshConfig.Host, sshConfig.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	writeJSON(w, http.StatusOK, map[string]string{"message": "SSH connected successfully"})
}

// handleSSHExecute 处理SSH命令执行请求
func (s *Server) handleSSHExecute(w http.ResponseWriter, r *http.Request) {
	type SSHExecuteRequest struct {
		SSHConfig vm.SSHConfig `json:"ssh_config"`
		Command   string       `json:"command"`
	}

	var req SSHExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(&req.SSHConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接
	if err := sshClient.Connect(req.SSHConfig.Host, req.SSHConfig.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	// 执行命令
	output, err := sshClient.ExecuteCommand(req.Command)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"output": output})
}

// ------------------------- File Management Handlers -------------------------

// handleFileUpload 处理文件上传请求
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	type FileUploadRequest struct {
		SSHConfig  vm.SSHConfig `json:"ssh_config"`
		LocalPath  string       `json:"local_path"`
		RemotePath string       `json:"remote_path"`
	}

	var req FileUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(&req.SSHConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接
	if err := sshClient.Connect(req.SSHConfig.Host, req.SSHConfig.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	// 创建文件管理器
	fileManager := vm.NewFileManager(sshClient)

	// 上传文件
	if err := fileManager.UploadFile(req.LocalPath, req.RemotePath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleFileDownload 处理文件下载请求
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	type FileDownloadRequest struct {
		SSHConfig  vm.SSHConfig `json:"ssh_config"`
		RemotePath string       `json:"remote_path"`
		LocalPath  string       `json:"local_path"`
	}

	var req FileDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(&req.SSHConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接
	if err := sshClient.Connect(req.SSHConfig.Host, req.SSHConfig.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	// 创建文件管理器
	fileManager := vm.NewFileManager(sshClient)

	// 下载文件
	if err := fileManager.DownloadFile(req.RemotePath, req.LocalPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File downloaded successfully"})
}

// handleFileList 处理列出文件的请求
func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	type FileListRequest struct {
		SSHConfig  vm.SSHConfig `json:"ssh_config"`
		RemotePath string       `json:"remote_path"`
	}

	var req FileListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(&req.SSHConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接
	if err := sshClient.Connect(req.SSHConfig.Host, req.SSHConfig.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	// 创建文件管理器
	fileManager := vm.NewFileManager(sshClient)

	// 获取文件列表
	files, err := fileManager.ListFiles(req.RemotePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, files)
}

// ------------------------- Permission Handlers -------------------------

// handleAddPermission 处理添加权限的请求
func (s *Server) handleAddPermission(w http.ResponseWriter, r *http.Request) {
	var permission vm.Permission
	if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.vmManager.GetPermissionManager().AddPermission(
		permission.VMName, permission.Username, permission.Permissions); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Permission added successfully"})
}

// handleRemovePermission 处理移除权限的请求
func (s *Server) handleRemovePermission(w http.ResponseWriter, r *http.Request) {
	var permission vm.Permission
	if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.vmManager.GetPermissionManager().RemovePermission(
		permission.VMName, permission.Username, permission.Permissions); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Permission removed successfully"})
}

// handleListPermissions 处理列出权限的请求
func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]

	permissions := s.vmManager.GetPermissionManager().ListPermissions(vmName)
	writeJSON(w, http.StatusOK, permissions)
}

// handleCheckPermission 处理检查权限的请求
func (s *Server) handleCheckPermission(w http.ResponseWriter, r *http.Request) {
	type CheckPermissionRequest struct {
		VMName     string `json:"vm_name"`
		Username   string `json:"username"`
		Permission string `json:"permission"`
	}

	var req CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hasPermission := s.vmManager.GetPermissionManager().CheckPermission(
		req.VMName, req.Username, req.Permission)

	writeJSON(w, http.StatusOK, map[string]bool{"has_permission": hasPermission})
}

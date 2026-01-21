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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/inspection"
	"github.com/casuallc/vigil/proc"
	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
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
	type SSHConnectRequest struct {
		VMName   string `json:"vm_name"`
		Password string `json:"password"`
	}

	var req SSHConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 获取指定的VM
	targetVM, err := s.vmManager.GetVM(req.VMName)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("VM not found: %v", err))
		return
	}

	// 创建SSH客户端配置
	sshConfig := &vm.SSHConfig{
		Host:     targetVM.IP,
		Port:     targetVM.Port,
		Username: targetVM.Username,
		Password: req.Password,
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(sshConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接到目标VM
	if err := sshClient.Connect(targetVM.IP, targetVM.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sshClient.Close()

	writeJSON(w, http.StatusOK, map[string]string{"message": "SSH connected successfully"})
}

// handleSSHExecute 处理SSH命令执行请求
func (s *Server) handleSSHExecute(w http.ResponseWriter, r *http.Request) {
	type SSHExecuteRequest struct {
		VMName  string `json:"vm_name"`
		Command string `json:"command"`
	}

	var req SSHExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 获取指定的VM
	targetVM, err := s.vmManager.GetVM(req.VMName)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("VM not found: %v", err))
		return
	}

	// 创建SSH客户端配置
	sshConfig := &vm.SSHConfig{
		Host:     targetVM.IP,
		Port:     targetVM.Port,
		Username: targetVM.Username,
		// 注意：VM对象中没有存储密码，这里需要从请求中获取密码
		// 或者在实际应用中，密码应该通过安全的方式存储和获取
		Password: "", // 暂时留空，实际应用中需要修改
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(sshConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接到目标VM
	if err := sshClient.Connect(targetVM.IP, targetVM.Port); err != nil {
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
	// 解析multipart/form-data请求
	err := r.ParseMultipartForm(10 << 20) // 10MB大小限制
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 获取目标路径
	targetPath := r.FormValue("target_path")
	if targetPath == "" {
		writeError(w, http.StatusBadRequest, "target_path is required")
		return
	}

	// 在Windows系统上，确保路径格式正确
	if runtime.GOOS == "windows" {
		// 直接检查是否以/tmp/开头，这样可以避免转义问题
		if strings.HasPrefix(targetPath, "/tmp/") {
			tmpDir := os.TempDir()
			targetPath = filepath.Join(tmpDir, filepath.Base(targetPath))
		} else {
			// 替换正斜杠为反斜杠
			targetPath = strings.ReplaceAll(targetPath, "/", "\\")
		}
	}

	// 获取上传的文件
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	// 创建文件管理器
	fileManager := vm.NewFileManager()

	// 上传文件
	if err := fileManager.UploadFileFromReader(file, targetPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleFileDownload 处理文件下载请求
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	type FileDownloadRequest struct {
		SourcePath string `json:"source_path"`
	}

	var req FileDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 在Windows系统上，确保路径格式正确
	if runtime.GOOS == "windows" {
		// 直接检查是否以/tmp/开头，这样可以避免转义问题
		if strings.HasPrefix(req.SourcePath, "/tmp/") {
			tmpDir := os.TempDir()
			log.Printf("Original path: %s, Temp dir: %s, Filename: %s", req.SourcePath, tmpDir, filepath.Base(req.SourcePath))
			req.SourcePath = filepath.Join(tmpDir, filepath.Base(req.SourcePath))
			log.Printf("Converted path: %s", req.SourcePath)
		} else {
			// 替换正斜杠为反斜杠
			req.SourcePath = strings.ReplaceAll(req.SourcePath, "/", "\\")
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(req.SourcePath); os.IsNotExist(err) {
		log.Printf("File not found at path: %s", req.SourcePath)
		writeError(w, http.StatusNotFound, "File not found")
		return
	}

	// 设置响应头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.SourcePath)))
	w.Header().Set("Content-Type", "application/octet-stream")

	// 打开文件并写入响应
	sourceFile, err := os.Open(req.SourcePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer sourceFile.Close()

	// 传输文件内容
	if _, err := io.Copy(w, sourceFile); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// handleFileList 处理列出文件的请求
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

	// 在Windows系统上，确保路径格式正确
	if runtime.GOOS == "windows" {
		// 如果路径以/tmp/开头，转换为Windows临时目录
		if strings.HasPrefix(req.Path, "/tmp/") {
			tmpDir := os.TempDir()
			req.Path = filepath.Join(tmpDir, filepath.Base(req.Path))
		} else if req.Path == "/tmp" {
			// 如果路径正好是/tmp，直接转换为Windows临时目录
			req.Path = os.TempDir()
		} else {
			// 替换正斜杠为反斜杠
			req.Path = strings.ReplaceAll(req.Path, "/", "\\")
		}
	}

	// 创建文件管理器
	fileManager := vm.NewFileManager()

	// 获取文件列表
	files, err := fileManager.ListFiles(req.Path, req.MaxDepth)
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

// WebSocket SSH相关

// websocket升级器配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有跨域请求（生产环境中应该限制）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleSSHWebSocket 处理WebSocket SSH连接请求
func (s *Server) handleSSHWebSocket(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	vmName := r.URL.Query().Get("vm_name")
	password := r.URL.Query().Get("password")

	if vmName == "" {
		writeError(w, http.StatusBadRequest, "vm_name parameter is required")
		return
	}

	// 获取指定的VM
	targetVM, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("VM not found: %v", err))
		return
	}

	// 创建SSH客户端配置
	sshConfig := &vm.SSHConfig{
		Host:     targetVM.IP,
		Port:     targetVM.Port,
		Username: targetVM.Username,
		Password: password,
	}

	// 创建SSH客户端
	sshClient, err := vm.NewSSHClient(sshConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 建立连接到目标VM
	if err := sshClient.Connect(targetVM.IP, targetVM.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sshClient.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	// 创建SSH会话
	session, err := sshClient.CreateSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}
	defer session.Close()

	// 获取会话的输入输出管道
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("Failed to get stdin pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		log.Printf("Failed to get stderr pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// 配置SSH会话的伪终端
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	}

	err = session.RequestPty("xterm-256color", 80, 24, modes)
	if err != nil {
		log.Printf("Failed to request PTY: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// 启动Shell
	if err := session.Shell(); err != nil {
		log.Printf("Failed to start shell: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// 设置通道关闭通知
	done := make(chan bool)

	// 从WebSocket读取数据并写入SSH会话
	go func() {
		defer func() {
			done <- true
		}()

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				return
			}

			if messageType == websocket.TextMessage {
				// 如果是窗口大小调整消息
				if strings.HasPrefix(string(p), "resize:") {
					// 解析窗口大小
					var resizeData struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					if err := json.Unmarshal(p[7:], &resizeData); err != nil {
						log.Printf("Failed to parse resize data: %v", err)
						continue
					}
					// 调整伪终端大小
					if err := session.WindowChange(resizeData.Rows, resizeData.Cols); err != nil {
						log.Printf("Failed to change window size: %v", err)
					}
					continue
				}

				// 否则写入SSH会话
				if _, err := stdin.Write(p); err != nil {
					log.Printf("Failed to write to SSH session: %v", err)
					return
				}
			}
		}
	}()

	// 从SSH会话读取数据并写入WebSocket
	go func() {
		defer func() {
			done <- true
		}()

		outputChan := make(chan []byte, 100)

		// 读取标准输出
		go func() {
			buffer := make([]byte, 1024)
			for {
				n, err := stdout.Read(buffer)
				if err != nil {
					if err != io.EOF {
						log.Printf("SSH stdout read error: %v", err)
					}
					close(outputChan)
					return
				}
				if n > 0 {
					outputChan <- buffer[:n]
				}
			}
		}()

		// 读取标准错误
		go func() {
			buffer := make([]byte, 1024)
			for {
				n, err := stderr.Read(buffer)
				if err != nil {
					if err != io.EOF {
						log.Printf("SSH stderr read error: %v", err)
					}
					return
				}
				if n > 0 {
					outputChan <- buffer[:n]
				}
			}
		}()

		// 将输出写入WebSocket
		for output := range outputChan {
			if err := conn.WriteMessage(websocket.BinaryMessage, output); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// 等待任一通道完成
	<-done

	// 等待SSH会话完成
	session.Wait()

	log.Printf("WebSocket SSH session closed for VM %s", vmName)
}

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
  "context"
  "database/sql"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/models"
  "io"
  "log"
  "net"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "time"

  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/file"
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
  // Try to get from cache first
  if resources, found := s.resourceMonitor.GetCachedSystemResources(); found {
    writeJSON(w, http.StatusOK, resources)
    return
  }

  // Fall back to real-time collection if cache is not available
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

  // Try to get from cache first
  if resources, found := s.resourceMonitor.GetCachedProcessResources(pid); found {
    writeJSON(w, http.StatusOK, resources)
    return
  }

  // Fall back to real-time collection if cache is not available
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
  if err := config.SaveConfig("./conf/config.yaml", &newConfig); err != nil {
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

  var updatedProcess models.ManagedProcess
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
  vars := mux.Vars(r)
  vmName := vars["name"]

  var newVM vm.VM
  if err := json.NewDecoder(r.Body).Decode(&newVM); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 使用URL参数中的name，忽略请求体中的name
  newVM.Name = vmName

  if err := s.vmManager.AddVM(&newVM); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusCreated, map[string]string{"message": "VM added successfully"})
}

// handleListVMs 处理列出所有VM的请求
func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
  vms := s.vmManager.ListVMs()

  // 创建一个不包含密码的VM列表副本
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

// handleGetVM 处理获取VM详情的请求
func (s *Server) handleGetVM(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]

  vmInstance, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建一个不包含密码的VM副本
  vmWithoutPassword := map[string]interface{}{
    "name":     vmInstance.Name,
    "ip":       vmInstance.IP,
    "port":     vmInstance.Port,
    "username": vmInstance.Username,
    "key_path": vmInstance.KeyPath,
  }

  writeJSON(w, http.StatusOK, vmWithoutPassword)
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

// handleUpdateVM 处理更新VM的请求
func (s *Server) handleUpdateVM(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 解析请求体
  var updateData struct {
    Password string `json:"password"`
    KeyPath  string `json:"key_path"`
  }
  if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 更新密码和密钥路径
  if updateData.Password != "" {
    vmInfo.Password = updateData.Password
  }
  if updateData.KeyPath != "" {
    vmInfo.KeyPath = updateData.KeyPath
  }

  // 保存VM信息
  if err := s.vmManager.SaveVMs(); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "VM updated successfully"})
}

// ------------------------- Group Management Handlers -------------------------

// handleAddGroup 处理添加VM组的请求
func (s *Server) handleAddGroup(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  groupName := vars["name"]

  // 解析请求体
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

  // 获取当前用户作为owner
  owner := s.getCurrentUser(r)

  // 添加组，使用URL参数中的name
  if err := s.vmManager.AddGroup(groupName, groupData.Description, owner, groupData.VMs, groupData.IsShared, groupData.SharedWith); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Group added successfully"})
}

// handleListGroups 处理列出VM组的请求
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
  // 获取所有组
  allGroups := s.vmManager.ListGroups()

  // 获取查询参数
  sharedOnly := r.URL.Query().Get("shared_only") == "true"
  mineOnly := r.URL.Query().Get("mine_only") == "true"
  username := s.getCurrentUser(r)
  isAdmin := s.isAdmin(r)

  // 过滤组
  var groups []*vm.Group
  for _, group := range allGroups {
    // 管理员可以看到所有组
    if isAdmin {
      groups = append(groups, group)
      continue
    }

    // 只获取共享组
    if sharedOnly {
      if group.IsShared {
        groups = append(groups, group)
      }
      continue
    }

    // 只获取自己创建的组
    if mineOnly {
      if group.Owner == username {
        groups = append(groups, group)
      }
      continue
    }

    // 默认：显示共享的组和自己创建的组
    if group.IsShared || group.Owner == username {
      groups = append(groups, group)
    }
  }

  writeJSON(w, http.StatusOK, groups)
}

// handleGetGroup 处理获取VM组的请求
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  groupName := vars["name"]

  // 获取组
  group, err := s.vmManager.GetGroup(groupName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, group)
}

// handleUpdateGroup 处理更新VM组的请求
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  groupName := vars["name"]

  // 解析请求体
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

  // 检查权限
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

  // 更新组
  if err := s.vmManager.UpdateGroup(groupName, updateData.Description, updateData.VMs, updateData.IsShared, updateData.SharedWith); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Group updated successfully"})
}

// handleDeleteGroup 处理删除VM组的请求
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  groupName := vars["name"]

  // 删除组
  if err := s.vmManager.RemoveGroup(groupName); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

// handleVmFileList 处理列出VM中的文件请求
func (s *Server) handleVmFileList(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
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

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建SSH客户端
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

  // 连接到SSH服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 获取文件列表
  files, err := sshClient.ListFiles(req.Path, req.MaxDepth)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, files)
}

// handleFileUpload 处理文件上传请求
func (s *Server) handleVmFileUpload(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

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

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建SSH客户端
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

  // 连接到SSH服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 获取上传的文件
  file, _, err := r.FormFile("file")
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }
  defer file.Close()

  // 上传文件
  if err := sshClient.UploadFile(file, targetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleVmFileDownload 处理文件下载请求
func (s *Server) handleVmFileDownload(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
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

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建SSH客户端
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

  // 连接到SSH服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 下载文件
  content, err := sshClient.DownloadFile(req.SourcePath)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  // 设置响应头
  w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.SourcePath)))
  w.Header().Set("Content-Type", "application/octet-stream")

  // 写入响应
  if _, err := w.Write(content); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
}

// handleVmFileDelete 处理删除文件请求
func (s *Server) handleVmFileDelete(w http.ResponseWriter, r *http.Request) {
  // 解析 URL 参数
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

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建 SSH 客户端
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

  // 连接到 SSH 服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 删除文件
  if _, err := sshClient.ExecuteCommand("rm " + req.Path); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}

// handleVmFileMkdir 处理创建目录请求
func (s *Server) handleVmFileMkdir(w http.ResponseWriter, r *http.Request) {
  // 解析 URL 参数
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

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建 SSH 客户端
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

  // 连接到 SSH 服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 创建目录
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

// handleVmFileTouch 处理创建文件请求
func (s *Server) handleVmFileTouch(w http.ResponseWriter, r *http.Request) {
  // 解析 URL 参数
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

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建 SSH 客户端
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

  // 连接到 SSH 服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 创建文件
  if _, err := sshClient.ExecuteCommand("touch " + req.Path); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File created successfully"})
}

// handleVmFileRmdir 处理删除目录请求
func (s *Server) handleVmFileRmdir(w http.ResponseWriter, r *http.Request) {
  // 解析 URL 参数
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

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建 SSH 客户端
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

  // 连接到 SSH 服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // 删除目录
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

// ------------------------- File Management Handlers -------------------------

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

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 获取文件列表
  files, err := fileManager.ListFiles(req.Path, req.MaxDepth)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, files)
}

// handleFileUpload 处理列出文件的请求
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

  // 获取上传的文件
  sourceFile, _, err := r.FormFile("file")
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }
  defer sourceFile.Close()

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 上传文件
  file, _, err := r.FormFile("file")
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }
  defer file.Close()

  if err := fileManager.UploadFile(file, targetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// handleFileDownload 处理列出文件的请求
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
  type FileDownloadRequest struct {
    SourcePath string `json:"source_path"`
  }

  var req FileDownloadRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 下载文件
  content, err := fileManager.DownloadFile(req.SourcePath)
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  // 设置响应头
  w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.SourcePath)))
  w.Header().Set("Content-Type", "application/octet-stream")

  // 写入响应
  if _, err := w.Write(content); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
}

// handleFileDelete 处理删除文件的请求
func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
  type FileDeleteRequest struct {
    Path string `json:"path"`
  }

  var req FileDeleteRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 删除文件
  if err := fileManager.DeleteFile(req.Path); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}

// handleFileCopy 处理复制文件的请求
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

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 复制文件
  if err := fileManager.CopyFile(req.SourcePath, req.TargetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File copied successfully"})
}

// handleFileMove 处理移动文件的请求
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

  // 创建文件管理器
  fileManager := file.NewManager("")

  // 移动文件
  if err := fileManager.MoveFile(req.SourcePath, req.TargetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File moved successfully"})
}

// ------------------------- Permission Handlers -------------------------

// handleAddPermission 处理添加权限的请求
func (s *Server) handleAddPermission(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
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

// handleRemovePermission 处理移除权限的请求
func (s *Server) handleRemovePermission(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
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

// handleListPermissions 处理列出权限的请求
func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]

  permissions := s.vmManager.GetPermissionManager().ListPermissions(vmName)
  writeJSON(w, http.StatusOK, permissions)
}

// handleCheckPermission 处理检查权限的请求
func (s *Server) handleCheckPermission(w http.ResponseWriter, r *http.Request) {
  // 解析URL参数
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

// handleSSHWebSocket 处理WebSocket SSH连接请求
func (s *Server) handleSSHWebSocket(w http.ResponseWriter, r *http.Request) {
  vmName := r.URL.Query().Get("vm_name")

  if vmName == "" {
    http.Error(w, "vm_name required", http.StatusBadRequest)
    return
  }

  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    http.Error(w, err.Error(), http.StatusNotFound)
    return
  }

  sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
    Host:     vmInfo.IP,
    Port:     vmInfo.Port,
    Username: vmInfo.Username,
    Password: vmInfo.Password,
    KeyPath:  vmInfo.KeyPath,
  })
  if err != nil {
    http.Error(w, err.Error(), 500)
    return
  }

  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    http.Error(w, err.Error(), 500)
    return
  }
  defer sshClient.Close()

  // Upgrade WS
  var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // 允许所有跨域请求（生产环境中应该限制）
    CheckOrigin: func(r *http.Request) bool {
      return true
    },
  }
  ws, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    return
  }
  defer ws.Close()

  session, err := sshClient.CreateSession()
  if err != nil {
    ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
    return
  }
  defer session.Close()

  // PTY
  if err := session.RequestPty(
    "xterm-256color",
    40,
    120,
    ssh.TerminalModes{
      ssh.ECHO:          1,
      ssh.TTY_OP_ISPEED: 14400,
      ssh.TTY_OP_OSPEED: 14400,
    },
  ); err != nil {
    ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
    return
  }

  stdin, _ := session.StdinPipe()
  stdout, _ := session.StdoutPipe()
  stderr, _ := session.StderrPipe()

  if err := session.Shell(); err != nil {
    ws.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
    return
  }

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // Generate a unique connection ID
  connID := fmt.Sprintf("%s-%d", vmName, time.Now().UnixNano())

  // Get client IP
  clientIP := r.RemoteAddr
  if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
    clientIP = forwardedFor
  }

  // Get username (authenticated user or anonymous)
  username := s.getConnectionUser(r)

  // Register the connection
  s.RegisterSSHConnection(connID, vmName, clientIP, username)
  log.Printf("SSH connection registered: ID=%s, VM=%s, ClientIP=%s, User=%s", connID, vmName, clientIP, username)

  // Ensure connection is unregistered when session ends
  defer func() {
    s.UnregisterSSHConnection(connID)
    log.Printf("SSH connection unregistered: ID=%s", connID)
  }()

  // ---------------- 输入：WS → SSH ----------------
  go func() {
    defer cancel()
    for {
      messageType, payload, err := ws.ReadMessage()
      if err != nil {
        // WebSocket connection closed or error occurred
        log.Printf("WebSocket read error for connection %s: %v", connID, err)
        return
      }

      if messageType == websocket.TextMessage {
        // 如果是窗口大小调整消息
        if strings.HasPrefix(string(payload), "resize:") {
          // 解析窗口大小
          var resizeData struct {
            Cols int `json:"cols"`
            Rows int `json:"rows"`
          }
          if err := json.Unmarshal(payload[7:], &resizeData); err != nil {
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
        if _, err := stdin.Write(payload); err != nil {
          log.Printf("Failed to write to SSH session: %v", err)
          return
        }
      } else {
        _, _ = stdin.Write(payload)
      }
    }
  }()

  // ---------------- 输出：SSH → WS ----------------
  // SSH → WebSocket（完全 raw）
  go func() {
    defer cancel()

    reader := io.MultiReader(stdout, stderr)
    buf := make([]byte, 4096)

    for {
      n, err := reader.Read(buf)
      if n > 0 {
        if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
          // WebSocket connection closed or error occurred
          log.Printf("WebSocket write error for connection %s: %v", connID, err)
          return
        }
      }
      if err != nil {
        log.Printf("SSH output error for connection %s: %v", connID, err)
        return
      }
    }
  }()

  // Wait for context cancellation (WebSocket disconnect or error)
  <-ctx.Done()
  log.Printf("WebSocket connection closed: ID=%s", connID)

  // Close session explicitly to clean up SSH resources
  // Use a timeout to prevent blocking forever if session.Close() hangs
  done := make(chan struct{})
  go func() {
    session.Close()
    close(done)
  }()

  select {
  case <-done:
    // Session closed normally
  case <-time.After(2 * time.Second):
    // Timeout - session close is stuck, but we still return to let defer run
    log.Printf("SSH session close timeout for connection %s", connID)
  }
}

// handleVMExec 处理 VM 命令执行请求
func (s *Server) handleVMExec(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  // 解析请求体
  var req struct {
    Command string `json:"command"`
    Timeout int    `json:"timeout"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 验证命令不为空
  if req.Command == "" {
    writeError(w, http.StatusBadRequest, "command cannot be empty")
    return
  }

  // 设置默认超时
  if req.Timeout <= 0 {
    req.Timeout = 30
  }

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 创建 SSH 客户端
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

  // 连接到 SSH 服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
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

  // 返回命令输出
  w.Header().Set("Content-Type", "text/plain; charset=utf-8")
  w.WriteHeader(http.StatusOK)
  w.Write([]byte(output))
}

// handleVMPing 处理 VM Ping 请求
func (s *Server) handleVMPing(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  // 获取 VM 信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // 测试 TCP 连接
  addr := fmt.Sprintf("%s:%d", vmInfo.IP, vmInfo.Port)
  start := time.Now()
  conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
  latency := time.Since(start)

  if err != nil {
    // 连接失败
    writeJSON(w, http.StatusOK, map[string]interface{}{
      "success": false,
      "status":  "TIMEOUT",
      "message": err.Error(),
    })
    return
  }
  defer conn.Close()

  // 连接成功
  writeJSON(w, http.StatusOK, map[string]interface{}{
    "success": true,
    "status":  "OK",
    "latency": latency.String(),
  })
}

// handleListSSHConnections handles the GET /api/vms/ssh/connections endpoint
func (s *Server) handleListSSHConnections(w http.ResponseWriter, r *http.Request) {
  connections := s.GetSSHConnections()

  // Get filter parameters from query
  vmName := r.URL.Query().Get("vm_name")
  userName := r.URL.Query().Get("user_name")
  clientIP := r.URL.Query().Get("client_ip")

  // Apply filters if provided
  var filteredConnections []*SSHConnectionInfo
  for _, conn := range connections {
    match := true

    if vmName != "" && conn.VMName != vmName {
      match = false
    }

    if userName != "" && conn.Username != userName {
      match = false
    }

    if clientIP != "" && conn.ClientIP != clientIP {
      match = false
    }

    if match {
      filteredConnections = append(filteredConnections, conn)
    }
  }

  writeJSON(w, http.StatusOK, filteredConnections)
}

// handleCloseSSHConnection handles the DELETE /api/vms/ssh/connections/{id} endpoint
func (s *Server) handleCloseSSHConnection(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  id := vars["id"]

  if id == "" {
    writeError(w, http.StatusBadRequest, "connection ID is required")
    return
  }

  success := s.CloseSSHConnection(id)
  if !success {
    writeError(w, http.StatusNotFound, "connection not found")
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "connection closed successfully"})
}

// handleCloseAllSSHConnections handles the DELETE /api/vms/ssh/connections endpoint
func (s *Server) handleCloseAllSSHConnections(w http.ResponseWriter, r *http.Request) {
  count := s.CloseAllSSHConnections()

  writeJSON(w, http.StatusOK, map[string]interface{}{
    "message": "all connections closed successfully",
    "count":   count,
  })
}

// handleRegisterUser handles the POST /api/users/register endpoint
func (s *Server) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
  // Only allow registration if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Parse request body
  var user models.User
  if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // Validate required fields
  if user.Username == "" || user.Password == "" {
    writeError(w, http.StatusBadRequest, "Username and password are required")
    return
  }

  // Check if user already exists
  if _, exists := s.userDatabase.GetUser(user.Username); exists {
    writeError(w, http.StatusConflict, "User already exists")
    return
  }

  // Set role to "user" by default if not specified
  if user.Role == "" {
    user.Role = "user"
  }

  // Generate a unique ID for the user
  user.ID = fmt.Sprintf("usr_%d", time.Now().Unix())

  // Create the user
  if err := s.userDatabase.CreateUser(&user); err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
    return
  }

  // Return success response (without password)
  responseUser := user
  responseUser.Password = "" // Don't return password
  writeJSON(w, http.StatusCreated, map[string]interface{}{
    "message": "User registered successfully",
    "user":    responseUser,
  })
}

// handleUserLogin handles the POST /api/users/login endpoint
func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
  // Check if user database is available
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Parse request body
  var loginReq struct {
    Username string `json:"username"`
    Password string `json:"password"`
  }
  if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // Validate required fields
  if loginReq.Username == "" || loginReq.Password == "" {
    writeError(w, http.StatusBadRequest, "Username and password are required")
    return
  }

  // Get client IP
  clientIP := r.RemoteAddr
  if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
    clientIP = forwardedFor
  }

  // Get User-Agent and device info
  userAgent := r.Header.Get("User-Agent")
  deviceInfo := ""
  if strings.Contains(userAgent, "Mobile") {
    deviceInfo = "mobile"
  } else if strings.Contains(userAgent, "Tablet") {
    deviceInfo = "tablet"
  } else {
    deviceInfo = "desktop"
  }

  // Validate password
  isValid, err := s.userDatabase.ValidatePassword(loginReq.Username, loginReq.Password)
  if err != nil {
    log.Printf("Error validating password for user %s: %v", loginReq.Username, err)
    writeError(w, http.StatusInternalServerError, "Failed to validate credentials")
    return
  }

  if !isValid {
    // Log failed login attempt
    if s.loginLogDatabase != nil {
      user, exists := s.userDatabase.GetUser(loginReq.Username)
      userID := ""
      if exists {
        userID = user.ID
      }
      s.loginLogDatabase.LogLogin(loginReq.Username, userID, clientIP, userAgent, deviceInfo, "failed")
    }
    writeError(w, http.StatusUnauthorized, "Invalid username or password")
    return
  }

  // Get user info
  user, exists := s.userDatabase.GetUser(loginReq.Username)
  if !exists {
    writeError(w, http.StatusNotFound, "User not found")
    return
  }

  // Update login status in users table
  if s.userDatabase != nil {
    if err := s.userDatabase.UpdateLoginStatus(loginReq.Username, clientIP); err != nil {
      log.Printf("Error updating login status for user %s: %v", loginReq.Username, err)
    }
  }

  // Log successful login
  if s.loginLogDatabase != nil {
    if err := s.loginLogDatabase.LogLogin(loginReq.Username, user.ID, clientIP, userAgent, deviceInfo, "success"); err != nil {
      log.Printf("Error logging login for user %s: %v", loginReq.Username, err)
    }
  }

  // Return success response with user info (without password)
  writeJSON(w, http.StatusOK, map[string]interface{}{
    "message": "Login successful",
    "user": map[string]interface{}{
      "id":         user.ID,
      "username":   user.Username,
      "email":      user.Email,
      "role":       user.Role,
      "avatar":     user.Avatar,
      "nickname":   user.Nickname,
      "region":     user.Region,
      "configs":    user.Configs,
      "created_at": user.CreatedAt,
      "updated_at": user.UpdatedAt,
    },
  })
}

// handleListUsers handles the GET /api/users endpoint
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Check if authentication is required based on config
  if s.config.BasicAuth.Enabled {
    // Check if requesting user has admin role
    username, _, ok := r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin (from config) or a registered admin
    isSuperAdmin := username == s.config.BasicAuth.Username
    isAdmin := false

    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(username); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    // Only super admin or admin users can list all users
    if !isSuperAdmin && !isAdmin {
      writeError(w, http.StatusForbidden, "Access denied: Admin role required to list users")
      return
    }
  }

  // Get all users
  users := s.userDatabase.GetAllUsers()

  // Create response without passwords
  responseUsers := make([]map[string]interface{}, len(users))
  for i, user := range users {
    responseUsers[i] = map[string]interface{}{
      "id":         user.ID,
      "username":   user.Username,
      "email":      user.Email,
      "role":       user.Role,
      "avatar":     user.Avatar,
      "nickname":   user.Nickname,
      "region":     user.Region,
      "configs":    user.Configs,
      "created_at": user.CreatedAt,
      "updated_at": user.UpdatedAt,
    }
  }

  writeJSON(w, http.StatusOK, responseUsers)
}

// handleGetUser handles the GET /api/users/{username} endpoint
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Get username from path variables
  vars := mux.Vars(r)
  targetUsername := vars["username"]

  if targetUsername == "" {
    writeError(w, http.StatusBadRequest, "Username is required")
    return
  }

  // Check if authentication is required based on config
  requestingUsername := ""
  if s.config.BasicAuth.Enabled {
    // Check if requesting user has permission to view this user
    var ok bool
    requestingUsername, _, ok = r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin (from config)
    isSuperAdmin := requestingUsername == s.config.BasicAuth.Username

    // Check if requesting user is an admin
    isAdmin := false
    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    // Allow access if it's the super admin, admin, or the user is requesting their own info
    if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
      writeError(w, http.StatusForbidden, "Access denied: Cannot access other user's information")
      return
    }
  } else {
    // When auth is disabled, allow access to user info (may need to adjust this logic based on requirements)
    requestingUsername = "anonymous" // Or some default value when auth is disabled
  }

  // Get the user
  user, exists := s.userDatabase.GetUser(targetUsername)
  if !exists {
    writeError(w, http.StatusNotFound, "User not found")
    return
  }

  // Create response without password
  responseUser := map[string]interface{}{
    "id":         user.ID,
    "username":   user.Username,
    "email":      user.Email,
    "role":       user.Role,
    "avatar":     user.Avatar,
    "nickname":   user.Nickname,
    "region":     user.Region,
    "configs":    user.Configs,
    "created_at": user.CreatedAt,
    "updated_at": user.UpdatedAt,
  }

  writeJSON(w, http.StatusOK, responseUser)
}

// handleUpdateUser handles the PUT /api/users/{username} endpoint
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Get username from path variables
  vars := mux.Vars(r)
  targetUsername := vars["username"]

  if targetUsername == "" {
    writeError(w, http.StatusBadRequest, "Username is required")
    return
  }

  // Parse request body
  var updateData struct {
    Email    string `json:"email"`
    Role     string `json:"role"`
    Password string `json:"password"`
    Avatar   string `json:"avatar"`
    Nickname string `json:"nickname"`
    Region   string `json:"region"`
    Configs  string `json:"configs"`
  }

  if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // Check if authentication is required based on config
  requestingUsername := ""
  if s.config.BasicAuth.Enabled {
    // Check if requesting user has permission to update this user
    var ok bool
    requestingUsername, _, ok = r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin (from config)
    isSuperAdmin := requestingUsername == s.config.BasicAuth.Username

    // Check if requesting user is an admin
    isAdmin := false
    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    // Determine if user can update (either super admin, admin, or updating their own account)
    canUpdate := isSuperAdmin || isAdmin || requestingUsername == targetUsername

    if !canUpdate {
      writeError(w, http.StatusForbidden, "Access denied: Cannot update other user's information")
      return
    }

    // Prevent non-admin users from changing roles
    if !isSuperAdmin && !isAdmin && updateData.Role != "" {
      writeError(w, http.StatusForbidden, "Access denied: Regular users cannot change roles")
      return
    }
  } else {
    // When auth is disabled, allow updates (may need to adjust this logic based on requirements)
    requestingUsername = "anonymous" // Or some default value when auth is disabled
  }

  // Prepare updated user data
  updatedUser := &models.User{
    Username: targetUsername,
    Email:    updateData.Email,
    Role:     updateData.Role,
    Password: updateData.Password, // Will be hashed in UpdateUser if not empty
    Avatar:   updateData.Avatar,
    Nickname: updateData.Nickname,
    Region:   updateData.Region,
    Configs:  updateData.Configs,
  }

  // Update the user
  if err := s.userDatabase.UpdateUser(targetUsername, updatedUser); err != nil {
    if os.IsNotExist(err) {
      writeError(w, http.StatusNotFound, "User not found")
      return
    }
    writeError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

// handleDeleteUser handles the DELETE /api/users/{username} endpoint
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Get username from path variables
  vars := mux.Vars(r)
  targetUsername := vars["username"]

  if targetUsername == "" {
    writeError(w, http.StatusBadRequest, "Username is required")
    return
  }

  // Check if authentication is required based on config
  requestingUsername := ""
  if s.config.BasicAuth.Enabled {
    // Check if requesting user has permission to delete this user
    var ok bool
    requestingUsername, _, ok = r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin (from config)
    isSuperAdmin := requestingUsername == s.config.BasicAuth.Username

    // Check if requesting user is an admin
    isAdmin := false
    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    // Only super admin or admin users can delete users
    if !isSuperAdmin && !isAdmin {
      writeError(w, http.StatusForbidden, "Access denied: Admin role required to delete users")
      return
    }

    // Prevent deletion of the super admin user
    if targetUsername == s.config.BasicAuth.Username {
      writeError(w, http.StatusForbidden, "Access denied: Cannot delete super admin user")
      return
    }
  } else {
    // When auth is disabled, allow deletion (may need to adjust this logic based on requirements)
    requestingUsername = "anonymous" // Or some default value when auth is disabled
  }

  // Delete the user
  if err := s.userDatabase.DeleteUser(targetUsername); err != nil {
    if os.IsNotExist(err) {
      writeError(w, http.StatusNotFound, "User not found")
      return
    }
    writeError(w, http.StatusInternalServerError, "Failed to delete user: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// handleGetUserConfigs handles the GET /api/users/{username}/configs endpoint
func (s *Server) handleGetUserConfigs(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Get username from path variables
  vars := mux.Vars(r)
  targetUsername := vars["username"]

  if targetUsername == "" {
    writeError(w, http.StatusBadRequest, "Username is required")
    return
  }

  // Check authentication
  if s.config.BasicAuth.Enabled {
    requestingUsername, _, ok := r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin, admin, or the user is requesting their own configs
    isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
    isAdmin := false
    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
      writeError(w, http.StatusForbidden, "Access denied: Cannot access other user's configuration")
      return
    }
  }

  // Get the user
  user, exists := s.userDatabase.GetUser(targetUsername)
  if !exists {
    writeError(w, http.StatusNotFound, "User not found")
    return
  }

  // Return configs (empty string if not set)
  writeJSON(w, http.StatusOK, map[string]string{
    "configs": user.Configs,
  })
}

// handleUpdateUserConfigs handles the PUT /api/users/{username}/configs endpoint
func (s *Server) handleUpdateUserConfigs(w http.ResponseWriter, r *http.Request) {
  // Only allow if user database exists
  if s.userDatabase == nil {
    writeError(w, http.StatusInternalServerError, "User database not available")
    return
  }

  // Get username from path variables
  vars := mux.Vars(r)
  targetUsername := vars["username"]

  if targetUsername == "" {
    writeError(w, http.StatusBadRequest, "Username is required")
    return
  }

  // Parse request body
  var req struct {
    Configs string `json:"configs"`
  }

  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // Check authentication
  requestingUsername := ""
  if s.config.BasicAuth.Enabled {
    var ok bool
    requestingUsername, _, ok = r.BasicAuth()
    if !ok {
      writeError(w, http.StatusUnauthorized, "Authentication required")
      return
    }

    // Check if it's the super admin, admin, or the user is updating their own configs
    isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
    isAdmin := false
    if !isSuperAdmin && s.userDatabase != nil {
      if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
        isAdmin = true
      }
    }

    if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
      writeError(w, http.StatusForbidden, "Access denied: Cannot update other user's configuration")
      return
    }
  }

  // Get the user to check existence
  _, exists := s.userDatabase.GetUser(targetUsername)
  if !exists {
    writeError(w, http.StatusNotFound, "User not found")
    return
  }

  // Update the user's configs
  updatedUser := &models.User{
    Username: targetUsername,
    Configs:  req.Configs,
  }

  if err := s.userDatabase.UpdateUser(targetUsername, updatedUser); err != nil {
    if os.IsNotExist(err) {
      writeError(w, http.StatusNotFound, "User not found")
      return
    }
    writeError(w, http.StatusInternalServerError, "Failed to update user configs: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "User configs updated successfully"})
}

// ------------------------- P0: Batch Operations Handlers -------------------------

// handleBatchExec 处理批量执行命令请求
func (s *Server) handleBatchExec(w http.ResponseWriter, r *http.Request) {
  var req BatchExecRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // 验证请求参数
  if len(req.VMNames) == 0 {
    writeError(w, http.StatusBadRequest, "vm_names is required")
    return
  }
  if req.Command == "" {
    writeError(w, http.StatusBadRequest, "command is required")
    return
  }
  if req.Timeout <= 0 {
    req.Timeout = 30
  }

  // 生成任务ID
  taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

  // 获取当前用户信息
  username := s.getCurrentUser(r)

  // 执行结果通道
  resultChan := make(chan BatchExecResult, len(req.VMNames))

  // 定义执行函数
  executeOnVM := func(vmName string) {
    start := time.Now()
    result := BatchExecResult{
      VMName: vmName,
      Status: "failed",
    }

    // 获取VM信息
    vmInfo, err := s.vmManager.GetVM(vmName)
    if err != nil {
      result.Error = "VM not found: " + err.Error()
      result.DurationMs = time.Since(start).Milliseconds()
      resultChan <- result
      return
    }

    // 创建SSH客户端
    sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
      Host:     vmInfo.IP,
      Port:     vmInfo.Port,
      Username: vmInfo.Username,
      Password: vmInfo.Password,
      KeyPath:  vmInfo.KeyPath,
    })
    if err != nil {
      result.Error = "Failed to create SSH client: " + err.Error()
      result.DurationMs = time.Since(start).Milliseconds()
      resultChan <- result
      return
    }

    // 连接到SSH服务器
    if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
      result.Error = "Failed to connect: " + err.Error()
      result.DurationMs = time.Since(start).Milliseconds()
      resultChan <- result
      return
    }
    defer sshClient.Close()

    // 设置超时上下文执行命令
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
    defer cancel()

    done := make(chan struct {
      output string
      err    error
    }, 1)

    go func() {
      output, err := sshClient.ExecuteCommand(req.Command)
      done <- struct {
        output string
        err    error
      }{output, err}
    }()

    select {
    case res := <-done:
      result.DurationMs = time.Since(start).Milliseconds()
      if res.err != nil {
        result.Error = res.err.Error()
      } else {
        result.Status = "success"
        result.Output = res.output
      }
    case <-ctx.Done():
      result.DurationMs = time.Since(start).Milliseconds()
      result.Error = "Command execution timeout"
    }

    resultChan <- result

    // 记录命令历史
    if s.commandHistoryDB != nil {
      status := "success"
      if result.Status != "success" {
        status = "failed"
      }
      s.recordCommandHistory(vmName, req.Command, username, status, result.DurationMs, result.Output, result.Error)
    }
  }

  // 执行命令
  if req.Parallel {
    // 并行执行，限制并发数
    semaphore := make(chan struct{}, 10)
    for _, vmName := range req.VMNames {
      go func(name string) {
        semaphore <- struct{}{}
        defer func() { <-semaphore }()
        executeOnVM(name)
      }(vmName)
    }
  } else {
    // 串行执行
    go func() {
      for _, vmName := range req.VMNames {
        executeOnVM(vmName)
      }
    }()
  }

  // 收集结果
  var results []BatchExecResult
  successCount := 0
  failedCount := 0

  for i := 0; i < len(req.VMNames); i++ {
    result := <-resultChan
    results = append(results, result)
    if result.Status == "success" {
      successCount++
    } else {
      failedCount++
    }
  }

  close(resultChan)

  response := BatchExecResponse{
    TaskID:  taskID,
    Total:   len(req.VMNames),
    Success: successCount,
    Failed:  failedCount,
    Results: results,
  }

  writeJSON(w, http.StatusOK, response)
}

// handleGetVMResources 获取单服务器资源监控
func (s *Server) handleGetVMResources(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "VM name is required")
    return
  }

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, "VM not found: "+err.Error())
    return
  }

  // 创建SSH客户端
  sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
    Host:     vmInfo.IP,
    Port:     vmInfo.Port,
    Username: vmInfo.Username,
    Password: vmInfo.Password,
    KeyPath:  vmInfo.KeyPath,
  })
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to create SSH client: "+err.Error())
    return
  }

  // 连接到SSH服务器
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to connect to VM: "+err.Error())
    return
  }
  defer sshClient.Close()

  // 收集资源信息
  resourceInfo := &VMResourceInfo{
    VMName:      vmName,
    CollectedAt: time.Now(),
  }

  // 获取CPU使用率
  cpuOutput, err := sshClient.ExecuteCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
  if err == nil && cpuOutput != "" {
    cpuOutput = strings.TrimSpace(cpuOutput)
    if cpu, err := strconv.ParseFloat(cpuOutput, 64); err == nil {
      resourceInfo.CPUUsage = cpu
    }
  }

  // 获取内存信息
  memOutput, err := sshClient.ExecuteCommand("free -m | awk 'NR==2{printf \"%.2f %.2f %.2f\", $2/1024,$3/1024,$3/$2*100}'")
  if err == nil {
    parts := strings.Fields(memOutput)
    if len(parts) >= 3 {
      if total, err := strconv.ParseFloat(parts[0], 64); err == nil {
        resourceInfo.MemoryTotalGB = total
      }
      if used, err := strconv.ParseFloat(parts[1], 64); err == nil {
        resourceInfo.MemoryUsedGB = used
      }
      if usage, err := strconv.ParseFloat(parts[2], 64); err == nil {
        resourceInfo.MemoryUsage = usage
      }
    }
  }

  // 获取磁盘信息
  diskOutput, err := sshClient.ExecuteCommand("df -h / | awk 'NR==2{print $2,$3,$5}' | sed 's/G//g' | sed 's/%//g'")
  if err == nil {
    parts := strings.Fields(diskOutput)
    if len(parts) >= 3 {
      if total, err := strconv.ParseFloat(parts[0], 64); err == nil {
        resourceInfo.DiskTotalGB = total
      }
      if used, err := strconv.ParseFloat(parts[1], 64); err == nil {
        resourceInfo.DiskUsedGB = used
      }
      if usage, err := strconv.ParseFloat(parts[2], 64); err == nil {
        resourceInfo.DiskUsage = usage
      }
    }
  }

  // 获取负载平均值
  loadOutput, err := sshClient.ExecuteCommand("uptime | awk -F'load average:' '{print $2}' | tr -d ','")
  if err == nil {
    parts := strings.Fields(loadOutput)
    for _, part := range parts {
      if load, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
        resourceInfo.LoadAverage = append(resourceInfo.LoadAverage, load)
      }
    }
  }

  // 获取运行时间
  uptimeOutput, err := sshClient.ExecuteCommand("uptime -p")
  if err == nil {
    resourceInfo.Uptime = strings.TrimSpace(uptimeOutput)
  }

  // 获取网络信息（需要两次采样）
  netOutput1, _ := sshClient.ExecuteCommand("cat /proc/net/dev | grep eth0 | awk '{print $2,$10}'")
  time.Sleep(1 * time.Second)
  netOutput2, _ := sshClient.ExecuteCommand("cat /proc/net/dev | grep eth0 | awk '{print $2,$10}'")

  if netOutput1 != "" && netOutput2 != "" {
    parts1 := strings.Fields(netOutput1)
    parts2 := strings.Fields(netOutput2)
    if len(parts1) >= 2 && len(parts2) >= 2 {
      if rx1, err := strconv.ParseInt(parts1[0], 10, 64); err == nil {
        if rx2, err := strconv.ParseInt(parts2[0], 10, 64); err == nil {
          resourceInfo.Network.RXBytesPerSec = rx2 - rx1
        }
      }
      if tx1, err := strconv.ParseInt(parts1[1], 10, 64); err == nil {
        if tx2, err := strconv.ParseInt(parts2[1], 10, 64); err == nil {
          resourceInfo.Network.TXBytesPerSec = tx2 - tx1
        }
      }
    }
  }

  writeJSON(w, http.StatusOK, resourceInfo)
}

// handleBatchGetVMResources 批量获取服务器资源
func (s *Server) handleBatchGetVMResources(w http.ResponseWriter, r *http.Request) {
  var req struct {
    VMNames []string `json:"vm_names"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  if len(req.VMNames) == 0 {
    writeError(w, http.StatusBadRequest, "vm_names is required")
    return
  }

  // 结果通道
  resultChan := make(chan VMResourceInfo, len(req.VMNames))

  // 限制并发数
  semaphore := make(chan struct{}, 10)

  for _, vmName := range req.VMNames {
    go func(name string) {
      semaphore <- struct{}{}
      defer func() { <-semaphore }()

      info := VMResourceInfo{
        VMName: name,
        Status: "error",
      }

      vmInfo, err := s.vmManager.GetVM(name)
      if err != nil {
        info.Error = "VM not found"
        resultChan <- info
        return
      }

      sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
        Host:     vmInfo.IP,
        Port:     vmInfo.Port,
        Username: vmInfo.Username,
        Password: vmInfo.Password,
        KeyPath:  vmInfo.KeyPath,
      })
      if err != nil {
        info.Error = "Failed to create SSH client"
        resultChan <- info
        return
      }

      if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
        info.Error = "Failed to connect"
        resultChan <- info
        return
      }
      defer sshClient.Close()

      info.CollectedAt = time.Now()

      // 获取CPU使用率（简化版）
      cpuOutput, _ := sshClient.ExecuteCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
      if cpuOutput != "" {
        if cpu, err := strconv.ParseFloat(strings.TrimSpace(cpuOutput), 64); err == nil {
          info.CPUUsage = cpu
        }
      }

      // 获取内存使用率
      memOutput, _ := sshClient.ExecuteCommand("free | awk 'NR==2{printf \"%.0f\", $3/$2*100}'")
      if memOutput != "" {
        if mem, err := strconv.ParseFloat(strings.TrimSpace(memOutput), 64); err == nil {
          info.MemoryUsage = mem
        }
      }

      // 获取磁盘使用率
      diskOutput, _ := sshClient.ExecuteCommand("df -h / | awk 'NR==2{print $5}' | sed 's/%//g'")
      if diskOutput != "" {
        if disk, err := strconv.ParseFloat(strings.TrimSpace(diskOutput), 64); err == nil {
          info.DiskUsage = disk
        }
      }

      // 确定状态
      if info.CPUUsage > 80 || info.MemoryUsage > 80 || info.DiskUsage > 85 {
        info.Status = "warning"
      } else {
        info.Status = "ok"
      }

      resultChan <- info
    }(vmName)
  }

  // 收集结果
  var results []VMResourceInfo
  for i := 0; i < len(req.VMNames); i++ {
    results = append(results, <-resultChan)
  }
  close(resultChan)

  writeJSON(w, http.StatusOK, results)
}

// getCurrentUser 获取当前请求的用户名
func (s *Server) getCurrentUser(r *http.Request) string {
  if s.config.BasicAuth.Enabled {
    if username, _, ok := r.BasicAuth(); ok {
      return username
    }
  }
  return "anonymous"
}

// recordCommandHistory 记录命令历史
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

// ------------------------- P1: Command Template Handlers -------------------------

// handleListCommandTemplates 列出命令模板
func (s *Server) handleListCommandTemplates(w http.ResponseWriter, r *http.Request) {
  if s.commandTemplateDB == nil {
    writeError(w, http.StatusInternalServerError, "Command template database not available")
    return
  }

  username := s.getCurrentUser(r)
  isAdmin := s.isAdmin(r)

  // 获取查询参数
  category := r.URL.Query().Get("category")
  sharedOnly := r.URL.Query().Get("shared") == "true"
  search := r.URL.Query().Get("search")

  query := "SELECT id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at FROM command_templates WHERE 1=1"
  var args []interface{}

  if category != "" {
    query += " AND category = ?"
    args = append(args, category)
  }

  if sharedOnly {
    query += " AND is_shared = 1"
  } else {
    // 非管理员只能看到共享的或自己创建的模板
    if !isAdmin {
      query += " AND (is_shared = 1 OR created_by = ?)"
      args = append(args, username)
    }
  }

  if search != "" {
    query += " AND (name LIKE ? OR description LIKE ? OR command LIKE ?)"
    searchPattern := "%" + search + "%"
    args = append(args, searchPattern, searchPattern, searchPattern)
  }

  query += " ORDER BY updated_at DESC"

  rows, err := s.commandTemplateDB.Query(query, args...)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to query templates: "+err.Error())
    return
  }
  defer rows.Close()

  var templates []CommandTemplate
  for rows.Next() {
    var tpl CommandTemplate
    var variablesJSON string
    var isShared int
    err := rows.Scan(
      &tpl.ID,
      &tpl.Name,
      &tpl.Description,
      &tpl.Command,
      &variablesJSON,
      &tpl.Category,
      &isShared,
      &tpl.CreatedBy,
      &tpl.CreatedAt,
      &tpl.UpdatedAt,
    )
    if err != nil {
      continue
    }
    tpl.IsShared = isShared == 1
    json.Unmarshal([]byte(variablesJSON), &tpl.Variables)
    templates = append(templates, tpl)
  }

  writeJSON(w, http.StatusOK, templates)
}

// handleCreateCommandTemplate 创建命令模板
func (s *Server) handleCreateCommandTemplate(w http.ResponseWriter, r *http.Request) {
  if s.commandTemplateDB == nil {
    writeError(w, http.StatusInternalServerError, "Command template database not available")
    return
  }

  var req CommandTemplate
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  if req.Name == "" || req.Command == "" {
    writeError(w, http.StatusBadRequest, "name and command are required")
    return
  }

  req.ID = fmt.Sprintf("tpl_%d", time.Now().UnixNano())
  req.CreatedBy = s.getCurrentUser(r)
  req.CreatedAt = time.Now()
  req.UpdatedAt = time.Now()

  variablesJSON, _ := json.Marshal(req.Variables)

  _, err := s.commandTemplateDB.Exec(
    "INSERT INTO command_templates (id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    req.ID,
    req.Name,
    req.Description,
    req.Command,
    string(variablesJSON),
    req.Category,
    boolToInt(req.IsShared),
    req.CreatedBy,
    req.CreatedAt,
    req.UpdatedAt,
  )

  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to create template: "+err.Error())
    return
  }

  writeJSON(w, http.StatusCreated, req)
}

// handleGetCommandTemplate 获取命令模板详情
func (s *Server) handleGetCommandTemplate(w http.ResponseWriter, r *http.Request) {
  if s.commandTemplateDB == nil {
    writeError(w, http.StatusInternalServerError, "Command template database not available")
    return
  }

  vars := mux.Vars(r)
  id := vars["id"]

  var tpl CommandTemplate
  var variablesJSON string
  var isShared int

  err := s.commandTemplateDB.QueryRow(
    "SELECT id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at FROM command_templates WHERE id = ?",
    id,
  ).Scan(
    &tpl.ID,
    &tpl.Name,
    &tpl.Description,
    &tpl.Command,
    &variablesJSON,
    &tpl.Category,
    &isShared,
    &tpl.CreatedBy,
    &tpl.CreatedAt,
    &tpl.UpdatedAt,
  )

  if err == sql.ErrNoRows {
    writeError(w, http.StatusNotFound, "Template not found")
    return
  }
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to get template: "+err.Error())
    return
  }

  tpl.IsShared = isShared == 1
  json.Unmarshal([]byte(variablesJSON), &tpl.Variables)

  writeJSON(w, http.StatusOK, tpl)
}

// handleUpdateCommandTemplate 更新命令模板
func (s *Server) handleUpdateCommandTemplate(w http.ResponseWriter, r *http.Request) {
  if s.commandTemplateDB == nil {
    writeError(w, http.StatusInternalServerError, "Command template database not available")
    return
  }

  vars := mux.Vars(r)
  id := vars["id"]

  var req CommandTemplate
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  // 检查权限
  var createdBy string
  err := s.commandTemplateDB.QueryRow("SELECT created_by FROM command_templates WHERE id = ?", id).Scan(&createdBy)
  if err == sql.ErrNoRows {
    writeError(w, http.StatusNotFound, "Template not found")
    return
  }

  username := s.getCurrentUser(r)
  if createdBy != username && !s.isAdmin(r) {
    writeError(w, http.StatusForbidden, "Permission denied")
    return
  }

  variablesJSON, _ := json.Marshal(req.Variables)

  _, err = s.commandTemplateDB.Exec(
    "UPDATE command_templates SET name = ?, description = ?, command = ?, variables = ?, category = ?, is_shared = ?, updated_at = ? WHERE id = ?",
    req.Name,
    req.Description,
    req.Command,
    string(variablesJSON),
    req.Category,
    boolToInt(req.IsShared),
    time.Now(),
    id,
  )

  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to update template: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Template updated successfully"})
}

// handleDeleteCommandTemplate 删除命令模板
func (s *Server) handleDeleteCommandTemplate(w http.ResponseWriter, r *http.Request) {
  if s.commandTemplateDB == nil {
    writeError(w, http.StatusInternalServerError, "Command template database not available")
    return
  }

  vars := mux.Vars(r)
  id := vars["id"]

  // 检查权限
  var createdBy string
  err := s.commandTemplateDB.QueryRow("SELECT created_by FROM command_templates WHERE id = ?", id).Scan(&createdBy)
  if err == sql.ErrNoRows {
    writeError(w, http.StatusNotFound, "Template not found")
    return
  }

  username := s.getCurrentUser(r)
  if createdBy != username && !s.isAdmin(r) {
    writeError(w, http.StatusForbidden, "Permission denied")
    return
  }

  _, err = s.commandTemplateDB.Exec("DELETE FROM command_templates WHERE id = ?", id)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to delete template: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Template deleted successfully"})
}

// ------------------------- P1: Command History Handlers -------------------------

// handleListCommandHistory 获取命令历史记录
func (s *Server) handleListCommandHistory(w http.ResponseWriter, r *http.Request) {
  if s.commandHistoryDB == nil {
    writeError(w, http.StatusInternalServerError, "Command history database not available")
    return
  }

  // 获取查询参数
  vmName := r.URL.Query().Get("vm_name")
  search := r.URL.Query().Get("search")
  pageStr := r.URL.Query().Get("page")
  pageSizeStr := r.URL.Query().Get("page_size")

  page := 1
  pageSize := 20
  if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
    page = p
  }
  if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
    pageSize = ps
  }

  query := "SELECT id, vm_name, command, executed_by, executed_at, status, duration_ms FROM command_history WHERE 1=1"
  countQuery := "SELECT COUNT(*) FROM command_history WHERE 1=1"
  var args []interface{}
  var countArgs []interface{}

  if vmName != "" {
    query += " AND vm_name = ?"
    countQuery += " AND vm_name = ?"
    args = append(args, vmName)
    countArgs = append(countArgs, vmName)
  }

  if search != "" {
    query += " AND command LIKE ?"
    countQuery += " AND command LIKE ?"
    searchPattern := "%" + search + "%"
    args = append(args, searchPattern)
    countArgs = append(countArgs, searchPattern)
  }

  // 获取总数
  var total int
  err := s.commandHistoryDB.QueryRow(countQuery, countArgs...).Scan(&total)
  if err != nil {
    total = 0
  }

  query += " ORDER BY executed_at DESC LIMIT ? OFFSET ?"
  args = append(args, pageSize, (page-1)*pageSize)

  rows, err := s.commandHistoryDB.Query(query, args...)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to query history: "+err.Error())
    return
  }
  defer rows.Close()

  var items []CommandHistory
  for rows.Next() {
    var h CommandHistory
    err := rows.Scan(
      &h.ID,
      &h.VMName,
      &h.Command,
      &h.ExecutedBy,
      &h.ExecutedAt,
      &h.Status,
      &h.DurationMs,
    )
    if err != nil {
      continue
    }
    items = append(items, h)
  }

  response := struct {
    Total    int              `json:"total"`
    Page     int              `json:"page"`
    PageSize int              `json:"page_size"`
    Items    []CommandHistory `json:"items"`
  }{
    Total:    total,
    Page:     page,
    PageSize: pageSize,
    Items:    items,
  }

  writeJSON(w, http.StatusOK, response)
}

// handleRecordCommandHistory 记录命令执行历史
func (s *Server) handleRecordCommandHistory(w http.ResponseWriter, r *http.Request) {
  if s.commandHistoryDB == nil {
    writeError(w, http.StatusInternalServerError, "Command history database not available")
    return
  }

  var req struct {
    VMName     string `json:"vm_name"`
    Command    string `json:"command"`
    Status     string `json:"status"`
    DurationMs int64  `json:"duration_ms"`
    Output     string `json:"output"`
    Error      string `json:"error"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  if req.VMName == "" || req.Command == "" {
    writeError(w, http.StatusBadRequest, "vm_name and command are required")
    return
  }

  history := CommandHistory{
    ID:         fmt.Sprintf("hist_%d", time.Now().UnixNano()),
    VMName:     req.VMName,
    Command:    req.Command,
    ExecutedBy: s.getCurrentUser(r),
    ExecutedAt: time.Now(),
    Status:     req.Status,
    DurationMs: req.DurationMs,
    Output:     req.Output,
    Error:      req.Error,
  }

  _, err := s.commandHistoryDB.Exec(
    "INSERT INTO command_history (id, vm_name, command, executed_by, executed_at, status, duration_ms, output, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
    history.ID,
    history.VMName,
    history.Command,
    history.ExecutedBy,
    history.ExecutedAt,
    history.Status,
    history.DurationMs,
    history.Output,
    history.Error,
  )

  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to record history: "+err.Error())
    return
  }

  writeJSON(w, http.StatusCreated, history)
}

// handleDeleteCommandHistory 删除命令历史记录
func (s *Server) handleDeleteCommandHistory(w http.ResponseWriter, r *http.Request) {
  if s.commandHistoryDB == nil {
    writeError(w, http.StatusInternalServerError, "Command history database not available")
    return
  }

  vars := mux.Vars(r)
  id := vars["id"]

  _, err := s.commandHistoryDB.Exec("DELETE FROM command_history WHERE id = ?", id)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to delete history: "+err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "History deleted successfully"})
}

// ------------------------- P2: File Transfer Handler -------------------------

// handleVMFileTransfer 处理跨服务器文件传输
func (s *Server) handleVMFileTransfer(w http.ResponseWriter, r *http.Request) {
  var req FileTransferRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
    return
  }

  if req.SourceVM == "" || req.SourcePath == "" || req.TargetVM == "" || req.TargetPath == "" {
    writeError(w, http.StatusBadRequest, "source_vm, source_path, target_vm, and target_path are required")
    return
  }

  start := time.Now()

  // 获取源VM信息
  sourceVM, err := s.vmManager.GetVM(req.SourceVM)
  if err != nil {
    writeError(w, http.StatusNotFound, "Source VM not found: "+err.Error())
    return
  }

  // 获取目标VM信息
  targetVM, err := s.vmManager.GetVM(req.TargetVM)
  if err != nil {
    writeError(w, http.StatusNotFound, "Target VM not found: "+err.Error())
    return
  }

  // 创建源SSH客户端
  sourceSSH, err := vm.NewSSHClient(&vm.SSHConfig{
    Host:     sourceVM.IP,
    Port:     sourceVM.Port,
    Username: sourceVM.Username,
    Password: sourceVM.Password,
    KeyPath:  sourceVM.KeyPath,
  })
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to create source SSH client: "+err.Error())
    return
  }

  if err := sourceSSH.Connect(sourceVM.IP, sourceVM.Port); err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to connect to source VM: "+err.Error())
    return
  }
  defer sourceSSH.Close()

  // 创建目标SSH客户端
  targetSSH, err := vm.NewSSHClient(&vm.SSHConfig{
    Host:     targetVM.IP,
    Port:     targetVM.Port,
    Username: targetVM.Username,
    Password: targetVM.Password,
    KeyPath:  targetVM.KeyPath,
  })
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to create target SSH client: "+err.Error())
    return
  }

  if err := targetSSH.Connect(targetVM.IP, targetVM.Port); err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to connect to target VM: "+err.Error())
    return
  }
  defer targetSSH.Close()

  // 检查源文件是否存在
  checkOutput, err := sourceSSH.ExecuteCommand(fmt.Sprintf("test -f %s && echo 'exists' || echo 'not found'", req.SourcePath))
  if err != nil || strings.TrimSpace(checkOutput) != "exists" {
    writeJSON(w, http.StatusNotFound, map[string]interface{}{
      "error":       "Source file not found",
      "source_vm":   req.SourceVM,
      "source_path": req.SourcePath,
    })
    return
  }

  // 获取文件大小
  sizeOutput, _ := sourceSSH.ExecuteCommand(fmt.Sprintf("stat -c %%s %s", req.SourcePath))
  fileSize := int64(0)
  if sizeOutput != "" {
    fileSize, _ = strconv.ParseInt(strings.TrimSpace(sizeOutput), 10, 64)
  }

  // 下载文件内容
  fileContent, err := sourceSSH.DownloadFile(req.SourcePath)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to download file from source: "+err.Error())
    return
  }

  // 确保目标目录存在
  targetDir := filepath.Dir(req.TargetPath)
  _, _ = targetSSH.ExecuteCommand(fmt.Sprintf("mkdir -p %s", targetDir))

  // 上传文件到目标
  err = targetSSH.UploadFileContent(fileContent, req.TargetPath)
  if err != nil {
    writeError(w, http.StatusInternalServerError, "Failed to upload file to target: "+err.Error())
    return
  }

  duration := time.Since(start)

  response := FileTransferResponse{
    Message:          "File transferred successfully",
    BytesTransferred: int64(len(fileContent)),
    DurationMs:       duration.Milliseconds(),
  }

  if fileSize > 0 {
    response.BytesTransferred = fileSize
  }

  writeJSON(w, http.StatusOK, response)
}

// ------------------------- Helper Functions -------------------------

// isAdmin 检查当前用户是否为管理员
func (s *Server) isAdmin(r *http.Request) bool {
  if !s.config.BasicAuth.Enabled {
    return true
  }

  username, _, ok := r.BasicAuth()
  if !ok {
    return false
  }

  // 检查是否为超级管理员
  if username == s.config.BasicAuth.Username {
    return true
  }

  // 检查用户角色
  if s.userDatabase != nil {
    if user, exists := s.userDatabase.GetUser(username); exists && user.Role == "admin" {
      return true
    }
  }

  return false
}

// boolToInt 将布尔值转换为整数
func boolToInt(b bool) int {
  if b {
    return 1
  }
  return 0
}

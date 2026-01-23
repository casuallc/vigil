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
  "encoding/json"
  "fmt"
  "io"
  "log"
  "net/http"
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
  // 解析请求体
  var groupData struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    VMs         []string `json:"vms"`
  }
  if err := json.NewDecoder(r.Body).Decode(&groupData); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 验证请求数据
  if groupData.Name == "" {
    writeError(w, http.StatusBadRequest, "group name is required")
    return
  }

  // 添加组
  if err := s.vmManager.AddGroup(groupData.Name, groupData.Description, groupData.VMs); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Group added successfully"})
}

// handleListGroups 处理列出VM组的请求
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
  // 获取所有组
  groups := s.vmManager.ListGroups()

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
  }
  if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 更新组
  if err := s.vmManager.UpdateGroup(groupName, updateData.Description, updateData.VMs); err != nil {
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
  type FileListRequest struct {
    VMName   string `json:"vm_name"`
    Path     string `json:"path"`
    MaxDepth int    `json:"max_depth"`
  }

  var req FileListRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 验证VM名称
  if req.VMName == "" {
    writeError(w, http.StatusBadRequest, "vm_name is required")
    return
  }

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(req.VMName)
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
  // 解析multipart/form-data请求
  err := r.ParseMultipartForm(10 << 20) // 10MB大小限制
  if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 获取VM名称
  vmName := r.FormValue("vm_name")
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm_name is required")
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
  type FileDownloadRequest struct {
    VMName     string `json:"vm_name"`
    SourcePath string `json:"source_path"`
  }

  var req FileDownloadRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // 验证VM名称
  if req.VMName == "" {
    writeError(w, http.StatusBadRequest, "vm_name is required")
    return
  }

  // 获取VM信息
  vmInfo, err := s.vmManager.GetVM(req.VMName)
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

  // ---------------- 输入：WS → SSH ----------------
  go func() {
    defer cancel()
    for {
      messageType, payload, err := ws.ReadMessage()
      if err != nil {
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
    buf := make([]byte, 32*1024)

    for {
      n, err := reader.Read(buf)
      if n > 0 {
        ws.WriteMessage(websocket.BinaryMessage, buf[:n])
      }
      if err != nil {
        return
      }
    }
  }()

  <-ctx.Done()
  session.Wait()
}

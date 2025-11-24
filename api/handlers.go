package api

import (
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/inspection"
  "github.com/casuallc/vigil/proc"
  "github.com/gorilla/mux"
  "net/http"
  "regexp"
  "strconv"
  "strings"
  "time"
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
    result := s.executeCheck(check, req.Env)
    results = append(results, result)

    // 统计检查结果
    switch strings.ToLower(result.Status) {
    case "ok", "pass":
      passedChecks++
    case "warning":
      warningChecks++
    case "error", "fail", "critical":
      errorChecks++
    }
  }

  // 构建响应
  // 确定整体状态
  var overallStatus string = "OK"
  if errorChecks > 0 {
    overallStatus = "CRITICAL"
  } else if warningChecks > 0 {
    overallStatus = "WARN"
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

// executeCheck 执行单个检查项
func (s *Server) executeCheck(check inspection.Check, envVars []string) inspection.CheckResult {
  // 记录开始时间
  startTime := time.Now()
  result := s.executeScriptCheck(check, envVars)

  // 计算执行时间
  result.DurationMs = int64(int(time.Since(startTime).Milliseconds()))

  return result
}

// executeScriptCheck 执行脚本检查
func (s *Server) executeScriptCheck(check inspection.Check, envVars []string) inspection.CheckResult {
  result := inspection.CheckResult{
    ID:       check.ID,
    Name:     check.Name,
    Type:     check.Type,
    Status:   "ok",
    Severity: "info",
  }

  // 获取命令
  commandLines, err := check.GetCommandLines()
  if err != nil || len(commandLines) == 0 {
    result.Status = "error"
    result.Severity = "critical"
    result.Message = fmt.Sprintf("Failed to get command: %v", err)
    return result
  }

  // 执行命令
  output, err := common.ExecuteCommand(commandLines[0], envVars)
  if err != nil {
    result.Status = "error"
    result.Severity = "critical"
    result.Message = fmt.Sprintf("Command execution failed: %v, output: %s", err, output)
    return result
  }

  // 解析输出
  result = s.parseCheckOutput(check, output, result)
  return result
}

// parseCheckOutput 解析检查输出
func (s *Server) parseCheckOutput(check inspection.Check, output string, result inspection.CheckResult) inspection.CheckResult {
  if check.Parse == nil {
    result.Message = output
    return result
  }

  switch check.Parse.Kind {
  case "regex":
    if check.Parse.Pattern != "" {
      re := regexp.MustCompile(check.Parse.Pattern)
      if matches := re.FindStringSubmatch(output); len(matches) > 1 {
        if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
          result.Value = val
        }
      }
    }
  case "json":
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(output), &data); err == nil {
      if val, ok := data[check.Parse.Path]; ok {
        if valFloat, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64); err == nil {
          result.Value = valFloat
        }
      }
    }
  case "threshold":
    // 阈值解析在executeScriptCheck中处理
  default:
    result.Message = output
  }

  return result
}

// evaluateThreshold 评估阈值
func (s *Server) evaluateThreshold(value float64, threshold *inspection.Threshold, result inspection.CheckResult) inspection.CheckResult {
  if threshold == nil {
    return result
  }

  switch threshold.Operator {
  case ">":
    if value > threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f exceeds threshold %.2f", value, threshold.Value)
    }
  case ">=":
    if value >= threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f exceeds or equals threshold %.2f", value, threshold.Value)
    }
  case "<":
    if value < threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f below threshold %.2f", value, threshold.Value)
    }
  case "<=":
    if value <= threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f below or equals threshold %.2f", value, threshold.Value)
    }
  case "==":
    if value == threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f equals threshold %.2f", value, threshold.Value)
    }
  case "!=":
    if value != threshold.Value {
      result.Status = "error"
      result.Severity = "critical"
      result.Message = fmt.Sprintf("Critical: value %.2f not equals threshold %.2f", value, threshold.Value)
    }
  default:
    result.Message = fmt.Sprintf("Unknown operator: %s", threshold.Operator)
  }

  return result
}

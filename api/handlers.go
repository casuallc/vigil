package api

import (
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/inspection"
  "github.com/casuallc/vigil/proc"
  "github.com/expr-lang/expr"
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
    case inspection.StatusOk:
      passedChecks++
    case inspection.StatusWarn:
      warningChecks++
    case inspection.StatusError:
      errorChecks++
    }
  }

  // 构建响应
  // 确定整体状态
  var overallStatus string = inspection.StatusOk
  if errorChecks > 0 {
    overallStatus = inspection.StatusError
  } else if warningChecks > 0 {
    overallStatus = inspection.StatusWarn
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

  if result.Value == nil || result.Status == inspection.StatusError || result.Status == inspection.StatusWarn {
    return result
  }

  // 根据 docs/inspection_rules.md 中定义的规则进行解析和结果判断
  if check.Expect != nil {
    // 处理 expect 匹配
    s.handleExpectMatch(check, &result)
  } else if len(check.Compare) > 0 {
    // 处理 compare 匹配
    s.handleCompare(check, &result)
  } else if len(check.Thresholds) > 0 {
    // 处理阈值判断
    s.handleThresholds(check, &result)
  } else {
    result.Status = inspection.StatusError
  }

  return result
}

// executeScriptCheck 执行脚本检查
func (s *Server) executeScriptCheck(check inspection.Check, envVars []string) inspection.CheckResult {
  result := inspection.CheckResult{
    ID:       check.ID,
    Name:     check.Name,
    Type:     check.Type,
    Status:   inspection.StatusOk,
    Severity: "info",
  }

  // 获取命令
  commandLines, err := check.GetCommandLines()
  if err != nil || len(commandLines) == 0 {
    result.Status = inspection.StatusError
    result.Severity = inspection.SeverityCritical
    result.Message = fmt.Sprintf("Failed to get command: %v", err)
    return result
  }

  // 执行命令
  output, err := common.ExecuteCommand(commandLines[0], envVars)
  if err != nil {
    result.Status = inspection.StatusError
    result.Severity = inspection.SeverityCritical
    result.Message = fmt.Sprintf("Command execution failed: %v, output: %s", err, output)
    return result
  }

  // 解析输出
  result = s.parseCheckOutput(check, output, result)

  return result
}

// parseCheckOutput 解析检查输出
func (s *Server) parseCheckOutput(check inspection.Check, output string, result inspection.CheckResult) inspection.CheckResult {
  result.Message = output
  if check.Parse == nil {
    return result
  }
  var parseErr error
  switch check.Parse.Kind {
  case "regex":
    if check.Parse.Pattern != "" {
      re := regexp.MustCompile(check.Parse.Pattern)
      if matches := re.FindStringSubmatch(output); len(matches) > 1 {
        if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
          result.Value = val
        } else {
          parseErr = err
        }
      }
    }
  case "json":
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(output), &data); err == nil {
      if val, ok := data[check.Parse.Path]; ok {
        if valFloat, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64); err == nil {
          result.Value = valFloat
        } else {
          parseErr = err
        }
      }
    }
  case "int":
    if val, err := strconv.ParseInt(output, 10, 64); err == nil {
      result.Value = val
    } else {
      parseErr = err
    }
  case "float":
    if val, err := strconv.ParseFloat(output, 64); err == nil {
      result.Value = val
    } else {
      parseErr = err
    }
  default:
    result.Value = output
  }
  // 获取不到值，则返回错误
  if result.Value == nil {
    errStr := ""
    if parseErr != nil {
      errStr = parseErr.Error()
    }
    result.Message = fmt.Sprintf("Can not parse value with %s, error: %s", check.Parse.Kind, errStr)
    result.Status = inspection.StatusError
  }

  return result
}

// handleExpectMatch 处理期望匹配
func (s *Server) handleExpectMatch(check inspection.Check, result *inspection.CheckResult) {
  output := fmt.Sprintf("%v", result.Value)

  expectLines, err := check.GetExpectLines()
  if err != nil {
    result.Status = inspection.StatusError
    result.Severity = inspection.SeverityCritical
    result.Message = fmt.Sprintf("Invalid expect configuration: %v", err)
    return
  }

  // 检查输出是否包含任一期望的行
  matched := false
  for _, expect := range expectLines {
    if strings.Contains(output, expect) {
      matched = true
      break
    }
  }

  if !matched {
    result.Status = inspection.StatusError
    result.Severity = string(check.Severity)
    if result.Severity == "" {
      result.Severity = inspection.SeverityWarn
    }
    result.Message = fmt.Sprintf("Output does not match expected pattern(s)")
  }
}

// handleCompare 处理数值比较
func (s *Server) handleCompare(check inspection.Check, result *inspection.CheckResult) {
  value, err := parseFloatValue(result.Value)
  if err != nil {
    result.Status = inspection.StatusError
    result.Severity = inspection.SeverityCritical
    result.Message = fmt.Sprintf("Failed to parse value: %v", err)
    return
  }

  // 使用expr-lang评估阈值表达式
  env := map[string]interface{}{
    "value": value,
  }
  exprStr := "value " + check.Compare
  // 评估表达式
  evalResult, err := expr.Eval(exprStr, env)
  if err != nil {
    return
  }

  // 如果表达式为真，则应用该阈值规则
  if match, ok := evalResult.(bool); ok && match {
    result.Status = "ok"
    result.Severity = "ok"
    result.Message = fmt.Sprintf("Threshold condition met: %s", check.Compare)
  } else {
    result.Status = "error"
    result.Severity = "error"
  }
}

// handleThresholds 处理阈值判断
func (s *Server) handleThresholds(check inspection.Check, result *inspection.CheckResult) {
  value, err := parseFloatValue(result.Value)
  if err != nil {
    result.Status = inspection.StatusError
    result.Severity = inspection.SeverityCritical
    result.Message = fmt.Sprintf("Failed to parse value: %v", err)
    return
  }

  // 使用expr-lang评估阈值表达式
  env := map[string]interface{}{
    "value": value,
  }

  // 检查每个阈值规则
  for _, threshold := range check.Thresholds {
    // 构建表达式
    exprStr := "value " + threshold.When

    // 评估表达式
    evalResult, err := expr.Eval(exprStr, env)
    if err != nil {
      continue // 跳过无效的表达式
    }

    // 如果表达式为真，则应用该阈值规则
    if match, ok := evalResult.(bool); ok && match {
      result.Status = string(threshold.Severity)
      result.Severity = string(threshold.Severity)
      if threshold.Message != "" {
        result.Message = threshold.Message
      } else {
        result.Message = fmt.Sprintf("Threshold condition met: %s", threshold.When)
      }
      break // 一旦匹配到阈值规则就停止检查
    }
  }
}

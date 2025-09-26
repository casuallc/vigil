package api

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "net/http"
  "strconv"
  "time"

  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/process"
  "github.com/gorilla/mux"
)

// Server represents the HTTP API server
type Server struct {
  config  *config.Config
  manager *process.Manager
  monitor *process.Monitor
}

// NewServerWithManager creates a new API server with an existing process manager
func NewServerWithManager(config *config.Config, manager *process.Manager) *Server {
  monitor := process.NewMonitor(manager)

  return &Server{
    config:  config,
    manager: manager,
    monitor: monitor,
  }
}

// 添加日志中间件函数
type loggingResponseWriter struct {
  http.ResponseWriter
  statusCode int
  body       []byte
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
  return &loggingResponseWriter{
    ResponseWriter: w,
    statusCode:     http.StatusOK,
  }
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
  lrw.statusCode = code
  lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
  lrw.body = append(lrw.body, b...)
  return lrw.ResponseWriter.Write(b)
}

// LoggingMiddleware 是一个HTTP中间件，用于记录请求和响应
func LoggingMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 创建一个可记录响应的ResponseWriter
    lrw := newLoggingResponseWriter(w)

    // 记录请求开始时间
    startTime := time.Now()

    // 读取并记录请求体
    var requestBody []byte
    if r.Body != nil {
      requestBody, _ = io.ReadAll(r.Body)
      // 重新设置请求体，以便后续处理可以读取
      r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
    }

    // 记录请求信息
    log.Printf("[REQUEST] Method: %s, URL: %s, RemoteAddr: %s, Body: %s",
      r.Method, r.URL.String(), r.RemoteAddr, string(requestBody))

    // 处理请求
    next.ServeHTTP(lrw, r)

    // 计算请求处理时间
    duration := time.Since(startTime)

    // 记录响应信息（对于大型响应，可以限制记录的长度）
    maxLogBodySize := 65536 // 限制日志中响应体的大小
    responseBody := lrw.body
    if len(responseBody) > maxLogBodySize {
      responseBody = responseBody[:maxLogBodySize]
    }

    log.Printf("[RESPONSE] Method: %s, URL: %s, Status: %d, Duration: %v, Body: %s",
      r.Method, r.URL.String(), lrw.statusCode, duration, string(responseBody))
  })
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
  // 获取路由
  r := s.Router()

  // 应用日志中间件到所有路由
  loggedRouter := LoggingMiddleware(r)

  log.Printf("Starting API server on %s", addr)
  return http.ListenAndServe(addr, loggedRouter)
}

// API response helpers
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(statusCode)
  json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
  writeJSON(w, statusCode, map[string]string{"error": message})
}

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

func (s *Server) handleManageProcess(w http.ResponseWriter, r *http.Request) {
  var process process.ManagedProcess
  if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  if err := s.manager.ManageProcess(process); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusCreated, map[string]string{"message": "Process managed successfully"})
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  name := vars["name"]

  if err := s.manager.StartProcess(name); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Process started successfully"})
}

func (s *Server) handleStopProcess(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  name := vars["name"]

  if err := s.manager.StopProcess(name); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Process stopped successfully"})
}

func (s *Server) handleRestartProcess(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  name := vars["name"]

  if err := s.manager.RestartProcess(name); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "Process restarted successfully"})
}

// 处理函数更新示例
func (s *Server) handleGetProcess(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  namespace := vars["namespace"]
  name := vars["name"]
  
  // 兼容旧版API，没有指定namespace时使用"default"
  if namespace == "" {
    namespace = "default"
  }

  process, err := s.manager.GetProcessStatus(namespace, name)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, process)
}

func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  namespace := vars["namespace"]
  
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
  namespace := vars["namespace"]
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
  resources, err := process.GetSystemResourceUsage()
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

  resources, err := process.GetProcessResourceUsage(pid)
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

// handleDeleteProcess handles the DELETE /api/processes/{name} endpoint
func (s *Server) handleDeleteProcess(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  name := vars["name"]

  err := s.manager.DeleteProcess(name)
  if err != nil {
    http.Error(w, err.Error(), http.StatusNotFound)
    return
  }

  w.WriteHeader(http.StatusOK)
  w.Write([]byte(fmt.Sprintf("Process %s deleted successfully", name)))
}

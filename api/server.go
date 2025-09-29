package api

import (
  "bytes"
  "encoding/json"
  "github.com/casuallc/vigil/process/monitor"
  "io"
  "log"
  "net/http"
  "time"

  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/process"
)

// Server represents the HTTP API server
type Server struct {
  config  *config.Config
  manager *process.Manager
  monitor *monitor.Monitor
}

// NewServerWithManager creates a new API server with an existing process manager
func NewServerWithManager(config *config.Config, manager *process.Manager) *Server {
  monitor := monitor.NewMonitor(manager)

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

// 获取 namespace
func getNamespace(vars map[string]string) string {
  namespace := vars["namespace"]

  // 兼容旧版API，没有指定namespace时使用"default"
  if namespace == "" {
    namespace = "default"
  }
  return namespace
}

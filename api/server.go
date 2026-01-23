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
  "bufio"
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "net"
  "net/http"
  "os"
  "path/filepath"
  "strings"
  "time"

  "github.com/casuallc/vigil/audit"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/proc"
  "github.com/casuallc/vigil/vm"
)

// Server represents the HTTP API server
type Server struct {
  config      *config.Config
  manager     *proc.Manager
  monitor     *proc.Monitor
  vmManager   *vm.Manager
  auditLogger *audit.Logger
}

// NewServerWithManager creates a new API server with an existing proc manager
func NewServerWithManager(config *config.Config, manager *proc.Manager) *Server {
  monitor := proc.NewMonitor(manager)
  vmManager := vm.NewManagerWithConfig("vms.json", config.Security.EncryptionKey)

  // 创建审计日志目录
  auditLogDir := filepath.Join("logs", "audit")

  // 初始化审计日志记录器
  auditLogger, err := audit.NewLogger(auditLogDir)
  if err != nil {
    log.Printf("Warning: failed to initialize audit logger: %v", err)
    // 继续运行，即使审计日志初始化失败
  }

  return &Server{
    config:      config,
    manager:     manager,
    monitor:     monitor,
    vmManager:   vmManager,
    auditLogger: auditLogger,
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

// Hijack 实现http.Hijacker接口，以支持WebSocket
func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
  if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
    return hijacker.Hijack()
  }
  return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
}

// AuditMiddleware 是一个HTTP中间件，用于记录审计日志
func (s *Server) AuditMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 获取客户端IP
    clientIP := r.RemoteAddr
    if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
      clientIP = forwardedFor
    }

    // 获取用户信息（这里简化处理，实际应用中可能需要从认证中间件获取）
    user := "anonymous"
    if auth := r.Header.Get("Authorization"); auth != "" {
      // 这里可以从认证头中提取用户信息
      user = "authenticated"
    }

    // 解析请求路径，确定操作类型
    path := r.URL.Path
    action := audit.ActionType("unknown")
    resource := ""

    // 根据路径确定操作类型和资源
    switch {
    case strings.HasPrefix(path, "/api/vms"):
      switch r.Method {
      case http.MethodPost:
        action = audit.ActionVMAdd
      case http.MethodGet:
        if strings.Count(path, "/") == 3 {
          action = audit.ActionVMGet
          resource = strings.TrimPrefix(path, "/api/vms/")
        } else {
          action = audit.ActionVMList
        }
      case http.MethodPut:
        action = audit.ActionVMUpdate
        resource = strings.TrimPrefix(path, "/api/vms/")
      case http.MethodDelete:
        action = audit.ActionVMDelete
        resource = strings.TrimPrefix(path, "/api/vms/")
      }
    case strings.HasPrefix(path, "/api/vms/groups"):
      switch r.Method {
      case http.MethodPost:
        action = audit.ActionGroupAdd
      case http.MethodGet:
        if strings.Count(path, "/") == 3 {
          action = audit.ActionGroupGet
          resource = strings.TrimPrefix(path, "/api/vms/groups/")
        } else {
          action = audit.ActionGroupList
        }
      case http.MethodPut:
        action = audit.ActionGroupUpdate
        resource = strings.TrimPrefix(path, "/api/vms/groups/")
      case http.MethodDelete:
        action = audit.ActionGroupDelete
        resource = strings.TrimPrefix(path, "/api/vms/groups/")
      }
    case strings.HasPrefix(path, "/api/ssh"):
      action = audit.ActionVMSSH
      resource = r.URL.Query().Get("vm")
    case strings.HasPrefix(path, "/api/file/upload"):
      action = audit.ActionFileUpload
      resource = r.URL.Query().Get("vm")
    case strings.HasPrefix(path, "/api/file/download"):
      action = audit.ActionFileDownload
      resource = r.URL.Query().Get("vm")
    case strings.HasPrefix(path, "/api/file/list"):
      action = audit.ActionFileList
      resource = r.URL.Query().Get("vm")
    case strings.HasPrefix(path, "/api/permissions"):
      switch r.Method {
      case http.MethodPost:
        action = audit.ActionPermissionAdd
      case http.MethodGet:
        if strings.Count(path, "/") == 3 {
          action = audit.ActionPermissionList
          resource = strings.TrimPrefix(path, "/api/permissions/")
        }
      case http.MethodDelete:
        action = audit.ActionPermissionRemove
        resource = strings.TrimPrefix(path, "/api/permissions/")
      }
    }

    // 创建一个可记录响应的ResponseWriter
    lrw := newLoggingResponseWriter(w)

    // 记录请求开始时间
    startTime := time.Now()

    // 调用下一个处理函数
    next.ServeHTTP(lrw, r)

    // 计算请求处理时间
    elapsedTime := time.Since(startTime)

    // 确定操作状态
    status := audit.StatusSuccess
    if lrw.statusCode >= 400 {
      status = audit.StatusFailed
    }

    // 创建审计日志条目
    message := fmt.Sprintf("%s %s %d %v", r.Method, r.URL.Path, lrw.statusCode, elapsedTime)
    details := map[string]interface{}{
      "method":      r.Method,
      "path":        r.URL.Path,
      "query":       r.URL.RawQuery,
      "status_code": lrw.statusCode,
      "elapsed_ms":  elapsedTime.Milliseconds(),
    }

    // 记录审计日志
    if s.auditLogger != nil {
      logEntry := audit.NewLogEntry(user, clientIP, action, resource, status, message, details)
      if err := s.auditLogger.Log(logEntry); err != nil {
        log.Printf("Error logging audit entry: %v", err)
      }
    }
  })
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

// BasicAuthMiddleware enforces HTTP Basic Auth.
func BasicAuthMiddleware(username, password string, next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    u, p, ok := r.BasicAuth()
    if !ok || u != username || p != password {
      w.Header().Set("WWW-Authenticate", `Basic realm="vigil"`)
      http.Error(w, "Unauthorized", http.StatusUnauthorized)
      return
    }
    next.ServeHTTP(w, r)
  })
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

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
  // 获取路由
  r := s.Router()

  // 应用日志中间件到所有路由
  loggedRouter := LoggingMiddleware(r)

  // 从 conf/app.conf 加载 Basic Auth 凭据
  confPath := filepath.Join("conf", "app.conf")
  kv, err := common.LoadKeyValues(confPath)
  if err != nil {
    log.Printf("warning: failed to load %s: %v", confPath, err)
  }
  user := kv["BASIC_AUTH_USER"]
  pass := kv["BASIC_AUTH_PASS"]

  var handler http.Handler = loggedRouter
  if user != "" && pass != "" {
    handler = BasicAuthMiddleware(user, pass, handler)
    log.Printf("Basic Auth enabled for API")
  } else {
    log.Printf("Basic Auth not configured; API runs without authentication")
  }

  // 检查是否配置了 HTTPS 证书和密钥
  if s.config.HTTPS.CertPath != "" && s.config.HTTPS.KeyPath != "" {
    // 检查证书和密钥文件是否存在
    if _, err := os.Stat(s.config.HTTPS.CertPath); os.IsNotExist(err) {
      log.Printf("Warning: HTTPS certificate file not found: %s", s.config.HTTPS.CertPath)
      log.Printf("Starting API server on %s (HTTP)", addr)
      return http.ListenAndServe(addr, handler)
    }
    if _, err := os.Stat(s.config.HTTPS.KeyPath); os.IsNotExist(err) {
      log.Printf("Warning: HTTPS private key file not found: %s", s.config.HTTPS.KeyPath)
      log.Printf("Starting API server on %s (HTTP)", addr)
      return http.ListenAndServe(addr, handler)
    }

    log.Printf("Starting API server on %s (HTTPS)", addr)
    return http.ListenAndServeTLS(addr, s.config.HTTPS.CertPath, s.config.HTTPS.KeyPath, handler)
  } else {
    log.Printf("Starting API server on %s (HTTP)", addr)
    return http.ListenAndServe(addr, handler)
  }
}

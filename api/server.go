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
  "crypto/tls"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "net"
  "net/http"
  "os"
  "path/filepath"
  "strings"
  "sync"
  "time"

  "github.com/casuallc/vigil/audit"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/models"
  "github.com/casuallc/vigil/proc"
  "github.com/casuallc/vigil/vm"
  _ "modernc.org/sqlite"
)

// Server represents the HTTP API server
type Server struct {
  config          *config.Config
  manager         *proc.Manager
  monitor         *proc.Monitor
  resourceMonitor *proc.ResourceMonitor
  vmManager       *vm.Manager
  userDatabase    *models.SQLiteUserDatabase
  auditLogger     *audit.Logger
  // SSH connection tracking
  sshConnections   map[string]*SSHConnectionInfo
  sshConnectionsMu sync.RWMutex
}

// SSHConnectionInfo represents an active SSH connection
type SSHConnectionInfo struct {
  ID          string    `json:"id"`
  VMName      string    `json:"vm_name"`
  ClientIP    string    `json:"client_ip"`
  Username    string    `json:"username"` // authenticated user when auth is enabled, anonymous when disabled
  ConnectedAt time.Time `json:"connected_at"`
  Duration    string    `json:"duration"` // formatted as human-readable string
}

// NewServerWithManager creates a new API server with an existing proc manager
func NewServerWithManager(config *config.Config, manager *proc.Manager) *Server {
  monitor := proc.NewMonitor(manager)
  // Initialize resource monitor with cache TTL of 5 seconds and collection interval of 3 seconds
  resourceMonitor := proc.NewResourceMonitor(manager, 5*time.Second, 3*time.Second, true, true)

  // Determine VM database path
  vmDBPath := "data/vms.db"
  vmManager := vm.NewManagerWithConfig(vmDBPath, config.Security.EncryptionKey)
  log.Printf("VM database initialized at %s", vmDBPath)

  // Create SQLite user database
  var userDatabase *models.SQLiteUserDatabase
  var err error

  // Determine database path from config or use default
  dbPath := config.Database.Path
  if dbPath == "" {
    dbPath = "data/users.db"
  }

  // Use SQLite if driver is sqlite or not specified (default to sqlite)
  if config.Database.Driver == "" || config.Database.Driver == "sqlite" {
    userDatabase, err = models.NewSQLiteUserDatabase(dbPath)
    if err != nil {
      log.Printf("Warning: failed to initialize SQLite user database: %v", err)
    } else {
      log.Printf("SQLite user database initialized at %s", dbPath)

      // Migrate from JSON file if it exists
      jsonPath := "conf/users.json"
      if _, err := os.Stat(jsonPath); err == nil {
        log.Printf("Found existing JSON user file, migrating to SQLite...")
        if err := models.MigrateJSONToSQLite(jsonPath, dbPath); err != nil {
          log.Printf("Warning: failed to migrate users from JSON: %v", err)
        } else {
          log.Printf("Users migrated successfully from JSON to SQLite")
          // Optionally remove the old JSON file after successful migration
          os.Remove(jsonPath)
        }
      }
    }
  }

  // Create audit log directory
  auditLogDir := filepath.Join("logs", "audit")

  // Initialize audit logger
  auditLogger, err := audit.NewLogger(auditLogDir)
  if err != nil {
    log.Printf("Warning: failed to initialize audit logger: %v", err)
    // Continue running even if audit logger initialization fails
  }

  server := &Server{
    config:          config,
    manager:         manager,
    monitor:         monitor,
    resourceMonitor: resourceMonitor,
    vmManager:       vmManager,
    userDatabase:    userDatabase,
    auditLogger:     auditLogger,
    // Initialize SSH connection tracking
    sshConnections: make(map[string]*SSHConnectionInfo),
  }

  // Start the resource monitor
  resourceMonitor.Start()

  return server
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

// LoggingMiddleware 是一个HTTP中间件，用于记录请求和响应
func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
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

    // 处理请求
    next.ServeHTTP(lrw, r)

    // 计算请求处理时间
    duration := time.Since(startTime)

    // 获取客户端IP
    clientIP := r.RemoteAddr
    if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
      clientIP = forwardedFor
    }

    // 获取用户信息
    user := "anonymous"
    if auth := r.Header.Get("Authorization"); auth != "" {
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
    case strings.HasPrefix(path, "/api/vms/ssh"):
      action = audit.ActionVMSSH
      resource = r.URL.Query().Get("vm")
    case strings.HasPrefix(path, "/api/vms/files"):
      switch {
      case strings.Contains(path, "/upload"):
        action = audit.ActionFileUpload
      case strings.Contains(path, "/download"):
        action = audit.ActionFileDownload
      case strings.Contains(path, "/list"):
        action = audit.ActionFileList
      case strings.Contains(path, "/delete"):
        action = audit.ActionFileDelete
      case strings.Contains(path, "/mkdir"):
        action = audit.ActionFileMkdir
      case strings.Contains(path, "/touch"):
        action = audit.ActionFileTouch
      case strings.Contains(path, "/rmdir"):
        action = audit.ActionFileRmdir
      }
      resource = r.URL.Query().Get("vm_name")
    case strings.HasPrefix(path, "/api/vms/permissions"):
      switch r.Method {
      case http.MethodPost:
        action = audit.ActionPermissionAdd
      case http.MethodGet:
        if strings.Count(path, "/") == 4 {
          action = audit.ActionPermissionList
          resource = strings.TrimPrefix(strings.TrimPrefix(path, "/api/vms/"), "/permissions")
        }
      case http.MethodDelete:
        action = audit.ActionPermissionRemove
        resource = strings.TrimPrefix(path, "/api/vms/permissions/")
      }
    case strings.HasPrefix(path, "/api/processes"):
      action = audit.ActionProcessManage
    case strings.HasPrefix(path, "/api/resources"):
      action = audit.ActionResourceMonitor
    case strings.HasPrefix(path, "/api/config"):
      action = audit.ActionConfigManage
    case strings.HasPrefix(path, "/api/exec"):
      action = audit.ActionCommandExecute
    }

    // 确定操作状态
    status := audit.StatusSuccess
    if lrw.statusCode >= 400 {
      status = audit.StatusFailed
    }

    // 创建审计日志条目
    message := fmt.Sprintf("%s %s %d %v", r.Method, r.URL.Path, lrw.statusCode, duration)
    details := map[string]interface{}{
      "method":      r.Method,
      "path":        r.URL.Path,
      "query":       r.URL.RawQuery,
      "status_code": lrw.statusCode,
      "elapsed_ms":  duration.Milliseconds(),
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

// BasicAuthMiddleware enforces HTTP Basic Auth with dual authentication.
// First checks super admin credentials from config, then checks user database.
func (s *Server) BasicAuthMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    u, p, ok := r.BasicAuth()
    if !ok {
      w.Header().Set("WWW-Authenticate", `Basic realm="vigil"`)
      http.Error(w, "Unauthorized", http.StatusUnauthorized)
      return
    }

    // First, check if it's the super admin user from config
    if s.config.BasicAuth.Enabled && u == s.config.BasicAuth.Username && p == s.config.BasicAuth.Password {
      // Super admin has full access - pass through
      next.ServeHTTP(w, r)
      return
    }

    // Second, check against registered users in the user database
    if s.userDatabase != nil {
      isValid, err := s.userDatabase.ValidatePassword(u, p)
      if err != nil {
        log.Printf("Error validating user password: %v", err)
        w.Header().Set("WWW-Authenticate", `Basic realm="vigil"`)
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
      }

      if isValid {
        // Valid registered user - pass through
        next.ServeHTTP(w, r)
        return
      }
    }

    // Neither super admin nor registered user matched
    w.Header().Set("WWW-Authenticate", `Basic realm="vigil"`)
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

// getConnectionUser extracts the authenticated user from the request context
func (s *Server) getConnectionUser(r *http.Request) string {
  // Check if authentication is enabled and get user from headers
  if s.config.BasicAuth.Enabled {
    if auth := r.Header.Get("Authorization"); auth != "" {
      // Extract user from Basic Auth header if possible
      return "authenticated" // Simplified - in real implementation, decode Basic Auth to extract user
    }
  }
  return "anonymous" // Use anonymous user when auth is disabled or no valid auth provided
}

// RegisterSSHConnection registers a new SSH connection
func (s *Server) RegisterSSHConnection(id string, vmName string, clientIP string, username string) {
  s.sshConnectionsMu.Lock()
  defer s.sshConnectionsMu.Unlock()

  s.sshConnections[id] = &SSHConnectionInfo{
    ID:          id,
    VMName:      vmName,
    ClientIP:    clientIP,
    Username:    username,
    ConnectedAt: time.Now(),
    Duration:    "0s",
  }
}

// UnregisterSSHConnection removes an SSH connection
func (s *Server) UnregisterSSHConnection(id string) {
  s.sshConnectionsMu.Lock()
  defer s.sshConnectionsMu.Unlock()

  delete(s.sshConnections, id)
}

// GetSSHConnections returns all active SSH connections
func (s *Server) GetSSHConnections() []*SSHConnectionInfo {
  s.sshConnectionsMu.RLock()
  defer s.sshConnectionsMu.RUnlock()

  connections := make([]*SSHConnectionInfo, 0, len(s.sshConnections))
  for _, conn := range s.sshConnections {
    // Calculate duration since connection started
    duration := time.Since(conn.ConnectedAt)
    conn.Duration = duration.Truncate(time.Second).String()
    connections = append(connections, conn)
  }
  return connections
}

// CloseSSHConnection closes a specific SSH connection
func (s *Server) CloseSSHConnection(id string) bool {
  s.sshConnectionsMu.Lock()
  defer s.sshConnectionsMu.Unlock()

  if _, exists := s.sshConnections[id]; exists {
    delete(s.sshConnections, id)
    return true
  }
  return false
}

// CloseAllSSHConnections closes all SSH connections
func (s *Server) CloseAllSSHConnections() int {
  s.sshConnectionsMu.Lock()
  defer s.sshConnectionsMu.Unlock()

  count := len(s.sshConnections)
  s.sshConnections = make(map[string]*SSHConnectionInfo)
  return count
}

// Start starts the HTTP server
func (s *Server) Start() error {
  addr := s.config.Addr
  // 获取路由
  r := s.Router()

  // 应用日志中间件到所有路由
  loggedRouter := s.LoggingMiddleware(r)

  var handler http.Handler = loggedRouter
  if s.config.BasicAuth.Enabled {
    handler = s.BasicAuthMiddleware(handler)
    log.Printf("Basic Auth enabled for API")
  } else {
    log.Printf("Basic Auth not configured; API runs without authentication")
  }

  // 检查是否配置了 HTTPS 证书和密钥
  if s.config.HTTPS.Enabled {
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

    // 尝试加载证书和密钥，验证它们是否匹配
    _, err := tls.LoadX509KeyPair(s.config.HTTPS.CertPath, s.config.HTTPS.KeyPath)
    if err != nil {
      log.Printf("Error: Failed to load HTTPS certificate and key: %v", err)
      log.Printf("This usually happens when the certificate and private key do not match.")
      log.Printf("Please generate a new certificate and key pair, or check your configuration.")
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

// Stop stops the HTTP server and cleans up resources
func (s *Server) Stop() {
  // Stop the resource monitor
  if s.resourceMonitor != nil {
    s.resourceMonitor.Stop()
  }

  // Close VM database connection
  if s.vmManager != nil {
    s.vmManager.Close()
  }
}

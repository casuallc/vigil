package models

import (
  "database/sql"
  "os"
  "path/filepath"
  "sync"
  "time"

  dbsql "github.com/casuallc/vigil/sql"
)

// LoginLog represents a login log entry
type LoginLog struct {
  ID         int       `json:"id"`
  Username   string    `json:"username"`
  UserID     string    `json:"user_id"`
  ClientIP   string    `json:"client_ip"`
  UserAgent  string    `json:"user_agent"`
  DeviceInfo string    `json:"device_info"`
  LoginTime  time.Time `json:"login_time"`
  Status     string    `json:"status"`
}

// LoginLogDatabase manages login log data with SQLite backend
type LoginLogDatabase struct {
  db   *sql.DB
  mu   sync.RWMutex
  path string
}

// NewLoginLogDatabase creates a new LoginLogDatabase instance
func NewLoginLogDatabase(path string) (*LoginLogDatabase, error) {
  // Ensure directory exists
  dir := filepath.Dir(path)
  if err := os.MkdirAll(dir, 0755); err != nil {
    return nil, err
  }

  // Open SQLite database
  db, err := sql.Open("sqlite", path)
  if err != nil {
    return nil, err
  }

  logDB := &LoginLogDatabase{
    db:   db,
    path: path,
  }

  // Initialize the database schema
  if err := logDB.initDB(); err != nil {
    db.Close()
    return nil, err
  }

  return logDB, nil
}

// initDB initializes the database schema
func (lld *LoginLogDatabase) initDB() error {
  schema, err := dbsql.LoadLoginLogsSchema()
  if err != nil {
    return err
  }

  _, err = lld.db.Exec(schema)
  return err
}

// Close closes the database connection
func (lld *LoginLogDatabase) Close() error {
  return lld.db.Close()
}

// LogLogin records a login attempt
func (lld *LoginLogDatabase) LogLogin(username, userID, clientIP, userAgent, deviceInfo, status string) error {
  lld.mu.Lock()
  defer lld.mu.Unlock()

  query := `INSERT INTO login_logs (username, user_id, client_ip, user_agent, device_info, status)
            VALUES (?, ?, ?, ?, ?, ?)`

  _, err := lld.db.Exec(query, username, userID, clientIP, userAgent, deviceInfo, status)
  return err
}

// GetUserLoginLogs retrieves login history for a specific user
func (lld *LoginLogDatabase) GetUserLoginLogs(username string, limit int) ([]*LoginLog, error) {
  lld.mu.RLock()
  defer lld.mu.RUnlock()

  query := `SELECT id, username, user_id, client_ip, user_agent, device_info, login_time, status
            FROM login_logs
            WHERE username = ?
            ORDER BY login_time DESC
            LIMIT ?`

  rows, err := lld.db.Query(query, username, limit)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var logs []*LoginLog
  for rows.Next() {
    var log LoginLog
    err := rows.Scan(
      &log.ID,
      &log.Username,
      &log.UserID,
      &log.ClientIP,
      &log.UserAgent,
      &log.DeviceInfo,
      &log.LoginTime,
      &log.Status,
    )
    if err == nil {
      logs = append(logs, &log)
    }
  }

  return logs, rows.Err()
}

// GetAllLoginLogs retrieves all login logs with optional limit
func (lld *LoginLogDatabase) GetAllLoginLogs(limit int) ([]*LoginLog, error) {
  lld.mu.RLock()
  defer lld.mu.RUnlock()

  query := `SELECT id, username, user_id, client_ip, user_agent, device_info, login_time, status
            FROM login_logs
            ORDER BY login_time DESC
            LIMIT ?`

  rows, err := lld.db.Query(query, limit)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var logs []*LoginLog
  for rows.Next() {
    var log LoginLog
    err := rows.Scan(
      &log.ID,
      &log.Username,
      &log.UserID,
      &log.ClientIP,
      &log.UserAgent,
      &log.DeviceInfo,
      &log.LoginTime,
      &log.Status,
    )
    if err == nil {
      logs = append(logs, &log)
    }
  }

  return logs, rows.Err()
}

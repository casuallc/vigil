package models

import (
  "database/sql"
  "encoding/json"
  "os"
  "path/filepath"
  "strings"
  "sync"
  "time"

  dbsql "github.com/casuallc/vigil/sql"
  "golang.org/x/crypto/bcrypt"
)

// SQLiteUserDatabase manages user data with SQLite backend
type SQLiteUserDatabase struct {
  db   *sql.DB
  mu   sync.RWMutex
  path string
}

// NewSQLiteUserDatabase creates a new SQLite user database instance
func NewSQLiteUserDatabase(path string) (*SQLiteUserDatabase, error) {
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

  sqliteDB := &SQLiteUserDatabase{
    db:   db,
    path: path,
  }

  // Initialize the database schema
  if err := sqliteDB.initDB(); err != nil {
    db.Close()
    return nil, err
  }

  return sqliteDB, nil
}

// initDB initializes the database schema
func (ud *SQLiteUserDatabase) initDB() error {
  schema, err := dbsql.LoadUsersSchema()
  if err != nil {
    return err
  }

  _, err = ud.db.Exec(schema)
  return err
}

// Close closes the database connection
func (ud *SQLiteUserDatabase) Close() error {
  return ud.db.Close()
}

// GetUser retrieves a user by username
func (ud *SQLiteUserDatabase) GetUser(username string) (*User, bool) {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  query := `SELECT id, username, password, email, role, created_at, updated_at, last_login_at, last_login_ip, login_count, avatar, nickname, region, configs
			  FROM users WHERE username = ?`

  var user User
  var lastLoginAt sql.NullTime
  err := ud.db.QueryRow(query, username).Scan(
    &user.ID,
    &user.Username,
    &user.Password,
    &user.Email,
    &user.Role,
    &user.CreatedAt,
    &user.UpdatedAt,
    &lastLoginAt,
    &user.LastLoginIP,
    &user.LoginCount,
    &user.Avatar,
    &user.Nickname,
    &user.Region,
    &user.Configs,
  )

  if err == sql.ErrNoRows {
    return nil, false
  }
  if err != nil {
    return nil, false
  }

  if lastLoginAt.Valid {
    user.LastLoginAt = &lastLoginAt.Time
  }

  return &user, true
}

// CreateUser creates a new user with a hashed password
func (ud *SQLiteUserDatabase) CreateUser(user *User) error {
  // Hash the password
  hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
  if err != nil {
    return err
  }

  user.Password = string(hashedPassword)
  user.CreatedAt = time.Now()
  user.UpdatedAt = time.Now()

  ud.mu.Lock()
  defer ud.mu.Unlock()

  query := `INSERT INTO users (id, username, password, email, role, created_at, updated_at, last_login_at, last_login_ip, login_count, avatar, nickname, region, configs)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

  _, err = ud.db.Exec(query,
    user.ID,
    user.Username,
    user.Password,
    user.Email,
    user.Role,
    user.CreatedAt,
    user.UpdatedAt,
    user.LastLoginAt,
    user.LastLoginIP,
    user.LoginCount,
    user.Avatar,
    user.Nickname,
    user.Region,
    user.Configs,
  )

  // Handle unique constraint violation
  if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
    return sql.ErrNoRows // Return a more meaningful error
  }

  return err
}

// UpdateUser updates an existing user
func (ud *SQLiteUserDatabase) UpdateUser(username string, updatedUser *User) error {
  ud.mu.Lock()
  defer ud.mu.Unlock()

  // Check if user exists
  var existingUsername string
  err := ud.db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&existingUsername)
  if err == sql.ErrNoRows {
    return os.ErrNotExist
  }
  if err != nil {
    return err
  }

  // Build dynamic update query
  var updates []string
  var args []interface{}

  if updatedUser.Password != "" {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedUser.Password), bcrypt.DefaultCost)
    if err != nil {
      return err
    }
    updates = append(updates, "password = ?")
    args = append(args, string(hashedPassword))
  }

  if updatedUser.Email != "" {
    updates = append(updates, "email = ?")
    args = append(args, updatedUser.Email)
  }

  if updatedUser.Role != "" {
    updates = append(updates, "role = ?")
    args = append(args, updatedUser.Role)
  }

  if updatedUser.Username != "" && updatedUser.Username != username {
    updates = append(updates, "username = ?")
    args = append(args, updatedUser.Username)
  }

  // Profile fields
  if updatedUser.Avatar != "" {
    updates = append(updates, "avatar = ?")
    args = append(args, updatedUser.Avatar)
  }
  if updatedUser.Nickname != "" {
    updates = append(updates, "nickname = ?")
    args = append(args, updatedUser.Nickname)
  }
  if updatedUser.Region != "" {
    updates = append(updates, "region = ?")
    args = append(args, updatedUser.Region)
  }
  if updatedUser.Configs != "" {
    updates = append(updates, "configs = ?")
    args = append(args, updatedUser.Configs)
  }

  // Always update updated_at
  updates = append(updates, "updated_at = ?")
  args = append(args, time.Now())

  // Add the WHERE clause parameter
  args = append(args, username)

  query := "UPDATE users SET " + joinStrings(updates, ", ") + " WHERE username = ?"
  _, err = ud.db.Exec(query, args...)

  return err
}

// DeleteUser removes a user by username
func (ud *SQLiteUserDatabase) DeleteUser(username string) error {
  ud.mu.Lock()
  defer ud.mu.Unlock()

  // Check if user exists
  var existingUsername string
  err := ud.db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&existingUsername)
  if err == sql.ErrNoRows {
    return os.ErrNotExist
  }
  if err != nil {
    return err
  }

  _, err = ud.db.Exec("DELETE FROM users WHERE username = ?", username)
  return err
}

// GetAllUsers returns all users
func (ud *SQLiteUserDatabase) GetAllUsers() []*User {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  query := `SELECT id, username, password, email, role, created_at, updated_at, last_login_at, last_login_ip, login_count, avatar, nickname, region, configs FROM users`

  rows, err := ud.db.Query(query)
  if err != nil {
    return nil
  }
  defer rows.Close()

  var users []*User
  for rows.Next() {
    var user User
    var lastLoginAt sql.NullTime
    err := rows.Scan(
      &user.ID,
      &user.Username,
      &user.Password,
      &user.Email,
      &user.Role,
      &user.CreatedAt,
      &user.UpdatedAt,
      &lastLoginAt,
      &user.LastLoginIP,
      &user.LoginCount,
      &user.Avatar,
      &user.Nickname,
      &user.Region,
      &user.Configs,
    )
    if err == nil {
      if lastLoginAt.Valid {
        user.LastLoginAt = &lastLoginAt.Time
      }
      users = append(users, &user)
    }
  }

  return users
}

// ValidatePassword validates a password against the stored hash
func (ud *SQLiteUserDatabase) ValidatePassword(username, password string) (bool, error) {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  query := `SELECT password FROM users WHERE username = ?`

  var storedHash string
  err := ud.db.QueryRow(query, username).Scan(&storedHash)
  if err == sql.ErrNoRows {
    return false, nil
  }
  if err != nil {
    return false, err
  }

  err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
  return err == nil, nil
}

// GetUserByID retrieves a user by ID
func (ud *SQLiteUserDatabase) GetUserByID(id string) (*User, bool) {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  query := `SELECT id, username, password, email, role, created_at, updated_at, last_login_at, last_login_ip, login_count, avatar, nickname, region, configs
			  FROM users WHERE id = ?`

  var user User
  var lastLoginAt sql.NullTime
  err := ud.db.QueryRow(query, id).Scan(
    &user.ID,
    &user.Username,
    &user.Password,
    &user.Email,
    &user.Role,
    &user.CreatedAt,
    &user.UpdatedAt,
    &lastLoginAt,
    &user.LastLoginIP,
    &user.LoginCount,
    &user.Avatar,
    &user.Nickname,
    &user.Region,
    &user.Configs,
  )

  if err == sql.ErrNoRows {
    return nil, false
  }
  if err != nil {
    return nil, false
  }

  if lastLoginAt.Valid {
    user.LastLoginAt = &lastLoginAt.Time
  }

  return &user, true
}

// GetUserCount returns the total number of users
func (ud *SQLiteUserDatabase) GetUserCount() (int, error) {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  var count int
  err := ud.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
  return count, err
}

// UpdateLoginStatus updates user login status after successful login
func (ud *SQLiteUserDatabase) UpdateLoginStatus(username, clientIP string) error {
  ud.mu.Lock()
  defer ud.mu.Unlock()

  // First get the current login count
  var currentCount int
  err := ud.db.QueryRow("SELECT login_count FROM users WHERE username = ?", username).Scan(&currentCount)
  if err == sql.ErrNoRows {
    return os.ErrNotExist
  }
  if err != nil {
    return err
  }

  // Update login status
  query := `UPDATE users SET last_login_at = ?, last_login_ip = ?, login_count = ? WHERE username = ?`
  _, err = ud.db.Exec(query, time.Now(), clientIP, currentCount+1, username)
  return err
}

// Helper function to join strings (since strings.Join is not available without import)
func joinStrings(strs []string, sep string) string {
  if len(strs) == 0 {
    return ""
  }
  result := strs[0]
  for i := 1; i < len(strs); i++ {
    result += sep + strs[i]
  }
  return result
}

// ExportToFile exports all users to a JSON file (for backup purposes)
func (ud *SQLiteUserDatabase) ExportToFile(path string) error {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  users := ud.GetAllUsers()

  data, err := json.MarshalIndent(users, "", "  ")
  if err != nil {
    return err
  }

  return os.WriteFile(path, data, 0600)
}

// ImportFromFile imports users from a JSON file (for migration purposes)
func (ud *SQLiteUserDatabase) ImportFromFile(path string) error {
  data, err := os.ReadFile(path)
  if err != nil {
    return err
  }

  var users []User
  if err := json.Unmarshal(data, &users); err != nil {
    return err
  }

  ud.mu.Lock()
  defer ud.mu.Unlock()

  tx, err := ud.db.Begin()
  if err != nil {
    return err
  }
  defer tx.Rollback()

  stmt, err := tx.Prepare(`INSERT OR REPLACE INTO users (id, username, password, email, role, created_at, updated_at, last_login_at, last_login_ip, login_count)
							 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
  if err != nil {
    return err
  }
  defer stmt.Close()

  for _, user := range users {
    _, err := stmt.Exec(user.ID, user.Username, user.Password, user.Email, user.Role, user.CreatedAt, user.UpdatedAt, user.LastLoginAt, user.LastLoginIP, user.LoginCount)
    if err != nil {
      return err
    }
  }

  return tx.Commit()
}

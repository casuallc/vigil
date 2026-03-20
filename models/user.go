package models

import (
  "encoding/json"
  "os"
  "path/filepath"
  "sync"
  "time"

  "golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
  ID        string    `json:"id"`
  Username  string    `json:"username"`
  Password  string    `json:"password"` // This will be the hashed password
  Email     string    `json:"email,omitempty"`
  Role      string    `json:"role"` // "admin" or "user"
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
}

// UserDatabase manages user data with concurrency safety
type UserDatabase struct {
  mu    sync.RWMutex
  users map[string]*User
  path  string
}

// NewUserDatabase creates a new user database instance
func NewUserDatabase(path string) (*UserDatabase, error) {
  db := &UserDatabase{
    users: make(map[string]*User),
    path:  path,
  }

  // Load existing users from file if it exists
  if _, err := os.Stat(path); err == nil {
    if err := db.loadFromFile(); err != nil {
      return nil, err
    }
  } else if !os.IsNotExist(err) {
    return nil, err
  }

  return db, nil
}

// loadFromFile loads users from the JSON file
func (ud *UserDatabase) loadFromFile() error {
  // First read the file without holding the lock
  data, err := os.ReadFile(ud.path)
  if err != nil {
    return err
  }

  var users []User
  if err := json.Unmarshal(data, &users); err != nil {
    return err
  }

  // Now acquire lock only to update the in-memory data structure
  ud.mu.Lock()
  defer ud.mu.Unlock()

  // Convert slice to map for easy lookup
  ud.users = make(map[string]*User)
  for i := range users {
    user := &users[i]
    ud.users[user.Username] = user
  }

  return nil
}

// saveToFile saves users to the JSON file
func (ud *UserDatabase) saveToFile() error {
  // First read and copy users without holding the lock, then save to file
  users := make([]User, 0, len(ud.users))
  ud.mu.RLock()
  for _, user := range ud.users {
    users = append(users, *user)
  }
  ud.mu.RUnlock()

  data, err := json.MarshalIndent(users, "", "  ")
  if err != nil {
    return err
  }

  // Ensure directory exists before writing the file
  dir := filepath.Dir(ud.path)
  if err := os.MkdirAll(dir, 0755); err != nil {
    return err
  }

  return os.WriteFile(ud.path, data, 0600)
}

// GetUser retrieves a user by username
func (ud *UserDatabase) GetUser(username string) (*User, bool) {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  user, exists := ud.users[username]
  if !exists {
    return nil, false
  }

  // Return a copy to prevent external modification
  userCopy := *user
  return &userCopy, true
}

// CreateUser creates a new user with a hashed password
func (ud *UserDatabase) CreateUser(user *User) error {
  // Hash the password before acquiring the lock
  hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
  if err != nil {
    return err
  }

  user.Password = string(hashedPassword)
  user.CreatedAt = time.Now()
  user.UpdatedAt = time.Now()

  // Acquire lock only for updating the in-memory storage
  ud.mu.Lock()
  ud.users[user.Username] = user
  ud.mu.Unlock()

  // Save to file after releasing the lock to avoid deadlock
  return ud.saveToFile()
}

// UpdateUser updates an existing user
func (ud *UserDatabase) UpdateUser(username string, updatedUser *User) error {
  // Check if password is being updated, hash it outside the lock
  var hashedPassword []byte
  var err error
  if updatedUser.Password != "" {
    hashedPassword, err = bcrypt.GenerateFromPassword([]byte(updatedUser.Password), bcrypt.DefaultCost)
    if err != nil {
      return err
    }
  }

  // Acquire lock only for updating the in-memory storage
  ud.mu.Lock()
  existingUser, exists := ud.users[username]
  if !exists {
    ud.mu.Unlock()
    return os.ErrNotExist
  }

  // Apply changes after checking existence
  if updatedUser.Password != "" {
    existingUser.Password = string(hashedPassword)
  }

  // Update other fields
  if updatedUser.Email != "" {
    existingUser.Email = updatedUser.Email
  }
  if updatedUser.Role != "" {
    existingUser.Role = updatedUser.Role
  }
  if updatedUser.Username != "" && updatedUser.Username != username {
    // Username is changing, update the map key
    delete(ud.users, username)
    existingUser.Username = updatedUser.Username
    ud.users[updatedUser.Username] = existingUser
  }

  existingUser.UpdatedAt = time.Now()
  ud.mu.Unlock()

  // Save to file after releasing the lock to avoid deadlock
  return ud.saveToFile()
}

// DeleteUser removes a user by username
func (ud *UserDatabase) DeleteUser(username string) error {
  ud.mu.Lock()
  defer ud.mu.Unlock()

  if _, exists := ud.users[username]; !exists {
    return os.ErrNotExist
  }

  delete(ud.users, username)

  return ud.saveToFile()
}

// GetAllUsers returns all users
func (ud *UserDatabase) GetAllUsers() []*User {
  ud.mu.RLock()
  defer ud.mu.RUnlock()

  users := make([]*User, 0, len(ud.users))
  for _, user := range ud.users {
    // Return a copy to prevent external modification
    userCopy := *user
    users = append(users, &userCopy)
  }

  return users
}

// ValidatePassword validates a password against the stored hash
func (ud *UserDatabase) ValidatePassword(username, password string) (bool, error) {
  ud.mu.RLock()
  user, exists := ud.users[username]
  ud.mu.RUnlock()

  if !exists {
    return false, nil
  }

  err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
  return err == nil, nil
}

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

package vm

import (
  "database/sql"
  "encoding/json"
  "fmt"
  "log"
  "os"
  "path/filepath"
  "sync"
  "time"

  "github.com/casuallc/vigil/crypto"
  dbsql "github.com/casuallc/vigil/sql"
)

// Manager 管理 VM 实例和相关操作
type Manager struct {
  vms           map[string]*VM
  groups        map[string]*Group
  permissions   *PermissionManager
  mu            sync.RWMutex
  db            *sql.DB // SQLite 数据库连接
  dbPath        string  // 数据库文件路径
  encryptionKey string  // 用于加密敏感数据的密钥
}

// NewManagerWithConfig 创建一个新的 VM 管理器，使用配置中的加密密钥
func NewManagerWithConfig(dbPath string, encryptionKey string) *Manager {
  manager := &Manager{
    vms:           make(map[string]*VM),
    groups:        make(map[string]*Group),
    permissions:   NewPermissionManager(),
    dbPath:        dbPath,
    encryptionKey: encryptionKey,
  }

  // 确保目录存在
  dir := filepath.Dir(dbPath)
  if err := os.MkdirAll(dir, 0755); err != nil {
    log.Printf("Failed to create VM database directory: %v", err)
    return manager
  }

  // 打开 SQLite 数据库
  db, err := sql.Open("sqlite", dbPath)
  if err != nil {
    log.Printf("Failed to open VM database: %v", err)
    return manager
  }

  manager.db = db

  // 初始化数据库 schema
  if err := manager.initDB(); err != nil {
    log.Printf("Failed to initialize VM database: %v", err)
    return manager
  }

  // 加载 VM 信息
  if err := manager.LoadVMs(); err != nil {
    log.Printf("Failed to load VMs: %v", err)
    // 继续运行，使用空的 VM 列表
  }

  return manager
}

// initDB 初始化数据库 schema
func (m *Manager) initDB() error {
  schema, err := dbsql.LoadVMsSchema()
  if err != nil {
    return err
  }

  _, err = m.db.Exec(schema)
  return err
}

// Close 关闭数据库连接
func (m *Manager) Close() {
  if m.db != nil {
    m.db.Close()
  }
}

// generateID 生成唯一 ID
func generateID() string {
  return time.Now().Format("20060102150405.000000000")
}

// GetVM 根据名称获取 VM 实例
func (m *Manager) GetVM(name string) (*VM, error) {
  m.mu.RLock()
  defer m.mu.RUnlock()

  vm, exists := m.vms[name]
  if !exists {
    return nil, fmt.Errorf("VM not found: %s", name)
  }
  return vm, nil
}

// ListVMs 列出所有 VM 实例
func (m *Manager) ListVMs() []*VM {
  m.mu.RLock()
  defer m.mu.RUnlock()

  var result []*VM
  for _, vm := range m.vms {
    result = append(result, vm)
  }
  return result
}

// AddVM 添加一个新的 VM 实例
func (m *Manager) AddVM(vm *VM) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  if _, exists := m.vms[vm.Name]; exists {
    return fmt.Errorf("VM already exists: %s", vm.Name)
  }

  // 加密敏感数据
  encryptedPassword := vm.Password
  encryptedKeyPath := vm.KeyPath

  if vm.Password != "" && m.encryptionKey != "" {
    var err error
    encryptedPassword, err = crypto.Encrypt(vm.Password, m.encryptionKey)
    if err != nil {
      return fmt.Errorf("failed to encrypt password: %v", err)
    }
  }

  if vm.KeyPath != "" && m.encryptionKey != "" {
    var err error
    encryptedKeyPath, err = crypto.Encrypt(vm.KeyPath, m.encryptionKey)
    if err != nil {
      return fmt.Errorf("failed to encrypt key path: %v", err)
    }
  }

  // 插入数据库
  query := `INSERT INTO vms (id, name, ip, port, username, password, key_path, status, description, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
  _, err := m.db.Exec(query,
    generateID(),
    vm.Name,
    vm.IP,
    vm.Port,
    vm.Username,
    encryptedPassword,
    encryptedKeyPath,
    vm.Status,
    vm.Description,
    vm.CreatedAt,
    vm.UpdatedAt,
  )

  if err != nil {
    return fmt.Errorf("failed to save VM to database: %v", err)
  }

  // 添加到内存
  m.vms[vm.Name] = vm
  log.Printf("Added new VM: %s", vm.Name)

  // 确保 VM 至少属于一个组，默认是 default 组
  defaultGroup, exists := m.groups["default"]
  if !exists {
    // 创建 default 组
    defaultGroup = NewGroup("default", "Default VM group", []string{vm.Name})
    m.groups["default"] = defaultGroup
    log.Printf("Created default group and added VM %s to it", vm.Name)

    // 保存到数据库
    if err := m.saveGroupToDB(defaultGroup); err != nil {
      log.Printf("Failed to save default group: %v", err)
    }
  } else {
    // 检查 VM 是否已经在 default 组中
    vmExistsInGroup := false
    for _, existingVM := range defaultGroup.VMs {
      if existingVM == vm.Name {
        vmExistsInGroup = true
        break
      }
    }
    if !vmExistsInGroup {
      // 将 VM 添加到 default 组
      defaultGroup.VMs = append(defaultGroup.VMs, vm.Name)
      defaultGroup.UpdatedAt = time.Now()
      log.Printf("Added VM %s to default group", vm.Name)

      // 保存到数据库
      if err := m.saveGroupToDB(defaultGroup); err != nil {
        log.Printf("Failed to update default group: %v", err)
      }
    }
  }

  return nil
}

// RemoveVM 移除一个 VM 实例
func (m *Manager) RemoveVM(name string) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  if _, exists := m.vms[name]; !exists {
    return fmt.Errorf("VM not found: %s", name)
  }

  // 从数据库删除
  _, err := m.db.Exec("DELETE FROM vms WHERE name = ?", name)
  if err != nil {
    return fmt.Errorf("failed to delete VM from database: %v", err)
  }

  // 从内存删除
  delete(m.vms, name)
  log.Printf("Removed VM: %s", name)

  // 从所有组中移除 VM
  for _, group := range m.groups {
    newVMs := []string{}
    for _, vmName := range group.VMs {
      if vmName != name {
        newVMs = append(newVMs, vmName)
      }
    }
    if len(newVMs) != len(group.VMs) {
      group.VMs = newVMs
      group.UpdatedAt = time.Now()
      log.Printf("Removed VM %s from group %s", name, group.Name)

      // 保存到数据库
      if err := m.saveGroupToDB(group); err != nil {
        log.Printf("Failed to update group %s: %v", group.Name, err)
      }
    }
  }

  return nil
}

// GetPermissionManager 获取权限管理器
func (m *Manager) GetPermissionManager() *PermissionManager {
  return m.permissions
}

// AddGroup 添加一个新的 VM 组
func (m *Manager) AddGroup(name, description, owner string, vms []string, isShared bool, sharedWith []string) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  // 检查组是否已存在
  if _, exists := m.groups[name]; exists {
    return fmt.Errorf("group already exists: %s", name)
  }

  // 检查 VM 是否存在
  for _, vmName := range vms {
    if _, exists := m.vms[vmName]; !exists {
      return fmt.Errorf("VM not found: %s", vmName)
    }
  }

  // 创建新组
  group := NewGroup(name, description, vms)
  group.Owner = owner
  group.IsShared = isShared
  group.SharedWith = sharedWith
  m.groups[name] = group
  log.Printf("Added group: %s with %d VMs, owner: %s, shared: %v", name, len(vms), owner, isShared)

  // 保存到数据库
  if err := m.saveGroupToDB(group); err != nil {
    return fmt.Errorf("failed to save group to database: %v", err)
  }

  return nil
}

// RemoveGroup 删除一个 VM 组
func (m *Manager) RemoveGroup(name string) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  // 检查组是否存在
  if _, exists := m.groups[name]; !exists {
    return fmt.Errorf("group not found: %s", name)
  }

  // 从数据库删除
  _, err := m.db.Exec("DELETE FROM groups WHERE name = ?", name)
  if err != nil {
    return fmt.Errorf("failed to delete group from database: %v", err)
  }

  // 从内存删除
  delete(m.groups, name)
  log.Printf("Removed group: %s", name)

  return nil
}

// GetGroup 获取一个 VM 组
func (m *Manager) GetGroup(name string) (*Group, error) {
  m.mu.RLock()
  defer m.mu.RUnlock()

  group, exists := m.groups[name]
  if !exists {
    return nil, fmt.Errorf("group not found: %s", name)
  }

  return group, nil
}

// ListGroups 列出所有 VM 组
func (m *Manager) ListGroups() []*Group {
  m.mu.RLock()
  defer m.mu.RUnlock()

  var groups []*Group
  for _, group := range m.groups {
    groups = append(groups, group)
  }

  return groups
}

// UpdateGroup 更新一个 VM 组
func (m *Manager) UpdateGroup(name, description string, vms []string, isShared bool, sharedWith []string) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  // 检查组是否存在
  group, exists := m.groups[name]
  if !exists {
    return fmt.Errorf("group not found: %s", name)
  }

  // 检查 VM 是否存在
  for _, vmName := range vms {
    if _, exists := m.vms[vmName]; !exists {
      return fmt.Errorf("VM not found: %s", vmName)
    }
  }

  // 更新组信息
  group.Description = description
  group.VMs = vms
  group.IsShared = isShared
  group.SharedWith = sharedWith
  group.UpdatedAt = time.Now()
  log.Printf("Updated group: %s with %d VMs, shared: %v", name, len(vms), isShared)

  // 保存到数据库
  if err := m.saveGroupToDB(group); err != nil {
    return fmt.Errorf("failed to update group in database: %v", err)
  }

  return nil
}

// saveGroupToDB 保存组到数据库（内部方法，不持有锁）
func (m *Manager) saveGroupToDB(group *Group) error {
  vmsJSON, err := json.Marshal(group.VMs)
  if err != nil {
    return err
  }

  sharedWithJSON, err := json.Marshal(group.SharedWith)
  if err != nil {
    return err
  }

  query := `INSERT OR REPLACE INTO groups (id, name, description, vms, owner, is_shared, shared_with, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
  _, err = m.db.Exec(query,
    generateID(),
    group.Name,
    group.Description,
    string(vmsJSON),
    group.Owner,
    boolToInt(group.IsShared),
    string(sharedWithJSON),
    group.CreatedAt,
    group.UpdatedAt,
  )

  return err
}

// boolToInt 将布尔值转换为整数
func boolToInt(b bool) int {
  if b {
    return 1
  }
  return 0
}

// LoadVMs 从数据库加载 VM 和组信息
func (m *Manager) LoadVMs() error {
  m.mu.Lock()
  defer m.mu.Unlock()

  // 清空现有数据
  m.vms = make(map[string]*VM)
  m.groups = make(map[string]*Group)

  // 加载 VMs
  rows, err := m.db.Query(`SELECT id, name, ip, port, username, password, key_path, status, description, created_at, updated_at FROM vms`)
  if err != nil {
    return fmt.Errorf("failed to query VMs: %v", err)
  }
  defer rows.Close()

  for rows.Next() {
    var id, name, ip, username, password, keyPath, status, description string
    var port int
    var createdAt, updatedAt time.Time

    err := rows.Scan(&id, &name, &ip, &port, &username, &password, &keyPath, &status, &description, &createdAt, &updatedAt)
    if err != nil {
      return fmt.Errorf("failed to scan VM row: %v", err)
    }

    vm := &VM{
      Name:        name,
      IP:          ip,
      Port:        port,
      Username:    username,
      Password:    password,
      KeyPath:     keyPath,
      Status:      status,
      Description: description,
      CreatedAt:   createdAt,
      UpdatedAt:   updatedAt,
    }

    // 解密敏感数据
    if password != "" && m.encryptionKey != "" {
      if decrypted, err := crypto.Decrypt(password, m.encryptionKey); err == nil {
        vm.Password = decrypted
      } else {
        log.Printf("Failed to decrypt password for VM %s: %v", name, err)
      }
    }

    if keyPath != "" && m.encryptionKey != "" {
      if decrypted, err := crypto.Decrypt(keyPath, m.encryptionKey); err == nil {
        vm.KeyPath = decrypted
      } else {
        log.Printf("Failed to decrypt key path for VM %s: %v", name, err)
      }
    }

    m.vms[name] = vm
  }

  // 加载 Groups
  rows, err = m.db.Query(`SELECT id, name, description, vms, owner, is_shared, shared_with, created_at, updated_at FROM groups`)
  if err != nil {
    return fmt.Errorf("failed to query groups: %v", err)
  }
  defer rows.Close()

  for rows.Next() {
    var id, name, description, vmsJSON, owner, sharedWithJSON string
    var isShared int
    var createdAt, updatedAt time.Time

    err := rows.Scan(&id, &name, &description, &vmsJSON, &owner, &isShared, &sharedWithJSON, &createdAt, &updatedAt)
    if err != nil {
      return fmt.Errorf("failed to scan group row: %v", err)
    }

    var vms []string
    if err := json.Unmarshal([]byte(vmsJSON), &vms); err != nil {
      vms = []string{}
    }

    var sharedWith []string
    if err := json.Unmarshal([]byte(sharedWithJSON), &sharedWith); err != nil {
      sharedWith = []string{}
    }

    group := &Group{
      Name:        name,
      Description: description,
      VMs:         vms,
      Owner:       owner,
      IsShared:    isShared == 1,
      SharedWith:  sharedWith,
      CreatedAt:   createdAt,
      UpdatedAt:   updatedAt,
    }

    m.groups[name] = group
  }

  log.Printf("Loaded %d VMs and %d groups from database", len(m.vms), len(m.groups))
  return nil
}

// SaveVMs 保存到数据库（此方法保留以兼容现有调用，但不再写入文件）
func (m *Manager) SaveVMs() error {
  // SQLite 数据库是实时保存的，不需要额外的保存操作
  // 此方法保留用于兼容现有代码调用
  log.Printf("SaveVMs called - SQLite auto-saves, no action needed")
  return nil
}

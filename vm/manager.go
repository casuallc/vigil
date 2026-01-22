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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/casuallc/vigil/crypto"
)

// Manager 管理VM实例和相关操作
type Manager struct {
	vms           map[string]*VM
	groups        map[string]*Group
	permissions   *PermissionManager
	mu            sync.RWMutex
	vmFile        string // 存储VM信息的文件路径
	encryptionKey string // 用于加密敏感数据的密钥
}

// NewManager 创建一个新的VM管理器
func NewManager() *Manager {
	// 默认使用当前目录下的vms.json文件存储VM信息
	vmFile := "vms.json"
	manager := &Manager{
		vms:         make(map[string]*VM),
		groups:      make(map[string]*Group),
		permissions: NewPermissionManager(),
		vmFile:      vmFile,
	}

	// 加载VM信息
	if err := manager.LoadVMs(); err != nil {
		log.Printf("Failed to load VMs: %v", err)
		// 继续运行，使用空的VM列表
	}

	return manager
}

// NewManagerWithFile 创建一个新的VM管理器，使用指定的文件存储VM信息
func NewManagerWithFile(vmFile string) *Manager {
	manager := &Manager{
		vms:         make(map[string]*VM),
		groups:      make(map[string]*Group),
		permissions: NewPermissionManager(),
		vmFile:      vmFile,
	}

	// 加载VM信息
	if err := manager.LoadVMs(); err != nil {
		log.Printf("Failed to load VMs: %v", err)
		// 继续运行，使用空的VM列表
	}

	return manager
}

// NewManagerWithConfig 创建一个新的VM管理器，使用配置中的加密密钥
func NewManagerWithConfig(vmFile string, encryptionKey string) *Manager {
	manager := &Manager{
		vms:           make(map[string]*VM),
		groups:        make(map[string]*Group),
		permissions:   NewPermissionManager(),
		vmFile:        vmFile,
		encryptionKey: encryptionKey,
	}

	// 加载VM信息
	if err := manager.LoadVMs(); err != nil {
		log.Printf("Failed to load VMs: %v", err)
		// 继续运行，使用空的VM列表
	}

	return manager
}

// GetVM 根据名称获取VM实例
func (m *Manager) GetVM(name string) (*VM, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vm, exists := m.vms[name]
	if !exists {
		return nil, fmt.Errorf("VM not found: %s", name)
	}
	return vm, nil
}

// ListVMs 列出所有VM实例
func (m *Manager) ListVMs() []*VM {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*VM
	for _, vm := range m.vms {
		result = append(result, vm)
	}
	return result
}

// AddVM 添加一个新的VM实例
func (m *Manager) AddVM(vm *VM) error {
	m.mu.Lock()

	if _, exists := m.vms[vm.Name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("VM already exists: %s", vm.Name)
	}

	m.vms[vm.Name] = vm
	log.Printf("Added new VM: %s", vm.Name)

	// 释放锁，以便SaveVMs可以获取读锁
	m.mu.Unlock()

	// 保存VM信息到文件
	if err := m.SaveVMs(); err != nil {
		log.Printf("Failed to save VMs: %v", err)
		// 继续运行，不返回错误
	}

	return nil
}

// RemoveVM 移除一个VM实例
func (m *Manager) RemoveVM(name string) error {
	m.mu.Lock()

	if _, exists := m.vms[name]; !exists {
		m.mu.Unlock()
		return fmt.Errorf("VM not found: %s", name)
	}

	delete(m.vms, name)
	log.Printf("Removed VM: %s", name)

	// 释放锁，以便SaveVMs可以获取读锁
	m.mu.Unlock()

	// 保存VM信息到文件
	if err := m.SaveVMs(); err != nil {
		log.Printf("Failed to save VMs: %v", err)
		// 继续运行，不返回错误
	}

	return nil
}

// GetPermissionManager 获取权限管理器
func (m *Manager) GetPermissionManager() *PermissionManager {
	return m.permissions
}

// AddGroup 添加一个新的VM组
// AI Modified
func (m *Manager) AddGroup(name, description string, vms []string) error {
	m.mu.Lock()

	// 检查组是否已存在
	if _, exists := m.groups[name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("group already exists: %s", name)
	}

	// 检查VM是否存在
	for _, vmName := range vms {
		if _, exists := m.vms[vmName]; !exists {
			m.mu.Unlock()
			return fmt.Errorf("VM not found: %s", vmName)
		}
	}

	// 创建新组
	group := NewGroup(name, description, vms)
	m.groups[name] = group
	log.Printf("Added group: %s with %d VMs", name, len(vms))

	// 释放锁，以便SaveVMs可以获取读锁
	m.mu.Unlock()

	// 保存VM和组信息到文件
	if err := m.SaveVMs(); err != nil {
		log.Printf("Failed to save VMs and groups: %v", err)
		// 继续运行，不返回错误
	}

	return nil
}

// RemoveGroup 删除一个VM组
// AI Modified
func (m *Manager) RemoveGroup(name string) error {
	m.mu.Lock()

	// 检查组是否存在
	if _, exists := m.groups[name]; !exists {
		m.mu.Unlock()
		return fmt.Errorf("group not found: %s", name)
	}

	// 删除组
	delete(m.groups, name)
	log.Printf("Removed group: %s", name)

	// 释放锁，以便SaveVMs可以获取读锁
	m.mu.Unlock()

	// 保存VM和组信息到文件
	if err := m.SaveVMs(); err != nil {
		log.Printf("Failed to save VMs and groups: %v", err)
		// 继续运行，不返回错误
	}

	return nil
}

// GetGroup 获取一个VM组
// AI Modified
func (m *Manager) GetGroup(name string) (*Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[name]
	if !exists {
		return nil, fmt.Errorf("group not found: %s", name)
	}

	return group, nil
}

// ListGroups 列出所有VM组
// AI Modified
func (m *Manager) ListGroups() []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var groups []*Group
	for _, group := range m.groups {
		groups = append(groups, group)
	}

	return groups
}

// UpdateGroup 更新一个VM组
// AI Modified
func (m *Manager) UpdateGroup(name, description string, vms []string) error {
	m.mu.Lock()

	// 检查组是否存在
	group, exists := m.groups[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("group not found: %s", name)
	}

	// 检查VM是否存在
	for _, vmName := range vms {
		if _, exists := m.vms[vmName]; !exists {
			m.mu.Unlock()
			return fmt.Errorf("VM not found: %s", vmName)
		}
	}

	// 更新组信息
	group.Description = description
	group.VMs = vms
	group.UpdatedAt = time.Now()
	log.Printf("Updated group: %s with %d VMs", name, len(vms))

	// 释放锁，以便SaveVMs可以获取读锁
	m.mu.Unlock()

	// 保存VM和组信息到文件
	if err := m.SaveVMs(); err != nil {
		log.Printf("Failed to save VMs and groups: %v", err)
		// 继续运行，不返回错误
	}

	return nil
}

// LoadVMs 从文件加载VM和组信息
func (m *Manager) LoadVMs() error {
	// 检查文件是否存在
	if _, err := os.Stat(m.vmFile); os.IsNotExist(err) {
		// 文件不存在，创建一个空文件
		file, err := os.Create(m.vmFile)
		if err != nil {
			return fmt.Errorf("failed to create VM file: %v", err)
		}
		defer file.Close()
		// 写入空的VMData结构体
		emptyData := VMData{VMs: []*VM{}, Groups: []*Group{}}
		if err := json.NewEncoder(file).Encode(emptyData); err != nil {
			return fmt.Errorf("failed to write to VM file: %v", err)
		}
		return nil
	}

	// 读取文件内容
	file, err := os.Open(m.vmFile)
	if err != nil {
		return fmt.Errorf("failed to open VM file: %v", err)
	}
	defer file.Close()

	// 解析JSON数据
	var vmData VMData
	if err := json.NewDecoder(file).Decode(&vmData); err != nil {
		// 尝试使用旧格式（仅VM）解码
		file.Seek(0, 0) // 重置文件指针
		var vms []*VM
		if err := json.NewDecoder(file).Decode(&vms); err != nil {
			return fmt.Errorf("failed to decode VM file: %v", err)
		}
		// 使用旧格式数据
		vmData = VMData{VMs: vms, Groups: []*Group{}}
	}

	// 将VM和组添加到管理器中
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有数据
	m.vms = make(map[string]*VM)
	m.groups = make(map[string]*Group)

	// 添加VM
	for _, vm := range vmData.VMs {
		// 解密密码
		if vm.Password != "" && m.encryptionKey != "" {
			decryptedPassword, err := crypto.Decrypt(vm.Password, m.encryptionKey)
			if err != nil {
				log.Printf("Failed to decrypt password for VM %s: %v", vm.Name, err)
			} else {
				vm.Password = decryptedPassword
			}
		}

		// 解密密钥路径
		if vm.KeyPath != "" && m.encryptionKey != "" {
			decryptedKeyPath, err := crypto.Decrypt(vm.KeyPath, m.encryptionKey)
			if err != nil {
				log.Printf("Failed to decrypt key path for VM %s: %v", vm.Name, err)
			} else {
				vm.KeyPath = decryptedKeyPath
			}
		}

		m.vms[vm.Name] = vm
	}

	// 添加组
	for _, group := range vmData.Groups {
		m.groups[group.Name] = group
	}

	log.Printf("Loaded %d VMs and %d groups from %s", len(vmData.VMs), len(vmData.Groups), m.vmFile)
	return nil
}

// VMData 存储VM和组信息
// AI Modified
type VMData struct {
	VMs    []*VM    `json:"vms"`
	Groups []*Group `json:"groups"`
}

// SaveVMs 将VM和组信息保存到文件
func (m *Manager) SaveVMs() error {
	m.mu.RLock()
	var vms []*VM
	for _, vm := range m.vms {
		// 创建一个临时VM结构体，用于存储加密后的敏感数据
		encryptedVM := *vm

		// 加密密码
		if encryptedVM.Password != "" && m.encryptionKey != "" {
			encryptedPassword, err := crypto.Encrypt(encryptedVM.Password, m.encryptionKey)
			if err != nil {
				log.Printf("Failed to encrypt password for VM %s: %v", encryptedVM.Name, err)
			} else {
				encryptedVM.Password = encryptedPassword
			}
		}

		// 加密密钥路径
		if encryptedVM.KeyPath != "" && m.encryptionKey != "" {
			encryptedKeyPath, err := crypto.Encrypt(encryptedVM.KeyPath, m.encryptionKey)
			if err != nil {
				log.Printf("Failed to encrypt key path for VM %s: %v", encryptedVM.Name, err)
			} else {
				encryptedVM.KeyPath = encryptedKeyPath
			}
		}

		vms = append(vms, &encryptedVM)
	}

	// 转换组信息
	var groups []*Group
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	m.mu.RUnlock()

	// 创建VM数据结构体
	vmData := &VMData{
		VMs:    vms,
		Groups: groups,
	}

	// 创建临时文件
	tempFile := m.vmFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary VM file: %v", err)
	}

	// 编码JSON数据
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vmData); err != nil {
		file.Close()
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode VM data: %v", err)
	}

	// 关闭文件
	if err := file.Close(); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to close temporary VM file: %v", err)
	}

	// 替换原文件
	if err := os.Rename(tempFile, m.vmFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temporary VM file: %v", err)
	}

	log.Printf("Saved %d VMs and %d groups to %s", len(vms), len(groups), m.vmFile)
	return nil
}

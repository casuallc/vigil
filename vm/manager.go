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
)

// Manager 管理VM实例和相关操作
type Manager struct {
	vms         map[string]*VM
	permissions *PermissionManager
	mu          sync.RWMutex
	vmFile      string // 存储VM信息的文件路径
}

// NewManager 创建一个新的VM管理器
func NewManager() *Manager {
	// 默认使用当前目录下的vms.json文件存储VM信息
	vmFile := "vms.json"
	manager := &Manager{
		vms:         make(map[string]*VM),
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

// LoadVMs 从文件加载VM信息
func (m *Manager) LoadVMs() error {
	// 检查文件是否存在
	if _, err := os.Stat(m.vmFile); os.IsNotExist(err) {
		// 文件不存在，创建一个空文件
		file, err := os.Create(m.vmFile)
		if err != nil {
			return fmt.Errorf("failed to create VM file: %v", err)
		}
		defer file.Close()
		// 写入空数组
		if _, err := file.Write([]byte("[]")); err != nil {
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
	var vms []*VM
	if err := json.NewDecoder(file).Decode(&vms); err != nil {
		return fmt.Errorf("failed to decode VM file: %v", err)
	}

	// 将VM添加到管理器中
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, vm := range vms {
		m.vms[vm.Name] = vm
	}

	log.Printf("Loaded %d VMs from %s", len(vms), m.vmFile)
	return nil
}

// SaveVMs 将VM信息保存到文件
func (m *Manager) SaveVMs() error {
	m.mu.RLock()
	var vms []*VM
	for _, vm := range m.vms {
		vms = append(vms, vm)
	}
	m.mu.RUnlock()

	// 创建临时文件
	tempFile := m.vmFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary VM file: %v", err)
	}

	// 编码JSON数据
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vms); err != nil {
		file.Close()
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode VMs: %v", err)
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

	log.Printf("Saved %d VMs to %s", len(vms), m.vmFile)
	return nil
}

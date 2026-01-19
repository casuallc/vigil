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
	"fmt"
	"log"
	"sync"
)

// Manager 管理VM实例和相关操作
type Manager struct {
	vms        map[string]*VM
	permissions *PermissionManager
	mu         sync.RWMutex
}

// NewManager 创建一个新的VM管理器
func NewManager() *Manager {
	return &Manager{
		vms:        make(map[string]*VM),
		permissions: NewPermissionManager(),
	}
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
	defer m.mu.Unlock()

	if _, exists := m.vms[vm.Name]; exists {
		return fmt.Errorf("VM already exists: %s", vm.Name)
	}

	m.vms[vm.Name] = vm
	log.Printf("Added new VM: %s", vm.Name)
	return nil
}

// RemoveVM 移除一个VM实例
func (m *Manager) RemoveVM(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[name]; !exists {
		return fmt.Errorf("VM not found: %s", name)
	}

	delete(m.vms, name)
	log.Printf("Removed VM: %s", name)
	return nil
}

// GetPermissionManager 获取权限管理器
func (m *Manager) GetPermissionManager() *PermissionManager {
	return m.permissions
}

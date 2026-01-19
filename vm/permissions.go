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

// Permission 表示VM的访问权限
type Permission struct {
	VMName     string   `json:"vm_name"`
	Username   string   `json:"username"`
	Permissions []string `json:"permissions"`
}

// PermissionManager 管理VM的访问权限
type PermissionManager struct {
	permissions map[string]map[string]map[string]bool
	mu          sync.RWMutex
}

// NewPermissionManager 创建一个新的权限管理器
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		permissions: make(map[string]map[string]map[string]bool),
	}
}

// AddPermission 添加VM访问权限
func (pm *PermissionManager) AddPermission(vmName, username string, permissions []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 确保VM的权限映射存在
	if _, exists := pm.permissions[vmName]; !exists {
		pm.permissions[vmName] = make(map[string]map[string]bool)
	}

	// 确保用户的权限映射存在
	if _, exists := pm.permissions[vmName][username]; !exists {
		pm.permissions[vmName][username] = make(map[string]bool)
	}

	// 添加权限
	for _, perm := range permissions {
		pm.permissions[vmName][username][perm] = true
	}

	log.Printf("Added permissions %v for user %s on VM %s", permissions, username, vmName)
	return nil
}

// RemovePermission 移除VM访问权限
func (pm *PermissionManager) RemovePermission(vmName, username string, permissions []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查VM是否存在
	if _, exists := pm.permissions[vmName]; !exists {
		return fmt.Errorf("VM not found: %s", vmName)
	}

	// 检查用户是否存在
	if _, exists := pm.permissions[vmName][username]; !exists {
		return fmt.Errorf("User not found: %s on VM %s", username, vmName)
	}

	// 移除权限
	for _, perm := range permissions {
		delete(pm.permissions[vmName][username], perm)
	}

	// 如果用户没有任何权限，移除用户
	if len(pm.permissions[vmName][username]) == 0 {
		delete(pm.permissions[vmName], username)
	}

	// 如果VM没有任何用户，移除VM
	if len(pm.permissions[vmName]) == 0 {
		delete(pm.permissions, vmName)
	}

	log.Printf("Removed permissions %v for user %s on VM %s", permissions, username, vmName)
	return nil
}

// CheckPermission 检查用户对VM的权限
func (pm *PermissionManager) CheckPermission(vmName, username, permission string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 检查VM是否存在
	vmPerms, exists := pm.permissions[vmName]
	if !exists {
		return false
	}

	// 检查用户是否存在
	userPerms, exists := vmPerms[username]
	if !exists {
		return false
	}

	// 检查是否有该权限或所有权限
	return userPerms[permission] || userPerms["all"]
}

// ListPermissions 列出VM的所有权限
func (pm *PermissionManager) ListPermissions(vmName string) []*Permission {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []*Permission

	// 检查VM是否存在
	vmPerms, exists := pm.permissions[vmName]
	if !exists {
		return result
	}

	// 收集所有用户的权限
	for username, userPerms := range vmPerms {
		var permissions []string
		for perm := range userPerms {
			permissions = append(permissions, perm)
		}

		result = append(result, &Permission{
			VMName:     vmName,
			Username:   username,
			Permissions: permissions,
		})
	}

	return result
}

// GetUserPermissions 获取用户对VM的权限
func (pm *PermissionManager) GetUserPermissions(vmName, username string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 检查VM是否存在
	vmPerms, exists := pm.permissions[vmName]
	if !exists {
		return nil
	}

	// 检查用户是否存在
	userPerms, exists := vmPerms[username]
	if !exists {
		return nil
	}

	// 收集用户的权限
	var permissions []string
	for perm := range userPerms {
		permissions = append(permissions, perm)
	}

	return permissions
}

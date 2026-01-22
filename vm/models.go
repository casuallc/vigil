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
	"time"
)

// VM 表示虚拟机实例的基本信息
type VM struct {
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	Password    string    `json:"password,omitempty"`
	KeyPath     string    `json:"key_path,omitempty"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewVM 创建一个新的VM实例
func NewVM(name, ip string, port int, username, password, keyPath string) *VM {
	now := time.Now()
	return &VM{
		Name:      name,
		IP:        ip,
		Port:      port,
		Username:  username,
		Password:  password,
		KeyPath:   keyPath,
		Status:    "stopped",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateStatus 更新VM的状态
func (vm *VM) UpdateStatus(status string) {
	vm.Status = status
	vm.UpdatedAt = time.Now()
}

// SSHConfig 表示SSH连接的配置信息
type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	KeyPath  string `json:"key_path,omitempty"`
}

// SSHForwardConfig 表示SSH转发服务的配置信息
type SSHForwardConfig struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	TargetHost       string `json:"target_host"`
	TargetPort       int    `json:"target_port"`
	TargetUsername   string `json:"target_username"`
	TargetPassword   string `json:"target_password,omitempty"`
	TargetPrivateKey string `json:"target_private_key,omitempty"`
	AuditLogPath     string `json:"audit_log_path,omitempty"`
}

// FileTransferConfig 表示文件传输的配置信息
type FileTransferConfig struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
}

// FileInfo 表示文件或目录的信息
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	Depth   int    `json:"depth"`
}

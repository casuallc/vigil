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

package proc

import (
  "github.com/casuallc/vigil/config"
)

// ProcessScanner 定义进程扫描相关操作
type ProcessScanner interface {
  // ScanProcesses 扫描系统进程
  ScanProcesses(query string) ([]ManagedProcess, error)
}

// ProcessLifecycle 定义进程生命周期管理相关操作
type ProcessLifecycle interface {
  // CreateProcess 创建并管理一个进程
  CreateProcess(process ManagedProcess) error
  // StartProcess 启动一个已管理的进程
  StartProcess(namespace, name string) error
  // StopProcess 停止一个已管理的进程
  StopProcess(namespace, name string) error
  // RestartProcess 重启一个已管理的进程
  RestartProcess(namespace, name string) error
  // DeleteProcess 删除一个已管理的进程
  DeleteProcess(namespace, name string) error
}

// ProcessInfo 定义进程信息查询相关操作
type ProcessInfo interface {
  // GetProcessStatus 获取进程状态
  GetProcessStatus(namespace, name string) (ManagedProcess, error)
  // ListManagedProcesses 获取所有已管理的进程
  ListManagedProcesses(namespace string) ([]ManagedProcess, error)
  // GetProcesses 获取所有进程的映射
  GetProcesses() map[string]*ManagedProcess
}

// ProcessConfig 定义进程配置相关操作
type ProcessConfig interface {
  // UpdateProcessConfig 更新进程配置
  UpdateProcessConfig(namespace, name string, config config.AppConfig) error
  // SaveManagedProcesses 保存所有已管理的进程到文件
  SaveManagedProcesses(filePath string) error
  // LoadManagedProcesses 从文件加载已管理的进程
  LoadManagedProcesses(filePath string) error
}

// ProcessMonitor 定义进程监控相关操作
type ProcessMonitor interface {
  // MonitorProcess 监控进程资源使用情况
  MonitorProcess(namespace, name string) (*ResourceStats, error)
  // StartMonitoring 开始监控进程
  StartMonitoring(namespace, name string)
  // TryMatchProcessByScript 尝试使用用户定义的脚本匹配系统中的进程
  TryMatchProcessByScript(managedProc *ManagedProcess) (*ManagedProcess, error)
}

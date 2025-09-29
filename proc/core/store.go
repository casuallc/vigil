package core

import (
  "fmt"
  "github.com/casuallc/vigil/proc"
  "gopkg.in/yaml.v3"
  "os"
  "time"
)

// SaveManagedProcesses 保存所有已管理的进程到文件
func (m *Manager) SaveManagedProcesses(filePath string) error {
  processesList := make([]proc.ManagedProcess, 0, len(m.Processes))

  // 过滤掉运行时的状态信息，只保存配置相关信息
  for _, p := range m.Processes {
    processCopy := *p
    // 重置运行时状态
    processCopy.Status.Phase = proc.PhaseFailed
    processCopy.Status.PID = 0
    processCopy.Status.StartTime = &time.Time{}
    processCopy.Status.ResourceStats = nil

    processesList = append(processesList, processCopy)
  }

  // 将进程信息转换为YAML
  data, err := yaml.Marshal(processesList)
  if err != nil {
    return fmt.Errorf("failed to marshal processes: %v", err)
  }

  // 确保目录存在
  dir := "proc"
  if err := os.MkdirAll(dir, 0755); err != nil {
    return fmt.Errorf("failed to create directory: %v", err)
  }

  // 保存到文件
  return os.WriteFile(filePath, data, 0644)
}

// LoadManagedProcesses 从文件加载已管理的进程
func (m *Manager) LoadManagedProcesses(filePath string) error {
  // 检查文件是否存在
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    return nil // 文件不存在，不执行加载
  }

  // 读取文件内容
  data, err := os.ReadFile(filePath)
  if err != nil {
    return fmt.Errorf("failed to read processes file: %v", err)
  }

  // 解析YAML
  var processesList []proc.ManagedProcess
  if err := yaml.Unmarshal(data, &processesList); err != nil {
    return fmt.Errorf("failed to unmarshal processes: %v", err)
  }

  // 将进程添加到管理器
  for _, process := range processesList {
    processCopy := process
    // 使用 namespace/name 作为键
    key := fmt.Sprintf("%s/%s", process.Metadata.Namespace, process.Metadata.Name)
    m.Processes[key] = &processCopy

    // 自动启动标记为需要重启的进程
    if process.Spec.RestartPolicy == proc.RestartPolicyAlways ||
      (process.Spec.RestartPolicy == proc.RestartPolicyOnFailure && process.Status.LastTerminationInfo.ExitCode != 0) {
      go func(namespace, name string) {
        // 延迟启动，避免启动时资源竞争
        time.Sleep(1 * time.Second)
        if err := m.StartProcess(namespace, name); err != nil {
          fmt.Printf("Failed to start proc %s/%s on startup: %v\n", namespace, name, err)
        }
      }(process.Metadata.Namespace, process.Metadata.Name)
    }
  }

  return nil
}

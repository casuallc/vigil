package process

import (
  "errors"
  "fmt"
)

// NewManager creates a new process manager
func NewManager() *Manager {
  return &Manager{
    processes: make(map[string]*ManagedProcess),
  }
}

// GetProcessStatus implements ProcManager interface
func (m *Manager) GetProcessStatus(name string) (ManagedProcess, error) {
  process, exists := m.processes[name]
  if !exists {
    return ManagedProcess{}, errors.New(fmt.Sprintf("Process %s is not managed", name))
  }
  return *process, nil
}

// ListManagedProcesses implements ProcManager interface
func (m *Manager) ListManagedProcesses() ([]ManagedProcess, error) {
  result := make([]ManagedProcess, 0, len(m.processes))
  for _, p := range m.processes {
    result = append(result, *p)
  }
  return result, nil
}

// MonitorProcess implements ProcManager interface
func (m *Manager) MonitorProcess(name string) (ResourceStats, error) {
  // 检查进程是否存在
  process, exists := m.processes[name]
  if !exists {
    return ResourceStats{}, errors.New(fmt.Sprintf("Process %s is not managed", name))
  }

  // 检查进程是否正在运行
  if process.Status != StatusRunning {
    return ResourceStats{}, errors.New(fmt.Sprintf("Process %s is not running", name))
  }

  pid := process.PID

  // 获取CPU和内存使用情况
  cpuUsage, memoryUsage, err := GetProcessCpuAndMemory(pid)
  if err != nil {
    return ResourceStats{}, fmt.Errorf("failed to get CPU and memory usage: %v", err)
  }

  // 获取磁盘IO统计信息
  diskIO, err := GetProcessDiskIO(pid)
  if err != nil {
    // 磁盘IO获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get disk IO: %v\n", err)
  }

  // 获取网络IO统计信息
  networkIO, err := GetProcessNetworkIO(pid)
  if err != nil {
    // 网络IO获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get network IO: %v\n", err)
  }

  // 获取监听端口信息
  listeningPorts, err := GetProcessListeningPorts(pid)
  if err != nil {
    // 监听端口获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get listening ports: %v\n", err)
  }

  // 创建并返回ResourceStats
  stats := ResourceStats{
    CPUUsage:       cpuUsage,
    MemoryUsage:    memoryUsage,
    DiskIO:         diskIO,
    NetworkIO:      networkIO,
    ListeningPorts: listeningPorts,
  }

  // 更新进程的Stats信息
  process.Stats = stats

  return stats, nil
}

// 注意：其他ProcessManager接口方法（ScanProcesses、ManageProcess等）需要根据实际实现添加

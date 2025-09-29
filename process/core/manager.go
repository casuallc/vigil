package core

import (
  "errors"
  "fmt"
  "github.com/casuallc/vigil/process"
)

// NewManager 修改NewManager函数
func NewManager() *process.Manager {
  return &process.Manager{
    Processes: make(map[string]*process.ManagedProcess),
  }
}

// GetProcesses 获取进程
func (m *process.Manager) GetProcesses() map[string]*process.ManagedProcess {
  return m.Processes
}

// GetProcessStatus 修改所有使用m.Processes的地方为m.Processes
// 例如：
func (m *process.Manager) GetProcessStatus(namespace, name string) (process.ManagedProcess, error) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return process.ManagedProcess{}, errors.New(fmt.Sprintf("Process %s/%s is not managed", namespace, name))
  }
  return *process, nil
}

// ListManagedProcesses implements ProcManager interface
func (m *process.Manager) ListManagedProcesses(namespace string) ([]process.ManagedProcess, error) {
  result := make([]process.ManagedProcess, 0)

  for _, p := range m.Processes {
    // 如果指定了namespace，则只返回该namespace的进程
    if namespace == "" || p.Metadata.Namespace == namespace {
      result = append(result, *p)
    }
  }
  return result, nil
}

// MonitorProcess implements ProcManager interface
func (m *process.Manager) MonitorProcess(namespace, name string) (*process.ResourceStats, error) {
  // 检查进程是否存在
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return nil, errors.New(fmt.Sprintf("Process %s/%s is not managed", namespace, name))
  }

  // 检查进程是否正在运行
  if process.Status.Phase != process.PhaseRunning {
    return nil, errors.New(fmt.Sprintf("Process %s/%s is not running", namespace, name))
  }

  pid := process.Status.PID

  // 获取CPU和内存使用情况
  cpuUsage, memoryUsage, err := process.GetProcessCpuAndMemory(pid)
  if err != nil {
    return nil, fmt.Errorf("failed to get CPU and memory usage: %v", err)
  }

  // 获取磁盘IO统计信息
  diskIO, err := process.GetProcessDiskIO(pid)
  if err != nil {
    // 磁盘IO获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get disk IO: %v\n", err)
  }

  // 获取网络IO统计信息
  networkIO, err := process.GetProcessNetworkIO(pid)
  if err != nil {
    // 网络IO获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get network IO: %v\n", err)
  }

  // 获取监听端口信息
  listeningPorts, err := process.GetProcessListeningPorts(pid)
  if err != nil {
    // 监听端口获取失败不应阻止整个监控过程
    fmt.Printf("Warning: failed to get listening ports: %v\n", err)
  }

  // 创建并返回ResourceStats
  stats := &process.ResourceStats{
    CPUUsage:       cpuUsage,
    MemoryUsage:    memoryUsage,
    DiskIO:         diskIO,
    NetworkIO:      networkIO,
    ListeningPorts: listeningPorts,
  }

  // 设置格式化的值
  stats.SetFormattedValues()

  // 更新进程的Stats信息
  process.Status.ResourceStats = stats

  return stats, nil
}

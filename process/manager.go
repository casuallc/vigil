package process

import (
  "errors"
  "fmt"
  "os"
  "strings"
  "time"

  "gopkg.in/yaml.v2"
)

// NewManager creates a new process manager
func NewManager() *Manager {
  return &Manager{
    processes: make(map[string]*ManagedProcess),
  }
}

// GetProcessStatus implements ProcManager interface
func (m *Manager) GetProcessStatus(namespace, name string) (ManagedProcess, error) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.processes[key]
  if !exists {
    return ManagedProcess{}, errors.New(fmt.Sprintf("Process %s/%s is not managed", namespace, name))
  }
  return *process, nil
}

// ListManagedProcesses implements ProcManager interface
func (m *Manager) ListManagedProcesses(namespace string) ([]ManagedProcess, error) {
  result := make([]ManagedProcess, 0)

  for _, p := range m.processes {
    // 如果指定了namespace，则只返回该namespace的进程
    if namespace == "" || p.Metadata.Namespace == namespace {
      result = append(result, *p)
    }
  }
  return result, nil
}

// MonitorProcess implements ProcManager interface
func (m *Manager) MonitorProcess(namespace, name string) (*ResourceStats, error) {
  // 检查进程是否存在
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.processes[key]
  if !exists {
    return nil, errors.New(fmt.Sprintf("Process %s/%s is not managed", namespace, name))
  }

  // 检查进程是否正在运行
  if process.Status.Phase != PhaseRunning {
    return nil, errors.New(fmt.Sprintf("Process %s/%s is not running", namespace, name))
  }

  pid := process.Status.PID

  // 获取CPU和内存使用情况
  cpuUsage, memoryUsage, err := GetProcessCpuAndMemory(pid)
  if err != nil {
    return nil, fmt.Errorf("failed to get CPU and memory usage: %v", err)
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
  stats := &ResourceStats{
    CPUUsage:       cpuUsage,
    MemoryUsage:    memoryUsage,
    DiskIO:         diskIO,
    NetworkIO:      networkIO,
    ListeningPorts: listeningPorts,
  }

  // 更新进程的Stats信息
  process.Status.ResourceStats = stats

  return stats, nil
}

// SaveManagedProcesses saves all managed processes to a file
func (m *Manager) SaveManagedProcesses(filePath string) error {
  processesList := make([]ManagedProcess, 0, len(m.processes))

  // 过滤掉运行时的状态信息，只保存配置相关信息
  for _, p := range m.processes {
    processCopy := *p
    // 重置运行时状态
    processCopy.Status.Phase = PhaseFailed
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
  dir := "process"
  if err := os.MkdirAll(dir, 0755); err != nil {
    return fmt.Errorf("failed to create directory: %v", err)
  }

  // 保存到文件
  return os.WriteFile(filePath, data, 0644)
}

// LoadManagedProcesses loads managed processes from a file
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
  var processesList []ManagedProcess
  if err := yaml.Unmarshal(data, &processesList); err != nil {
    return fmt.Errorf("failed to unmarshal processes: %v", err)
  }

  // 将进程添加到管理器
  for _, process := range processesList {
    processCopy := process
    // 使用 namespace/name 作为键
    key := fmt.Sprintf("%s/%s", process.Metadata.Namespace, process.Metadata.Name)
    m.processes[key] = &processCopy

    // 自动启动标记为需要重启的进程
    if process.Spec.RestartPolicy == RestartPolicyAlways ||
      (process.Spec.RestartPolicy == RestartPolicyOnFailure && process.Status.LastTerminationInfo.ExitCode != 0) {
      go func(namespace, name string) {
        // 延迟启动，避免启动时资源竞争
        time.Sleep(1 * time.Second)
        if err := m.StartProcess(namespace, name); err != nil {
          fmt.Printf("Failed to start process %s/%s on startup: %v\n", namespace, name, err)
        }
      }(process.Metadata.Namespace, process.Metadata.Name)
    }
  }

  return nil
}

// ScanProcesses implements ProcManager interface to scan system processes
func (m *Manager) ScanProcesses(query string) ([]ManagedProcess, error) {
  // 根据查询类型选择不同的扫描方法
  if strings.HasPrefix(query, ScriptPrefix) {
    // 直接执行内联脚本
    scriptContent := strings.TrimPrefix(query, ScriptPrefix)
    return m.scanWithScript(scriptContent)
  } else if strings.HasPrefix(query, FileScriptPrefix) {
    // 从文件加载脚本并执行
    scriptPath := strings.TrimPrefix(query, FileScriptPrefix)
    content, err := os.ReadFile(scriptPath)
    if err != nil {
      return nil, fmt.Errorf("failed to read script file: %v", err)
    }
    return m.scanWithScript(string(content))
  } else {
    // 使用标准的Unix进程扫描
    return m.scanUnixProcesses(query)
  }
}

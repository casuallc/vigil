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
  "bytes"
  "fmt"
  "os"
  "os/exec"
  "strconv"
  "strings"
  "time"

  "github.com/casuallc/vigil/config"
  "github.com/shirou/gopsutil/v3/process"
)

// NewManager 创建一个新的进程管理器
func NewManager() *Manager {
  return &Manager{
    Processes:         make(map[string]*ManagedProcess),
    monitoringStarted: make(map[string]bool),
  }
}

// Manager 实现了所有进程管理相关的接口
// 它实现了 ProcessScanner, ProcessLifecycle, ProcessInfo, ProcessConfig 和 ProcessMonitor 接口
type Manager struct {
  Processes map[string]*ManagedProcess
  // 新增：监控协程启动标记，避免重复启动
  monitoringStarted map[string]bool
}

// GetProcesses 获取所有进程的映射
func (m *Manager) GetProcesses() map[string]*ManagedProcess {
  return m.Processes
}

// GetProcessStatus 获取进程状态
func (m *Manager) GetProcessStatus(namespace, name string) (ManagedProcess, error) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  managedProcess, exists := m.Processes[key]
  if !exists {
    return ManagedProcess{}, fmt.Errorf("process %s/%s is not managed", namespace, name)
  }
  return *managedProcess, nil
}

// ListManagedProcesses 获取所有已管理的进程
func (m *Manager) ListManagedProcesses(namespace string) ([]ManagedProcess, error) {
  result := make([]ManagedProcess, 0)

  for _, p := range m.Processes {
    // 如果指定了namespace，则只返回该namespace的进程
    if namespace == "" || p.Metadata.Namespace == namespace {
      result = append(result, *p)
    }
  }
  return result, nil
}

// MonitorProcess 监控进程资源使用情况
func (m *Manager) MonitorProcess(namespace, name string) (*ResourceStats, error) {
  // 检查进程是否存在
  key := fmt.Sprintf("%s/%s", namespace, name)
  managedProcess, exists := m.Processes[key]
  if !exists {
    return nil, fmt.Errorf("Process %s/%s is not managed", namespace, name)
  }

  // 检查进程是否正在运行
  if managedProcess.Status.Phase != PhaseRunning {
    return nil, fmt.Errorf("Process %s/%s is not running", namespace, name)
  }

  pid := managedProcess.Status.PID
  stats, err := GetUnixProcessResourceUsage(pid)
  if err != nil {
    fmt.Printf("Warning: failed to get process resource usage: %v\n", err)
    return nil, err
  }

  // 设置格式化的值
  stats.SetFormattedValues()

  // 更新进程的Stats信息
  managedProcess.Status.ResourceStats = stats

  return stats, nil
}

// ScanProcesses 扫描系统进程
func (m *Manager) ScanProcesses(query string) ([]ManagedProcess, error) {
  // 根据查询类型选择不同的扫描方法
  if strings.HasPrefix(query, ScriptPrefix) {
    // 直接执行内联脚本
    scriptContent := strings.TrimPrefix(query, ScriptPrefix)
    return ScanWithScript(scriptContent)
  } else if strings.HasPrefix(query, FileScriptPrefix) {
    // 从文件加载脚本并执行
    scriptPath := strings.TrimPrefix(query, FileScriptPrefix)
    content, err := os.ReadFile(scriptPath)
    if err != nil {
      return nil, fmt.Errorf("failed to read script file: %v", err)
    }
    return ScanWithScript(string(content))
  } else {
    // 使用标准的Unix进程扫描
    return ScanUnixProcesses(query)
  }
}

// ProcessesFilePath 添加一个全局变量来保存进程配置文件路径
var ProcessesFilePath = "proc/managed_processes.yaml"

// UpdateProcessConfig 实现ProcessManager接口，更新进程配置
func (m *Manager) UpdateProcessConfig(namespace, name string, config config.AppConfig) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  managedProcess, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("进程 %s/%s 未被纳管", namespace, name)
  }

  // 保存旧配置
  oldConfig := managedProcess.Spec.Config

  // 更新配置
  managedProcess.Spec.Config = config

  // 如果进程正在运行，需要重启来应用新配置
  if managedProcess.Status.Phase == PhaseRunning {
    // 重启进程
    if err := m.RestartProcess(namespace, name); err != nil {
      // 如果重启失败，恢复旧配置
      managedProcess.Spec.Config = oldConfig
      return err
    }
  }

  return nil
}

// StartMonitoring 启动进程监控
func (m *Manager) StartMonitoring(namespace, name string) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  if m.monitoringStarted[key] {
    return
  }
  m.monitoringStarted[key] = true
  go m.startMonitoring(namespace, name)
}

// startMonitoring 开始监控进程
func (m *Manager) startMonitoring(namespace, name string) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  managedProcess, exists := m.Processes[key]
  if !exists {
    return
  }

  // 定期更新进程状态和资源使用情况
  ticker := time.NewTicker(5 * time.Second)
  defer ticker.Stop()

  // 添加重关联检查的计时器
  checkTicker := time.NewTicker(30 * time.Second)
  defer checkTicker.Stop()

  for {
    select {
    case <-ticker.C:
      if managedProcess.Status.Phase == PhaseRunning {
        // 更新资源使用情况
        stats, err := m.MonitorProcess(namespace, name)
        if err == nil {
          managedProcess.Status.ResourceStats = stats
        }
      }

    case <-checkTicker.C:
      // 扩展：运行态与非运行态都做重关联检查
      if managedProcess.Status.Phase == PhaseRunning {
        // 运行态：校验当前 PID 是否仍对应同一逻辑进程（防 PID 复用）
        currentPID := managedProcess.Status.PID
        valid := false

        if currentPID > 0 {
          if pidExists, err := process.PidExists(int32(currentPID)); err == nil && pidExists {
            if p, err := process.NewProcess(int32(currentPID)); err == nil {
              if m.matchProcess(managedProcess, p) {
                valid = true
              }
            }
          }
        }

        if !valid {
          // 尝试重关联到新的 PID
          reAssociated, _ := m.tryCheckProcess(managedProcess)
          if !reAssociated {
            // 重关联失败，标记停止并按策略后续处理
            managedProcess.Status.Phase = PhaseStopped
            managedProcess.Status.PID = 0
          }
        }
      } else {
        // 非运行态（Failed/Stopped/Pending）：尝试发现进程是否已重新起来并重关联
        reAssociated, _ := m.tryCheckProcess(managedProcess)
        if reAssociated {
          // 已自动重新关联到新 PID，状态在 tryCheckProcess 中置为 Running
          // 可选：此处无需额外处理
          fmt.Printf("re associated process %s status, current: %s\n", managedProcess.Metadata.Name, managedProcess.Status.Phase)
        } else {
          // 保持原状态（Failed/Stopped/Pending），等待下次检查或人工启动
        }
      }
    }
  }
}

// matchProcess 根据多个特征匹配进程，优先使用用户定义的识别脚本
func (m *Manager) matchProcess(managedProc *ManagedProcess, sysProcess *process.Process) bool {
  // 如果用户定义了识别脚本，优先使用它
  if managedProc.Spec.ProcessMatcher != nil {
    return m.matchProcessByScript(managedProc, sysProcess)
  }

  // 否则使用传统的匹配方法
  return m.matchProcessByAttributes(managedProc, sysProcess)
}

// matchProcessByScript 使用用户定义的脚本匹配进程
func (m *Manager) matchProcessByScript(managedProc *ManagedProcess, sysProcess *process.Process) bool {
  // 获取进程的PID
  pid := int(sysProcess.Pid)

  // 检查该PID是否与脚本匹配到的PID相同
  matchedPID, err := m.getMatchedPIDByScript(managedProc)
  if err != nil {
    fmt.Printf("Error getting matched PID: %v\n", err)
    return false
  }

  return matchedPID == pid
}

// matchProcessByAttributes 使用进程属性匹配进程
func (m *Manager) matchProcessByAttributes(managedProc *ManagedProcess, sysProcess *process.Process) bool {
  // 匹配命令行参数
  cmdline, err := sysProcess.Cmdline()
  if err == nil {
    for _, arg := range managedProc.Spec.Exec.Args {
      if !strings.Contains(cmdline, arg) {
        return false
      }
    }
  }

  // 匹配工作目录
  cwd, err := sysProcess.Cwd()
  if err == nil && managedProc.Spec.WorkingDir != "" {
    if cwd != managedProc.Spec.WorkingDir {
      return false
    }
  }

  // 4. 匹配环境变量（可选，对于关键环境变量进行匹配）
  //for _, envVar := range managedProc.Spec.Env {
  //  // 这里可以添加环境变量匹配逻辑
  //  // 为简化实现，我们可以跳过这一步
  //}

  // 如果通过了以上检查，则认为是匹配的进程
  return true
}

// getMatchedPIDByScript 直接通过脚本获取匹配的进程ID
func (m *Manager) getMatchedPIDByScript(managedProc *ManagedProcess) (int, error) {
  if managedProc.Spec.ProcessMatcher == nil {
    return 0, fmt.Errorf("no process matcher script defined")
  }

  fullCmd := fmt.Sprintf("%s %s",
    managedProc.Spec.ProcessMatcher.Command,
    strings.Join(managedProc.Spec.ProcessMatcher.Args, " "))
  cmd := exec.Command("/bin/bash", "-c", fullCmd)

  // 设置环境变量
  cmd.Env = os.Environ()
  for _, envVar := range managedProc.Spec.Env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
  }

  // 设置工作目录
  if managedProc.Spec.WorkingDir != "" {
    cmd.Dir = managedProc.Spec.WorkingDir
  }

  // 设置超时
  timeout := managedProc.Spec.ProcessMatcher.Timeout
  if timeout == 0 {
    timeout = 5 * time.Second // 默认5秒超时
  }

  // 捕获输出
  var stdout bytes.Buffer
  cmd.Stdout = &stdout

  // 执行脚本并等待结果
  done := make(chan error, 1)
  go func() {
    done <- cmd.Run()
  }()

  // 等待命令完成或超时
  select {
  case err := <-done:
    if err != nil {
      return 0, fmt.Errorf("process matcher script failed: %w", err)
    }
    // 解析输出获取PID
    pidStr := strings.TrimSpace(stdout.String())
    if pidStr == "" {
      return 0, fmt.Errorf("process matcher script returned empty output")
    }
    pid, err := strconv.Atoi(pidStr)
    if err != nil {
      return 0, fmt.Errorf("invalid PID returned by process matcher script: %s, error: %w", pidStr, err)
    }
    return pid, nil
  case <-time.After(timeout):
    // 超时，强制终止命令
    if cmd.Process != nil {
      cmd.Process.Kill()
    }
    return 0, fmt.Errorf("process matcher script timed out after %v", timeout)
  }
}

// TryMatchProcessByScript 尝试使用用户定义的脚本直接匹配进程ID
func (m *Manager) TryMatchProcessByScript(managedProc *ManagedProcess) (*ManagedProcess, error) {
  if managedProc.Spec.ProcessMatcher == nil {
    return nil, fmt.Errorf("no process matcher script defined for proc")
  }

  // 直接通过脚本获取匹配的进程ID
  pid, err := m.getMatchedPIDByScript(managedProc)
  if err != nil {
    return nil, fmt.Errorf("failed to get matched process ID: %w", err)
  }

  // 检查进程是否存在
  if pidExists, err := process.PidExists(int32(pid)); err != nil || !pidExists {
    return nil, fmt.Errorf("matched process ID %d does not exist or cannot be accessed", pid)
  }

  // 找到匹配的进程，更新信息
  matchedProcess := *managedProc
  matchedProcess.Status.PID = pid
  matchedProcess.Status.Phase = PhaseRunning

  // 获取并更新进程的其他信息
  if err := m.updateProcessInfoFromSystem(&matchedProcess); err != nil {
    // 更新失败不影响匹配结果，但记录警告
    fmt.Printf("Warning: failed to update proc info: %v\n", err)
  }

  return &matchedProcess, nil
}

// updateProcessInfoFromSystem 从系统中更新进程的详细信息
func (m *Manager) updateProcessInfoFromSystem(managedProc *ManagedProcess) error {
  // 获取系统进程对象
  sysProc, err := process.NewProcess(int32(managedProc.Status.PID))
  if err != nil {
    return err
  }

  // 更新进程的基本信息
  if err = FillBasicInfo(managedProc, sysProc); err != nil {
    return fmt.Errorf("failed to fill basic info: %w", err)
  }

  // 更新资源统计信息
  stats, err := m.MonitorProcess(managedProc.Metadata.Namespace, managedProc.Metadata.Name)
  if err == nil {
    managedProc.Status.ResourceStats = stats
  }

  return nil
}

// CheckProcesses 检查并重新关联可能已重启的进程
func (m *Manager) CheckProcesses() {
  // 遍历所有纳管的进程
  for key, managedProc := range m.Processes {
    // 只处理应该运行但当前未运行的进程
    if managedProc.Status.Phase != PhaseRunning &&
      (managedProc.Spec.RestartPolicy == RestartPolicyAlways ||
        managedProc.Spec.RestartPolicy == RestartPolicyOnFailure) {

      // 尝试通过标识重新发现进程
      checked, err := m.tryCheckProcess(managedProc)
      if err == nil && checked {
        fmt.Printf("Successfully Checkd proc %s\n", key)
      } else if err != nil {
        fmt.Printf("Failed to Check proc %s: %v\n", key, err)
      }
    }
  }
}

// tryCheckProcess 尝试通过进程特征检查进程状态
func (m *Manager) tryCheckProcess(managedProc *ManagedProcess) (bool, error) {
  // 如果定义了进程匹配脚本，直接使用脚本获取进程ID
  if managedProc.Spec.ProcessMatcher != nil {
    pid, err := m.getMatchedPIDByScript(managedProc)
    if err != nil {
      return false, fmt.Errorf("failed to get matched process ID: %w", err)
    }

    // 检查进程是否存在
    if pidExists, err := process.PidExists(int32(pid)); err != nil || !pidExists {
      return false, nil // 进程不存在，返回false但不报错
    }

    // 找到匹配的进程，更新状态
    managedProc.Status.PID = pid
    managedProc.Status.Phase = PhaseRunning
    now := time.Now()
    managedProc.Status.StartTime = &now

    // 更新资源监控信息
    stats, err := m.MonitorProcess(managedProc.Metadata.Namespace, managedProc.Metadata.Name)
    if err == nil {
      managedProc.Status.ResourceStats = stats
    }

    return true, nil
  } else {
    // 如果没有定义进程匹配脚本，使用传统方式遍历所有进程
    allProcesses, err := process.Processes()
    if err != nil {
      return false, fmt.Errorf("failed to get all processes: %w", err)
    }

    // 遍历所有进程，寻找匹配的进程
    for _, sysProcess := range allProcesses {
      // 尝试匹配进程特征
      if m.matchProcessByAttributes(managedProc, sysProcess) {
        // 找到匹配的进程，更新状态
        managedProc.Status.PID = int(sysProcess.Pid)
        managedProc.Status.Phase = PhaseRunning
        now := time.Now()
        managedProc.Status.StartTime = &now

        // 更新资源监控信息
        stats, err := m.MonitorProcess(managedProc.Metadata.Namespace, managedProc.Metadata.Name)
        if err == nil {
          managedProc.Status.ResourceStats = stats
        }

        return true, nil
      }
    }
  }

  return false, nil
}

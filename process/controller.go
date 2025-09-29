package process

import (
  "errors"
  "fmt"
  "github.com/casuallc/vigil/process/scanner"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strconv"
  "strings"
  "syscall"
  "time"

  "github.com/casuallc/vigil/config"
)

// ProcessesFilePath 添加一个全局变量来保存进程配置文件路径
var ProcessesFilePath = "process/managed_processes.yaml"

// CreateProcess implements ProcManager interface to manage a process
func (m *Manager) CreateProcess(process ManagedProcess) error {
  // 如果未指定namespace，使用默认namespace
  if process.Metadata.Namespace == "" {
    process.Metadata.Namespace = "default"
  }

  key := fmt.Sprintf("%s/%s", process.Metadata.Namespace, process.Metadata.Name)
  if _, exists := m.Processes[key]; exists {
    return fmt.Errorf("process %s/%s is already managed", process.Metadata.Namespace, process.Metadata.Name)
  }

  // Generate ID for the process
  process.Metadata.ID = fmt.Sprintf("%s-%d", process.Metadata.Name, time.Now().UnixNano())

  // Store the process
  m.Processes[key] = &process

  // 保存进程信息
  if err := m.SaveManagedProcesses(ProcessesFilePath); err != nil {
    fmt.Printf("Warning: failed to save managed processes: %v\n", err)
  }

  // Start monitoring the process
  go m.startMonitoring(process.Metadata.Namespace, process.Metadata.Name)

  return nil
}

// UpdateProcessConfig 实现ProcessManager接口，更新进程配置
func (m *Manager) UpdateProcessConfig(namespace, name string, config config.AppConfig) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("进程 %s/%s 未被纳管", namespace, name)
  }

  // 保存旧配置
  oldConfig := process.Spec.Config

  // 更新配置
  process.Spec.Config = config

  // 如果进程正在运行，需要重启来应用新配置
  if process.Status.Phase == PhaseRunning {
    // 重启进程
    if err := m.RestartProcess(namespace, name); err != nil {
      // 如果重启失败，恢复旧配置
      process.Spec.Config = oldConfig
      return err
    }
  }

  return nil
}

// startMonitoring 开始监控进程
func (m *Manager) startMonitoring(namespace, name string) {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
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
      if process.Status.Phase == PhaseRunning {
        // 更新资源使用情况
        stats, err := m.MonitorProcess(namespace, name)
        if err == nil {
          process.Status.ResourceStats = stats
        }
      }

    case <-checkTicker.C:
      // 检查进程是否还在运行
      if process.Status.Phase == PhaseRunning {
        sysProcess, err := os.FindProcess(process.Status.PID)
        if err != nil {
          // 进程不存在了，标记为已停止
          process.Status.Phase = PhaseStopped
          process.Status.PID = 0

          // 尝试重新关联进程
          if process.Spec.RestartPolicy == RestartPolicyAlways ||
            process.Spec.RestartPolicy == RestartPolicyOnFailure {
            go m.tryCheckProcess(process)
          }
        } else {
          // 在Unix系统上，我们可以发送0信号来检查进程是否存在
          if err := sysProcess.Signal(syscall.Signal(0)); err != nil {
            // 进程不存在了
            process.Status.Phase = PhaseStopped
            process.Status.PID = 0

            // 尝试重新关联进程
            if process.Spec.RestartPolicy == RestartPolicyAlways ||
              process.Spec.RestartPolicy == RestartPolicyOnFailure {
              go m.tryCheckProcess(process)
            }
          }
        }
      }
    }
  }
}

// matchProcess 根据多个特征匹配进程，优先使用用户定义的识别脚本
func (m *Manager) matchProcess(managedProc *ManagedProcess, proc *process.Process) bool {
  // 如果用户定义了识别脚本，优先使用它
  if managedProc.Spec.CheckAlive != nil {
    return m.matchProcessByScript(managedProc, proc)
  }

  // 否则使用传统的匹配方法
  return m.matchProcessByAttributes(managedProc, proc)
}

// matchProcessByScript 使用用户定义的脚本匹配进程
func (m *Manager) matchProcessByScript(managedProc *ManagedProcess, proc *process.Process) bool {
  // 获取进程的PID
  pid := int(proc.Pid)

  // 构建识别脚本命令，将PID作为参数传递
  cmd := exec.Command(managedProc.Spec.CheckAlive.Command,
    append(managedProc.Spec.CheckAlive.Args, strconv.Itoa(pid))...)

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
  timeout := managedProc.Spec.CheckAlive.Timeout
  if timeout == 0 {
    timeout = 5 * time.Second // 默认5秒超时
  }

  // 执行脚本并等待结果
  done := make(chan error, 1)
  go func() {
    done <- cmd.Run()
  }()

  // 等待命令完成或超时
  select {
  case err := <-done:
    // 脚本返回0表示匹配成功
    if err == nil {
      return true
    }
    // 检查退出码
    if exitErr, ok := err.(*exec.ExitError); ok {
      if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus() == 0
      }
    }
    return false
  case <-time.After(timeout):
    // 超时，强制终止命令
    if cmd.Process != nil {
      cmd.Process.Kill()
    }
    return false
  }
}

// matchProcessByAttributes 使用进程属性匹配进程
func (m *Manager) matchProcessByAttributes(managedProc *ManagedProcess, proc *process.Process) bool {
  // 匹配命令行参数
  cmdline, err := proc.Cmdline()
  if err == nil {
    for _, arg := range managedProc.Spec.Exec.Args {
      if !strings.Contains(cmdline, arg) {
        return false
      }
    }
  }

  // 匹配工作目录
  cwd, err := proc.Cwd()
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

// TryMatchProcessByScript 尝试使用用户定义的脚本匹配系统中的进程
func (m *Manager) TryMatchProcessByScript(managedProc *ManagedProcess) (*ManagedProcess, error) {
  if managedProc.Spec.CheckAlive == nil {
    return nil, fmt.Errorf("no identify script defined for process")
  }

  // 获取所有系统进程
  allProcesses, err := process.Processes()
  if err != nil {
    return nil, fmt.Errorf("failed to get all processes: %w", err)
  }

  // 遍历所有进程，使用脚本进行匹配
  for _, proc := range allProcesses {
    if m.matchProcessByScript(managedProc, proc) {
      // 找到匹配的进程，更新信息
      matchedProcess := *managedProc
      matchedProcess.Status.PID = int(proc.Pid)
      matchedProcess.Status.Phase = PhaseRunning

      // 获取并更新进程的其他信息
      if err := m.updateProcessInfoFromSystem(&matchedProcess); err != nil {
        // 更新失败不影响匹配结果，但记录警告
        fmt.Printf("Warning: failed to update process info: %v\n", err)
      }

      return &matchedProcess, nil
    }
  }

  return nil, fmt.Errorf("no matching process found")
}

// updateProcessInfoFromSystem 从系统中更新进程的详细信息
func (m *Manager) updateProcessInfoFromSystem(managedProc *ManagedProcess) error {
  // 获取系统进程对象
  proc, err := process.NewProcess(int32(managedProc.Status.PID))
  if err != nil {
    return err
  }

  // 更新进程的基本信息
  if err := scanner.fillBasicInfo(managedProc, proc); err != nil {
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
  for key, proc := range m.Processes {
    // 只处理应该运行但当前未运行的进程
    if proc.Status.Phase != PhaseRunning &&
      (proc.Spec.RestartPolicy == RestartPolicyAlways ||
        proc.Spec.RestartPolicy == RestartPolicyOnFailure) {

      // 尝试通过标识重新发现进程
      checked, err := m.tryCheckProcess(proc)
      if err == nil && checked {
        fmt.Printf("Successfully Checkd process %s\n", key)
      } else if err != nil {
        fmt.Printf("Failed to Check process %s: %v\n", key, err)
      }
    }
  }
}

// tryCheckProcess 尝试通过进程特征检查进程状态
func (m *Manager) tryCheckProcess(managedProc *ManagedProcess) (bool, error) {
  // 获取当前系统中的所有进程
  allProcesses, err := process.Processes()
  if err != nil {
    return false, fmt.Errorf("failed to get all processes: %w", err)
  }

  // 遍历所有进程，寻找匹配的进程
  for _, proc := range allProcesses {
    // 尝试匹配进程特征
    if m.matchProcess(managedProc, proc) {
      // 找到匹配的进程，更新状态
      managedProc.Status.PID = int(proc.Pid)
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

  return false, nil
}

package process

import (
  "errors"
  "fmt"
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

// DeleteProcess 删除一个纳管的进程
func (m *Manager) DeleteProcess(namespace, name string) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("进程 %s/%s 未被纳管", namespace, name)
  }

  // 如果进程正在运行，先停止它
  if process.Status.Phase == PhaseRunning {
    if err := m.StopProcess(namespace, name); err != nil {
      return fmt.Errorf("停止进程失败: %w", err)
    }
  }

  // 从管理列表中删除进程 - 修复使用正确的键
  delete(m.Processes, key)

  // 保存更新后的进程列表
  if err := m.SaveManagedProcesses(ProcessesFilePath); err != nil {
    fmt.Printf("Warning: failed to save managed processes: %v\n", err)
  }

  return nil
}

// StartProcess implements ProcManager interface to start a process
func (m *Manager) StartProcess(namespace, name string) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("process %s/%s is not managed", namespace, name)
  }

  if process.Status.Phase == PhaseRunning {
    return fmt.Errorf("process %s is already running", name)
  }

  // Set process status to pending
  process.Status.Phase = PhasePending

  // 构建命令 - 使用 CommandConfig
  cmd := exec.Command(process.Spec.Exec.Command, process.Spec.Exec.Args...)

  // Set working directory
  if process.Spec.WorkingDir != "" {
    cmd.Dir = process.Spec.WorkingDir
  } else {
    // Use current directory by default
    currentDir, err := os.Getwd()
    if err != nil {
      process.Status.Phase = PhaseFailed
      return err
    }
    cmd.Dir = currentDir
  }

  // Set environment variables - 从 EnvVar 数组构建
  cmd.Env = os.Environ()
  // 从配置中添加环境变量
  for _, envVar := range process.Spec.Env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
  }

  // 确保日志目录存在
  if process.Spec.Log.Dir != "" {
    if err := os.MkdirAll(process.Spec.Log.Dir, 0755); err != nil {
      process.Status.Phase = PhaseFailed
      return err
    }
  }

  // Capture standard output and error
  logDir := process.Spec.Log.Dir
  if logDir == "" {
    logDir = cmd.Dir
  }

  stdout, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s.stdout.log", name)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    process.Status.Phase = PhaseFailed
    return err
  }
  defer stdout.Close()

  stderr, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s.stderr.log", name)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    process.Status.Phase = PhaseFailed
    return err
  }
  defer stderr.Close()

  cmd.Stdout = stdout
  cmd.Stderr = stderr

  // Start process with timeout support
  done := make(chan error, 1)
  go func() {
    done <- cmd.Start()
  }()

  // 使用命令配置中的超时时间
  timeout := 30 * time.Second

  // Wait for start or timeout
  select {
  case err := <-done:
    if err != nil {
      process.Status.Phase = PhaseFailed
      return err
    }
  case <-time.After(timeout):
    // Timeout, kill the process
    process.Status.Phase = PhaseFailed
    if cmd.Process != nil {
      cmd.Process.Kill()
    }
    return fmt.Errorf("process start timed out after %v", timeout)
  }

  // Update process information
  process.Status.PID = cmd.Process.Pid
  process.Status.Phase = PhaseRunning
  now := time.Now()
  process.Status.StartTime = &now
  process.Status.RestartCount++

  // 异步等待进程退出，增加重启策略和最大重启次数的支持
  go func() {
    err := cmd.Wait()
    if err != nil {
      // 如果有退出码，记录下来
      var exitErr *exec.ExitError
      if errors.As(err, &exitErr) {
        if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
          if process.Status.LastTerminationInfo == nil {
            process.Status.LastTerminationInfo = &TerminationInfo{}
          }
          process.Status.LastTerminationInfo.ExitCode = status.ExitStatus()
        }
      }
    }

    process.Status.Phase = PhaseStopped
    process.Status.PID = 0

    // 应用重启策略
    shouldRestart := false
    switch process.Spec.RestartPolicy {
    case RestartPolicyAlways:
      shouldRestart = true
    case RestartPolicyOnFailure:
      shouldRestart = err != nil
    case RestartPolicyOnSuccess:
      shouldRestart = err == nil
    case RestartPolicyNever:
      shouldRestart = false
    }

    // 检查是否达到最大重启次数
    if shouldRestart {
      // 应用重启间隔
      restartInterval := process.Spec.RestartInterval
      if restartInterval == 0 {
        restartInterval = 5 * time.Second // 默认 5 秒
      }

      go func() {
        time.Sleep(restartInterval)
        m.StartProcess(namespace, name)
      }()
    }
  }()

  return nil
}

// StopProcess 实现ProcessManager接口，停止一个进程
func (m *Manager) StopProcess(namespace, name string) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("进程 %s/%s 未被纳管", namespace, name)
  }

  if process.Status.Phase == PhaseStopped {
    return fmt.Errorf("进程 %s 已经停止", name)
  }

  // 设置进程状态为停止中
  process.Status.Phase = PhaseStopping

  // 如果有自定义停止命令，使用它
  if process.Spec.Exec.StopCommand != nil {
    cmd := exec.Command(process.Spec.Exec.StopCommand.Command, process.Spec.Exec.StopCommand.Args...)
    cmd.Env = os.Environ()
    for _, envVar := range process.Spec.Env {
      cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
    }

    // 设置工作目录
    if process.Spec.WorkingDir != "" {
      cmd.Dir = process.Spec.WorkingDir
    }

    // 设置超时
    timeout := process.Spec.Exec.StopCommand.Timeout
    if timeout == 0 {
      timeout = 30 * time.Second
    }

    // 执行停止命令
    done := make(chan error, 1)
    go func() {
      done <- cmd.Run()
    }()

    // 等待命令完成或超时
    select {
    case err := <-done:
      if err != nil {
        // 停止命令失败，尝试强制终止
        return m.forceStopProcess(process)
      }
    case <-time.After(timeout):
      // 超时，尝试强制终止
      return m.forceStopProcess(process)
    }

    process.Status.Phase = PhaseStopped
    process.Status.PID = 0
    return nil
  } else {
    // 没有自定义停止命令，使用强制终止
    return m.forceStopProcess(process)
  }
}

// 新增一个辅助方法，用于强制终止进程
func (m *Manager) forceStopProcess(process *ManagedProcess) error {
  // 获取进程
  sysProcess, err := os.FindProcess(process.Status.PID)
  if err != nil {
    return err
  }

  // 在Windows系统上，我们使用Terminate方法
  // 在Unix/Linux系统上，我们先发送SIGTERM，然后在必要时发送SIGKILL

  // 发送终止信号
  if runtime.GOOS == "windows" {
    if err := sysProcess.Kill(); err != nil {
      return err
    }
  } else {
    // 先发送SIGTERM
    if err := sysProcess.Signal(syscall.SIGTERM); err != nil {
      // 如果失败，尝试直接杀死进程
      if err := sysProcess.Kill(); err != nil {
        return err
      }
    } else {
      // 等待进程退出，最多等待10秒
      timer := time.NewTimer(10 * time.Second)
      defer timer.Stop()

      // 检查进程是否还在运行
      for {
        select {
        case <-timer.C:
          // 超时，强制杀死进程
          if err := sysProcess.Kill(); err != nil {
            return err
          }
          return nil
        default:
          // 检查进程是否存在
          sysProcess.Signal(syscall.Signal(0)) // 发送0信号检查进程是否存在
          time.Sleep(100 * time.Millisecond)
        }
      }
    }
  }

  process.Status.Phase = PhaseStopped
  process.Status.PID = 0
  return nil
}

// RestartProcess 实现ProcessManager接口，重启一个进程
func (m *Manager) RestartProcess(namespace, name string) error {
  // 先停止进程
  if err := m.StopProcess(namespace, name); err != nil {
    // 如果进程已经停止，继续启动
    if !errors.Is(err, fmt.Errorf("进程 %s/%s 已经停止", namespace, name)) {
      return err
    }
  }

  // 等待一段时间
  time.Sleep(1 * time.Second)

  // 然后启动进程
  return m.StartProcess(namespace, name)
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
  if err := fillBasicInfo(managedProc, proc); err != nil {
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

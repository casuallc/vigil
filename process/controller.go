package process

import (
  "errors"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "syscall"
  "time"

  "github.com/casuallc/vigil/config"
)

// ManageProcess implements ProcManager interface to manage a process
func (m *Manager) ManageProcess(process ManagedProcess) error {
  if _, exists := m.processes[process.Name]; exists {
    return fmt.Errorf("process %s is already managed", process.Name)
  }

  // Generate ID for the process
  process.ID = fmt.Sprintf("%s-%d", process.Name, time.Now().UnixNano())

  // Set start time
  process.StartTime = time.Now()

  // Store the process
  m.processes[process.Name] = &process

  // Start monitoring the process
  go m.startMonitoring(process.Name)

  return nil
}

// StartProcess implements ProcManager interface to start a process
func (m *Manager) StartProcess(name string) error {
  process, exists := m.processes[name]
  if !exists {
    return fmt.Errorf("process %s is not managed", name)
  }

  if process.Status == StatusRunning {
    return fmt.Errorf("process %s is already running", name)
  }

  // Set process status to starting
  process.Status = StatusStarting

  // 构建命令 - 现在使用 CommandConfig
  cmd := exec.Command(process.StartCommand.Command, process.StartCommand.Args...)

  // Set working directory
  if process.WorkingDir != "" {
    cmd.Dir = process.WorkingDir
  } else {
    // Use current directory by default
    currentDir, err := os.Getwd()
    if err != nil {
      process.Status = StatusFailed
      return err
    }
    cmd.Dir = currentDir
  }

  // Set environment variables - 现在从 EnvVar 数组构建
  cmd.Env = os.Environ()
  // 从配置中添加环境变量
  for key, value := range process.Config.Env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
  }
  // 从 EnvVar 数组添加环境变量
  for _, envVar := range process.Env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
  }

  // 确保日志目录存在
  if process.LogDir != "" {
    if err := os.MkdirAll(process.LogDir, 0755); err != nil {
      process.Status = StatusFailed
      return err
    }
  }

  // Capture standard output and error
  logDir := process.LogDir
  if logDir == "" {
    logDir = cmd.Dir
  }

  stdout, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s.stdout.log", name)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    process.Status = StatusFailed
    return err
  }
  defer stdout.Close()

  stderr, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s.stderr.log", name)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    process.Status = StatusFailed
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
  timeout := process.StartCommand.Timeout

  // 如果没有设置超时时间，默认使用 30 秒
  if timeout == 0 {
    timeout = 30 * time.Second
  }

  // Wait for start or timeout
  select {
  case err := <-done:
    if err != nil {
      process.Status = StatusFailed
      return err
    }
  case <-time.After(timeout):
    // Timeout, kill the process
    process.Status = StatusFailed
    if cmd.Process != nil {
      cmd.Process.Kill()
    }
    return fmt.Errorf("process start timed out after %v", timeout)
  }

  // Update process information
  process.PID = cmd.Process.Pid
  process.Status = StatusRunning
  process.StartTime = time.Now()
  process.RestartCount++

  // 异步等待进程退出，增加重启策略和最大重启次数的支持
  go func() {
    err := cmd.Wait()
    if err != nil {
      // 如果有退出码，记录下来
      if exitErr, ok := err.(*exec.ExitError); ok {
        if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
          process.LastExitCode = status.ExitStatus()
        }
      }
    }

    process.Status = StatusStopped

    // 应用重启策略
    shouldRestart := false
    switch process.RestartPolicy {
    case RestartPolicyAlways:
      shouldRestart = true
    case RestartPolicyOnFailure:
      shouldRestart = err != nil
    case RestartPolicyOnSuccess:
      shouldRestart = err == nil
    case RestartPolicyNever:
      shouldRestart = false
    default:
      // 如果没有设置重启策略，回退到旧的 Restart 字段
      shouldRestart = process.Config.Restart
    }

    // 检查是否达到最大重启次数
    if shouldRestart && (process.MaxRestarts <= 0 || process.RestartCount < process.MaxRestarts) {
      // 应用重启间隔
      restartInterval := process.RestartInterval
      if restartInterval == 0 {
        restartInterval = 5 * time.Second // 默认 5 秒
      }

      go func() {
        time.Sleep(restartInterval)
        m.StartProcess(name)
      }()
    }
  }()

  return nil
}

// StopProcess 实现ProcessManager接口，停止一个进程
func (m *Manager) StopProcess(name string) error {
  process, exists := m.processes[name]
  if !exists {
    return fmt.Errorf("进程 %s 未被纳管", name)
  }

  if process.Status == StatusStopped {
    return fmt.Errorf("进程 %s 已经停止", name)
  }

  // 设置进程状态为停止中
  process.Status = StatusStopping

  // 如果有自定义停止命令，使用它
  if process.StopCommand != nil {
    cmd := exec.Command(process.StopCommand.Command, process.StopCommand.Args...)
    cmd.Env = os.Environ()
    for _, envVar := range process.Env {
      cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
    }

    // 设置工作目录
    if process.WorkingDir != "" {
      cmd.Dir = process.WorkingDir
    }

    // 设置超时
    timeout := process.StopCommand.Timeout
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

    process.Status = StatusStopped
    return nil
  } else {
    // 没有自定义停止命令，使用强制终止
    return m.forceStopProcess(process)
  }
}

// 新增一个辅助方法，用于强制终止进程
func (m *Manager) forceStopProcess(process *ManagedProcess) error {
  // 获取进程
  sysProcess, err := os.FindProcess(process.PID)
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

  process.Status = StatusStopped
  return nil
}

// RestartProcess 实现ProcessManager接口，重启一个进程
func (m *Manager) RestartProcess(name string) error {
  // 先停止进程
  if err := m.StopProcess(name); err != nil {
    // 如果进程已经停止，继续启动
    if !errors.Is(err, fmt.Errorf("进程 %s 已经停止", name)) {
      return err
    }
  }

  // 等待一段时间
  time.Sleep(1 * time.Second)

  // 然后启动进程
  return m.StartProcess(name)
}

// UpdateProcessConfig 实现ProcessManager接口，更新进程配置
func (m *Manager) UpdateProcessConfig(name string, config config.AppConfig) error {
  process, exists := m.processes[name]
  if !exists {
    return fmt.Errorf("进程 %s 未被纳管", name)
  }

  // 保存旧配置
  oldConfig := process.Config

  // 更新配置
  process.Config = config

  // 如果进程正在运行，需要重启来应用新配置
  if process.Status == StatusRunning {
    // 重启进程
    if err := m.RestartProcess(name); err != nil {
      // 如果重启失败，恢复旧配置
      process.Config = oldConfig
      return err
    }
  }

  return nil
}

// startMonitoring 开始监控进程
func (m *Manager) startMonitoring(name string) {
  process, exists := m.processes[name]
  if !exists {
    return
  }

  // 定期更新进程状态和资源使用情况
  ticker := time.NewTicker(5 * time.Second)
  defer ticker.Stop()

  for {
    select {
    case <-ticker.C:
      if process.Status == StatusRunning {
        // 更新资源使用情况
        stats, err := m.MonitorProcess(name)
        if err == nil {
          process.Stats = stats
        }
      }
    default:
      // 检查进程是否还存在
      if process.Status == StatusRunning {
        _, err := os.FindProcess(process.PID)
        if err != nil {
          // 进程不存在了
          process.Status = StatusStopped
        }
      }
    }
  }
}

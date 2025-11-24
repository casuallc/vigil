package proc

import (
  "errors"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "syscall"
  "time"
)

// CreateProcess implements ProcManager interface to manage a proc
func (m *Manager) CreateProcess(process ManagedProcess) error {
  // 如果未指定namespace，使用默认namespace
  if process.Metadata.Namespace == "" {
    process.Metadata.Namespace = "default"
  }

  key := fmt.Sprintf("%s/%s", process.Metadata.Namespace, process.Metadata.Name)
  if _, exists := m.Processes[key]; exists {
    return fmt.Errorf("proc %s/%s is already managed", process.Metadata.Namespace, process.Metadata.Name)
  }

  // Generate ID for the proc
  process.Metadata.ID = fmt.Sprintf("%s-%d", process.Metadata.Name, time.Now().UnixNano())

  // Store the proc
  m.Processes[key] = &process

  // 保存进程信息
  if err := m.SaveManagedProcesses(ProcessesFilePath); err != nil {
    fmt.Printf("Warning: failed to save managed processes: %v\n", err)
  }

  // Start monitoring the proc
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

// StartProcess implements ProcManager interface to start a proc
func (m *Manager) StartProcess(namespace, name string) error {
  key := fmt.Sprintf("%s/%s", namespace, name)
  process, exists := m.Processes[key]
  if !exists {
    return fmt.Errorf("proc %s/%s is not managed", namespace, name)
  }

  if process.Status.Phase == PhaseRunning {
    return fmt.Errorf("proc %s is already running", name)
  }

  // Set proc status to pending
  process.Status.Phase = PhasePending

  // 在 Linux 下应用目录挂载（bind/tmpfs/named）
  if err := applyMounts(process.Spec.Mounts); err != nil {
    process.Status.Phase = PhaseFailed
    return fmt.Errorf("failed to apply mounts: %w", err)
  }

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
    // 启动失败时清理挂载
    cleanupMounts(process.Spec.Mounts)
    return err
  }
  defer stdout.Close()

  stderr, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s.stderr.log", name)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    process.Status.Phase = PhaseFailed
    // 启动失败时清理挂载
    cleanupMounts(process.Spec.Mounts)
    return err
  }
  defer stderr.Close()

  cmd.Stdout = stdout
  cmd.Stderr = stderr

  // Start proc with timeout support
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
      // 启动失败时清理挂载
      cleanupMounts(process.Spec.Mounts)
      return err
    }
  case <-time.After(timeout):
    // Timeout, kill the proc
    process.Status.Phase = PhaseFailed
    if cmd.Process != nil {
      cmd.Process.Kill()
    }
    // 超时也清理挂载
    cleanupMounts(process.Spec.Mounts)
    return fmt.Errorf("proc start timed out after %v", timeout)
  }

  // Update proc information
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

    // 进程退出后清理挂载（Linux）
    cleanupMounts(process.Spec.Mounts)

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
        if err := m.forceStopProcess(process); err != nil {
          return err
        }
      }
    case <-time.After(timeout):
      // 超时，尝试强制终止
      if err := m.forceStopProcess(process); err != nil {
        return err
      }
    }

    process.Status.Phase = PhaseStopped
    process.Status.PID = 0

    // 停止后清理挂载
    cleanupMounts(process.Spec.Mounts)
    return nil
  } else {
    // 没有自定义停止命令，使用强制终止
    if err := m.forceStopProcess(process); err != nil {
      return err
    }
    // 停止后清理挂载
    cleanupMounts(process.Spec.Mounts)
    return nil
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

// Linux 下应用挂载（bind/tmpfs/named）
func applyMounts(mounts []Mount) error {
  if len(mounts) == 0 || runtime.GOOS != "linux" {
    return nil
  }

  for _, m := range mounts {
    // 解析类型，默认 bind
    t := m.Type
    if t == "" {
      t = "bind"
    }
    if m.Target == "" {
      return fmt.Errorf("invalid mount: target is required")
    }

    // 目标目录准备
    if m.CreateTarget {
      if err := os.MkdirAll(m.Target, 0755); err != nil {
        return fmt.Errorf("failed to create target dir %s: %w", m.Target, err)
      }
      // 权限/所有者
      if m.Mode != "" {
        if perm, err := ParseFileMode(m.Mode); err == nil {
          _ = os.Chmod(m.Target, perm)
        }
      }
      if m.UID != 0 || m.GID != 0 {
        _ = os.Chown(m.Target, m.UID, m.GID)
      }
    }

    switch t {
    case "bind":
      if m.Source == "" {
        return fmt.Errorf("bind mount requires source")
      }
      // 递归绑定
      args := []string{"--bind"}
      if m.Recursive {
        args = []string{"--rbind"}
      }
      cmd := exec.Command("mount", append(args, m.Source, m.Target)...)
      if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to bind mount %s -> %s: %w", m.Source, m.Target, err)
      }
      // 只读 remount
      if m.ReadOnly {
        ro := exec.Command("mount", "-o", "remount,ro", m.Target)
        if err := ro.Run(); err != nil {
          return fmt.Errorf("failed to remount ro for %s: %w", m.Target, err)
        }
      }
      // 传播选项
      if m.Propagation != "" {
        make := exec.Command("mount", "--make-"+m.Propagation, m.Target)
        _ = make.Run() // 非致命
      }

    case "tmpfs":
      // tmpfs: mount -t tmpfs -o size=NNM target
      opts := "size=64M"
      if m.TmpfsSizeMB > 0 {
        opts = fmt.Sprintf("size=%dM", m.TmpfsSizeMB)
      }
      cmd := exec.Command("mount", "-t", "tmpfs", "-o", opts, "tmpfs", m.Target)
      if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to mount tmpfs on %s: %w", m.Target, err)
      }

    case "named":
      if m.Name == "" {
        return fmt.Errorf("named volume requires name")
      }
      // 简化的命名卷目录：./volumes/<name>
      volDir := filepath.Join(".", "volumes", m.Name)
      if err := os.MkdirAll(volDir, 0755); err != nil {
        return fmt.Errorf("failed to create volume dir %s: %w", volDir, err)
      }
      cmd := exec.Command("mount", "--bind", volDir, m.Target)
      if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to bind named volume %s -> %s: %w", volDir, m.Target, err)
      }
      if m.ReadOnly {
        ro := exec.Command("mount", "-o", "remount,ro", m.Target)
        _ = ro.Run()
      }

    default:
      return fmt.Errorf("unsupported mount type: %s", t)
    }
  }

  return nil
}

// Linux 下卸载挂载（使用懒卸载以尽量避免繁忙状态）
func cleanupMounts(mounts []Mount) {
  if len(mounts) == 0 || runtime.GOOS != "linux" {
    return
  }
  for _, m := range mounts {
    if m.Target == "" {
      continue
    }
    // umount -l target
    _ = exec.Command("umount", "-l", m.Target).Run()
  }
}

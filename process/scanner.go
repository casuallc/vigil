package process

import (
  "bufio"
  "bytes"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "os/user"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"
  "syscall"
  "time"
)

// 定义扫描类型常量
const (
  // ScriptPrefix 特殊前缀用于识别脚本
  ScriptPrefix = "script://"
  // FileScriptPrefix 特殊前缀用于识别文件脚本
  FileScriptPrefix = "file://"
)

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

// scanWithScript scans processes using a custom script
func (m *Manager) scanWithScript(script string) ([]ManagedProcess, error) {
  // 创建一个临时脚本文件
  // 在实际实现中，应该使用更安全的方式处理临时文件
  // 这里为了简化示例，直接执行脚本内容

  // 执行脚本
  cmd := exec.Command("sh", "-c", script)
  var output bytes.Buffer
  cmd.Stdout = &output
  cmd.Stderr = &output

  err := cmd.Run()
  if err != nil {
    return nil, fmt.Errorf("script execution failed: %v, output: %s", err, output.String())
  }

  // 解析脚本输出，期望每行包含一个PID
  var processes []ManagedProcess
  lines := strings.Split(output.String(), "\n")

  for _, line := range lines {
    line = strings.TrimSpace(line)
    if line == "" {
      continue
    }

    // 尝试将每行解析为PID
    pid, err := strconv.Atoi(line)
    if err != nil {
      // 如果不是纯PID，则忽略该行或记录警告
      continue
    }

    // 通过PID获取进程信息
    process, err := m.getProcessByPID(pid)
    if err != nil {
      // 如果无法获取进程信息，则忽略该PID或记录警告
      continue
    }

    processes = append(processes, *process)
  }

  return processes, nil
}

// scanUnixProcesses scans processes on Unix/Linux/macOS systems
func (m *Manager) scanUnixProcesses(query string) ([]ManagedProcess, error) {
  dirs, err := os.ReadDir("/proc")
  if err != nil {
    return nil, fmt.Errorf("failed to read /proc: %v", err)
  }

  var processes []ManagedProcess
  // Compile regex for query matching
  queryRegex, err := regexp.Compile(query)
  if err != nil {
    // If not a valid regex, use as plain string match
    queryRegex, _ = regexp.Compile(regexp.QuoteMeta(query))
  }

  for _, dir := range dirs {
    if !dir.IsDir() {
      continue
    }
    // Parse PID
    pid, err := strconv.Atoi(dir.Name())
    if err != nil {
      continue
    }
    cmdlinePath := filepath.Join("/proc", dir.Name(), "cmdline")
    content, err := os.ReadFile(cmdlinePath)
    if err != nil {
      continue
    }
    cmdLine := common.ParseToString(content, 0)
    if !queryRegex.MatchString(cmdLine) {
      continue
    }

    // 通过PID获取进程信息
    managedProcess, err := m.getProcessByPID(pid)
    if err != nil {
      // 如果无法获取进程信息，则忽略该PID或记录警告
      continue
    }
    processes = append(processes, *managedProcess)
  }
  return processes, nil
}

// getProcessByPID 获取指定PID的进程详细信息
func (m *Manager) getProcessByPID(pid int) (*ManagedProcess, error) {
  // 创建基础结构体
  manageProcess := &ManagedProcess{
    Spec: Spec{},
    Status: Status{
      PID: pid,
    },
  }

  // 获取 gopsutil 的进程对象
  proc, err := process.NewProcess(int32(pid))
  if err != nil {
    return nil, fmt.Errorf("failed to create process object for PID %d: %w", pid, err)
  }

  // 填充基本信息
  if err := fillBasicInfo(manageProcess, proc); err != nil {
    return nil, fmt.Errorf("failed to fill basic info: %w", err)
  }

  // 填充命令和参数信息
  if err := fillCommandInfo(manageProcess, proc); err != nil {
    return nil, fmt.Errorf("failed to fill command info: %w", err)
  }

  // 填充环境变量
  if err := fillEnvironmentInfo(manageProcess, proc); err != nil {
    // 环境变量可能因为权限问题无法读取，这里不作为致命错误
    fmt.Printf("Warning: failed to read environment variables for PID %d: %v\n", pid, err)
  }

  // 填充工作目录
  if err := fillWorkingDir(manageProcess, proc); err != nil {
    // 工作目录可能因为权限问题无法读取
    fmt.Printf("Warning: failed to read working directory for PID %d: %v\n", pid, err)
  }

  // 填充用户和用户组信息
  if err := fillUserGroupInfo(manageProcess, proc); err != nil {
    fmt.Printf("Warning: failed to read user/group info for PID %d: %v\n", pid, err)
  }

  // 填充资源统计信息
  if err := fillResourceStats(manageProcess, proc); err != nil {
    fmt.Printf("Warning: failed to read resource stats for PID %d: %v\n", pid, err)
  }

  // 填充监听端口信息
  if err := fillListeningPorts(manageProcess, proc); err != nil {
    fmt.Printf("Warning: failed to read listening ports for PID %d: %v\n", pid, err)
  }

  // 设置状态
  manageProcess.Status.Phase = PhaseRunning

  return manageProcess, nil
}

// fillBasicInfo 填充基本信息
func fillBasicInfo(mp *ManagedProcess, proc *process.Process) error {
  // 进程名称
  name, err := proc.Name()
  if err != nil {
    return fmt.Errorf("failed to get process name: %w", err)
  }
  mp.Metadata.Name = name

  // 启动时间
  createTimeMs, err := proc.CreateTime()
  if err != nil {
    return fmt.Errorf("failed to get create time: %w", err)
  }
  // 转换为 time.Time
  createTime := time.UnixMilli(createTimeMs)
  mp.Status.StartTime = &createTime

  // 退出码（对于正在运行的进程，这个通常是0或未设置）
  // gopsutil 不直接提供最后退出码，这里保持默认值0

  return nil
}

// fillCommandInfo 填充命令信息
func fillCommandInfo(mp *ManagedProcess, proc *process.Process) error {
  // 可执行路径
  exe, err := proc.Exe()
  if err != nil {
    return fmt.Errorf("failed to get executable path: %w", err)
  }

  // 命令行参数
  cmdline, err := proc.CmdlineSlice()
  if err != nil {
    return fmt.Errorf("failed to get command line: %w", err)
  }

  if len(cmdline) > 0 {
    mp.Spec.Exec.StopCommand = &CommandConfig{}
    mp.Spec.Exec.Command = exe
    if len(cmdline) > 1 {
      mp.Spec.Exec.Args = cmdline[1:]
    }
  }

  return nil
}

// fillEnvironmentInfo 填充环境变量信息
func fillEnvironmentInfo(mp *ManagedProcess, proc *process.Process) error {
  envVars, err := proc.Environ()
  if err != nil {
    return fmt.Errorf("failed to get environment variables: %w", err)
  }

  mp.Spec.Env = make([]EnvVar, 0, len(envVars))
  for _, envVar := range envVars {
    parts := strings.SplitN(envVar, "=", 2)
    if len(parts) == 2 {
      mp.Spec.Env = append(mp.Spec.Env, EnvVar{
        Name:  parts[0],
        Value: parts[1],
      })
    }
  }

  return nil
}

// fillWorkingDir 填充工作目录
func fillWorkingDir(mp *ManagedProcess, proc *process.Process) error {
  cwd, err := proc.Cwd()
  if err != nil {
    return fmt.Errorf("failed to get working directory: %w", err)
  }
  mp.Spec.WorkingDir = cwd
  return nil
}

// fillUserGroupInfo 填充用户和用户组信息
func fillUserGroupInfo(mp *ManagedProcess, proc *process.Process) error {
  // 用户ID
  uids, err := proc.Uids()
  if err != nil {
    return fmt.Errorf("failed to get UIDs: %w", err)
  }
  if len(uids) > 0 {
    uid := uids[0]
    if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
      mp.Spec.User = u.Username
    } else {
      mp.Spec.User = strconv.FormatUint(uint64(uid), 10)
    }
  }

  // 用户组ID
  gids, err := proc.Gids()
  if err != nil {
    return fmt.Errorf("failed to get GIDs: %w", err)
  }
  if len(gids) > 0 {
    gid := gids[0]
    // gopsutil 可能不直接提供组名查找，这里只存储GID
    mp.Spec.UserGroup = strconv.FormatUint(uint64(gid), 10)
  }

  return nil
}

// fillResourceStats 填充资源统计信息
func fillResourceStats(mp *ManagedProcess, proc *process.Process) error {
  // CPU 使用率
  cpuPercent, err := proc.CPUPercent()
  if err != nil {
    return fmt.Errorf("failed to get CPU percent: %w", err)
  }
  mp.Status.ResourceStats = &ResourceStats{}
  mp.Status.ResourceStats.CPUUsage = cpuPercent

  // 内存使用量
  memInfo, err := proc.MemoryInfo()
  if err != nil {
    return fmt.Errorf("failed to get memory info: %w", err)
  }
  mp.Status.ResourceStats.MemoryUsage = memInfo.RSS // Resident Set Size

  // 磁盘IO
  ioCounters, err := proc.IOCounters()
  if err != nil {
    // IO counters 可能不可用，不作为致命错误
    mp.Status.ResourceStats.DiskIO = 0
  } else {
    mp.Status.ResourceStats.DiskIO = ioCounters.ReadBytes + ioCounters.WriteBytes
  }

  // 网络IO - gopsutil 不直接提供进程级别的网络IO
  // 这需要更复杂的实现，这里暂时设为0
  mp.Status.ResourceStats.NetworkIO = 0

  return nil
}

// fillListeningPorts 填充监听端口信息
func fillListeningPorts(mp *ManagedProcess, proc *process.Process) error {
  connections, err := proc.Connections()
  if err != nil {
    return fmt.Errorf("failed to get connections: %w", err)
  }

  var listeningPorts []PortInfo
  for _, conn := range connections {
    if conn.Status == "LISTEN" {
      portInfo := PortInfo{
        Port:     int(conn.Laddr.Port),
        Protocol: socketTypeToProtocol(conn.Type),
        Address:  conn.Laddr.IP,
      }
      listeningPorts = append(listeningPorts, portInfo)
    }
  }

  mp.Status.ResourceStats.ListeningPorts = listeningPorts
  return nil
}

// 获取协议内容
func socketTypeToProtocol(t uint32) string {
  switch t {
  case syscall.SOCK_STREAM:
    return "TCP"
  case syscall.SOCK_DGRAM:
    return "UDP"
  default:
    return "UNKNOWN"
  }
}

// Helper function to get process status from /proc (alternative method)
func getProcessStatusFromProc(pid int) (Phase, error) {
  statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
  file, err := os.Open(statusPath)
  if err != nil {
    return PhaseUnknown, err
  }
  defer file.Close()

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "State:") {
      fields := strings.Fields(line)
      if len(fields) > 1 {
        state := fields[1]
        switch state {
        case "R", "S", "D": // Running, Sleeping, Uninterruptible sleep
          return PhaseRunning, nil
        case "Z": // Zombie
          return PhaseFailed, nil
        case "T": // Stopped
          return PhaseStopped, nil
        default:
          return PhaseUnknown, nil
        }
      }
    }
  }

  return PhaseUnknown, scanner.Err()
}

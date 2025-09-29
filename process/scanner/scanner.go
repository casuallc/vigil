package scanner

import (
  "bufio"
  "bytes"
  "fmt"
  "github.com/casuallc/vigil/common"
  process2 "github.com/casuallc/vigil/process"
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
func (m *process.Manager) ScanProcesses(query string) ([]process.ManagedProcess, error) {
  // 根据查询类型选择不同的扫描方法
  if strings.HasPrefix(query, scanner.ScriptPrefix) {
    // 直接执行内联脚本
    scriptContent := strings.TrimPrefix(query, scanner.ScriptPrefix)
    return m.scanWithScript(scriptContent)
  } else if strings.HasPrefix(query, scanner.FileScriptPrefix) {
    // 从文件加载脚本并执行
    scriptPath := strings.TrimPrefix(query, scanner.FileScriptPrefix)
    content, err := os.ReadFile(scriptPath)
    if err != nil {
      return nil, fmt.Errorf("failed to read script file: %v", err)
    }
    return m.scanWithScript(string(content))
  } else {
    // 使用标准的Unix进程扫描
    return m.scanUnixProcesses(query)
  }



// fillBasicInfo 填充基本信息
func fillBasicInfo(mp *process2.ManagedProcess, proc *process.Process) error {
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
  createTime := time.UnixMilli(createTimeMs)
  mp.Status.StartTime = &createTime

  // 重启策略
  mp.Spec.RestartPolicy = process2.RestartPolicyOnFailure
  mp.Spec.RestartInterval = 5

  // 退出码（对于正在运行的进程，这个通常是0或未设置）
  // gopsutil 不直接提供最后退出码，这里保持默认值0

  return nil
}

// fillCommandInfo 填充命令信息
func fillCommandInfo(mp *process2.ManagedProcess, proc *process.Process) error {
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
    mp.Spec.Exec.StopCommand = &process2.CommandConfig{}
    mp.Spec.Exec.Command = exe
    if len(cmdline) > 1 {
      mp.Spec.Exec.Args = cmdline[1:]
    }
  }

  return nil
}

// fillEnvironmentInfo 填充环境变量信息
func fillEnvironmentInfo(mp *process2.ManagedProcess, proc *process.Process) error {
  envVars, err := proc.Environ()
  if err != nil {
    return fmt.Errorf("failed to get environment variables: %w", err)
  }

  mp.Spec.Env = make([]process2.EnvVar, 0, len(envVars))
  for _, envVar := range envVars {
    parts := strings.SplitN(envVar, "=", 2)
    if len(parts) == 2 {
      mp.Spec.Env = append(mp.Spec.Env, process2.EnvVar{
        Name:  parts[0],
        Value: parts[1],
      })
    }
  }

  return nil
}

// fillWorkingDir 填充工作目录
func fillWorkingDir(mp *process2.ManagedProcess, proc *process.Process) error {
  cwd, err := proc.Cwd()
  if err != nil {
    return fmt.Errorf("failed to get working directory: %w", err)
  }
  mp.Spec.WorkingDir = cwd
  return nil
}

// fillUserGroupInfo 填充用户和用户组信息
func fillUserGroupInfo(mp *process2.ManagedProcess, proc *process.Process) error {
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
func fillResourceStats(mp *process2.ManagedProcess, proc *process.Process) error {
  // CPU 使用率
  cpuPercent, err := proc.CPUPercent()
  if err != nil {
    return fmt.Errorf("failed to get CPU percent: %w", err)
  }
  mp.Status.ResourceStats = &process2.ResourceStats{}
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
func fillListeningPorts(mp *process2.ManagedProcess, proc *process.Process) error {
  connections, err := proc.Connections()
  if err != nil {
    return fmt.Errorf("failed to get connections: %w", err)
  }

  var listeningPorts []process2.PortInfo
  for _, conn := range connections {
    if conn.Status == "LISTEN" {
      portInfo := process2.PortInfo{
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
func getProcessStatusFromProc(pid int) (process2.Phase, error) {
  statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
  file, err := os.Open(statusPath)
  if err != nil {
    return process2.PhaseUnknown, err
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
          return process2.PhaseRunning, nil
        case "Z": // Zombie
          return process2.PhaseFailed, nil
        case "T": // Stopped
          return process2.PhaseStopped, nil
        default:
          return process2.PhaseUnknown, nil
        }
      }
    }
  }

  return process2.PhaseUnknown, scanner.Err()
}

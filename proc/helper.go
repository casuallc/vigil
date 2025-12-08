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
  "fmt"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "os/user"
  "strconv"
  "strings"
  "syscall"
  "time"
)

// FormatCPUUsage 将CPU使用率格式化为百分比字符串
func FormatCPUUsage(usage float64) string {
  return fmt.Sprintf("%.1f%%", usage)
}

// FormatBytes 将字节数格式化为人类可读的字符串（如 1K, 1M, 1G）
func FormatBytes(bytes uint64) string {
  const (
    B  = 1
    KB = 1024 * B
    MB = 1024 * KB
    GB = 1024 * MB
    TB = 1024 * GB
  )

  switch {
  case bytes >= TB:
    return fmt.Sprintf("%.2fTiB", float64(bytes)/TB)
  case bytes >= GB:
    return fmt.Sprintf("%.2fGiB", float64(bytes)/GB)
  case bytes >= MB:
    return fmt.Sprintf("%.2fMiB", float64(bytes)/MB)
  case bytes >= KB:
    return fmt.Sprintf("%.2fKiB", float64(bytes)/KB)
  default:
    return fmt.Sprintf("%dB", bytes)
  }
}

// ParseBytes 将人类可读的字节字符串（如 1K, 1M, 1G）解析为字节数
func ParseBytes(s string) (uint64, error) {
  var (
    value float64
    unit  string
  )

  // 解析数字和单位
  n, err := fmt.Sscanf(s, "%f%s", &value, &unit)
  if err != nil || (n != 1 && n != 2) {
    return 0, fmt.Errorf("invalid format: %s", s)
  }

  // 默认单位是字节
  multiplier := uint64(1)

  // 根据单位设置乘数
  switch strings.ToUpper(unit) {
  case "B":
    multiplier = 1
  case "KB", "K":
    multiplier = 1024
  case "MB", "M":
    multiplier = 1024 * 1024
  case "GB", "G":
    multiplier = 1024 * 1024 * 1024
  case "TB", "T":
    multiplier = 1024 * 1024 * 1024 * 1024
  default:
    if n == 2 {
      return 0, fmt.Errorf("unknown unit: %s", unit)
    }
  }

  return uint64(value) * multiplier, nil
}

// FillBasicInfo 填充基本信息
func FillBasicInfo(mp *ManagedProcess, process *process.Process) error {
  // 进程名称
  name, err := process.Name()
  if err != nil {
    return fmt.Errorf("failed to get process name: %w", err)
  }
  mp.Metadata.Name = name

  // 启动时间
  createTimeMs, err := process.CreateTime()
  if err != nil {
    return fmt.Errorf("failed to get create time: %w", err)
  }
  // 转换为 time.Time
  createTime := time.UnixMilli(createTimeMs)
  mp.Status.StartTime = &createTime

  mp.Spec.RestartPolicy = RestartPolicyAlways
  mp.Spec.RestartInterval = 10 * time.Second

  // 退出码（对于正在运行的进程，这个通常是0或未设置）
  // gopsutil 不直接提供最后退出码，这里保持默认值0

  return nil
}

// FillCommandInfo 填充命令信息
func FillCommandInfo(mp *ManagedProcess, process *process.Process) error {
  // 可执行路径
  exe, err := process.Exe()
  if err != nil {
    return fmt.Errorf("failed to get executable path: %w", err)
  }

  // 命令行参数
  cmdline, err := process.CmdlineSlice()
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

// FillEnvironmentInfo 填充环境变量信息
func FillEnvironmentInfo(mp *ManagedProcess, process *process.Process) error {
  envVars, err := process.Environ()
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

// FillWorkingDir 填充工作目录
func FillWorkingDir(mp *ManagedProcess, process *process.Process) error {
  cwd, err := process.Cwd()
  if err != nil {
    return fmt.Errorf("failed to get working directory: %w", err)
  }
  mp.Spec.WorkingDir = cwd
  return nil
}

// FillUserGroupInfo 填充用户和用户组信息
func FillUserGroupInfo(mp *ManagedProcess, process *process.Process) error {
  // 用户ID
  uids, err := process.Uids()
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
  gids, err := process.Gids()
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

// FillResourceStats 填充资源统计信息
func FillResourceStats(mp *ManagedProcess, process *process.Process) error {
  stats, err := GetUnixProcessResourceUsage(int(process.Pid))
  if err != nil {
    return err
  }

  mp.Status.ResourceStats = stats
  return nil
}

// FillListeningPorts 填充监听端口信息
func FillListeningPorts(mp *ManagedProcess, process *process.Process) error {
  connections, err := process.Connections()
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

// GetProcessCpuAndMemory 获取进程的CPU和内存使用情况
func GetProcessCpuAndMemory(pid int) (float64, uint64, error) {
  // 使用ps命令获取进程的CPU和内存使用情况
  cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu,rss")
  output, err := cmd.CombinedOutput()
  if err != nil {
    return 0, 0, fmt.Errorf("failed to get proc CPU and memory: %v, output: %s", err, string(output))
  }

  // 解析输出
  lines := strings.Split(string(output), "\n")
  if len(lines) < 2 {
    return 0, 0, fmt.Errorf("invalid proc stats output")
  }

  // 解析第二行（第一行是表头）
  line := strings.TrimSpace(lines[1])
  fields := strings.Fields(line)
  if len(fields) < 2 {
    return 0, 0, fmt.Errorf("invalid proc stats format")
  }

  // 提取CPU使用率（百分比）
  cpuUsage, err := strconv.ParseFloat(fields[0], 64)
  if err != nil {
    return 0, 0, fmt.Errorf("failed to parse CPU usage: %v", err)
  }

  // 提取内存使用量（KB）并转换为字节
  rss, err := strconv.ParseUint(fields[1], 10, 64)
  if err != nil {
    return 0, 0, fmt.Errorf("failed to parse memory usage: %v", err)
  }

  // 转换为字节
  memoryUsage := rss * 1024

  return cpuUsage, memoryUsage, nil
}

// ParseFileMode 解析权限字符串（如 "0755"）
func ParseFileMode(s string) (os.FileMode, error) {
  if s == "" {
    return 0, fmt.Errorf("empty mode")
  }
  // 支持 0755 或 755 两种格式
  var v uint64
  var err error
  if s[0] == '0' {
    v, err = strconv.ParseUint(s, 8, 32)
  } else {
    v, err = strconv.ParseUint(s, 10, 32)
  }
  if err != nil {
    return 0, err
  }
  return os.FileMode(v), nil
}

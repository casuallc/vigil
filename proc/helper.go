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
  "github.com/casuallc/vigil/models"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "os/user"
  "strconv"
  "strings"
  "syscall"
  "time"
)

// FillBasicInfo 填充基本信息
func FillBasicInfo(mp *models.ManagedProcess, process *process.Process) error {
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

  mp.Spec.RestartPolicy = models.RestartPolicyAlways
  mp.Spec.RestartInterval = 10 * time.Second

  // 退出码（对于正在运行的进程，这个通常是0或未设置）
  // gopsutil 不直接提供最后退出码，这里保持默认值0

  return nil
}

// FillCommandInfo 填充命令信息
func FillCommandInfo(mp *models.ManagedProcess, process *process.Process) error {
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
    mp.Spec.Exec.StopCommand = &models.CommandConfig{}
    mp.Spec.Exec.Command = exe
    if len(cmdline) > 1 {
      mp.Spec.Exec.Args = cmdline[1:]
    }
  }

  return nil
}

// FillEnvironmentInfo 填充环境变量信息
func FillEnvironmentInfo(mp *models.ManagedProcess, process *process.Process) error {
  envVars, err := process.Environ()
  if err != nil {
    return fmt.Errorf("failed to get environment variables: %w", err)
  }

  mp.Spec.Env = make([]models.EnvVar, 0, len(envVars))
  for _, envVar := range envVars {
    parts := strings.SplitN(envVar, "=", 2)
    if len(parts) == 2 {
      mp.Spec.Env = append(mp.Spec.Env, models.EnvVar{
        Name:  parts[0],
        Value: parts[1],
      })
    }
  }

  return nil
}

// FillWorkingDir 填充工作目录
func FillWorkingDir(mp *models.ManagedProcess, process *process.Process) error {
  cwd, err := process.Cwd()
  if err != nil {
    return fmt.Errorf("failed to get working directory: %w", err)
  }
  mp.Spec.WorkingDir = cwd
  return nil
}

// FillUserGroupInfo 填充用户和用户组信息
func FillUserGroupInfo(mp *models.ManagedProcess, process *process.Process) error {
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
func FillResourceStats(mp *models.ManagedProcess, process *process.Process) error {
  stats, err := GetUnixProcessResourceUsage(int(process.Pid))
  if err != nil {
    return err
  }

  mp.Status.ResourceStats = stats
  return nil
}

// FillListeningPorts 填充监听端口信息
func FillListeningPorts(mp *models.ManagedProcess, process *process.Process) error {
  connections, err := process.Connections()
  if err != nil {
    return fmt.Errorf("failed to get connections: %w", err)
  }

  var listeningPorts []models.PortInfo
  for _, conn := range connections {
    if conn.Status == "LISTEN" {
      portInfo := models.PortInfo{
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

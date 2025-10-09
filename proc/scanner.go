package proc

import (
  "bufio"
  "bytes"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"
)

// 定义扫描类型常量
const (
  // ScriptPrefix 特殊前缀用于识别脚本
  ScriptPrefix = "script://"
  // FileScriptPrefix 特殊前缀用于识别文件脚本
  FileScriptPrefix = "file://"
)

// ScanWithScript scans processes using a custom script
func ScanWithScript(script string) ([]ManagedProcess, error) {
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
    process, err := GetProcessByPID(pid)
    if err != nil {
      // 如果无法获取进程信息，则忽略该PID或记录警告
      continue
    }

    processes = append(processes, *process)
  }

  return processes, nil
}

// ScanUnixProcesses scans processes on Unix/Linux/macOS systems
func ScanUnixProcesses(query string) ([]ManagedProcess, error) {
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
    managedProc, err := GetProcessByPID(pid)
    if err != nil {
      // 如果无法获取进程信息，则忽略该PID或记录警告
      continue
    }
    processes = append(processes, *managedProc)
  }
  return processes, nil
}

// GetProcessByPID 获取指定PID的进程详细信息
func GetProcessByPID(pid int) (*ManagedProcess, error) {
  // 创建基础结构体
  manageProcess := &ManagedProcess{
    Spec: Spec{},
    Status: Status{
      PID: pid,
    },
  }

  // 获取 gopsutil 的进程对象
  sysProcess, err := process.NewProcess(int32(pid))
  if err != nil {
    return nil, fmt.Errorf("failed to create proc object for PID %d: %w", pid, err)
  }

  // 填充基本信息
  if err := FillBasicInfo(manageProcess, sysProcess); err != nil {
    return nil, fmt.Errorf("failed to fill basic info: %w", err)
  }

  // 填充命令和参数信息
  if err := FillCommandInfo(manageProcess, sysProcess); err != nil {
    return nil, fmt.Errorf("failed to fill command info: %w", err)
  }

  // 填充环境变量
  if err := FillEnvironmentInfo(manageProcess, sysProcess); err != nil {
    // 环境变量可能因为权限问题无法读取，这里不作为致命错误
    fmt.Printf("Warning: failed to read environment variables for PID %d: %v\n", pid, err)
  }

  // 填充工作目录
  if err := FillWorkingDir(manageProcess, sysProcess); err != nil {
    // 工作目录可能因为权限问题无法读取
    fmt.Printf("Warning: failed to read working directory for PID %d: %v\n", pid, err)
  }

  // 填充用户和用户组信息
  if err := FillUserGroupInfo(manageProcess, sysProcess); err != nil {
    fmt.Printf("Warning: failed to read user/group info for PID %d: %v\n", pid, err)
  }

  // 填充资源统计信息
  if err := FillResourceStats(manageProcess, sysProcess); err != nil {
    fmt.Printf("Warning: failed to read resource stats for PID %d: %v\n", pid, err)
  }

  // 填充监听端口信息
  if err := FillListeningPorts(manageProcess, sysProcess); err != nil {
    fmt.Printf("Warning: failed to read listening ports for PID %d: %v\n", pid, err)
  }

  // 设置状态
  manageProcess.Status.Phase = PhaseRunning

  return manageProcess, nil
}

// Helper function to get proc status from /proc (alternative method)
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

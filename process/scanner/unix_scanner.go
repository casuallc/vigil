package scanner

import (
  "bytes"
  "fmt"
  "github.com/casuallc/vigil/common"
  process2 "github.com/casuallc/vigil/process"
  "github.com/shirou/gopsutil/v3/process"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"
)

// scanWithScript scans processes using a custom script
func (m *process2.Manager) scanWithScript(script string) ([]process2.ManagedProcess, error) {
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
  var processes []process2.ManagedProcess
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
func (m *process2.Manager) scanUnixProcesses(query string) ([]process2.ManagedProcess, error) {
  dirs, err := os.ReadDir("/proc")
  if err != nil {
    return nil, fmt.Errorf("failed to read /proc: %v", err)
  }

  var processes []process2.ManagedProcess
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
func (m *process2.Manager) getProcessByPID(pid int) (*process2.ManagedProcess, error) {
  // 创建基础结构体
  manageProcess := &process2.ManagedProcess{
    Spec: process2.Spec{},
    Status: process2.Status{
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
  manageProcess.Status.Phase = process2.PhaseRunning

  return manageProcess, nil
}

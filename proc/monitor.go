package proc

import (
  "fmt"
  "os/exec"
  "regexp"
  "runtime"
  "strconv"
  "strings"
  "time"

  "github.com/shirou/gopsutil/v3/cpu"
  "github.com/shirou/gopsutil/v3/process"
)

// Monitor provides resource monitoring functionality
type Monitor struct {
  manager *Manager
}

// NewMonitor creates a new monitor
func NewMonitor(manager *Manager) *Monitor {
  return &Monitor{
    manager: manager,
  }
}

// GetSystemResourceUsage gets system resource usage
func GetSystemResourceUsage() (ResourceStats, error) {
  var stats ResourceStats
  var err error

  if runtime.GOOS == "windows" {
    stats, err = getWindowsSystemResourceUsage()
  } else {
    stats, err = getUnixSystemResourceUsage()
  }

  return stats, err
}

// GetProcessDiskIO 获取进程的磁盘IO统计信息
func GetProcessDiskIO(pid int) (uint64, error) {
  // 在Linux系统上，我们可以使用iostat或直接读取/proc文件系统
  // 这里使用一个简化的实现，实际应用中可能需要根据不同系统进行适配

  // 尝试读取/proc/{pid}/io文件（Linux系统）
  cmd := exec.Command("cat", fmt.Sprintf("/proc/%d/io", pid))
  output, err := cmd.CombinedOutput()
  if err == nil {
    // 解析输出
    readBytes := parseProcIOOutput(string(output), "read_bytes")
    writeBytes := parseProcIOOutput(string(output), "write_bytes")
    return readBytes + writeBytes, nil
  }

  // 如果无法读取/proc文件系统，尝试使用lsof命令（Unix/Linux/macOS通用）	// 注意：lsof可能需要root权限才能获取完整信息
  cmd = exec.Command("lsof", "-p", strconv.Itoa(pid), "-a", "-d", "^txt")
  output, err = cmd.CombinedOutput()
  if err != nil {
    return 0, fmt.Errorf("failed to get disk IO: %v, output: %s", err, string(output))
  }

  // 注意：lsof的输出格式复杂，这里仅作简化处理
  // 默认返回0，表示无法获取准确的磁盘IO信息
  return 0, nil
}

// parseProcIOOutput 解析/proc/{pid}/io文件的输出
func parseProcIOOutput(output string, field string) uint64 {
  lines := strings.Split(output, "\n")
  for _, line := range lines {
    if strings.HasPrefix(line, field) {
      parts := strings.Split(line, ":")
      if len(parts) > 1 {
        value, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
        if err == nil {
          return value
        }
      }
    }
  }
  return 0
}

// GetProcessNetworkIO 获取进程的网络IO统计信息
func GetProcessNetworkIO(pid int) (uint64, error) {
  // 网络IO的获取在不同系统上差异较大
  // 这里使用一个简化的实现
  // 默认返回0，表示无法获取准确的网络IO信息
  return 0, nil
}

// GetProcessListeningPorts 获取进程监听的端口信息
func GetProcessListeningPorts(pid int) ([]PortInfo, error) {
  var ports []PortInfo

  // 使用lsof命令获取进程打开的网络连接
  cmd := exec.Command("lsof", "-i", "-P", "-n", "-p", strconv.Itoa(pid))
  output, err := cmd.CombinedOutput()
  if err != nil {
    return ports, fmt.Errorf("failed to get listening ports: %v, output: %s", err, string(output))
  }

  // 解析lsof输出
  lines := strings.Split(string(output), "\n")
  for i := 1; i < len(lines); i++ { // 跳过表头
    line := strings.TrimSpace(lines[i])
    if line == "" {
      continue
    }

    fields := strings.Fields(line)
    if len(fields) < 8 {
      continue
    }

    // 检查是否为LISTEN状态（表示监听端口）
    if !strings.Contains(fields[7], "LISTEN") {
      continue
    }

    // 解析地址和端口
    addressPort := fields[7]
    // 格式通常为：协议->地址:端口
    // 例如：TCP->127.0.0.1:8080

    // 简单解析端口信息
    re := regexp.MustCompile(`:([0-9]+)`)
    matches := re.FindStringSubmatch(addressPort)
    if len(matches) < 2 {
      continue
    }

    port, err := strconv.Atoi(matches[1])
    if err != nil {
      continue
    }

    // 解析协议类型
    protocol := "TCP"
    if strings.Contains(addressPort, "UDP") {
      protocol = "UDP"
    }

    // 解析地址
    address := "0.0.0.0"
    addrRe := regexp.MustCompile(`->([^:]+):`)
    addrMatches := addrRe.FindStringSubmatch(addressPort)
    if len(addrMatches) > 1 {
      address = addrMatches[1]
    }

    ports = append(ports, PortInfo{
      Port:     port,
      Protocol: protocol,
      Address:  address,
    })
  }

  return ports, nil
}

// getWindowsSystemResourceUsage gets Windows system resource usage
func getWindowsSystemResourceUsage() (ResourceStats, error) {
  var stats ResourceStats

  // Get CPU usage
  cpuPercent, err := cpu.Percent(time.Second, false)
  if err == nil && len(cpuPercent) > 0 {
    stats.CPUUsage = cpuPercent[0]
  }

  // Get memory usage
  // 这里简化实现，实际项目中可以使用更复杂的方法

  return stats, nil
}

// getUnixSystemResourceUsage gets Unix/Linux/macOS system resource usage
func getUnixSystemResourceUsage() (ResourceStats, error) {
  var stats ResourceStats

  // Get CPU usage
  cpuPercent, err := cpu.Percent(time.Second, false)
  if err == nil && len(cpuPercent) > 0 {
    stats.CPUUsage = cpuPercent[0]
  }

  // Get memory usage using gopsutil
  _, err = process.NewProcess(0) // 使用0表示获取系统内存信息
  if err == nil {
    // 使用系统级别的内存信息
    // 注意：这里使用一个简化的方式获取系统内存使用情况
    // 在实际项目中，应该使用专门的内存信息获取方法
    // 由于gopsutil库没有直接的系统内存信息获取函数，我们可以使用exec.Command来获取
    cmd := exec.Command("free", "-b")
    output, err := cmd.CombinedOutput()
    if err == nil {
      lines := strings.Split(string(output), "\n")
      for _, line := range lines {
        if strings.HasPrefix(line, "Mem:") {
          fields := strings.Fields(line)
          if len(fields) >= 3 {
            // 提取已使用内存（第二列是总量，第三列是已使用量）
            usedMem, err := strconv.ParseUint(fields[2], 10, 64)
            if err == nil {
              stats.MemoryUsage = usedMem
            }
          }
          break
        }
      }
    }
  }

  // Get disk IO usage
  // 在Linux系统上，我们可以使用iostat命令获取磁盘IO统计信息
  cmd := exec.Command("iostat", "-d", "-k", "1", "1")
  output, err := cmd.CombinedOutput()
  if err == nil {
    // 解析iostat输出，提取磁盘IO信息
    // 这里使用一个简化的实现，实际应用中可能需要根据不同系统进行适配
    // 注意：完整的实现需要累加所有磁盘的IO数据
    lines := strings.Split(string(output), "\n")
    var totalRead, totalWrite uint64

    // 标记是否开始读取数据行
    dataStarted := false
    for _, line := range lines {
      if strings.Contains(line, "Device") {
        dataStarted = true
        continue
      }

      if dataStarted && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "\t") {
        fields := strings.Fields(line)
        if len(fields) >= 5 {
          // 读取和写入的KB数（假设第3和第4列是读写数据）
          readKB, _ := strconv.ParseUint(fields[2], 10, 64)
          writeKB, _ := strconv.ParseUint(fields[3], 10, 64)
          totalRead += readKB * 1024   // 转换为字节
          totalWrite += writeKB * 1024 // 转换为字节
        }
      }
    }

    // 累加读写IO总量
    stats.DiskIO = totalRead + totalWrite
  }

  return stats, nil
}

// GetUnixProcessResourceUsage gets Unix/Linux/macOS proc resource usage
func GetUnixProcessResourceUsage(pid int) (*ResourceStats, error) {
  var stats ResourceStats

  // Get proc resource usage on Unix/Linux/macOS
  newProc, err := process.NewProcess(int32(pid))
  if err != nil {
    return &stats, err
  }

  // Get CPU usage
  cpuPercent, err := newProc.CPUPercent()
  if err == nil {
    stats.CPUUsage = cpuPercent
  }

  // Get memory usage
  memInfo, err := newProc.MemoryInfo()
  if err == nil {
    stats.MemoryUsage = memInfo.RSS
  }

  // Get disk IO for the process
  diskIO, err := GetProcessDiskIO(pid)
  if err == nil {
    stats.DiskIO = diskIO
  }

  // Get network IO for the process
  networkIO, err := GetProcessNetworkIO(pid)
  if err == nil {
    stats.NetworkIO = networkIO
  }

  // Get listening ports for the process
  listeningPorts, err := GetProcessListeningPorts(pid)
  if err == nil {
    stats.ListeningPorts = listeningPorts
  }

  return &stats, nil
}

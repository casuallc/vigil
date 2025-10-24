package proc

import (
  "fmt"
  "os"
  "os/exec"
  "regexp"
  "runtime"
  "strconv"
  "strings"
  "time"

  "github.com/shirou/gopsutil/v3/cpu"
  "github.com/shirou/gopsutil/v3/disk"
  "github.com/shirou/gopsutil/v3/load"
  "github.com/shirou/gopsutil/v3/mem"
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

  // CPU
  if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
    stats.CPUUsage = cpuPercent[0]
  }

  // Memory
  if vm, err := mem.VirtualMemory(); err == nil {
    stats.MemoryUsage = vm.Used
    stats.MemoryTotal = vm.Total
    stats.MemoryUsedPercent = vm.UsedPercent
  }

  // Disk usage per partition
  if parts, err := disk.Partitions(true); err == nil {
    for _, p := range parts {
      if du, err := disk.Usage(p.Mountpoint); err == nil {
        stats.DiskUsages = append(stats.DiskUsages, DiskUsageInfo{
          Device:      p.Device,
          Mountpoint:  p.Mountpoint,
          Fstype:      p.Fstype,
          Total:       du.Total,
          Used:        du.Used,
          Free:        du.Free,
          UsedPercent: du.UsedPercent,
        })
      }
    }
  }

  // Disk IO per device
  if ioMap, err := disk.IOCounters(); err == nil {
    var totalIO uint64
    for dev, v := range ioMap {
      stats.DiskIODevices = append(stats.DiskIODevices, DiskIOInfo{
        Device:     dev,
        ReadBytes:  v.ReadBytes,
        WriteBytes: v.WriteBytes,
        ReadCount:  v.ReadCount,
        WriteCount: v.WriteCount,
      })
      totalIO += v.ReadBytes + v.WriteBytes
    }
    stats.DiskIO = totalIO
  }

  // Load avg: Windows不可用（留空）
  // FD、Kernel params: Windows不适用

  return stats, nil
}

// getUnixSystemResourceUsage gets Unix/Linux/macOS system resource usage
func getUnixSystemResourceUsage() (ResourceStats, error) {
  var stats ResourceStats

  // CPU
  if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
    stats.CPUUsage = cpuPercent[0]
  }

  // Memory
  if vm, err := mem.VirtualMemory(); err == nil {
    stats.MemoryUsage = vm.Used
    stats.MemoryTotal = vm.Total
    stats.MemoryUsedPercent = vm.UsedPercent
  }

  // Disk usage per partition（过滤虚拟FS与容器相关挂载）
  excludeFSTypes := map[string]bool{
    "tmpfs": true, "devtmpfs": true, "overlay": true, "proc": true, "sysfs": true,
    "cgroup": true, "aufs": true, "nsfs": true, "squashfs": true, "ramfs": true,
    "debugfs": true, "fusectl": true, "fuse.lxcfs": true,
  }
  excludeMountContains := []string{
    "/proc", "/sys", "/dev", "/run", "/snap", "/docker", "/containerd", "/var/lib/docker", "/var/lib/containerd",
  }

  if parts, err := disk.Partitions(true); err == nil {
    for _, p := range parts {
      fst := strings.ToLower(p.Fstype)
      if excludeFSTypes[fst] {
        continue
      }
      mpLower := strings.ToLower(p.Mountpoint)
      skip := false
      for _, m := range excludeMountContains {
        if strings.Contains(mpLower, m) {
          skip = true
          break
        }
      }
      devLower := strings.ToLower(p.Device)
      if strings.HasPrefix(devLower, "overlay") || strings.Contains(devLower, "docker") || strings.Contains(devLower, "containerd") {
        skip = true
      }
      if skip {
        continue
      }

      if du, err := disk.Usage(p.Mountpoint); err == nil {
        stats.DiskUsages = append(stats.DiskUsages, DiskUsageInfo{
          Device:      p.Device,
          Mountpoint:  p.Mountpoint,
          Fstype:      p.Fstype,
          Total:       du.Total,
          Used:        du.Used,
          Free:        du.Free,
          UsedPercent: du.UsedPercent,
        })
      }
    }
  }

  // Disk IO per device
  if ioMap, err := disk.IOCounters(); err == nil {
    var totalIO uint64
    for dev, v := range ioMap {
      stats.DiskIODevices = append(stats.DiskIODevices, DiskIOInfo{
        Device:     dev,
        ReadBytes:  v.ReadBytes,
        WriteBytes: v.WriteBytes,
        ReadCount:  v.ReadCount,
        WriteCount: v.WriteCount,
      })
      totalIO += v.ReadBytes + v.WriteBytes
    }
    stats.DiskIO = totalIO
  }

  // System Load
  if la, err := load.Avg(); err == nil {
    stats.Load = LoadAvg{Load1: la.Load1, Load5: la.Load5, Load15: la.Load15}
  }

  // File Descriptor Check (/proc/sys/fs/file-nr)
  if data, err := os.ReadFile("/proc/sys/fs/file-nr"); err == nil {
    fields := strings.Fields(strings.TrimSpace(string(data)))
    if len(fields) >= 3 {
      allocated, _ := strconv.ParseUint(fields[0], 10, 64)
      unused, _ := strconv.ParseUint(fields[1], 10, 64)
      max, _ := strconv.ParseUint(fields[2], 10, 64)
      inUse := allocated - unused
      var pct float64
      if max > 0 {
        pct = float64(inUse) / float64(max) * 100
      }
      stats.FD = FDCheck{
        CurrentAllocated: allocated,
        InUse:            inUse,
        Max:              max,
        UsagePercent:     pct,
      }
    }
  }

  // Kernel parameter check
  params := []string{
    "/proc/sys/vm/max_map_count",
    "/proc/sys/fs/file-max",
    "/proc/sys/net/core/somaxconn",
    "/proc/sys/net/ipv4/tcp_tw_reuse",
    "/proc/sys/net/ipv4/ip_local_port_range",
    "/proc/sys/net/core/netdev_max_backlog",
    "/proc/sys/vm/swappiness",
  }
  for _, p := range params {
    if val, err := readSysParam(p); err == nil {
      stats.KernelParams = append(stats.KernelParams, KernelParam{Key: p, Value: val})
    }
  }

  return stats, nil
}

func readSysParam(path string) (string, error) {
  b, err := os.ReadFile(path)
  if err != nil {
    return "", err
  }
  return strings.TrimSpace(string(b)), nil
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

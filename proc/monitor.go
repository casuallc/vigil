package proc

import (
  "fmt"
  "github.com/shirou/gopsutil/v3/host"
  gnet "github.com/shirou/gopsutil/v3/net"
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
    stats.CPUUsageHuman = FormatCPUUsage(cpuPercent[0])
  }

  // Memory
  if vm, err := mem.VirtualMemory(); err == nil {
    stats.MemoryUsage = vm.Used
    stats.MemoryUsageHuman = FormatBytes(vm.Used)
    stats.MemoryTotal = vm.Total
    stats.MemoryUsedPercent = vm.UsedPercent
    // 新增：可用内存
    stats.MemoryAvailable = vm.Available
  }
  // 新增：Swap 使用统计
  if sm, err := mem.SwapMemory(); err == nil {
    stats.SwapTotal = sm.Total
    stats.SwapUsed = sm.Used
    stats.SwapFree = sm.Free
  }
  // 新增：内存压力（PSI，仅 Linux）
  if runtime.GOOS == "linux" {
    if psi, err := readPressureStallInfo("memory"); err == nil {
      stats.MemoryPressure = psi
    }
  }

  // 预采样：网络与磁盘IO（用于速率/利用率估算）
  var (
    netSample1  map[string]gnet.IOCountersStat
    diskSample1 map[string]disk.IOCountersStat
    tSample1    time.Time
  )
  if ns, err := gnet.IOCounters(true); err == nil {
    netSample1 = make(map[string]gnet.IOCountersStat, len(ns))
    for _, v := range ns {
      netSample1[v.Name] = v
    }
  }
  if dm, err := disk.IOCounters(); err == nil {
    diskSample1 = dm
  }
  tSample1 = time.Now()

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
        info := DiskUsageInfo{
          Device:      p.Device,
          Mountpoint:  p.Mountpoint,
          Fstype:      p.Fstype,
          Total:       du.Total,
          Used:        du.Used,
          Free:        du.Free,
          UsedPercent: du.UsedPercent,
        }
        // 新增：Inode 统计（gopsutil 提供）
        info.InodesTotal = du.InodesTotal
        info.InodesUsed = du.InodesUsed
        info.InodesFree = du.InodesFree
        info.InodesUsedPercent = du.InodesUsedPercent

        stats.DiskUsages = append(stats.DiskUsages, info)
      }
    }
  }

  // Disk IO per device（首样）
  if ioMap, err := disk.IOCounters(); err == nil {
    var totalIO uint64
    stats.DiskIODevices = stats.DiskIODevices[:0]
    for dev, v := range ioMap {
      d := DiskIOInfo{
        Device:     dev,
        ReadBytes:  v.ReadBytes,
        WriteBytes: v.WriteBytes,
        ReadCount:  v.ReadCount,
        WriteCount: v.WriteCount,
        // 新增：读写耗时/设备忙碌时间（毫秒）
        ReadTimeMS:  v.ReadTime,
        WriteTimeMS: v.WriteTime,
        BusyTimeMS:  v.IoTime,
      }
      // 新增：平均读写延迟（毫秒）
      if v.ReadCount > 0 {
        d.AvgReadLatencyMS = float64(v.ReadTime) / float64(v.ReadCount)
      }
      if v.WriteCount > 0 {
        d.AvgWriteLatencyMS = float64(v.WriteTime) / float64(v.WriteCount)
      }

      stats.DiskIODevices = append(stats.DiskIODevices, d)
      totalIO += v.ReadBytes + v.WriteBytes
    }
    stats.DiskIO = totalIO
    stats.DiskIOHuman = FormatBytes(totalIO)
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

  // 新增：系统运行时间（秒）
  if up, err := host.Uptime(); err == nil {
    stats.SystemUptimeSeconds = float64(up)
  }

  // 二次采样：网络与磁盘IO（估算速率/利用率）
  time.Sleep(1 * time.Second)
  tSample2 := time.Now()
  intervalSec := tSample2.Sub(tSample1).Seconds()
  if intervalSec <= 0 {
    intervalSec = 1.0
  }

  // 网络速率与错误/丢包（系统聚合）
  if ns2, err := gnet.IOCounters(true); err == nil {
    var rx1, tx1, rx2, tx2 uint64
    var rxPkts, txPkts, rxErrs, txErrs, rxDrop, txDrop uint64
    // 聚合所有接口
    for _, v := range ns2 {
      rx2 += v.BytesRecv
      tx2 += v.BytesSent
      rxPkts += v.PacketsRecv
      txPkts += v.PacketsSent
      rxErrs += v.Errin
      txErrs += v.Errout
      rxDrop += v.Dropin
      txDrop += v.Dropout
    }
    for _, v := range netSample1 {
      rx1 += v.BytesRecv
      tx1 += v.BytesSent
    }

    if rx2 >= rx1 {
      stats.NetRxBytesPerSec = float64(rx2-rx1) / intervalSec
    }
    if tx2 >= tx1 {
      stats.NetTxBytesPerSec = float64(tx2-tx1) / intervalSec
    }
    stats.NetRxPackets = rxPkts
    stats.NetTxPackets = txPkts
    stats.NetRxErrors = rxErrs
    stats.NetTxErrors = txErrs
    stats.NetRxDropped = rxDrop
    stats.NetTxDropped = txDrop
  }

  // 磁盘 I/O 利用率与吞吐（按设备，基于 delta）
  if dm2, err := disk.IOCounters(); err == nil && diskSample1 != nil {
    // 构建索引以便填充 stats.DiskIODevices
    idx := make(map[string]int, len(stats.DiskIODevices))
    for i, d := range stats.DiskIODevices {
      idx[d.Device] = i
    }
    intervalMS := intervalSec * 1000.0
    for dev, v2 := range dm2 {
      v1, ok := diskSample1[dev]
      if !ok {
        continue
      }
      // 吞吐（B/s）
      readBps := float64Delta(v2.ReadBytes, v1.ReadBytes) / intervalSec
      writeBps := float64Delta(v2.WriteBytes, v1.WriteBytes) / intervalSec
      // 利用率（%）：IoTime 的增量相对采样间隔
      util := float64Delta(v2.IoTime, v1.IoTime) / intervalMS * 100.0
      if util < 0 {
        util = 0
      } else if util > 100 {
        util = 100
      }

      // 写回对应设备项
      if i, ok := idx[dev]; ok {
        stats.DiskIODevices[i].ReadThroughputBps = readBps
        stats.DiskIODevices[i].WriteThroughputBps = writeBps
        stats.DiskIODevices[i].UtilizationPercent = util
      }
    }
  }

  // 新增：TCP 连接状态计数（Linux）
  if runtime.GOOS == "linux" {
    if m, err := readTCPStateCounts(); err == nil {
      stats.TCPStateCounts = m
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

// 新增：读取 PSI（/proc/pressure/<kind>），kind=memory/cpu/io
func readPressureStallInfo(kind string) (PressureStallInfo, error) {
  var psi PressureStallInfo
  // 仅 Linux 支持
  if runtime.GOOS != "linux" {
    return psi, fmt.Errorf("PSI not supported on %s", runtime.GOOS)
  }
  path := fmt.Sprintf("/proc/pressure/%s", kind)
  data, err := os.ReadFile(path)
  if err != nil {
    return psi, err
  }
  // 解析 "some avg10=0.00 avg60=0.00 avg300=0.00 total=0" 行
  lines := strings.Split(strings.TrimSpace(string(data)), "\n")
  for _, line := range lines {
    if strings.HasPrefix(line, "some ") {
      psi = parsePSILine(line)
      break
    }
  }
  return psi, nil
}

// 新增：解析 PSI 行
func parsePSILine(line string) PressureStallInfo {
  var psi PressureStallInfo
  // 去掉前导 "some "
  parts := strings.Fields(strings.TrimPrefix(line, "some "))
  for _, p := range parts {
    kv := strings.SplitN(p, "=", 2)
    if len(kv) != 2 {
      continue
    }
    key, val := kv[0], kv[1]
    switch key {
    case "avg10":
      psi.Avg10 = parseFloat(val)
    case "avg60":
      psi.Avg60 = parseFloat(val)
    case "avg300":
      psi.Avg300 = parseFloat(val)
    case "total":
      psi.Total = parseUint(val)
    }
  }
  return psi
}

// 新增：读取 TCP 状态计数（/proc/net/tcp*）
func readTCPStateCounts() (map[string]int, error) {
  paths := []string{"/proc/net/tcp", "/proc/net/tcp6"}
  counts := map[string]int{}
  for _, p := range paths {
    b, err := os.ReadFile(p)
    if err != nil {
      // 某些系统不存在 tcp6，无需报错
      continue
    }
    // 按行解析，跳过首行表头
    lines := strings.Split(strings.TrimSpace(string(b)), "\n")
    for i := 1; i < len(lines); i++ {
      fields := strings.Fields(lines[i])
      if len(fields) < 4 {
        continue
      }
      // 第4列为 state 的十六进制码
      code := fields[3]
      name := tcpStateName(code)
      counts[name]++
    }
  }
  return counts, nil
}

// 新增：十六进制 TCP 状态码转名称
func tcpStateName(hex string) string {
  switch strings.ToUpper(hex) {
  case "01":
    return "ESTABLISHED"
  case "02":
    return "SYN_SENT"
  case "03":
    return "SYN_RECV"
  case "04":
    return "FIN_WAIT1"
  case "05":
    return "FIN_WAIT2"
  case "06":
    return "TIME_WAIT"
  case "07":
    return "CLOSE"
  case "08":
    return "CLOSE_WAIT"
  case "09":
    return "LAST_ACK"
  case "0A":
    return "LISTEN"
  case "0B":
    return "CLOSING"
  case "0C":
    return "NEW_SYN_RECV"
  default:
    return "UNKNOWN"
  }
}

// 新增：安全的 uint64 差计算转为浮点
func float64Delta(a, b uint64) float64 {
  if a >= b {
    return float64(a - b)
  }
  return 0
}

// 新增：Parse helpers
func parseFloat(s string) float64 {
  f, _ := strconv.ParseFloat(s, 64)
  return f
}
func parseUint(s string) uint64 {
  u, _ := strconv.ParseUint(s, 10, 64)
  return u
}

func readHeapFromSmaps(pid int) (uint64, error) {
  b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/smaps")
  if err != nil {
    return 0, err
  }
  lines := strings.Split(string(b), "\n")
  reHeader := regexp.MustCompile(`^[0-9a-fA-F]+-[0-9a-fA-F]+\s`)
  inHeap := false
  var sumKB uint64 = 0
  for _, line := range lines {
    if reHeader.MatchString(line) {
      inHeap = strings.Contains(line, "[heap]")
      continue
    }
    if inHeap && strings.HasPrefix(line, "Rss:") {
      f := strings.Fields(line)
      if len(f) >= 2 {
        v, _ := strconv.ParseUint(f[1], 10, 64)
        sumKB += v
      }
    }
  }
  return sumKB * 1024, nil
}

func readPriorityAndPolicy(pid int) (int32, string, error) {
  b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
  if err != nil {
    return 0, "", err
  }
  s := string(b)
  l := strings.Index(s, "(")
  r := strings.LastIndex(s, ")")
  if l < 0 || r < 0 || r <= l {
    return 0, "", fmt.Errorf("invalid stat format")
  }
  part1 := strings.Fields(s[:l])
  part2 := strings.Fields(s[r+1:])
  tokens := append(part1, part2...)

  // field indices (1-based): priority=18, policy=41
  getInt := func(idx int) int64 {
    if idx < len(tokens) {
      v, _ := strconv.ParseInt(tokens[idx], 10, 64)
      return v
    }
    return 0
  }
  priority := int32(getInt(17))
  policyNum := getInt(40)

  policy := map[int64]string{
    0: "SCHED_NORMAL",
    1: "SCHED_FIFO",
    2: "SCHED_RR",
    3: "SCHED_BATCH",
    5: "SCHED_IDLE",
    6: "SCHED_DEADLINE",
  }[policyNum]
  if policy == "" {
    policy = "UNKNOWN"
  }
  return priority, policy, nil
}

func GetUnixProcessResourceUsage(pid int) (*ResourceStats, error) {
  p, err := process.NewProcess(int32(pid))
  if err != nil {
    return nil, err
  }

  stats := &ResourceStats{}

  // CPU 使用率（采样1s）
  if cpuPercent, err := p.Percent(time.Second); err == nil {
    stats.CPUUsage = cpuPercent
  }
  // CPU 时间（user/system/total）
  if times, err := p.Times(); err == nil {
    stats.CPUUserTime = times.User
    stats.CPUSystemTime = times.System
    stats.CPUTotalTime = times.User + times.System
  }

  // 内存信息（RSS/VMS/Shared/Heap）
  if mi, err := p.MemoryInfo(); err == nil {
    stats.MemoryUsage = mi.RSS
    stats.MemoryRSS = mi.RSS
    stats.MemoryVMS = mi.VMS
    //stats.MemoryShared = mi.Shared
  }
  if runtime.GOOS == "linux" {
    if heap, err := readHeapFromSmaps(pid); err == nil {
      stats.MemoryHeap = heap
    }
  }

  // 进程级 IO 统计
  if ioc, err := p.IOCounters(); err == nil && ioc != nil {
    stats.IOReadBytes = ioc.ReadBytes
    stats.IOWriteBytes = ioc.WriteBytes
    stats.IOReadCount = ioc.ReadCount
    stats.IOWriteCount = ioc.WriteCount
    stats.DiskIO = ioc.ReadBytes + ioc.WriteBytes
  } else {
    if v, err2 := GetProcessDiskIO(pid); err2 == nil {
      stats.DiskIO = v
    }
  }

  // 打开文件描述符数
  if n, err := p.NumFDs(); err == nil {
    stats.OpenFDs = n
  }

  // 线程数与状态
  if t, err := p.NumThreads(); err == nil {
    stats.ThreadCount = t
  }
  if st, err := p.Status(); err == nil {
    stats.ProcessStatus = strings.Join(st, ",")
  }

  // 上下文切换次数
  if cs, err := p.NumCtxSwitches(); err == nil && cs != nil {
    stats.CtxSwitchesVoluntary = cs.Voluntary
    stats.CtxSwitchesInvoluntary = cs.Involuntary
  }

  // nice 值
  if n, err := p.Nice(); err == nil {
    stats.Nice = n
  }

  // 调度策略与优先级（仅 Linux）
  if runtime.GOOS == "linux" {
    if prio, policy, err := readPriorityAndPolicy(pid); err == nil {
      stats.SchedulerPriority = prio
      stats.SchedulerPolicy = policy
    }
  }

  // CPU 亲和性
  if aff, err := p.CPUAffinity(); err == nil {
    stats.CPUAffinity = aff
  }

  // 打开文件与网络连接数量；同时填充监听端口
  if files, err := p.OpenFiles(); err == nil {
    stats.OpenFilesCount = len(files)
  }
  if conns, err := p.Connections(); err == nil {
    stats.NetworkConnectionsCount = len(conns)
    var ports []PortInfo
    for _, c := range conns {
      if c.Status == "LISTEN" {
        ports = append(ports, PortInfo{
          Port:     int(c.Laddr.Port),
          Protocol: socketTypeToProtocol(c.Type),
          Address:  c.Laddr.IP,
        })
      }
    }
    stats.ListeningPorts = ports
  } else {
    if ports, err2 := GetProcessListeningPorts(pid); err2 == nil {
      stats.ListeningPorts = ports
    }
  }

  // 系统内存与 PSI（用于补充展示）
  if vm, err := mem.VirtualMemory(); err == nil {
    stats.MemoryTotal = vm.Total
    stats.MemoryUsedPercent = vm.UsedPercent
    stats.MemoryAvailable = vm.Available
  }
  if sm, err := mem.SwapMemory(); err == nil {
    stats.SwapTotal = sm.Total
    stats.SwapUsed = sm.Used
    stats.SwapFree = sm.Free
  }
  if runtime.GOOS == "linux" {
    if psi, err := readPressureStallInfo("memory"); err == nil {
      stats.MemoryPressure = psi
    }
  }

  // 父进程与启动时间
  if ppid, err := p.Ppid(); err == nil {
    stats.ParentPID = int(ppid)
  }
  if ct, err := p.CreateTime(); err == nil {
    ts := time.UnixMilli(ct)
    stats.StartTime = &ts
  }

  // 系统运行时间（秒）
  if up, err := host.Uptime(); err == nil {
    stats.SystemUptimeSeconds = float64(up)
  }

  return stats, nil
}

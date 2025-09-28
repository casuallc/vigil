package process

import (
  "runtime"
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

  // Get memory usage
  // 这里简化实现，实际项目中可以使用更复杂的方法

  return stats, nil
}

// GetProcessResourceUsage gets resource usage of a specific process
func GetProcessResourceUsage(pid int) (ResourceStats, error) {
  var stats ResourceStats
  var err error

  if runtime.GOOS == "windows" {
    stats, err = getWindowsProcessResourceUsage(pid)
  } else {
    stats, err = getUnixProcessResourceUsage(pid)
  }

  return stats, err
}

// getWindowsProcessResourceUsage gets Windows process resource usage
func getWindowsProcessResourceUsage(pid int) (ResourceStats, error) {
  var stats ResourceStats

  // Get process resource usage on Windows
  proc, err := process.NewProcess(int32(pid))
  if err != nil {
    return stats, err
  }

  // Get CPU usage
  cpuPercent, err := proc.CPUPercent()
  if err == nil {
    stats.CPUUsage = cpuPercent
  }

  // Get memory usage
  memInfo, err := proc.MemoryInfo()
  if err == nil {
    stats.MemoryUsage = memInfo.RSS
  }

  return stats, nil
}

// getUnixProcessResourceUsage gets Unix/Linux/macOS process resource usage
func getUnixProcessResourceUsage(pid int) (ResourceStats, error) {
  var stats ResourceStats

  // Get process resource usage on Unix/Linux/macOS
  proc, err := process.NewProcess(int32(pid))
  if err != nil {
    return stats, err
  }

  // Get CPU usage
  cpuPercent, err := proc.CPUPercent()
  if err == nil {
    stats.CPUUsage = cpuPercent
  }

  // Get memory usage
  memInfo, err := proc.MemoryInfo()
  if err == nil {
    stats.MemoryUsage = memInfo.RSS
  }

  return stats, nil
}

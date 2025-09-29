package process

import (
  "fmt"
  "strings"
)

// SetFormattedValues 设置所有格式化的字段值
func (rs *ResourceStats) SetFormattedValues() {
  rs.CPUUsageHuman = FormatCPUUsage(rs.CPUUsage)
  rs.MemoryUsageHuman = FormatBytes(rs.MemoryUsage)
  rs.DiskIOHuman = FormatBytes(rs.DiskIO)
  rs.NetworkIOHuman = FormatBytes(rs.NetworkIO)
}

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

package cli

import (
  "bytes"
  "fmt"
  "github.com/casuallc/vigil/proc"
  "strings"
  "time"
)

// 辅助函数：格式化时间间隔
func formatDuration(d time.Duration) string {
  if d.Seconds() < 60 {
    return fmt.Sprintf("%.1fs", d.Seconds())
  } else if d.Minutes() < 60 {
    return fmt.Sprintf("%.1fm", d.Minutes())
  } else if d.Hours() < 24 {
    return fmt.Sprintf("%.1fh", d.Hours())
  } else {
    return fmt.Sprintf("%.1fd", d.Hours()/24)
  }
}

// 辅助函数：截断字符串
func truncateString(s string, maxLength int) string {
  if len(s) <= maxLength {
    return s
  }
  return s[:maxLength-3] + "..."
}

// 辅助函数：居中显示文本
func centerText(text string, width int) string {
  if len(text) >= width {
    return text
  }
  padding := (width - len(text)) / 2
  return string(bytes.Repeat([]byte(" "), padding)) + text
}

// selectProcessInteractively 通用的交互式进程选择函数
func (c *CLI) selectProcessInteractively(namespace string, label string) (proc.ManagedProcess, error) {
  // 获取所有进程
  processes, err := c.client.ListProcesses(namespace)
  if err != nil {
    return proc.ManagedProcess{}, err
  }

  if len(processes) == 0 {
    return proc.ManagedProcess{}, fmt.Errorf("没有找到进程，请先注册一个进程")
  }

  // 提取进程名称列表
  processNames := make([]string, len(processes))
  for i, p := range processes {
    processNames[i] = fmt.Sprintf("%s (Namespace: %s, Status: %s, PID: %d)",
      p.Metadata.Name, p.Metadata.Namespace, p.Status.Phase, p.Status.PID)
  }

  idx, _, err := Select(SelectConfig{
    Label: label,
    Items: processNames,
  })
  if err != nil {
    return proc.ManagedProcess{}, err
  }

  return processes[idx], nil
}

func firstNonEmptyLine(msg string) string {
  for _, line := range strings.Split(msg, "\n") {
    if trimmed := strings.TrimSpace(line); trimmed != "" {
      return trimmed
    }
  }
  return ""
}

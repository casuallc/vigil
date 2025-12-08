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

// firstNonEmptyLine 获取字符串的第一行非空行
func firstNonEmptyLine(msg string) string {
  for _, line := range strings.Split(msg, "\n") {
    if trimmed := strings.TrimSpace(line); trimmed != "" {
      return trimmed
    }
  }
  return ""
}

// SplitStringByFixedWidth 按固定宽度使用换行分割字符串
func SplitStringByFixedWidth(s string, lineLen int) string {
  if lineLen <= 0 || len(s) <= lineLen {
    return s
  }

  r := []rune(s)
  var b strings.Builder

  for i := 0; i < len(r); i += lineLen {
    end := i + lineLen
    if end > len(r) {
      end = len(r)
    }
    b.WriteString(string(r[i:end]))
    if end < len(r) {
      b.WriteString("\n")
    }
  }

  return b.String()
}

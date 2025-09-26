package cli

import (
	"fmt"
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
package common

import (
  "bufio"
  "os"
  "strconv"
  "strings"
  "time"
)

// ParseProcessStartTime 解析ps命令输出的启动时间字符串
// ps命令在不同系统上输出的时间格式可能不同，这里支持多种常见格式
func ParseProcessStartTime(startTimeStr string) time.Time {
  // 尝试解析HH:MM格式（今天的时间）
  if t, err := time.Parse("15:04", startTimeStr); err == nil {
    today := time.Now()
    return time.Date(today.Year(), today.Month(), today.Day(), t.Hour(), t.Minute(), 0, 0, today.Location())
  }

  // 尝试解析MM-DD格式（今年的日期）
  if t, err := time.Parse("01-02", startTimeStr); err == nil {
    year := time.Now().Year()
    return time.Date(year, t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
  }

  // 尝试解析MM-DD HH:MM格式（今年的日期和时间）
  if t, err := time.Parse("01-02 15:04", startTimeStr); err == nil {
    year := time.Now().Year()
    return time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
  }

  // 尝试解析完整的日期时间格式
  layouts := []string{
    time.RFC3339,
    time.RFC1123,
    time.RFC1123Z,
    time.ANSIC,
    time.UnixDate,
  }

  for _, layout := range layouts {
    if t, err := time.Parse(layout, startTimeStr); err == nil {
      return t
    }
  }

  // 如果所有解析都失败，返回当前时间
  return time.Now()
}

// IsNumber 是否是数字
func IsNumber(str string) bool {
  _, err := strconv.Atoi(str)
  return err == nil
}

// ParseToString 转换成字符串，替换其中的分隔符为空格
// 这里主要是处理 linux 下按照 \0 分隔的内容
func ParseToString(content []byte, split byte) string {
  var result strings.Builder
  for i := 0; i < len(content); i++ {
    if content[i] == split {
      result.WriteByte(' ')
    } else {
      result.WriteByte(content[i])
    }
  }
  return strings.TrimSpace(result.String())
}

// ParsePropertyArray 把 k=v,k2=v2 转换成二维数组
func ParsePropertyArray(str string) [][]string {
  if str == "" {
    return nil
  }

  array := strings.Split(str, ",")
  if len(array) == 0 {
    return nil
  }

  result := make([][]string, len(array))
  for i := 0; i < len(array); i++ {
    kv := strings.SplitN(array[i], "=", 2)
    if len(kv) != 2 {
      continue
    }
    kv[0] = strings.TrimSpace(kv[0])
    kv[1] = strings.TrimSpace(kv[1])
    result[i] = kv
  }
  return result
}

// LoadKeyValues loads simple key=value lines from a file.
func LoadKeyValues(filePath string) (map[string]string, error) {
  result := make(map[string]string)

  f, err := os.Open(filePath)
  if err != nil {
    return result, err
  }
  defer f.Close()

  scanner := bufio.NewScanner(f)
  for scanner.Scan() {
    line := strings.TrimSpace(scanner.Text())
    if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
      continue
    }
    if strings.HasPrefix(line, "export ") {
      line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
    }
    kv := strings.SplitN(line, "=", 2)
    if len(kv) != 2 {
      continue
    }
    key := strings.TrimSpace(kv[0])
    val := strings.TrimSpace(kv[1])
    val = strings.Trim(val, `"'`)
    result[key] = val
  }
  return result, scanner.Err()
}

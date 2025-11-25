package api

import (
  "fmt"
  "strconv"
)

// parseFloatValue 转换成数值
func parseFloatValue(v interface{}) (float64, error) {
  // 尝试将Value转换为float64
  var value float64
  var err error

  switch v := v.(type) {
  case float64:
    value = v
  case float32:
    value = float64(v)
  case int:
    value = float64(v)
  case int64:
    value = float64(v)
  case string:
    value, err = strconv.ParseFloat(v, 64)
    if err != nil {
      return 0, err // 无法转换为数值，跳过阈值判断
    }
  default:
    // 如果Value是其他类型，尝试字符串转换
    valueStr := fmt.Sprintf("%v", v)
    value, err = strconv.ParseFloat(valueStr, 64)
    if err != nil {
      return 0, err // 无法转换为数值，跳过阈值判断
    }
  }
  return value, nil
}

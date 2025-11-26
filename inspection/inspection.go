package inspection

import (
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/expr-lang/expr"
  "regexp"
  "strconv"
  "strings"
  "time"
)

// ExecuteCheck 执行单个检查项
func ExecuteCheck(check Check, envVars []string) CheckResult {
  // 记录开始时间
  startTime := time.Now()
  result := executeScriptCheck(check, envVars)
  // 计算执行时间
  result.DurationMs = int64(int(time.Since(startTime).Milliseconds()))

  if result.Value == nil || result.Status == StatusError {
    return result
  }

  // 根据 docs/inspection_rules.md 中定义的规则进行解析和结果判断
  if check.Expect != nil {
    // 处理 expect 匹配
    handleExpectMatch(check, &result)
  } else if len(check.Compare) > 0 {
    // 处理 compare 匹配
    handleCompare(check, &result)
  } else if len(check.Thresholds) > 0 {
    // 处理阈值判断
    handleThresholds(check, &result)
  } else {
    // 不需要比较
    result.Status = StatusOk
  }

  return result
}

// executeScriptCheck 执行脚本检查
func executeScriptCheck(check Check, envVars []string) CheckResult {
  result := CheckResult{
    ID:          check.ID,
    Name:        check.Name,
    Type:        check.Type,
    Remediation: check.Remediation,
    Status:      StatusOk,
    Severity:    SeverityWarn,
  }

  // 获取命令
  commandLines, err := check.GetCommandLines()
  if err != nil || len(commandLines) == 0 {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Failed to get command: %v", err)
    return result
  }

  // 执行命令
  output, err := common.ExecuteCommand(commandLines[0], envVars)
  if err != nil {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Command execution failed: %v, output: %s", err, output)
    return result
  }

  // 解析输出
  result = parseCheckOutput(check, output, result)

  return result
}

// parseCheckOutput 解析检查输出
func parseCheckOutput(check Check, output string, result CheckResult) CheckResult {
  var parseErr error
  result.Value = output
  if check.Parse != nil {
    switch check.Parse.Kind {
    case "regex":
      if check.Parse.Pattern != "" {
        re := regexp.MustCompile(check.Parse.Pattern)
        if matches := re.FindStringSubmatch(output); len(matches) > 1 {
          if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
            result.Value = val
          } else {
            parseErr = err
          }
        }
      }
    case "json":
      var data map[string]interface{}
      if err := json.Unmarshal([]byte(output), &data); err == nil {
        if val, ok := data[check.Parse.Path]; ok {
          if valFloat, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64); err == nil {
            result.Value = valFloat
          } else {
            parseErr = err
          }
        }
      }
    case "int":
      if val, err := strconv.ParseInt(output, 10, 64); err == nil {
        result.Value = val
      } else {
        parseErr = err
      }
    case "float":
      if val, err := strconv.ParseFloat(output, 64); err == nil {
        result.Value = val
      } else {
        parseErr = err
      }
    default:
      result.Value = output
    }
  }
  // 获取不到值，则返回错误
  if result.Value == nil || parseErr != nil {
    errStr := ""
    if parseErr != nil {
      errStr = parseErr.Error()
    }
    result.Message = fmt.Sprintf("Can not parse value, error: %s", errStr)
    result.Status = StatusError
  }

  return result
}

// handleExpectMatch 处理期望匹配
func handleExpectMatch(check Check, result *CheckResult) {
  output := fmt.Sprintf("%v", result.Value)

  expectLines, err := check.GetExpectLines()
  if err != nil {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Invalid expect configuration: %v", err)
    return
  }

  // 检查输出是否包含任一期望的行
  matched := false
  for _, expect := range expectLines {
    if strings.Contains(output, expect) {
      matched = true
      break
    }
  }

  if !matched {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Output does not match expected pattern(s), value: %s", output)
  }
}

// handleCompare 处理数值比较
func handleCompare(check Check, result *CheckResult) {
  value, err := common.ParseFloatValue(result.Value)
  if err != nil {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Failed to parse value: %v", err)
    return
  }

  // 使用expr-lang评估阈值表达式
  env := map[string]interface{}{
    "value": value,
  }
  exprStr := "value " + check.Compare
  // 评估表达式
  evalResult, err := expr.Eval(exprStr, env)
  if err != nil {
    return
  }

  // 如果表达式为真，则应用该阈值规则
  if match, ok := evalResult.(bool); ok && match {
    result.Status = StatusOk
    result.Message = fmt.Sprintf("Threshold condition met: %s", check.Compare)
  } else {
    result.Status = StatusError
  }
}

// handleThresholds 处理阈值判断
func handleThresholds(check Check, result *CheckResult) {
  value, err := common.ParseFloatValue(result.Value)
  if err != nil {
    result.Status = StatusError
    result.Message = fmt.Sprintf("Failed to parse value: %v", err)
    return
  }

  // 使用expr-lang评估阈值表达式
  env := map[string]interface{}{
    "value": value,
  }

  // 检查每个阈值规则
  for _, threshold := range check.Thresholds {
    // 构建表达式
    exprStr := "value " + threshold.When

    // 评估表达式
    evalResult, err := expr.Eval(exprStr, env)
    if err != nil {
      continue // 跳过无效的表达式
    }

    // 如果表达式为真，则应用该阈值规则
    if match, ok := evalResult.(bool); ok && match {
      result.Status = string(threshold.Severity)
      result.Severity = string(threshold.Severity)
      if threshold.Message != "" {
        result.Message = threshold.Message
      } else {
        result.Message = fmt.Sprintf("Threshold condition met: %s", threshold.When)
      }
      break // 一旦匹配到阈值规则就停止检查
    }
  }
}

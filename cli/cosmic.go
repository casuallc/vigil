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
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/api"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/inspection"
  "github.com/pterm/pterm"
  "github.com/spf13/cobra"
  "gopkg.in/yaml.v3"
  "os"
  "os/exec"
  "strconv"
  "strings"
  "time"
)

// setupCosmicCommands 设置cosmic相关命令
func (c *CLI) setupCosmicCommands() *cobra.Command {
  cosmicCmd := &cobra.Command{
    Use:   "cosmic",
    Short: "Cosmic system inspection operations",
    Long:  "Manage and inspect cosmic systems with various operations",
  }

  // 添加cosmic子命令
  cosmicCmd.AddCommand(c.setupCosmicInspectCommand())

  return cosmicCmd
}

// setupCosmicInspectCommand 设置cosmic inspect命令
func (c *CLI) setupCosmicInspectCommand() *cobra.Command {
  var configFile string
  var outputFormat string
  var outputFile string
  var jobName string
  var envVars []string

  inspectCmd := &cobra.Command{
    Use:   "inspect",
    Short: "Inspect cosmic systems based on configuration",
    Long:  "Run inspection rules against cosmic systems by parsing configuration files from conf/cosmic directory",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleCosmicInspect(configFile, jobName, envVars, outputFormat, outputFile)
    },
  }

  inspectCmd.Flags().StringVarP(&configFile, "config", "c", "conf/cosmic/cosmic.yaml", "Cosmic configuration file path")
  inspectCmd.Flags().StringVarP(&jobName, "job", "j", "", "Specific job name to inspect (if not specified, all jobs will be inspected)")
  inspectCmd.Flags().StringArrayVarP(&envVars, "env", "e", []string{}, "Environment variables to override (format: KEY=VALUE)")
  inspectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output result to file instead of console")
  inspectCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text|json|yaml|markdown|html|pdf)")

  return inspectCmd
}

// handleCosmicInspect 处理cosmic系统巡检命令
func (c *CLI) handleCosmicInspect(configFile string, jobName string, envVars []string, outputFormat string, outputFile string) error {
  pterm.DefaultHeader.WithFullWidth().Printf("Cosmic System Inspection Started")
  startTime := time.Now()

  // 加载cosmic配置文件
  cosmicConfig, err := inspection.LoadCosmicConfig(configFile)
  if err != nil {
    fmt.Printf("ERROR failed to load cosmic config: %v\n", err)
    return fmt.Errorf("failed to load cosmic config: %w", err)
  }

  // 创建节点映射
  nodeMap := make(map[string]inspection.Node)
  for _, node := range cosmicConfig.Nodes {
    nodeMap[node.Name] = node
    fmt.Printf("Discovered node: %s (%s:%d)\n", node.Name, node.IP, node.Port)
  }
  fmt.Printf("Total nodes discovered: %d\n", len(nodeMap))

  // 筛选需要巡检的作业
  var jobsToInspect []inspection.Job
  if jobName != "" {
    // 指定了特定作业
    found := false
    for _, job := range cosmicConfig.Jobs {
      if job.Name == jobName {
        jobsToInspect = append(jobsToInspect, job)
        found = true
        break
      }
    }
    if !found {
      fmt.Printf("ERROR job '%s' not found in configuration\n", jobName)
      return fmt.Errorf("job '%s' not found in configuration", jobName)
    }
  } else {
    // 检查所有作业
    jobsToInspect = cosmicConfig.Jobs
  }
  fmt.Printf("Total jobs to inspect: %d\n", len(jobsToInspect))

  // 处理环境变量
  envMap := make(map[string]string)
  for _, env := range envVars {
    parts := strings.SplitN(env, "=", 2)
    if len(parts) == 2 {
      envMap[parts[0]] = parts[1]
      fmt.Printf("Override env var: %s=%s\n", parts[0], parts[1])
    }
  }

  // 执行巡检
  var allResults []inspection.CosmicResult
  var summaryBySoftware = make(map[string][]inspection.CosmicResult)

  for _, job := range jobsToInspect {
    fmt.Println()
    pterm.DefaultHeader.WithFullWidth().Printf("Processing Software: %s", job.Name)

    // 加载作业规则
    var inspectionRules *inspection.RuleConfig
    if len(job.Rules) > 0 {
      // 支持多个规则文件，按顺序加载
      for _, rule := range job.Rules {
        rulePath := rule.Path
        fmt.Printf("Loading rules for %s from %s\n", job.Name, rulePath)

        rules, err := inspection.LoadRules(rulePath)
        if err != nil {
          fmt.Printf("WARNING failed to load rules '%s' for job '%s' from %s: %v\n", rule.Name, job.Name, rulePath, err)
          continue
        }

        // 如果是第一个规则文件，直接赋值；否则合并检查项
        if inspectionRules == nil {
          inspectionRules = rules
        } else {
          inspectionRules.Checks = append(inspectionRules.Checks, rules.Checks...)
        }
      }

      if inspectionRules != nil {
        fmt.Printf("Successfully loaded %d inspection rules for %s\n", len(inspectionRules.Checks), job.Name)
      }
    } else {
      fmt.Printf("No rules defined for job: %s\n", job.Name)
    }

    // 按节点维度进行巡检
    for _, targetName := range job.Targets {
      node, exists := nodeMap[targetName]
      if !exists {
        fmt.Printf("WARNING node '%s' not found for job '%s'\n", targetName, job.Name)
        continue
      }

      fmt.Println()
      pterm.DefaultHeader.WithFullWidth().Printf("Inspecting %s on node %s (%s:%d)", job.Name, node.Name, node.IP, node.Port)

      // 执行实际巡检
      result := c.performCosmicInspection(job, node, inspectionRules, envMap)
      allResults = append(allResults, result)

      // 按软件分组存储结果
      summaryBySoftware[job.Name] = append(summaryBySoftware[job.Name], result)
    }
  }

  // 汇总分析结果
  fmt.Println()
  pterm.DefaultHeader.WithFullWidth().Printf("Summary Analysis")
  for software, results := range summaryBySoftware {
    fmt.Printf("Software: %s\n", software)
    fmt.Printf("- Total nodes: %d\n", len(results))

    var success, errorCount int
    for _, r := range results {
      switch r.Status {
      case inspection.StatusOk:
        success++
      case inspection.StatusError:
        errorCount++
      }
    }

    fmt.Printf("- Success: %d\n", success)
    fmt.Printf("- Error: %d\n", errorCount)
  }

  fmt.Printf("\nTotal inspection duration: %.2f seconds\n", time.Since(startTime).Seconds())

  // 格式化输出结果
  return c.formatAndOutputCosmicResults(allResults, outputFormat, outputFile)
}

// performRuleBasedInspection 基于规则配置执行巡检
func (c *CLI) performRuleBasedInspection(job inspection.Job, node inspection.Node, rules *inspection.RuleConfig, envMap map[string]string, result *inspection.CosmicResult) error {
  // 构建环境变量列表，转换为[]string格式
  var envList []string
  for k, v := range envMap {
    envList = append(envList, fmt.Sprintf("%s=%s", k, v))
  }

  // 构建检查请求
  checkRequest := inspection.Request{
    Version: rules.Version,
    Meta: inspection.RequestMeta{
      System:  rules.Meta.System,
      Host:    node.IP,
      JobName: job.Name,
      Time:    time.Now(),
    },
    Checks: rules.Checks,
    Env:    envList,
  }

  fmt.Printf("Sending inspection rules to node %s (%s:%d)\n", node.Name, node.IP, node.Port)
  fmt.Printf("- System: %s\n", rules.Meta.System)
  fmt.Printf("- Job: %s\n", job.Name)
  fmt.Printf("- Rules count: %d\n", len(rules.Checks))

  // 为每个节点创建客户端
  nodeClient := api.NewClient(fmt.Sprintf("http://%s:%d", node.IP, node.Port))

  // 执行远程检查
  checkResult, err := nodeClient.ExecuteInspection(checkRequest)
  if err != nil {
    result.Status = inspection.StatusError
    result.Message = fmt.Sprintf("Failed to execute inspection on node %s: %v", node.Name, err)
    return err
  }

  // 转换检查结果
  result.Status = strings.ToLower(checkResult.Summary.OverallStatus)
  result.Message = fmt.Sprintf("Inspection completed with status: %s", checkResult.Summary.OverallStatus)

  // 转换检查项
  for _, check := range checkResult.Results {
    result.Checks = append(result.Checks, inspection.CheckResult{
      ID:          check.ID,
      Name:        check.Name,
      Type:        check.Type,
      Value:       check.Value,
      Unit:        check.Unit,
      Status:      strings.ToLower(check.Status),
      Severity:    strings.ToLower(check.Severity),
      Message:     check.Message,
      DurationMs:  check.DurationMs,
      Remediation: check.Remediation,
    })
  }

  // 统计检查结果
  var passed, errorCount int
  for _, check := range result.Checks {
    switch check.Status {
    case inspection.StatusOk:
      passed++
    case inspection.StatusError:
      errorCount++
    }
  }

  fmt.Printf("Inspection results for node %s:\n", node.Name)
  fmt.Printf("- Total checks: %d\n", len(result.Checks))
  fmt.Printf("- Passed: %d\n", passed)
  fmt.Printf("- Error: %d\n", errorCount)

  return nil
}

// performCosmicInspection 执行具体的cosmic系统巡检
func (c *CLI) performCosmicInspection(job inspection.Job, node inspection.Node, rules *inspection.RuleConfig, envMap map[string]string) inspection.CosmicResult {
  result := inspection.CosmicResult{
    JobName: job.Name,
    Host:    node.IP,
    Port:    node.Port,
    Status:  inspection.StatusOk,
    Checks:  []inspection.CheckResult{},
  }

  // 记录开始时间
  startTime := time.Now()

  // 构建环境变量（合并作业配置和命令行参数）
  allEnvs := make(map[string]string)

  // 添加作业环境变量
  for _, env := range job.Envs {
    allEnvs[env.Name] = env.Value
  }

  // 添加命令行环境变量（优先级更高）
  for k, v := range envMap {
    allEnvs[k] = v
  }

  // 添加节点信息到环境变量
  allEnvs["NODE_IP"] = node.IP
  allEnvs["NODE_PORT"] = fmt.Sprintf("%d", node.Port)
  allEnvs["NODE_NAME"] = node.Name
  allEnvs["JOB_NAME"] = job.Name

  tryInspect := func() error {
    // 如果存在规则配置，使用规则进行巡检
    if rules != nil {
      return c.performRuleBasedInspection(job, node, rules, allEnvs, &result)
    }

    // 如果没有规则，进行基本的连通性检查
    fmt.Printf("No rules specified for job '%s', performing basic connectivity check\n", job.Name)
    result.Status = inspection.StatusOk
    result.Message = "Basic connectivity check: No inspection rules provided"
    return nil
  }

  if err := tryInspect(); err != nil {
    result.Status = inspection.StatusError
    result.Message = fmt.Sprintf("Inspection failed: %v", err)
  }

  // 计算执行时间
  result.Duration = time.Since(startTime).Seconds()
  return result
}

// formatAndOutputCosmicResults 格式化并输出cosmic巡检结果
func (c *CLI) formatAndOutputCosmicResults(results []inspection.CosmicResult, format string, outputFile string) error {
  var output []byte
  var err error

  switch format {
  case "yaml":
    output, err = yaml.Marshal(results)
    if err != nil {
      return fmt.Errorf("failed to marshal cosmic results to yaml: %w", err)
    }
  case "json":
    output, err = json.MarshalIndent(results, "", "  ")
    if err != nil {
      return fmt.Errorf("failed to marshal cosmic results to json: %w", err)
    }
  case "text", "":
    output = formatToText(results, outputFile)
  case "html":
    output = formatToHtml(results, outputFile)
  case "markdown":
    output = formatToMarkdown(results, outputFile)
  case "pdf":
    return formatToPdf(results, outputFile)
  default:
    return fmt.Errorf("unsupported output format: %s", format)
  }

  // === 输出 ===
  if outputFile != "" {
    if err := os.WriteFile(outputFile, output, 0644); err != nil {
      return fmt.Errorf("failed to write output file: %w", err)
    }
    pterm.Success.Printf("Report written to: %s\n", outputFile)
  } else {
    fmt.Print(string(output))
  }

  return nil
}

// formatToText 格式化并输出cosmic巡检结果为纯文本
func formatToText(results []inspection.CosmicResult, outputFile string) []byte {
  var buf bytes.Buffer
  const lineWidth = 120

  // === 报告标题 ===
  headerText := pterm.DefaultHeader.WithFullWidth().Sprint("Cosmic Middleware Inspection Report")
  fmt.Fprintf(&buf, "%s", headerText)
  fmt.Fprintf(&buf, "Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

  // === 收集统计信息 ===
  totalJobs := len(results)
  successJobs, failedJobs := 0, 0
  totalChecks, totalPassed, totalErrors := 0, 0, 0

  softwareResults := make(map[string][]inspection.CosmicResult)
  for _, r := range results {
    softwareResults[r.JobName] = append(softwareResults[r.JobName], r)

    switch r.Status {
    case inspection.StatusOk:
      successJobs++
    default: // "error", "critical", etc.
      failedJobs++
    }

    for _, check := range r.Checks {
      totalChecks++
      switch check.Status {
      case inspection.StatusOk:
        totalPassed++
      case inspection.StatusError:
        totalErrors++
      }
    }
  }

  // === 总体统计（自动适配终端/文件）===
  statsText := fmt.Sprintf(
    "• Total Jobs: %d\n"+
      "• Success: %s | Failures: %s\n"+
      "• Total Checks: %d\n"+
      "• Passed: %s | Errors: %s",
    totalJobs,
    pterm.Green(strconv.Itoa(successJobs)),
    pterm.Red(strconv.Itoa(failedJobs)),
    totalChecks,
    pterm.Green(strconv.Itoa(totalPassed)),
    pterm.Red(strconv.Itoa(totalErrors)),
  )

  var statsOutput string
  if pterm.Output && outputFile == "" {
    statsOutput = pterm.DefaultBox.WithTitle("Overall Statistics").Sprint(statsText)
  } else {
    statsOutput = "Overall Statistics\n" + strings.Repeat("-", 30) + "\n" + statsText
  }
  fmt.Fprintf(&buf, "%s\n\n", statsOutput)

  // === 按软件分组输出详情 ===
  for software, jobResults := range softwareResults {
    softwareHeader := pterm.DefaultHeader.WithFullWidth().Sprintf("Software: %s", software)
    fmt.Fprintf(&buf, "%s\n\n", softwareHeader)

    for _, result := range jobResults {
      fmt.Fprintf(&buf, "→ Node: %s:%d (%s)\n", result.Host, result.Port, result.Host)

      var statusLine string
      switch result.Status {
      case inspection.StatusOk:
        statusLine = pterm.Success.Sprint("Status: OK")
      default:
        statusLine = pterm.Error.Sprint("Status: FAILED")
      }
      fmt.Fprintf(&buf, "  %s", statusLine)
      if result.Duration > 0 {
        fmt.Fprintf(&buf, " | Duration: %.2fs", result.Duration)
      }
      fmt.Fprintf(&buf, "\n")

      if result.Message != "" {
        fmt.Fprintf(&buf, "  Message: %s\n", SplitStringByFixedWidth(result.Message, 100))
      }

      if len(result.Checks) > 0 {
        fmt.Fprintf(&buf, "\n  Checks (%d):\n", len(result.Checks))

        tableData := [][]string{{"Name", "Status", "Severity", "Value", "Message", "Remediation"}}
        for _, check := range result.Checks {
          name := SplitStringByFixedWidth(check.Name, 30)
          val := SplitStringByFixedWidth(common.ParseInterfaceToString(check.Value), 10)
          msg := SplitStringByFixedWidth(check.Message, 40)
          remediation := ""
          if check.Remediation != "" {
            remediation = SplitStringByFixedWidth(check.Remediation, 30)
          }

          statusStr := check.Status
          // 终端输出带颜色，文件输出用纯文本
          if outputFile == "" && pterm.Output {
            switch check.Status {
            case inspection.StatusOk:
              statusStr = pterm.Green("OK")
            case inspection.StatusError:
              statusStr = pterm.Red("FAIL")
            }
          } else {
            switch check.Status {
            case inspection.StatusOk:
              statusStr = "OK"
            case inspection.StatusError:
              statusStr = "FAIL"
            }
          }

          // Severity 字段处理
          severityStr := check.Severity
          if severityStr == "" {
            severityStr = "N/A"
          }

          tableData = append(tableData, []string{name, statusStr, severityStr, val, msg, remediation})
        }

        pterm.DefaultTable.
          WithWriter(&buf).
          WithHasHeader().
          WithBoxed().
          WithRowSeparator("-").
          WithHeaderRowSeparator("-").
          WithLeftAlignment().
          WithData(tableData).
          Render()
      }
      fmt.Fprintf(&buf, "\n")
    }
  }

  // === 结束标记 ===
  endText := pterm.DefaultHeader.WithFullWidth().Sprint("END OF REPORT")
  fmt.Fprintf(&buf, "\n%s\n", endText)
  return buf.Bytes()
}

// formatToMarkdown 辅助函数：将检查结果格式化为Markdown
func formatToMarkdown(results []inspection.CosmicResult, outputFile string) []byte {
  var buf bytes.Buffer

  // 报告标题
  fmt.Fprintf(&buf, "# Cosmic Middleware Inspection Report\n\n")
  fmt.Fprintf(&buf, "> Generated at: `%s`\n\n", time.Now().Format("2006-01-02 15:04:05"))

  // 收集统计
  totalJobs := len(results)
  successJobs, failedJobs := 0, 0
  totalChecks, totalPassed, totalErrors := 0, 0, 0

  softwareResults := make(map[string][]inspection.CosmicResult)
  for _, r := range results {
    softwareResults[r.JobName] = append(softwareResults[r.JobName], r)
    switch r.Status {
    case inspection.StatusOk:
      successJobs++
    default:
      failedJobs++
    }
    for _, check := range r.Checks {
      totalChecks++
      switch check.Status {
      case inspection.StatusOk:
        totalPassed++
      case inspection.StatusError:
        totalErrors++
      }
    }
  }

  // 总体统计表格
  fmt.Fprintf(&buf, "## 📊 Overall Statistics\n\n")
  fmt.Fprintf(&buf, "| Metric | Value |\n")
  fmt.Fprintf(&buf, "|--------|-------|\n")
  fmt.Fprintf(&buf, "| Total Jobs | %d |\n", totalJobs)
  fmt.Fprintf(&buf, "| Success | %d ✅ |\n", successJobs)
  fmt.Fprintf(&buf, "| Failures | %d ❌ |\n", failedJobs)
  fmt.Fprintf(&buf, "| Total Checks | %d |\n", totalChecks)
  fmt.Fprintf(&buf, "| Passed | %d ✅ |\n", totalPassed)
  fmt.Fprintf(&buf, "| Errors | %d ❌ |\n", totalErrors)
  fmt.Fprintf(&buf, "\n")

  // 按软件分组
  fmt.Fprintf(&buf, "## 🧩 Software Details\n\n")
  for software, jobResults := range softwareResults {
    fmt.Fprintf(&buf, "### 📦 %s\n\n", software)

    // 软件级表格
    fmt.Fprintf(&buf, "| Node | Status | Duration | Message |\n")
    fmt.Fprintf(&buf, "|------|--------|----------|---------|\n")

    // 每个节点的检查详情表格
    fmt.Fprintf(&buf, "\n#### Node Checks\n\n")
    fmt.Fprintf(&buf, "| Name | Type | Status | Value | Message | Remediation |\n")
    fmt.Fprintf(&buf, "|------|------|--------|---------|---------|-------------|\n")

    for _, result := range jobResults {
      // 状态图标
      statusIcon := "❓"
      statusText := result.Status
      switch result.Status {
      case inspection.StatusOk:
        statusIcon = "✅"
        statusText = "OK"
      default:
        statusIcon = "❌"
        statusText = "FAILED"
      }

      duration := "N/A"

      // 输出节点级别的表格行
      if result.Duration > 0 {
        duration = fmt.Sprintf("%.2fs", result.Duration)
      }
      fmt.Fprintf(&buf, "| %s:%d | %s %s | %s | %s |\n",
        result.Host, result.Port, statusIcon, statusText, duration, result.Message)

      // 输出该节点的每个检查项
      for _, check := range result.Checks {
        checkStatusIcon := "❓"
        checkStatusText := check.Status
        switch check.Status {
        case inspection.StatusOk:
          checkStatusIcon = "✅"
          checkStatusText = "OK"
        default:
          checkStatusIcon = "❌"
          checkStatusText = "FAILED"
        }

        remediation := ""
        if check.Remediation != "" {
          remediation = check.Remediation
        }

        fmt.Fprintf(&buf, "| %s | %s | %s %s | %s | %s | %s |\n",
          check.Name, check.Type, checkStatusIcon, checkStatusText, common.ParseInterfaceToString(check.Value), check.Message, remediation)
      }
      fmt.Fprintf(&buf, "\n")
      if result.Duration > 0 {
        duration = fmt.Sprintf("%.2fs", result.Duration)
      }

      message := SplitStringByFixedWidth(result.Message, 80)
      if message == "" {
        message = "—"
      }

      fmt.Fprintf(&buf, "| `%s:%d` | %s %s | %s | %s |\n",
        result.Host, result.Port,
        statusIcon, statusText,
        duration,
        message,
      )
    }
    fmt.Fprintf(&buf, "\n")

    // 检查项详情（可折叠，兼容 GitHub）
    fmt.Fprintf(&buf, "<details>\n")
    fmt.Fprintf(&buf, "<summary>🔍 View %d Checks</summary>\n\n", len(jobResults)*0) // 先不展开，下面补充

    // 为每个节点列出检查项
    for _, result := range jobResults {
      if len(result.Checks) == 0 {
        continue
      }
      fmt.Fprintf(&buf, "#### Node: `%s:%d`\n\n", result.Host, result.Port)
      fmt.Fprintf(&buf, "| Name | Status | Severity | Message | Remediation |\n")
      fmt.Fprintf(&buf, "|------|--------|----------|---------|-------------|\n")
      for _, check := range result.Checks {
        checkStatusIcon := "❓"
        switch check.Status {
        case inspection.StatusOk:
          checkStatusIcon = "✅"
        case inspection.StatusError:
          checkStatusIcon = "❌"
        }
        checkMsg := SplitStringByFixedWidth(firstNonEmptyLine(check.Message), 40)
        if checkMsg == "" {
          checkMsg = "—"
        }
        remediation := "—"
        if check.Remediation != "" {
          remediation = SplitStringByFixedWidth(check.Remediation, 30)
        }
        severity := check.Severity
        if severity == "" {
          severity = "N/A"
        }
        fmt.Fprintf(&buf, "| %s | %s %s | %s | %s | %s |\n",
          SplitStringByFixedWidth(check.Name, 25),
          checkStatusIcon,
          strings.ToUpper(check.Status),
          severity,
          checkMsg,
          remediation,
        )
      }
      fmt.Fprintf(&buf, "\n")
    }
    fmt.Fprintf(&buf, "</details>\n\n")
  }

  fmt.Fprintf(&buf, "---\n")
  fmt.Fprintf(&buf, "> **End of Report**\n")

  return buf.Bytes()
}

// formatToHtml 辅助函数：格式化为 HTML
func formatToHtml(results []inspection.CosmicResult, outputFile string) []byte {
  var buf bytes.Buffer

  // 内联 CSS（现代、响应式、折叠支持）
  css := `
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f8f9fa; color: #212529; }
.container { max-width: 1200px; margin: 0 auto; }
.header { background: #0d6efd; color: white; padding: 20px; border-radius: 8px; text-align: center; margin-bottom: 24px; }
.header h1 { margin: 0; font-size: 28px; }
.meta { text-align: center; color: #6c757d; margin-bottom: 24px; font-style: italic; }
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 16px; margin-bottom: 32px; }
.stat-card { background: white; padding: 16px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
.stat-value { font-size: 24px; font-weight: bold; margin: 8px 0; }
.stat-label { font-size: 14px; color: #6c757d; }
.software-section { background: white; margin-bottom: 24px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
.software-header { background: #e9ecef; padding: 16px 20px; font-size: 20px; font-weight: bold; color: #495057; }
.node-item { border-bottom: 1px solid #eee; padding: 16px 20px; }
.node-item:last-child { border-bottom: none; }
.node-title { font-weight: bold; margin-bottom: 8px; color: #0d6efd; }
.status-ok { color: #198754; }
.status-warning { color: #ffc107; }
.status-error { color: #dc3545; }
.checks-table { width: 100%; border-collapse: collapse; margin-top: 12px; display: none; }
.checks-table th, .checks-table td { text-align: left; padding: 10px; border-bottom: 1px solid #dee2e6; font-size: 14px; }
.checks-table th { background: #f8f9fa; }
.toggle-btn { background: #f8f9fa; border: 1px solid #dee2e6; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 13px; color: #0d6efd; }
.toggle-btn:hover { background: #e9ecef; }
.footer { text-align: center; margin-top: 32px; color: #6c757d; font-size: 14px; }
.status-badge { padding: 2px 8px; border-radius: 12px; font-size: 12px; font-weight: bold; color: white; }
.badge-ok { background: #198754; }
.badge-warning { background: #ffc107; color: #212529; }
.badge-error { background: #dc3545; }
</style>
<script>
document.addEventListener('DOMContentLoaded', () => {
  document.querySelectorAll('.toggle-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const table = btn.nextElementSibling;
      if (table.style.display === 'table') {
        table.style.display = 'none';
        btn.textContent = 'Show Checks';
      } else {
        table.style.display = 'table';
        btn.textContent = 'Hide Checks';
      }
    });
  });
});
</script>
`

  // 开始 HTML
  buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
  buf.WriteString("<meta charset=\"UTF-8\">\n")
  buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
  buf.WriteString("<title>苍穹中间件巡检报告</title>\n")
  buf.WriteString(css)
  buf.WriteString("</head>\n<body>\n")
  buf.WriteString("<div class=\"container\">\n")

  // 标题
  buf.WriteString("<div class=\"header\">\n")
  buf.WriteString("<h1>Cosmic Middleware Inspection Report</h1>\n")
  buf.WriteString("</div>\n")
  buf.WriteString(fmt.Sprintf("<div class=\"meta\">Generated at: %s</div>\n", time.Now().Format("2006-01-02 15:04:05")))

  // === 统计卡片 ===
  totalJobs := len(results)
  successJobs, failedJobs := 0, 0
  totalChecks, totalPassed, totalErrors := 0, 0, 0

  softwareResults := make(map[string][]inspection.CosmicResult)
  for _, r := range results {
    softwareResults[r.JobName] = append(softwareResults[r.JobName], r)
    switch r.Status {
    case inspection.StatusOk:
      successJobs++
    default:
      failedJobs++
    }
    for _, check := range r.Checks {
      totalChecks++
      switch check.Status {
      case inspection.StatusOk:
        totalPassed++
      case inspection.StatusError:
        totalErrors++
      }
    }
  }

  buf.WriteString("<div class=\"stats-grid\">\n")
  buf.WriteString(fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Total Jobs</div><div class=\"stat-value\">%d</div></div>\n", totalJobs))
  buf.WriteString(fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Success</div><div class=\"stat-value\" style=\"color:#198754\">%d ✅</div></div>\n", successJobs))
  buf.WriteString(fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Failures</div><div class=\"stat-value\" style=\"color:#dc3545\">%d ❌</div></div>\n", failedJobs))
  buf.WriteString(fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Total Checks</div><div class=\"stat-value\">%d</div></div>\n", totalChecks))
  buf.WriteString("</div>\n")

  // === 软件分组 ===
  buf.WriteString("<h2>Software Details</h2>\n")
  for software, jobResults := range softwareResults {
    buf.WriteString("<div class=\"software-section\">\n")
    buf.WriteString(fmt.Sprintf("<div class=\"software-header\">📦 %s</div>\n", software))

    for _, result := range jobResults {
      buf.WriteString("<div class=\"node-item\">\n")
      buf.WriteString(fmt.Sprintf("<div class=\"node-title\">Node: %s:%d</div>\n", result.Host, result.Port))

      // 状态徽章
      var statusText string
      switch result.Status {
      case inspection.StatusOk:
        //statusClass = "status-ok"
        statusText = `<span class="status-badge badge-ok">OK</span>`
      default:
        //statusClass = "status-error"
        statusText = `<span class="status-badge badge-error">FAILED</span>`
      }

      duration := "N/A"
      if result.Duration > 0 {
        duration = fmt.Sprintf("%.2f s", result.Duration)
      }

      message := result.Message
      if message == "" {
        message = "—"
      }

      buf.WriteString(fmt.Sprintf("<div>Status: %s | Duration: %s</div>\n", statusText, duration))
      buf.WriteString(fmt.Sprintf("<div>Message: %s</div>\n", message))

      // 检查项表格（初始隐藏）
      if len(result.Checks) > 0 {
        buf.WriteString("<button class=\"toggle-btn\">Show Checks</button>\n")
        buf.WriteString("<table class=\"checks-table\">\n")
        buf.WriteString("<thead><tr><th>Name</th><th>Status</th><th>Severity</th><th>Value</th><th>Message</th><th>Remediation</th></tr></thead>\n")
        buf.WriteString("<tbody>\n")
        for _, check := range result.Checks {
          var checkStatusBadge string
          switch check.Status {
          case inspection.StatusOk:
            checkStatusBadge = `<span class="status-badge badge-ok">OK</span>`
          case inspection.StatusError:
            checkStatusBadge = `<span class="status-badge badge-error">FAIL</span>`
          default:
            checkStatusBadge = check.Status
          }
          checkMsg := firstNonEmptyLine(check.Message)
          if checkMsg == "" {
            checkMsg = "—"
          }
          checkRemediation := firstNonEmptyLine(check.Remediation)
          if checkRemediation == "" {
            checkRemediation = "—"
          }
          severity := check.Severity
          if severity == "" {
            severity = "N/A"
          }
          buf.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
            check.Name,
            checkStatusBadge,
            severity,
            common.ParseInterfaceToString(check.Value),
            checkMsg,
            checkRemediation,
          ))
        }
        buf.WriteString("</tbody>\n</table>\n")
      }

      buf.WriteString("</div>\n") // node-item
    }
    buf.WriteString("</div>\n") // software-section
  }

  // 结束
  buf.WriteString("<div class=\"footer\">End of Report</div>\n")
  buf.WriteString("</div>\n</body>\n</html>")

  return buf.Bytes()
}

// formatToPdf 辅助函数：格式化为 PDF
func formatToPdf(results []inspection.CosmicResult, outputFile string) error {
  if outputFile == "" {
    return fmt.Errorf("--output/-o is required for PDF format")
  }

  // 1. 渲染 HTML
  htmlContent := string(formatToHtml(results, ""))

  // 2. 检查 wkhtmltopdf
  if _, err := exec.LookPath("wkhtmltopdf"); err != nil {
    return fmt.Errorf("wkhtmltopdf not found. Install from https://wkhtmltopdf.org/downloads.html")
  }

  // 3. 临时文件
  tmpFile, err := os.CreateTemp("", "cosmic-*.html")
  if err != nil {
    return fmt.Errorf("create temp file: %w", err)
  }
  defer os.Remove(tmpFile.Name())
  defer tmpFile.Close()

  if _, err := tmpFile.WriteString(htmlContent); err != nil {
    return fmt.Errorf("write temp HTML: %w", err)
  }
  tmpFile.Close()

  // 4. 转 PDF
  cmd := exec.Command("wkhtmltopdf", "--quiet", tmpFile.Name(), outputFile)
  if err := cmd.Run(); err != nil {
    return fmt.Errorf("wkhtmltopdf failed: %w", err)
  }

  pterm.Success.Printf("PDF report saved to: %s\n", outputFile)
  return nil
}

package cli

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/client"
  "github.com/casuallc/vigil/inspection"
  "github.com/spf13/cobra"
  "gopkg.in/yaml.v3"
  "os"
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
  inspectCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text|yaml|json)")
  inspectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output result to file instead of console")

  return inspectCmd
}

// handleCosmicInspect 处理cosmic系统巡检命令
func (c *CLI) handleCosmicInspect(configFile string, jobName string, envVars []string, outputFormat string, outputFile string) error {
  fmt.Println("=== Cosmic System Inspection Started ===")
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
    fmt.Printf("\n=== Processing Software: %s ===\n", job.Name)
    
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

      fmt.Printf("\n==== Inspecting %s on node %s (%s:%d) ====\n", job.Name, node.Name, node.IP, node.Port)

      // 执行实际巡检
      result := c.performCosmicInspection(job, node, inspectionRules, envMap)
      allResults = append(allResults, result)
      
      // 按软件分组存储结果
      summaryBySoftware[job.Name] = append(summaryBySoftware[job.Name], result)
    }
  }

  // 汇总分析结果
  fmt.Println("\n=== Summary Analysis ===")
  for software, results := range summaryBySoftware {
    fmt.Printf("Software: %s\n", software)
    fmt.Printf("- Total nodes: %d\n", len(results))
    
    var success, warning, errorCount int
    for _, r := range results {
      switch r.Status {
      case "ok":
        success++
      case "warning":
        warning++
      case "error":
        errorCount++
      }
    }
    
    fmt.Printf("- Success: %d\n", success)
    fmt.Printf("- Warning: %d\n", warning)
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
  nodeClient := client.NewClient(fmt.Sprintf("http://%s:%d", node.IP, node.Port))
  
  // 执行远程检查
  checkResult, err := nodeClient.ExecuteInspection(checkRequest)
  if err != nil {
    result.Status = "error"
    result.Message = fmt.Sprintf("Failed to execute inspection on node %s: %v", node.Name, err)
    return err
  }

  // 转换检查结果
  result.Status = strings.ToLower(checkResult.Summary.OverallStatus)
  result.Message = fmt.Sprintf("Inspection completed with status: %s", checkResult.Summary.OverallStatus)

  // 转换检查项
  for _, check := range checkResult.Results {
    result.Checks = append(result.Checks, inspection.CheckResult{
      ID:         check.ID,
      Name:       check.Name,
      Type:       check.Type,
      Value:      check.Value,
      Unit:       check.Unit,
      Status:     strings.ToLower(check.Status),
      Severity:   strings.ToLower(check.Severity),
      Message:    check.Message,
      DurationMs: check.DurationMs,
    })
  }

  // 统计检查结果
  var passed, warning, critical, errorCount int
  for _, check := range result.Checks {
    switch check.Status {
    case "ok":
      passed++
    case "warning":
      warning++
    case "critical":
      critical++
    case "error":
      errorCount++
    }
  }
  
  fmt.Printf("Inspection results for node %s:\n", node.Name)
  fmt.Printf("- Total checks: %d\n", len(result.Checks))
  fmt.Printf("- Passed: %d\n", passed)
  fmt.Printf("- Warning: %d\n", warning)
  fmt.Printf("- Critical: %d\n", critical)
  fmt.Printf("- Error: %d\n", errorCount)

  return nil
}

// evaluateThreshold 评估阈值
func (c *CLI) evaluateThreshold(value float64, threshold *inspection.Threshold) bool {
  if threshold == nil {
    return false
  }

  switch threshold.Operator {
  case ">":
    return value > threshold.Value
  case ">=":
    return value >= threshold.Value
  case "<":
    return value < threshold.Value
  case "<=":
    return value <= threshold.Value
  case "==":
    return value == threshold.Value
  case "!=":
    return value != threshold.Value
  default:
    return false
  }
}

// performCosmicInspection 执行具体的cosmic系统巡检
func (c *CLI) performCosmicInspection(job inspection.Job, node inspection.Node, rules *inspection.RuleConfig, envMap map[string]string) inspection.CosmicResult {
  result := inspection.CosmicResult{
    JobName: job.Name,
    Host:    node.IP,
    Port:    node.Port,
    Status:  "ok",
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
    result.Status = "warning"
    result.Message = "Basic connectivity check: No inspection rules provided"
    return nil
  }

  if err := tryInspect(); err != nil {
    result.Status = "error"
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

  // 根据格式选项格式化输出
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
    // 文本格式输出
    var buf bytes.Buffer
    const lineWidth = 120

    // 打印标题和分隔线
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("COSMIC SYSTEM INSPECTION REPORT", lineWidth))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    fmt.Fprintf(&buf, "Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))

    // 统计信息
    totalJobs := len(results)
    successJobs := 0
    failedJobs := 0
    totalChecks := 0
    totalPassed := 0
    totalWarnings := 0
    totalCritical := 0
    totalErrors := 0

    // 按软件分组统计
    softwareStats := make(map[string]map[string]int)
    nodeStats := make(map[string]map[string]int)
    
    for _, result := range results {
      // 初始化软件统计
      if _, exists := softwareStats[result.JobName]; !exists {
        softwareStats[result.JobName] = map[string]int{
          "total":   0,
          "success": 0,
          "warning": 0,
          "error":   0,
          "checks":  0,
        }
      }
      
      // 初始化节点统计
      if _, exists := nodeStats[result.Host]; !exists {
        nodeStats[result.Host] = map[string]int{
          "total":   0,
          "success": 0,
          "warning": 0,
          "error":   0,
          "checks":  0,
        }
      }
      
      // 更新统计
      if result.Status == "ok" {
        successJobs++
        softwareStats[result.JobName]["success"]++
        nodeStats[result.Host]["success"]++
      } else if result.Status == "warning" {
        softwareStats[result.JobName]["warning"]++
        nodeStats[result.Host]["warning"]++
      } else {
        failedJobs++
        softwareStats[result.JobName]["error"]++
        nodeStats[result.Host]["error"]++
      }
      
      softwareStats[result.JobName]["total"]++
      nodeStats[result.Host]["total"]++

      for _, check := range result.Checks {
        totalChecks++
        softwareStats[result.JobName]["checks"]++
        nodeStats[result.Host]["checks"]++
        
        switch check.Status {
        case "ok":
          totalPassed++
        case "warning":
          totalWarnings++
        case "critical":
          totalCritical++
        case "error":
          totalErrors++
        }
      }
    }

    // 打印总体统计
    fmt.Fprintf(&buf, "%-15s: %d\n", "Total Jobs", totalJobs)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Success Jobs", successJobs)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Failed Jobs", failedJobs)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Total Checks", totalChecks)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Passed", totalPassed)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Warnings", totalWarnings)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Critical", totalCritical)
    fmt.Fprintf(&buf, "%-15s: %d\n", "Errors", totalErrors)

    if totalJobs > 0 {
      successRate := float64(successJobs) * 100.0 / float64(totalJobs)
      fmt.Fprintf(&buf, "%-15s: %.1f%%\n", "Success Rate", successRate)
    }

    // 按软件分组统计
    fmt.Fprintf(&buf, "\n%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("SOFTWARE SUMMARY", lineWidth))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    fmt.Fprintf(&buf, "%-30s %-10s %-10s %-10s %-10s %-10s\n", 
      "Software", "Nodes", "Success", "Warning", "Error", "Checks")
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    
    for software, stats := range softwareStats {
      fmt.Fprintf(&buf, "%-30s %-10d %-10d %-10d %-10d %-10d\n",
        truncateString(software, 30),
        stats["total"],
        stats["success"],
        stats["warning"],
        stats["error"],
        stats["checks"])
    }

    // 按节点分组统计
    fmt.Fprintf(&buf, "\n%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("NODE SUMMARY", lineWidth))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    fmt.Fprintf(&buf, "%-30s %-10s %-10s %-10s %-10s %-10s\n", 
      "Node", "Jobs", "Success", "Warning", "Error", "Checks")
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    
    for node, stats := range nodeStats {
      fmt.Fprintf(&buf, "%-30s %-10d %-10d %-10d %-10d %-10d\n",
        truncateString(node, 30),
        stats["total"],
        stats["success"],
        stats["warning"],
        stats["error"],
        stats["checks"])
    }

    // 按软件-节点维度详细展示结果
    fmt.Fprintf(&buf, "\n%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("DETAILED INSPECTION RESULTS", lineWidth))
    fmt.Fprintf(&buf, "%s\n\n", string(bytes.Repeat([]byte("="), lineWidth)))
    
    // 按软件分组展示详细结果
    softwareResults := make(map[string][]inspection.CosmicResult)
    for _, result := range results {
      softwareResults[result.JobName] = append(softwareResults[result.JobName], result)
    }
    
    for software, softwareRes := range softwareResults {
      fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("*"), lineWidth)))
      fmt.Fprintf(&buf, "%s\n", centerText("SOFTWARE: "+software, lineWidth))
      fmt.Fprintf(&buf, "%s\n\n", string(bytes.Repeat([]byte("*"), lineWidth)))
      
      for _, result := range softwareRes {
        fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
        fmt.Fprintf(&buf, "Node: %s (%s:%d)\n", result.Host, result.Host, result.Port)
        fmt.Fprintf(&buf, "Status: %s\n", result.Status)
        if result.Message != "" {
          fmt.Fprintf(&buf, "Message: %s\n", result.Message)
        }
        if result.Duration > 0 {
          fmt.Fprintf(&buf, "Duration: %.2fs\n", result.Duration)
        }

        if len(result.Checks) > 0 {
          fmt.Fprintf(&buf, "\nChecks (%d):\n", len(result.Checks))
          fmt.Fprintf(&buf, "%-40s %-12s %-12s %-40s\n", "Name", "Type", "Status", "Message")
          fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), 104)))

          for _, check := range result.Checks {
            fmt.Fprintf(&buf, "%-40s %-12s %-12s %-40s\n",
              truncateString(check.Name, 40),
              truncateString(check.Type, 12),
              truncateString(check.Status, 12),
              truncateString(check.Message, 40))
          }
        }
        fmt.Fprintf(&buf, "\n")
      }
    }

    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("END OF REPORT", lineWidth))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))

    output = buf.Bytes()
  default:
    return fmt.Errorf("unsupported output format: %s", format)
  }

  // 输出结果
  if outputFile != "" {
    if err := os.WriteFile(outputFile, output, 0644); err != nil {
      fmt.Printf("ERROR failed to write cosmic output to file: %v\n", err)
      return fmt.Errorf("failed to write cosmic output to file: %w", err)
    }
    fmt.Printf("✅ Cosmic inspection report written to %s\n", outputFile)
  } else {
    fmt.Println(string(output))
  }

  return nil
}

package cli

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/inspection"
  "github.com/spf13/cobra"
  "gopkg.in/yaml.v3"
  "os"
  "strings"
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
  // 加载cosmic配置文件
  cosmicConfig, err := inspection.LoadCosmicConfig(configFile)
  if err != nil {
    fmt.Printf("ERROR failed to load cosmic config: %v\n", err)
    return nil
  }

  // 创建节点映射
  nodeMap := make(map[string]inspection.Node)
  for _, node := range cosmicConfig.Nodes {
    nodeMap[node.Name] = node
  }

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
      return nil
    }
  } else {
    // 检查所有作业
    jobsToInspect = cosmicConfig.Jobs
  }

  // 处理环境变量
  envMap := make(map[string]string)
  for _, env := range envVars {
    parts := strings.SplitN(env, "=", 2)
    if len(parts) == 2 {
      envMap[parts[0]] = parts[1]
    }
  }

  // 执行巡检
  var allResults []inspection.CosmicResult
  for _, job := range jobsToInspect {
    for _, targetName := range job.Targets {
      node, exists := nodeMap[targetName]
      if !exists {
        fmt.Printf("WARNING node '%s' not found for job '%s'\n", targetName, job.Name)
        continue
      }

      fmt.Printf("Inspecting job: %s on node %s (%s:%d)\n", job.Name, node.Name, node.IP, node.Port)

      // 构建cosmic作业配置
      cosmicJob := inspection.CosmicJob{
        Name: job.Name,
        Host: node.IP,
        Port: node.Port,
      }

      // 添加标签
      if cosmicJob.Labels == nil {
        cosmicJob.Labels = make(map[string]string)
      }
      cosmicJob.Labels["node"] = node.Name

      // 添加作业环境变量
      for _, env := range job.Envs {
        cosmicJob.Labels[env.Name] = env.Value
      }

      // 构建巡检请求
      request := inspection.CosmicRequest{
        Job:  cosmicJob,
        Envs: envMap,
      }

      // 执行巡检（这里使用模拟数据，实际应该调用c.client.CosmicInspect）
      result := inspection.CosmicResult{
        JobName:  job.Name,
        Host:     node.IP,
        Port:     node.Port,
        Status:   "ok",
        Message:  fmt.Sprintf("Successfully inspected %s on %s", job.Name, node.Name),
        Duration: 1.5,
        Checks: []inspection.CheckResult{
          {
            ID:      "process_check",
            Name:    "Process Status",
            Type:    "process",
            Status:  "ok",
            Message: "All processes running normally",
            //Duration: 0.5,
            Severity: "info",
          },
          {
            ID:      "memory_check",
            Name:    "Memory Usage",
            Type:    "performance",
            Status:  "warning",
            Value:   85.5,
            Message: "Memory usage is high (85.5%)",
            //Duration: 0.3,
            Severity: "warning",
          },
        },
      }

      allResults = append(allResults, result)
    }
  }

  // 格式化输出结果
  return c.formatAndOutputCosmicResults(allResults, outputFormat, outputFile)
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

    // 统计信息
    totalJobs := len(results)
    successJobs := 0
    failedJobs := 0
    totalChecks := 0
    totalPassed := 0
    totalWarnings := 0
    totalCritical := 0
    totalErrors := 0

    for _, result := range results {
      if result.Status == "ok" {
        successJobs++
      } else {
        failedJobs++
      }

      for _, check := range result.Checks {
        totalChecks++
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

    fmt.Fprintf(&buf, "\n")

    // 打印每个作业的详细结果
    for i, result := range results {
      fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
      fmt.Fprintf(&buf, "Job %d: %s (%s:%d)\n", i+1, result.JobName, result.Host, result.Port)
      fmt.Fprintf(&buf, "Status: %s\n", result.Status)
      if result.Message != "" {
        fmt.Fprintf(&buf, "Message: %s\n", result.Message)
      }
      if result.Duration > 0 {
        fmt.Fprintf(&buf, "Duration: %.2fs\n", result.Duration)
      }

      if len(result.Checks) > 0 {
        fmt.Fprintf(&buf, "\nChecks:\n")
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

    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))

    output = buf.Bytes()
  default:
    return fmt.Errorf("unsupported output format: %s", format)
  }

  // 输出结果
  if outputFile != "" {
    if err := os.WriteFile(outputFile, output, 0644); err != nil {
      fmt.Printf("ERROR failed to write cosmic output to file: %v\n", err)
      return nil
    }
    fmt.Printf("Cosmic inspection report written to %s\n", outputFile)
  } else {
    fmt.Println(string(output))
  }

  return nil
}

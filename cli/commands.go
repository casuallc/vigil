package cli

import (
  "fmt"
  "github.com/casuallc/vigil/client"
  "github.com/spf13/cobra"
)

// setupCommands configures and returns the root command with all subcommands
func (c *CLI) setupCommands() *cobra.Command {
  // Root command
  var apiHost string

  rootCmd := &cobra.Command{
    Use:     "vigil",
    Short:   "Process Management Program",
    Long:    "Vigil - A powerful process management and monitoring tool",
    Version: "1.0.0",
  }

  // Global flag for API host
  rootCmd.PersistentFlags().StringVarP(&apiHost, "host", "H", "http://localhost:8080", "API server host address")

  // Override PreRun to create client with the provided host
  rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Only create a new client if we're not in the root command
    if cmd != rootCmd {
      c.client = client.NewClient(apiHost)
    }
    return nil
  }

  // Scan command
  var query string
  var registerAfterScan bool
  scanCmd := &cobra.Command{
    Use:   "scan",
    Short: "Scan processes",
    Long:  "Scan system processes based on query string or regex",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleScan(query, registerAfterScan)
    },
  }
  scanCmd.Flags().StringVarP(&query, "query", "q", "", "Search query string or regex")
  scanCmd.Flags().BoolVarP(&registerAfterScan, "register", "r", false, "Register a process after scanning")
  scanCmd.MarkFlagRequired("query")

  // Manage command
  var (
    processName string
    commandPath string
  )
  manageCmd := &cobra.Command{
    Use:   "manage",
    Short: "Manage process",
    Long:  "Manage an existing process or create a new managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleManage(processName, commandPath)
    },
  }
  manageCmd.Flags().StringVarP(&processName, "name", "n", "", "Process name")
  manageCmd.Flags().StringVarP(&commandPath, "command", "c", "", "Command path")
  manageCmd.MarkFlagRequired("name")

  // Start command
  var startName string
  startCmd := &cobra.Command{
    Use:   "start",
    Short: "Start process",
    Long:  "Start a managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleStart(startName)
    },
  }
  startCmd.Flags().StringVarP(&startName, "name", "n", "", "Process name")
  startCmd.MarkFlagRequired("name")

  // Stop command
  var stopName string
  stopCmd := &cobra.Command{
    Use:   "stop",
    Short: "Stop process",
    Long:  "Stop a managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleStop(stopName)
    },
  }
  stopCmd.Flags().StringVarP(&stopName, "name", "n", "", "Process name")
  stopCmd.MarkFlagRequired("name")

  // Restart command
  var restartName string
  restartCmd := &cobra.Command{
    Use:   "restart",
    Short: "Restart process",
    Long:  "Restart a managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRestart(restartName)
    },
  }
  restartCmd.Flags().StringVarP(&restartName, "name", "n", "", "Process name")
  restartCmd.MarkFlagRequired("name")

  // List command
  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List processes",
    Long:  "List all managed processes",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleList()
    },
  }

  // Status command
  var statusName string
  statusCmd := &cobra.Command{
    Use:   "status",
    Short: "Check process status",
    Long:  "Check the status of a managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleStatus(statusName)
    },
  }
  statusCmd.Flags().StringVarP(&statusName, "name", "n", "", "Process name")
  statusCmd.MarkFlagRequired("name")

  // Resource commands
  systemResourceCmd := &cobra.Command{
    Use:   "system-resources",
    Short: "Get system resources",
    Long:  "Get system resource usage information",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleSystemResources()
    },
  }

  processResourceCmd := &cobra.Command{
    Use:   "process-resources",
    Short: "Get process resources",
    Long:  "Get resource usage information for a specific process",
    RunE: func(cmd *cobra.Command, args []string) error {
      pid := 0
      if len(args) > 0 {
        fmt.Sscanf(args[0], "%d", &pid)
      }
      return c.handleProcessResources(pid)
    },
  }

  // Config commands
  getConfigCmd := &cobra.Command{
    Use:   "get-config",
    Short: "Get configuration",
    Long:  "Get the current configuration",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleGetConfig()
    },
  }

  // Get command - 新增支持YAML格式输出的get命令
  var getFormat string
  var getProcessName string
  getCmd := &cobra.Command{
    Use:   "get",
    Short: "Get process details",
    Long:  "Get detailed information about a managed process",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleGet(getProcessName, getFormat)
    },
  }
  getCmd.Flags().StringVarP(&getProcessName, "name", "n", "", "Process name")
  getCmd.Flags().StringVarP(&getFormat, "format", "f", "yaml", "Output format (yaml|text)")
  getCmd.MarkFlagRequired("name")

  // Add commands to root command
  rootCmd.AddCommand(scanCmd)
  rootCmd.AddCommand(manageCmd)
  rootCmd.AddCommand(startCmd)
  rootCmd.AddCommand(stopCmd)
  rootCmd.AddCommand(restartCmd)
  rootCmd.AddCommand(listCmd)
  rootCmd.AddCommand(statusCmd)
  rootCmd.AddCommand(systemResourceCmd)
  rootCmd.AddCommand(processResourceCmd)
  rootCmd.AddCommand(getConfigCmd)
  rootCmd.AddCommand(getCmd) // 添加新命令到根命令

  return rootCmd
}

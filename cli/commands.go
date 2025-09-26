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

  // Global flags
  rootCmd.PersistentFlags().StringVarP(&apiHost, "host", "H", "http://localhost:8080", "API server host address")
  // namespace不再作为全局变量定义，而是在各命令中处理

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
  var scanNamespace string
  scanCmd := &cobra.Command{
    Use:   "scan",
    Short: "Scan processes",
    Long:  "Scan system processes based on query string or regex",
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleScan(query, registerAfterScan, namespaceFlag)
    },
  }
  scanCmd.Flags().StringVarP(&query, "query", "q", "", "Search query string or regex")
  scanCmd.Flags().BoolVarP(&registerAfterScan, "register", "r", false, "Register a process after scanning")
  scanCmd.Flags().StringVarP(&scanNamespace, "namespace", "n", "default", "Process namespace")
  scanCmd.MarkFlagRequired("query")

  // Manage command
  var processName string
  var commandPath string
  var manageNamespace string
  manageCmd := &cobra.Command{
    Use:   "manage [name]",
    Short: "Manage process",
    Long:  "Manage an existing process or create a new managed process",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      // 如果提供了位置参数，使用位置参数作为name
      if len(args) > 0 && processName == "" {
        processName = args[0]
      }
      return c.handleManage(processName, commandPath, namespaceFlag)
    },
  }
  manageCmd.Flags().StringVarP(&processName, "name", "N", "", "Process name (alternative to positional argument)")
  manageCmd.Flags().StringVarP(&commandPath, "command", "c", "", "Command path")
  manageCmd.Flags().StringVarP(&manageNamespace, "namespace", "n", "default", "Process namespace")

  // Start command
  var startNamespace string
  startCmd := &cobra.Command{
    Use:   "start [name]",
    Short: "Start process",
    Long:  "Start a managed process",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleStart(args[0], namespaceFlag)
    },
  }
  startCmd.Flags().StringVarP(&startNamespace, "namespace", "n", "default", "Process namespace")

  // Stop command
  var stopNamespace string
  stopCmd := &cobra.Command{
    Use:   "stop [name]",
    Short: "Stop process",
    Long:  "Stop a managed process",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleStop(args[0], namespaceFlag)
    },
  }
  stopCmd.Flags().StringVarP(&stopNamespace, "namespace", "n", "default", "Process namespace")

  // Restart command
  var restartNamespace string
  restartCmd := &cobra.Command{
    Use:   "restart [name]",
    Short: "Restart process",
    Long:  "Restart a managed process",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleRestart(args[0], namespaceFlag)
    },
  }
  restartCmd.Flags().StringVarP(&restartNamespace, "namespace", "n", "default", "Process namespace")

  // List command
  var listNamespace string
  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List processes",
    Long:  "List all managed processes",
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleList(namespaceFlag)
    },
  }
  listCmd.Flags().StringVarP(&listNamespace, "namespace", "n", "default", "Process namespace")

  // Status command
  var statusNamespace string
  statusCmd := &cobra.Command{
    Use:   "status [name]",
    Short: "Check process status",
    Long:  "Check the status of a managed process",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleStatus(args[0], namespaceFlag)
    },
  }
  statusCmd.Flags().StringVarP(&statusNamespace, "namespace", "n", "default", "Process namespace")

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
    Use:   "process-resources [pid]",
    Short: "Get process resources",
    Long:  "Get resource usage information for a specific process",
    Args:  cobra.MaximumNArgs(1),
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
  var getNamespace string
  getCmd := &cobra.Command{
    Use:   "get [name]",
    Short: "Get process details",
    Long:  "Get detailed information about a managed process",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      formatFlag, _ := cmd.Flags().GetString("format")
      return c.handleGet(args[0], formatFlag, namespaceFlag)
    },
  }
  getCmd.Flags().StringVarP(&getFormat, "format", "f", "yaml", "Output format (yaml|text)")
  getCmd.Flags().StringVarP(&getNamespace, "namespace", "n", "default", "Process namespace")

  // Delete command
  var deleteNamespace string
  deleteCmd := &cobra.Command{
    Use:   "delete [name]",
    Short: "Delete a managed process",
    Long:  "Delete a process from the managed list. If the process is running, it will be stopped first.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      return c.handleDelete(args[0], namespaceFlag)
    },
  }
  deleteCmd.Flags().StringVarP(&deleteNamespace, "namespace", "n", "default", "Process namespace")

  // Add all commands to root command
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
  rootCmd.AddCommand(deleteCmd)
  rootCmd.AddCommand(getCmd)

  return rootCmd
}

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
    Long:    "Vigil - A powerful proc management and monitoring tool",
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
  scanCmd.Flags().BoolVarP(&registerAfterScan, "register", "r", false, "Register a proc after scanning")
  scanCmd.Flags().StringVarP(&scanNamespace, "namespace", "n", "default", "Process namespace")
  scanCmd.MarkFlagRequired("query")

  // Manage command
  var processName string
  var commandPath string
  var createNamespace string
  createCmd := &cobra.Command{
    Use:   "create [name]",
    Short: "Manage proc",
    Long:  "Manage an existing proc or create a new managed proc",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      // 如果提供了位置参数，使用位置参数作为name
      if len(args) > 0 && processName == "" {
        processName = args[0]
      }
      return c.handleCreate(processName, commandPath, namespaceFlag)
    },
  }
  createCmd.Flags().StringVarP(&processName, "name", "N", "", "Process name (alternative to positional argument)")
  createCmd.Flags().StringVarP(&commandPath, "command", "c", "", "Command path")
  createCmd.Flags().StringVarP(&createNamespace, "namespace", "n", "default", "Process namespace")

  // Start command
  var startNamespace string
  startCmd := &cobra.Command{
    Use:   "start [name]",
    Short: "Start proc",
    Long:  "Start a managed proc. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleStartInteractive(namespaceFlag)
      }

      return c.handleStart(args[0], namespaceFlag)
    },
  }
  startCmd.Flags().StringVarP(&startNamespace, "namespace", "n", "default", "Process namespace")

  // Stop command
  var stopNamespace string
  stopCmd := &cobra.Command{
    Use:   "stop [name]",
    Short: "Stop proc",
    Long:  "Stop a managed proc. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleStopInteractive(namespaceFlag)
      }

      return c.handleStop(args[0], namespaceFlag)
    },
  }
  stopCmd.Flags().StringVarP(&stopNamespace, "namespace", "n", "default", "Process namespace")

  // Restart command
  var restartNamespace string
  restartCmd := &cobra.Command{
    Use:   "restart [name]",
    Short: "Restart proc",
    Long:  "Restart a managed proc. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleRestartInteractive(namespaceFlag)
      }

      return c.handleRestart(args[0], namespaceFlag)
    },
  }
  restartCmd.Flags().StringVarP(&restartNamespace, "namespace", "n", "default", "Process namespace")

  // Delete command
  var deleteNamespace string
  deleteCmd := &cobra.Command{
    Use:   "delete [name]",
    Short: "Delete a managed proc",
    Long:  "Delete a proc from the managed list. If the proc is running, it will be stopped first. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleDeleteInteractive(namespaceFlag)
      }

      return c.handleDelete(args[0], namespaceFlag)
    },
  }
  deleteCmd.Flags().StringVarP(&deleteNamespace, "namespace", "n", "default", "Process namespace")

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
    Short: "Check proc status",
    Long:  "Check the status of a managed proc",
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
    Use:   "proc-resources [pid]",
    Short: "Get proc resources",
    Long:  "Get resource usage information for a specific proc",
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
    Short: "Get proc details",
    Long:  "Get detailed information about a managed proc. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")
      formatFlag, _ := cmd.Flags().GetString("format")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleGetInteractive(formatFlag, namespaceFlag)
      }

      return c.handleGet(args[0], formatFlag, namespaceFlag)
    },
  }
  getCmd.Flags().StringVarP(&getFormat, "format", "f", "yaml", "Output format (yaml|text)")
  getCmd.Flags().StringVarP(&getNamespace, "namespace", "n", "default", "Process namespace")

  // Edit command
  var editNamespace string
  editCmd := &cobra.Command{
    Use:   "edit [name]",
    Short: "Edit proc definition",
    Long:  "Edit the definition of a managed proc using vim editor. If no name is provided, an interactive selection will be shown.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      namespaceFlag, _ := cmd.Flags().GetString("namespace")

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleEditInteractive(namespaceFlag)
      }

      return c.handleEdit(args[0], namespaceFlag)
    },
  }
  editCmd.Flags().StringVarP(&editNamespace, "namespace", "n", "default", "Process namespace")

  // Exec command - 新增执行脚本命令
  var isFile bool
  var envVars []string
  var outputFile string
  // 注意：原有的-f参数已经被用作--file标志，所以使用-r作为--result的短选项
  execCmd := &cobra.Command{
    Use:   "exec [command/script]",
    Short: "Execute a command or script",
    Long:  "Execute a command or script file on the server, with optional environment variables and output to file.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleExec(args[0], isFile, envVars, outputFile)
    },
  }
  execCmd.Flags().BoolVarP(&isFile, "file", "f", false, "Treat the argument as a script file path")
  execCmd.Flags().StringArrayVarP(&envVars, "env", "e", []string{}, "Environment variables to set (format: KEY=VALUE)")
  execCmd.Flags().StringVarP(&outputFile, "result", "r", "", "Output result to file instead of console")

  // Add all commands to root command
  rootCmd.AddCommand(scanCmd)
  rootCmd.AddCommand(createCmd)
  rootCmd.AddCommand(startCmd)
  rootCmd.AddCommand(stopCmd)
  rootCmd.AddCommand(restartCmd)
  rootCmd.AddCommand(deleteCmd)
  rootCmd.AddCommand(listCmd)
  rootCmd.AddCommand(statusCmd)
  rootCmd.AddCommand(systemResourceCmd)
  rootCmd.AddCommand(processResourceCmd)
  rootCmd.AddCommand(getConfigCmd)
  rootCmd.AddCommand(getCmd)
  rootCmd.AddCommand(editCmd)
  rootCmd.AddCommand(execCmd)

  return rootCmd
}

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

  // Override PreRun to create client with the provided host
  rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Only create a new client if we're not in the root command
    if cmd != rootCmd {
      c.client = client.NewClient(apiHost)
    }
    return nil
  }

  // Add subcommands
  procCmd := c.setupProcCommands()
  rootCmd.AddCommand(procCmd)

  resourceCmd := c.setupResourceCommands()
  rootCmd.AddCommand(resourceCmd)

  configCmd := c.setupConfigCommands()
  rootCmd.AddCommand(configCmd)

  execCmd := c.setupExecCommand()
  rootCmd.AddCommand(execCmd)

  // Add Redis commands
  redisCmd := c.setupRedisCommands()
  rootCmd.AddCommand(redisCmd)

  // Add RabbitMQ commands
  rabbitCmd := c.setupRabbitCommands()
  rootCmd.AddCommand(rabbitCmd)

  // Add Zookeeper commands
  zkCmd := c.setupZkCommands()
  rootCmd.AddCommand(zkCmd)

  // Add RocketMQ commands
  rocketCmd := c.setupRocketCommands()
  rootCmd.AddCommand(rocketCmd)

  // Add Kafka commands
  kafkaCmd := c.setupKafkaCommands()
  rootCmd.AddCommand(kafkaCmd)

  // Add MQTT commands
  mqttCmd := c.setupMqttCommands()
  rootCmd.AddCommand(mqttCmd)

  // Add Pulsar commands
  pulsarCmd := c.setupPulsarCommands()
  rootCmd.AddCommand(pulsarCmd)

  return rootCmd
}

// setupProcCommands 设置所有进程管理相关的命令
func (c *CLI) setupProcCommands() *cobra.Command {
  // Proc command - 作为父命令来组织所有进程相关的子命令
  procCmd := &cobra.Command{
    Use:   "proc",
    Short: "Process management operations",
    Long:  "Manage and monitor processes with various operations",
  }

  // 添加各个子命令
  procCmd.AddCommand(c.setupScanCommand())
  procCmd.AddCommand(c.setupCreateCommand())
  procCmd.AddCommand(c.setupStartCommand())
  procCmd.AddCommand(c.setupStopCommand())
  procCmd.AddCommand(c.setupRestartCommand())
  procCmd.AddCommand(c.setupDeleteCommand())
  procCmd.AddCommand(c.setupListCommand())
  procCmd.AddCommand(c.setupStatusCommand())
  procCmd.AddCommand(c.setupEditCommand())
  procCmd.AddCommand(c.setupGetCommand())

  return procCmd
}

// setupScanCommand 设置scan命令
func (c *CLI) setupScanCommand() *cobra.Command {
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

  return scanCmd
}

// setupCreateCommand 设置create命令
func (c *CLI) setupCreateCommand() *cobra.Command {
  var processName string
  var commandPath string
  var createNamespace string

  createCmd := &cobra.Command{
    Use:   "create [name]",
    Short: "Create process",
    Long:  "Create a new managed process",
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

  return createCmd
}

// setupStartCommand 设置start命令
func (c *CLI) setupStartCommand() *cobra.Command {
  var startNamespace string

  startCmd := &cobra.Command{
    Use:   "start [name]",
    Short: "Start process",
    Long:  "Start a managed process. If no name is provided, an interactive selection will be shown.",
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

  return startCmd
}

// setupStopCommand 设置stop命令
func (c *CLI) setupStopCommand() *cobra.Command {
  var stopNamespace string

  stopCmd := &cobra.Command{
    Use:   "stop [name]",
    Short: "Stop process",
    Long:  "Stop a managed process. If no name is provided, an interactive selection will be shown.",
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

  return stopCmd
}

// setupRestartCommand 设置restart命令
func (c *CLI) setupRestartCommand() *cobra.Command {
  var restartNamespace string

  restartCmd := &cobra.Command{
    Use:   "restart [name]",
    Short: "Restart process",
    Long:  "Restart a managed process. If no name is provided, an interactive selection will be shown.",
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

  return restartCmd
}

// setupDeleteCommand 设置delete命令
func (c *CLI) setupDeleteCommand() *cobra.Command {
  var deleteNamespace string

  deleteCmd := &cobra.Command{
    Use:   "delete [name]",
    Short: "Delete a managed process",
    Long:  "Delete a process from the managed list. If the process is running, it will be stopped first. If no name is provided, an interactive selection will be shown.",
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

  return deleteCmd
}

// setupListCommand 设置list命令
func (c *CLI) setupListCommand() *cobra.Command {
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

  return listCmd
}

// setupStatusCommand 设置status命令
func (c *CLI) setupStatusCommand() *cobra.Command {
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

  return statusCmd
}

// setupEditCommand 设置edit命令
func (c *CLI) setupEditCommand() *cobra.Command {
  var editNamespace string

  editCmd := &cobra.Command{
    Use:   "edit [name]",
    Short: "Edit process definition",
    Long:  "Edit the definition of a managed process using vim editor. If no name is provided, an interactive selection will be shown.",
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

  return editCmd
}

// setupGetCommand 设置get命令
func (c *CLI) setupGetCommand() *cobra.Command {
  var getFormat string
  var getNamespace string

  getCmd := &cobra.Command{
    Use:   "get [name]",
    Short: "Get process details",
    Long:  "Get detailed information about a managed process. If no name is provided, an interactive selection will be shown.",
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

  return getCmd
}

// setupResourceCommands 设置资源相关命令
func (c *CLI) setupResourceCommands() *cobra.Command {
  resourceCmd := &cobra.Command{
    Use:   "resources",
    Short: "Resource management operations",
    Long:  "View and manage system and process resources",
  }

  // System resources command
  systemResourceCmd := &cobra.Command{
    Use:   "system",
    Short: "Get system resources",
    Long:  "Get system resource usage information",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleSystemResources()
    },
  }

  // Process resources command
  processResourceCmd := &cobra.Command{
    Use:   "process [pid]",
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

  // Add subcommands to resources command
  resourceCmd.AddCommand(systemResourceCmd)
  resourceCmd.AddCommand(processResourceCmd)

  return resourceCmd
}

// setupConfigCommands 设置配置相关命令
func (c *CLI) setupConfigCommands() *cobra.Command {
  // Config command - 作为父命令来组织所有配置相关的子命令
  configCmd := &cobra.Command{
    Use:   "config",
    Short: "Configuration operations",
    Long:  "View and manage system configuration",
  }

  // Get config command
  getConfigCmd := &cobra.Command{
    Use:   "get",
    Short: "Get configuration",
    Long:  "Get the current configuration",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleGetConfig()
    },
  }

  // Add subcommands to config command
  configCmd.AddCommand(getConfigCmd)

  // 可以根据需要添加更多配置相关的命令

  return configCmd
}

// setupExecCommand 设置exec命令
func (c *CLI) setupExecCommand() *cobra.Command {
  var isFile bool
  var envVars []string
  var outputFile string

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

  return execCmd
}

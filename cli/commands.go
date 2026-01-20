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
  "fmt"
  "github.com/casuallc/vigil/api"
  "github.com/casuallc/vigil/version"
  "github.com/spf13/cobra"
)

// setupCommands configures and returns the root command with all subcommands
func (c *CLI) setupCommands() *cobra.Command {
  // Root command
  var apiHost string

  rootCmd := &cobra.Command{
    Use:     "bbx-cli",
    Short:   "Process Management Program",
    Long:    "BBX - A powerful process management and monitoring tool",
    Version: version.Version,
  }

  // 设置版本信息
  rootCmd.SetVersionTemplate(fmt.Sprintf(`Version:   %s
BuildTime: %s
GitCommit: %s
GitBranch: %s
GoVersion: %s
OS/Arch:   %s/%s
`,
    version.Version,
    version.BuildTime,
    version.GitCommit,
    version.GitBranch,
    version.GetVersionInfo().GoVersion,
    version.GetVersionInfo().OS,
    version.GetVersionInfo().Arch,
  ))

  // Add subcommands
  procCmd := c.setupProcCommands()
  rootCmd.AddCommand(procCmd)

  resourceCmd := c.setupResourceCommands()
  rootCmd.AddCommand(resourceCmd)

  configCmd := c.setupConfigCommands()
  rootCmd.AddCommand(configCmd)

  execCmd := c.setupExecCommand()
  rootCmd.AddCommand(execCmd)

  // Add VM commands
  vmCmd := c.setupVMCommands()
  rootCmd.AddCommand(vmCmd)

  // Global flags
  rootCmd.PersistentFlags().StringVarP(&apiHost, "host", "H", "http://127.0.0.1:8181", "API server host address")

  // Override PreRun to create client with the provided host
  rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Create a new client if we're in any of the main commands or their subcommands
    // Check if the command or any of its parents is one of the main commands
    currentCmd := cmd
    for currentCmd != nil {
      if currentCmd == procCmd ||
        currentCmd == resourceCmd ||
        currentCmd == configCmd ||
        currentCmd == execCmd ||
        currentCmd == vmCmd {
        c.client = api.NewClient(apiHost)
        break
      }
      currentCmd = currentCmd.Parent()
    }
    return nil
  }

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

  // Add Cosmic commands
  cosmicCmd := c.setupCosmicCommands()
  rootCmd.AddCommand(cosmicCmd)

  // Add Test commands
  testCmd := c.setupIntegrationTestingCommands()
  rootCmd.AddCommand(testCmd)

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

  // 新增挂载命令组
  procCmd.AddCommand(c.setupMountCommands())

  return procCmd
}

// setupScanCommand 设置scan命令
func (c *CLI) setupScanCommand() *cobra.Command {
  var query string
  var registerAfterScan bool
  var scanNamespace string
  var configFile string
  var batchMode bool

  scanCmd := &cobra.Command{
    Use:   "scan",
    Short: "Scan processes",
    Long:  "Scan system processes based on query string or regex",
    RunE: func(cmd *cobra.Command, args []string) error {
      // 批量模式：从配置文件加载并扫描
      if batchMode {
        return c.handleBatchScan(configFile, registerAfterScan, scanNamespace)
      }

      // 单个查询模式
      return c.handleScan(query, registerAfterScan, scanNamespace)
    },
  }
  scanCmd.Flags().StringVarP(&query, "query", "q", "", "Search query string or regex; support prefix: script://, file://")
  scanCmd.Flags().BoolVarP(&registerAfterScan, "register", "r", false, "Register a process after scanning")
  scanCmd.Flags().StringVarP(&scanNamespace, "namespace", "n", "default", "Process namespace")
  scanCmd.Flags().StringVarP(&configFile, "config", "c", "conf/scan.yaml", "Configuration file for batch scanning")
  scanCmd.Flags().BoolVarP(&batchMode, "batch", "b", false, "Enable batch scanning mode using configuration file")

  // 只有在非批量模式下才需要query参数
  scanCmd.MarkFlagRequired("query")
  // 如果启用了批量模式，则query参数不再是必需的
  scanCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
    if batchMode {
      cmd.Flags().SetAnnotation("query", cobra.BashCompOneRequiredFlag, []string{"false"})
    }
    return nil
  }

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
      // 如果提供了位置参数，使用位置参数作为name
      if len(args) > 0 && processName == "" {
        processName = args[0]
      }
      return c.handleCreate(processName, commandPath, createNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleStartInteractive(startNamespace)
      }

      return c.handleStart(args[0], startNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleStopInteractive(stopNamespace)
      }

      return c.handleStop(args[0], stopNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleRestartInteractive(restartNamespace)
      }

      return c.handleRestart(args[0], restartNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleDeleteInteractive(deleteNamespace)
      }

      return c.handleDelete(args[0], deleteNamespace)
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
      return c.handleList(listNamespace)
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
      return c.handleStatus(args[0], statusNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleEditInteractive(editNamespace)
      }

      return c.handleEdit(args[0], editNamespace)
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
      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleGetInteractive(getFormat, getNamespace)
      }

      return c.handleGet(args[0], getFormat, getNamespace)
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

// setupMountCommands 设置挂载相关子命令 (add/remove/list)
func (c *CLI) setupMountCommands() *cobra.Command {
  mountCmd := &cobra.Command{
    Use:   "mount",
    Short: "Manage process mounts (bind/tmpfs/volume)",
    Long:  "Manage mounts for a managed process: add/remove/list, similar to Docker volumes.",
  }

  mountCmd.AddCommand(c.setupMountAddCommand())
  mountCmd.AddCommand(c.setupMountRemoveCommand())
  mountCmd.AddCommand(c.setupMountListCommand())

  return mountCmd
}

// setupMountAddCommand 设置挂载添加命令
func (c *CLI) setupMountAddCommand() *cobra.Command {
  var mType string
  var target string
  var source string
  var volumeName string
  var readOnly bool
  var options []string
  var mountAddNamespace string
  var mountID string

  cmd := &cobra.Command{
    Use:   "add [name]",
    Short: "Add a mount to a process",
    Long:  "Add a mount to a managed process. Supports type=bind/tmpfs/volume.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      var name string
      if len(args) > 0 {
        name = args[0]
      }
      if name == "" {
        return fmt.Errorf("process name is required")
      }

      if mountID == "" {
        return fmt.Errorf("mount name (id) is required")
      }

      if target == "" {
        return fmt.Errorf("target is required")
      }

      switch mType {
      case "bind":
        if source == "" {
          return fmt.Errorf("source is required for bind mount")
        }
      case "volume":
        if volumeName == "" {
          return fmt.Errorf("volume name is required for volume mount")
        }
      case "tmpfs":
        // no additional required fields
      default:
        return fmt.Errorf("unsupported mount type: %s (use bind|tmpfs|volume)", mType)
      }

      return c.handleProcMountAdd(name, mountAddNamespace, mountID, mType, target, source, volumeName, readOnly, options)
    },
  }

  cmd.Flags().StringVarP(&mType, "type", "t", "bind", "Mount type (bind|tmpfs|volume)")
  cmd.Flags().StringVarP(&mountID, "name", "N", "", "Mount identifier (unique per process)")
  cmd.Flags().StringVarP(&target, "target", "T", "", "Target path inside process (required)")
  cmd.Flags().StringVarP(&source, "source", "s", "", "Source path for bind mount")
  cmd.Flags().StringVarP(&volumeName, "volume", "v", "", "Named volume name for volume mount")
  cmd.Flags().BoolVarP(&readOnly, "read-only", "r", false, "Mount as read-only")
  cmd.Flags().StringArrayVarP(&options, "option", "o", []string{}, "Additional mount options (can be repeated)")
  cmd.Flags().StringVarP(&mountAddNamespace, "namespace", "n", "default", "Process namespace")
  cmd.MarkFlagRequired("name")
  cmd.MarkFlagRequired("target")

  return cmd
}

// setupMountRemoveCommand 设置挂载移除命令
func (c *CLI) setupMountRemoveCommand() *cobra.Command {
  var mountRemoveNamespace string
  var target string
  var index int

  cmd := &cobra.Command{
    Use:   "remove [name]",
    Short: "Remove mount(s) from a process",
    Long:  "Remove a mount from a process by target or index.",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      var name string
      if len(args) > 0 {
        name = args[0]
      }
      if name == "" {
        return fmt.Errorf("process name is required")
      }

      if target == "" && index < 0 {
        return fmt.Errorf("either --target or --index must be specified")
      }

      return c.handleProcMountRemove(name, mountRemoveNamespace, target, index)
    },
  }

  cmd.Flags().StringVarP(&mountRemoveNamespace, "namespace", "n", "default", "Process namespace")
  cmd.Flags().StringVarP(&target, "target", "T", "", "Target path to remove")
  cmd.Flags().IntVarP(&index, "index", "i", -1, "Mount index to remove")
  return cmd
}

// setupMountListCommand 设置挂载列表命令
func (c *CLI) setupMountListCommand() *cobra.Command {
  var mountListNamespace string

  cmd := &cobra.Command{
    Use:   "list [name]",
    Short: "List mounts of a process",
    Long:  "List all mounts configured for a managed process.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleProcMountList(args[0], mountListNamespace)
    },
  }

  cmd.Flags().StringVarP(&mountListNamespace, "namespace", "n", "default", "Process namespace")
  return cmd
}

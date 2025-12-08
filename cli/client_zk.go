package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/client/zookeeper"

  "github.com/spf13/cobra"
)

// setupZkCommands 设置Zookeeper相关命令
func (c *CLI) setupZkCommands() *cobra.Command {
  zkCmd := &cobra.Command{
    Use:   "zookeeper",
    Short: "Zookeeper related commands",
    Long:  `Perform Zookeeper operations like creating nodes, getting node data, etc.`,
  }

  // 为父命令添加持久化标志
  var config zookeeper.ServerConfig
  zkCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "localhost", "Zookeeper server host")
  zkCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 2181, "Zookeeper server port")
  zkCmd.PersistentFlags().IntVarP(&config.Timeout, "timeout", "t", 30, "Connection timeout in seconds")

  // 存储配置到上下文
  zkCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "zkConfig", &config))
  }

  // 添加子命令
  zkCmd.AddCommand(c.setupZkCreateCommand())
  zkCmd.AddCommand(c.setupZkDeleteCommand())
  zkCmd.AddCommand(c.setupZkExistsCommand())
  zkCmd.AddCommand(c.setupZkGetCommand())
  zkCmd.AddCommand(c.setupZkSetCommand())

  return zkCmd
}

// 修改子命令，移除重复的连接标志
func (c *CLI) setupZkCreateCommand() *cobra.Command {
  var path, data string

  cmd := &cobra.Command{
    Use:   "create",
    Short: "Create a Zookeeper node",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("zkConfig").(*zookeeper.ServerConfig)
      return c.handleZkCreate(path, data, config)
    },
  }

  cmd.Flags().StringVar(&path, "path", "", "Node path")
  cmd.Flags().StringVarP(&data, "data", "d", "", "Node data")
  cmd.MarkFlagRequired("path")

  return cmd
}

func (c *CLI) handleZkCreate(path, data string, config *zookeeper.ServerConfig) error {
  client := &zookeeper.Client{Config: config}

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Zookeeper:", err.Error())
    return nil
  }
  defer client.Disconnect()

  if err := client.Create(path, []byte(data)); err != nil {
    fmt.Println("ERROR failed to create node:", err.Error())
    return nil
  }

  fmt.Printf("Node created at path: %s\n", path)
  return nil
}

// setupZkDeleteCommand 设置删除节点命令
func (c *CLI) setupZkDeleteCommand() *cobra.Command {
  var path string

  cmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete a Zookeeper node",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("zkConfig").(*zookeeper.ServerConfig)
      return c.handleZkDelete(path, config)
    },
  }

  cmd.Flags().StringVar(&path, "path", "", "Node path")
  cmd.MarkFlagRequired("path")

  return cmd
}

// handleZkDelete 处理删除节点命令
func (c *CLI) handleZkDelete(path string, config *zookeeper.ServerConfig) error {
  client := &zookeeper.Client{Config: config}

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Zookeeper:", err.Error())
    return nil
  }
  defer client.Disconnect()

  if err := client.Delete(path); err != nil {
    fmt.Println("ERROR failed to delete node:", err.Error())
    return nil
  }

  fmt.Printf("Node deleted successfully at path: %s\n", path)
  return nil
}

// setupZkExistsCommand 设置检查节点是否存在命令
func (c *CLI) setupZkExistsCommand() *cobra.Command {
  var path string

  cmd := &cobra.Command{
    Use:   "exists",
    Short: "Check if a Zookeeper node exists",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("zkConfig").(*zookeeper.ServerConfig)
      return c.handleZkExists(path, config)
    },
  }

  cmd.Flags().StringVar(&path, "path", "", "Node path")
  cmd.MarkFlagRequired("path")

  return cmd
}

// handleZkExists 处理检查节点是否存在命令
func (c *CLI) handleZkExists(path string, config *zookeeper.ServerConfig) error {
  client := &zookeeper.Client{Config: config}

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Zookeeper:", err.Error())
    return nil
  }
  defer client.Disconnect()

  exists, err := client.Exists(path)
  if err != nil {
    fmt.Println("ERROR failed to check node existence:", err.Error())
    return nil
  }

  if exists {
    fmt.Printf("Node exists at path: %s\n", path)
  } else {
    fmt.Printf("Node does not exist at path: %s\n", path)
  }
  return nil
}

// setupZkGetCommand 设置获取节点数据命令
func (c *CLI) setupZkGetCommand() *cobra.Command {
  var path string

  cmd := &cobra.Command{
    Use:   "get",
    Short: "Get data from a Zookeeper node",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("zkConfig").(*zookeeper.ServerConfig)
      return c.handleZkGet(path, config)
    },
  }

  cmd.Flags().StringVar(&path, "path", "", "Node path")
  cmd.MarkFlagRequired("path")

  return cmd
}

// handleZkGet 处理获取节点数据命令
func (c *CLI) handleZkGet(path string, config *zookeeper.ServerConfig) error {
  client := &zookeeper.Client{Config: config}

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Zookeeper:", err.Error())
    return nil
  }
  defer client.Disconnect()

  data, err := client.Get(path)
  if err != nil {
    fmt.Println("ERROR failed to get node data:", err.Error())
    return nil
  }

  fmt.Printf("Data at path %s:\n%s\n", path, string(data))
  return nil
}

// setupZkSetCommand 设置设置节点数据命令
func (c *CLI) setupZkSetCommand() *cobra.Command {
  var path, data string

  cmd := &cobra.Command{
    Use:   "set",
    Short: "Set data for a Zookeeper node",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("zkConfig").(*zookeeper.ServerConfig)
      return c.handleZkSet(path, data, config)
    },
  }

  cmd.Flags().StringVar(&path, "path", "", "Node path")
  cmd.Flags().StringVarP(&data, "data", "d", "", "Node data")
  cmd.MarkFlagRequired("path")
  cmd.MarkFlagRequired("data")

  return cmd
}

// handleZkSet 处理设置节点数据命令
func (c *CLI) handleZkSet(path, data string, config *zookeeper.ServerConfig) error {
  client := &zookeeper.Client{
    Config: config,
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Zookeeper: %w", err)
  }
  defer client.Disconnect()

  if err := client.Set(path, []byte(data)); err != nil {
    return fmt.Errorf("failed to set node data: %w", err)
  }

  fmt.Printf("Data set successfully at path: %s\n", path)
  return nil
}

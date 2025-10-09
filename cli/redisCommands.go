package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/redis"
  "github.com/spf13/cobra"
)

// setupRedisCommands 设置Redis相关命令
func (c *CLI) setupRedisCommands() *cobra.Command {
  redisCmd := &cobra.Command{
    Use:   "redis",
    Short: "Redis related commands",
    Long:  `Perform Redis operations like get, set, delete, info, etc.`,
  }

  // 为父命令添加持久化标志（子命令会继承这些标志）
  var config redis.ServerConfig
  redisCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "localhost", "Redis server host")
  redisCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 6379, "Redis server port")
  redisCmd.PersistentFlags().StringVarP(&config.Password, "password", "P", "", "Redis password")
  redisCmd.PersistentFlags().IntVarP(&config.DB, "db", "d", 0, "Redis database")

  // 存储配置到上下文，供子命令使用
  redisCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "redisConfig", &config))
  }

  // 添加子命令
  redisCmd.AddCommand(c.setupRedisGetCommand())
  redisCmd.AddCommand(c.setupRedisSetCommand())
  redisCmd.AddCommand(c.setupRedisDeleteCommand())
  redisCmd.AddCommand(c.setupRedisInfoCommand())

  return redisCmd
}

// 修改setupRedisGetCommand，不再重复设置连接标志
func (c *CLI) setupRedisGetCommand() *cobra.Command {
  var key string

  cmd := &cobra.Command{
    Use:   "get",
    Short: "Get value from Redis",
    RunE: func(cmd *cobra.Command, args []string) error {
      // 从上下文中获取配置
      config := cmd.Context().Value("redisConfig").(*redis.ServerConfig)
      return c.handleRedisGet(key, config)
    },
  }

  cmd.Flags().StringVarP(&key, "key", "k", "", "Redis key")
  cmd.MarkFlagRequired("key")

  return cmd
}

// 修改handleRedisGet函数签名
func (c *CLI) handleRedisGet(key string, config *redis.ServerConfig) error {
  client := &redis.Client{
    Config: config,
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Redis: %w", err)
  }
  defer client.Disconnect()

  value, err := client.Get(key)
  if err != nil {
    return fmt.Errorf("failed to get key: %w", err)
  }

  fmt.Printf("Key: %s, Value: %s\n", key, value)
  return nil
}

// setupRedisSetCommand 设置设置Redis键值命令
func (c *CLI) setupRedisSetCommand() *cobra.Command {
  var key, value string
  var server string
  var port int
  var password string
  var db int

  cmd := &cobra.Command{
    Use:   "set",
    Short: "Set value in Redis",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRedisSet(key, value, server, port, password, db)
    },
  }

  cmd.Flags().StringVarP(&key, "key", "k", "", "Redis key")
  cmd.Flags().StringVarP(&value, "value", "v", "", "Redis value")
  cmd.MarkFlagRequired("key")
  cmd.MarkFlagRequired("value")

  return cmd
}

// handleRedisSet 处理设置Redis键值命令
func (c *CLI) handleRedisSet(key, value, server string, port int, password string, db int) error {
  client := &redis.Client{
    Config: &redis.ServerConfig{
      Server:   server,
      Port:     port,
      Password: password,
      DB:       db,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Redis: %w", err)
  }
  defer client.Disconnect()

  if err := client.Set(key, value); err != nil {
    return fmt.Errorf("failed to set key: %w", err)
  }

  fmt.Printf("Key: %s set successfully\n", key)
  return nil
}

// setupRedisDeleteCommand 设置删除Redis键命令
func (c *CLI) setupRedisDeleteCommand() *cobra.Command {
  var key string
  var server string
  var port int
  var password string
  var db int

  cmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete key from Redis",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRedisDelete(key, server, port, password, db)
    },
  }

  cmd.Flags().StringVarP(&key, "key", "k", "", "Redis key")
  cmd.MarkFlagRequired("key")

  return cmd
}

// handleRedisDelete 处理删除Redis键命令
func (c *CLI) handleRedisDelete(key, server string, port int, password string, db int) error {
  client := &redis.Client{
    Config: &redis.ServerConfig{
      Server:   server,
      Port:     port,
      Password: password,
      DB:       db,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Redis: %w", err)
  }
  defer client.Disconnect()

  if err := client.Delete(key); err != nil {
    return fmt.Errorf("failed to delete key: %w", err)
  }

  fmt.Printf("Key: %s deleted successfully\n", key)
  return nil
}

// setupRedisInfoCommand 设置获取Redis信息命令
func (c *CLI) setupRedisInfoCommand() *cobra.Command {
  var server string
  var port int
  var password string
  var db int

  cmd := &cobra.Command{
    Use:   "info",
    Short: "Get Redis server information",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRedisInfo(server, port, password, db)
    },
  }

  return cmd
}

// handleRedisInfo 处理获取Redis信息命令
func (c *CLI) handleRedisInfo(server string, port int, password string, db int) error {
  client := &redis.Client{
    Config: &redis.ServerConfig{
      Server:   server,
      Port:     port,
      Password: password,
      DB:       db,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Redis: %w", err)
  }
  defer client.Disconnect()

  info, err := client.Info()
  if err != nil {
    return fmt.Errorf("failed to get Redis info: %w", err)
  }

  fmt.Println("Redis Server Information:")
  fmt.Printf("  Build Date: %s\n", info.BuildDate)
  fmt.Printf("  Used Memory: %s (%s)\n", info.UsedMemory, info.UsedMemoryHuman)
  fmt.Printf("  Max Memory: %s (%s)\n", info.MaxMemory, info.MaxMemoryHuman)
  fmt.Printf("  Total System Memory: %s (%s)\n", info.TotalMemory, info.TotalMemoryHuman)
  fmt.Printf("  Role: %s\n", info.Role)
  fmt.Printf("  DB0: %s\n", info.Db0)
  return nil
}

package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/rocketmq"
  "github.com/spf13/cobra"
)

// setupRocketCommands 设置RocketMQ相关命令
func (c *CLI) setupRocketCommands() *cobra.Command {
  rocketCmd := &cobra.Command{
    Use:   "rocketmq",
    Short: "RocketMQ related commands",
    Long:  `Perform RocketMQ operations like sending and receiving messages.`,
  }

  // 为父命令添加持久化标志
  var config rocketmq.ServerConfig
  rocketCmd.PersistentFlags().StringVar(&config.Server, "server", "localhost", "RocketMQ server host")
  rocketCmd.PersistentFlags().IntVar(&config.Port, "port", 9876, "RocketMQ server port")
  rocketCmd.PersistentFlags().StringVar(&config.User, "user", "", "Username for authentication")
  rocketCmd.PersistentFlags().StringVar(&config.Password, "password", "", "Password for authentication")

  // 存储配置到上下文
  rocketCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "rocketConfig", &config))
  }

  // 添加子命令
  rocketCmd.AddCommand(c.setupRocketSendCommand())
  rocketCmd.AddCommand(c.setupRocketReceiveCommand())

  return rocketCmd
}

// 修改子命令，移除重复的连接标志
func (c *CLI) setupRocketSendCommand() *cobra.Command {
  var groupName string
  var topic string
  var tags string
  var keys string
  var message string
  var repeat int
  var interval int

  cmd := &cobra.Command{
    Use:   "send",
    Short: "Send message to RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      //config := cmd.Context().Value("rocketConfig").(*rocketmq.ServerConfig)
      //return c.handleRocketSend(config, groupName, topic, tags, keys, message, repeat, interval)
      return nil
    },
  }

  cmd.Flags().StringVar(&groupName, "group", "default_group", "Producer group name")
  cmd.Flags().StringVar(&topic, "topic", "", "Message topic")
  cmd.Flags().StringVar(&tags, "tags", "", "Message tags")
  cmd.Flags().StringVar(&keys, "keys", "", "Message keys")
  cmd.Flags().StringVar(&message, "message", "", "Message content")
  cmd.Flags().IntVar(&repeat, "repeat", 1, "Number of times to repeat sending")
  cmd.Flags().IntVar(&interval, "interval", 0, "Interval between messages")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagRequired("message")

  return cmd
}

// setupRocketReceiveCommand 设置接收消息命令
func (c *CLI) setupRocketReceiveCommand() *cobra.Command {
  var server string
  var port int
  var user string
  var password string
  var groupName string
  var topic string
  var tags string
  var timeout int

  cmd := &cobra.Command{
    Use:   "receive",
    Short: "Receive messages from RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRocketReceive(server, port, user, password, groupName, topic, tags, timeout)
    },
  }

  cmd.Flags().StringVar(&groupName, "group", "default_consumer_group", "Consumer group name")
  cmd.Flags().StringVar(&topic, "topic", "", "Message topic to subscribe")
  cmd.Flags().StringVar(&tags, "tags", "*", "Message tags filter (use * for all)")
  cmd.Flags().IntVar(&timeout, "timeout", 0, "Consumer timeout in seconds (0 for no timeout)")

  // 设置必填参数
  cmd.MarkFlagRequired("topic")

  return cmd
}

// handleRocketSend 处理发送消息
func (c *CLI) handleRocketSend(server string, port int, user string, password string, groupName string, topic string, tags string, keys string, message string, repeat int, interval int) error {
  // 创建配置
  config := &rocketmq.ServerConfig{
    Server:   server,
    Port:     port,
    User:     user,
    Password: password,
  }

  // 创建客户端
  client := rocketmq.NewClient(config)
  defer client.Close()

  // 连接到服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RocketMQ server: %w", err)
  }

  // 创建生产者配置
  producerConfig := &rocketmq.ProducerConfig{
    GroupName: groupName,
    Topic:     topic,
    Tags:      tags,
    Keys:      keys,
    Message:   message,
    Repeat:    repeat,
    Interval:  interval,
  }

  // 发送消息
  if err := client.SendMessage(producerConfig); err != nil {
    return fmt.Errorf("failed to send message: %w", err)
  }

  fmt.Printf("Successfully sent %d messages to topic %s\n", repeat, topic)
  return nil
}

// handleRocketReceive 处理接收消息
func (c *CLI) handleRocketReceive(server string, port int, user string, password string, groupName string, topic string, tags string, timeout int) error {
  // 创建配置
  config := &rocketmq.ServerConfig{
    Server:   server,
    Port:     port,
    User:     user,
    Password: password,
  }

  // 创建客户端
  client := rocketmq.NewClient(config)
  defer client.Close()

  // 连接到服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RocketMQ server: %w", err)
  }

  // 创建消费者配置
  consumerConfig := &rocketmq.ConsumerConfig{
    GroupName: groupName,
    Topic:     topic,
    Tags:      tags,
    Timeout:   timeout,
    Handler: func(msg *rocketmq.Message) {
      fmt.Printf("\nReceived message:\n")
      fmt.Printf("  Topic: %s\n", msg.Topic)
      fmt.Printf("  Tags: %s\n", msg.Tags)
      fmt.Printf("  Keys: %s\n", msg.Keys)
      fmt.Printf("  MsgID: %s\n", msg.MsgID)
      fmt.Printf("  QueueID: %d\n", msg.QueueID)
      fmt.Printf("  Body: %s\n", msg.Body)
    },
  }

  fmt.Printf("Starting to receive messages from topic '%s'\n", topic)
  if timeout > 0 {
    fmt.Printf("Consumer will stop after %d seconds if no messages are received\n", timeout)
  } else {
    fmt.Printf("Consumer running continuously. Press Ctrl+C to stop...\n")
  }

  // 接收消息
  if err := client.ReceiveMessage(consumerConfig); err != nil {
    return fmt.Errorf("error receiving messages: %w", err)
  }

  return nil
}

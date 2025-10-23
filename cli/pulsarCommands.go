package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/pulsar"
  "github.com/spf13/cobra"
  "time"
)

// setupPulsarCommands 设置Pulsar相关命令
func (c *CLI) setupPulsarCommands() *cobra.Command {
  pulsarCmd := &cobra.Command{
    Use:   "pulsar",
    Short: "Pulsar related commands",
    Long:  `Perform Apache Pulsar operations like sending and receiving messages.`,
  }

  // 为父命令添加持久化标志
  var config pulsar.ServerConfig
  pulsarCmd.PersistentFlags().StringVar(&config.URL, "url", "pulsar://localhost:6650", "Pulsar service URL")
  pulsarCmd.PersistentFlags().StringVar(&config.Auth.Token, "token", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhZG1xIn0.ybJge7zTfy_RDdAtB3w6nIPDHPT6-kbB6sNzgPt8sKQ", "Authentication token")
  pulsarCmd.PersistentFlags().IntVarP(&config.Timeout, "timeout", "o", 30, "Connection timeout in seconds")

  // 存储配置到上下文
  pulsarCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "pulsarConfig", &config))
  }

  // 添加子命令
  pulsarCmd.AddCommand(c.setupPulsarSendCommand())
  pulsarCmd.AddCommand(c.setupPulsarReceiveCommand())

  return pulsarCmd
}

// setupPulsarSendCommand 设置发送消息命令
func (c *CLI) setupPulsarSendCommand() *cobra.Command {
  var topic string
  var message string
  var key string
  var sendTimeout int
  var enableBatching bool
  var batchingMaxDelay int
  var batchingMaxMessages int
  var messageLength int
  var repeat int
  var interval int
  var printLog bool
  var delayTime int
  var deliverTimeStr string
  var enableCompression bool
  var properties string

  cmd := &cobra.Command{
    Use:   "send",
    Short: "Send message to Pulsar",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("pulsarConfig").(*pulsar.ServerConfig)

      // 解析定时消息时间
      var deliverTime *time.Time
      if deliverTimeStr != "" {
        t, err := time.Parse("2006-01-02 15:04:05", deliverTimeStr)
        if err != nil {
          return fmt.Errorf("invalid deliver time format: %v", err)
        }
        deliverTime = &t
      }

      return c.handlePulsarSend(config, topic, message, key, sendTimeout, enableBatching, batchingMaxDelay, batchingMaxMessages, messageLength, repeat, interval, printLog, delayTime, deliverTime, enableCompression, properties)
    },
  }

  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().StringVarP(&key, "key", "k", "", "Message key")
  cmd.Flags().IntVar(&sendTimeout, "send-timeout", 30000, "Send timeout in milliseconds")
  cmd.Flags().BoolVar(&enableBatching, "enable-batching", false, "Enable message batching")
  cmd.Flags().IntVar(&batchingMaxDelay, "batching-max-delay", 10, "Max batching delay in milliseconds")
  cmd.Flags().IntVar(&batchingMaxMessages, "batching-max-messages", 1000, "Max messages per batch")
  cmd.Flags().IntVar(&messageLength, "message-length", 0, "Message length, will pad with spaces if necessary")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 1, "Number of times to repeat sending")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().IntVar(&delayTime, "delay-time", 0, "Delay time for delayed messages in milliseconds")
  cmd.Flags().StringVar(&deliverTimeStr, "deliver-time", "", "Deliver time for scheduled messages (format: 2006-01-02 15:04:05)")
  cmd.Flags().BoolVar(&enableCompression, "enable-compression", false, "Enable message compression")
  cmd.Flags().StringVar(&properties, "properties", "", "Message properties in format key=val,key=val")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagRequired("message")

  return cmd
}

// handlePulsarSend 处理发送消息
func (c *CLI) handlePulsarSend(config *pulsar.ServerConfig, topic string, message string, key string, sendTimeout int, enableBatching bool, batchingMaxDelay int, batchingMaxMessages int, messageLength int, repeat int, interval int, printLog bool, delayTime int, deliverTime *time.Time, enableCompression bool, properties string) error {
  client := pulsar.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Pulsar server:", err.Error())
    return nil
  }

  // 创建生产者配置
  producerConfig := pulsar.ProducerConfig{
    Topic:                   topic,
    Message:                 message,
    Key:                     key,
    SendTimeout:             sendTimeout,
    EnableBatching:          enableBatching,
    BatchingMaxPublishDelay: batchingMaxDelay,
    BatchingMaxMessages:     batchingMaxMessages,
    MessageLength:           messageLength,
    Repeat:                  repeat,
    Interval:                interval,
    PrintLog:                printLog,
    DelayTime:               delayTime,
    DeliverTime:             deliverTime,
    EnableCompression:       enableCompression,
    Properties:              properties,
  }

  // 发送消息
  if err := client.SendMessage(producerConfig); err != nil {
    fmt.Println("ERROR failed to send message:", err.Error())
    return nil
  }

  fmt.Printf("Successfully sent %d messages to topic %s\n", repeat, topic)
  return nil
}

// setupPulsarReceiveCommand 设置接收消息命令
func (c *CLI) setupPulsarReceiveCommand() *cobra.Command {
  var topic string
  var subscription string
  var subscriptionType string
  var receiveTimeout int
  var messageTimeout int
  var initialPosition string
  var autoAck bool
  var count int

  cmd := &cobra.Command{
    Use:   "receive",
    Short: "Receive messages from Pulsar",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("pulsarConfig").(*pulsar.ServerConfig)
      return c.handlePulsarReceive(config, topic, subscription, subscriptionType, receiveTimeout, messageTimeout, initialPosition, autoAck, count)
    },
  }

  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVarP(&subscription, "subscription", "s", "default-subscription", "Subscription name")
  cmd.Flags().StringVar(&subscriptionType, "subscription-type", "Exclusive", "Subscription type: Exclusive, Shared, Failover, Key_Shared")
  cmd.Flags().IntVar(&receiveTimeout, "receive-timeout", 10000, "Receive timeout in milliseconds")
  cmd.Flags().IntVar(&messageTimeout, "message-timeout", 0, "Message processing timeout in seconds (0 for no timeout)")
  cmd.Flags().StringVar(&initialPosition, "initial-position", "Latest", "Initial position: Earliest, Latest")
  cmd.Flags().BoolVar(&autoAck, "auto-ack", true, "Auto acknowledge messages")
  cmd.Flags().IntVarP(&count, "count", "c", 0, "Number of messages to receive (0 for unlimited)")

  cmd.MarkFlagRequired("topic")

  return cmd
}

// handlePulsarReceive 处理接收消息
func (c *CLI) handlePulsarReceive(config *pulsar.ServerConfig, topic string, subscription string, subscriptionType string, receiveTimeout int, messageTimeout int, initialPosition string, autoAck bool, count int) error {
  client := pulsar.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Pulsar server:", err.Error())
    return nil
  }

  // 创建消费者配置
  consumerConfig := pulsar.ConsumerConfig{
    Topic:            topic,
    Subscription:     subscription,
    SubscriptionType: subscriptionType,
    ReceiveTimeout:   receiveTimeout,
    MessageTimeout:   messageTimeout,
    InitialPosition:  initialPosition,
    AutoAck:          autoAck,
  }

  fmt.Printf("Starting to receive messages from topic '%s'\n", topic)
  if messageTimeout > 0 {
    fmt.Printf("Consumer will stop after %d seconds if no messages are received\n", messageTimeout)
  } else {
    fmt.Printf("Consumer running continuously. Press Ctrl+C to stop...\n")
  }

  // 计数器
  receivedCount := 0

  // 接收消息
  err := client.ReceiveMessage(consumerConfig, func(msg *pulsar.Message) bool {
    fmt.Printf("\nReceived message:\n")
    fmt.Printf("  Topic: %s\n", msg.Topic)
    fmt.Printf("  Key: %s\n", msg.Key)
    fmt.Printf("  MessageID: %s\n", msg.MessageID)
    fmt.Printf("  PublishTime: %s\n", msg.PublishTime.Format("2006-01-02 15:04:05"))
    fmt.Printf("  Body: %s\n", string(msg.Payload))

    // 打印属性
    if len(msg.Properties) > 0 {
      fmt.Printf("  Properties:\n")
      for k, v := range msg.Properties {
        fmt.Printf("    %s: %s\n", k, v)
      }
    }

    // 更新计数器
    receivedCount++

    // 检查是否达到指定的消息数量
    if count > 0 && receivedCount >= count {
      fmt.Printf("Received %d messages, stopping...\n", count)
      return false
    }

    return true
  })
  if err != nil {
    fmt.Println("ERROR failed to receive message:", err.Error())
    return nil
  }

  fmt.Println("Receive completed.")
  return nil
}

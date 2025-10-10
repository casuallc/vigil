package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/kafka"
  "github.com/spf13/cobra"
)

// setupKafkaCommands 设置Kafka相关命令
func (c *CLI) setupKafkaCommands() *cobra.Command {
  kafkaCmd := &cobra.Command{
    Use:   "kafka",
    Short: "Kafka related commands",
    Long:  `Perform Kafka operations like sending and receiving messages.`,
  }

  // 为父命令添加持久化标志
  var config kafka.ServerConfig
  kafkaCmd.PersistentFlags().StringVar(&config.Servers, "servers", "localhost", "Kafka server addresses (comma separated)")
  kafkaCmd.PersistentFlags().IntVar(&config.Port, "port", 9092, "Kafka server port")
  kafkaCmd.PersistentFlags().StringVar(&config.User, "user", "", "Username for authentication")
  kafkaCmd.PersistentFlags().StringVar(&config.Password, "password", "", "Password for authentication")
  kafkaCmd.PersistentFlags().StringVar(&config.SASLMechanism, "sasl-mechanism", "", "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
  kafkaCmd.PersistentFlags().StringVar(&config.SASLProtocol, "sasl-protocol", "SASL_PLAINTEXT", "SASL protocol")
  kafkaCmd.PersistentFlags().IntVar(&config.Timeout, "timeout", 30, "Connection timeout in seconds")

  // 存储配置到上下文
  kafkaCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "kafkaConfig", &config))
  }

  // 添加子命令
  kafkaCmd.AddCommand(c.setupKafkaSendCommand())
  kafkaCmd.AddCommand(c.setupKafkaReceiveCommand())

  return kafkaCmd
}

// setupKafkaSendCommand 设置发送消息命令
func (c *CLI) setupKafkaSendCommand() *cobra.Command {
  var topic string
  var message string
  var key string
  var repeat int
  var interval int
  var printLog bool
  var acks string
  var messageLength int
  var compression string
  var headers string

  cmd := &cobra.Command{
    Use:   "send",
    Short: "Send message to Kafka",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("kafkaConfig").(*kafka.ServerConfig)
      return c.handleKafkaSend(config, topic, message, key, repeat, interval, printLog, acks, messageLength, compression, headers)
    },
  }

  cmd.Flags().StringVar(&topic, "topic", "", "Message topic")
  cmd.Flags().StringVar(&message, "message", "", "Message content")
  cmd.Flags().StringVar(&key, "key", "", "Message key")
  cmd.Flags().IntVar(&repeat, "repeat", 1, "Number of times to repeat sending")
  cmd.Flags().IntVar(&interval, "interval", 1000, "Interval between messages in milliseconds")
  cmd.Flags().BoolVar(&printLog, "print-log", false, "Print detailed logs")
  cmd.Flags().StringVar(&acks, "acks", "1", "Acknowledgment level (0, 1, -1/all)")
  cmd.Flags().IntVar(&messageLength, "message-length", 0, "Message length, will pad with spaces if necessary")
  cmd.Flags().StringVar(&compression, "compression", "", "Compression type (gzip, snappy, lz4, zstd)")
  cmd.Flags().StringVar(&headers, "headers", "", "Message headers in format name=value,name2=value2")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagRequired("message")

  return cmd
}

// setupKafkaReceiveCommand 设置接收消息命令
func (c *CLI) setupKafkaReceiveCommand() *cobra.Command {
  var topic string
  var groupID string
  var partition int32
  var offset int64
  var offsetType string
  var timeout int
  var printLog bool
  var maxMessages int
  var commitInterval int

  cmd := &cobra.Command{
    Use:   "receive",
    Short: "Receive messages from Kafka",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("kafkaConfig").(*kafka.ServerConfig)
      return c.handleKafkaReceive(config, topic, groupID, partition, offset, offsetType, timeout, printLog, maxMessages, commitInterval)
    },
  }

  cmd.Flags().StringVar(&topic, "topic", "", "Message topic to subscribe")
  cmd.Flags().StringVar(&groupID, "group-id", "default_consumer_group", "Consumer group ID")
  cmd.Flags().Int32Var(&partition, "partition", -1, "Partition number (-1 for all partitions)")
  cmd.Flags().Int64Var(&offset, "offset", 0, "Offset value (only valid if offset-type is 'specific')")
  cmd.Flags().StringVar(&offsetType, "offset-type", "latest", "Offset type (earliest, latest, specific)")
  cmd.Flags().IntVar(&timeout, "timeout", 0, "Consumer timeout in seconds (0 for no timeout)")
  cmd.Flags().BoolVar(&printLog, "print-log", false, "Print detailed logs")
  cmd.Flags().IntVar(&maxMessages, "max-messages", 0, "Maximum number of messages to receive (0 for unlimited)")
  cmd.Flags().IntVar(&commitInterval, "commit-interval", 0, "Commit interval in milliseconds")

  cmd.MarkFlagRequired("topic")

  return cmd
}

// handleKafkaSend 处理发送消息
func (c *CLI) handleKafkaSend(config *kafka.ServerConfig, topic, message, key string, repeat, interval int, printLog bool, acks string, messageLength int, compression, headers string) error {
  // 创建Kafka客户端
  client := kafka.NewClient(config)

  // 连接到Kafka服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Kafka: %w", err)
  }
  defer client.Close()

  // 设置生产者配置
  producerConfig := &kafka.ProducerConfig{
    Topic:         topic,
    Message:       message,
    Key:           key,
    Repeat:        repeat,
    Interval:      interval,
    PrintLog:      printLog,
    Acks:          acks,
    MessageLength: messageLength,
    Compression:   compression,
    Headers:       headers,
  }

  // 发送消息
  if err := client.SendMessage(producerConfig); err != nil {
    return fmt.Errorf("failed to send message: %w", err)
  }

  fmt.Printf("Successfully sent %d messages to Kafka topic '%s'\n", repeat, topic)
  return nil
}

// handleKafkaReceive 处理接收消息
func (c *CLI) handleKafkaReceive(config *kafka.ServerConfig, topic, groupID string, partition int32, offset int64, offsetType string, timeout int, printLog bool, maxMessages, commitInterval int) error {
  // 创建Kafka客户端
  client := kafka.NewClient(config)

  // 连接到Kafka服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to Kafka: %w", err)
  }
  defer client.Close()

  // 设置消费者配置
  consumerConfig := &kafka.ConsumerConfig{
    Topic:          topic,
    GroupID:        groupID,
    Partition:      partition,
    Offset:         offset,
    OffsetType:     offsetType,
    Timeout:        timeout,
    PrintLog:       printLog,
    MaxMessages:    maxMessages,
    CommitInterval: commitInterval,
  }

  // 接收消息
  if err := client.ReceiveMessage(consumerConfig); err != nil {
    return fmt.Errorf("failed to receive message: %w", err)
  }

  fmt.Printf("Finished receiving messages from Kafka topic '%s'\n", topic)
  return nil
}

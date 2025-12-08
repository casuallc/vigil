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
  "context"
  "fmt"
  "github.com/casuallc/vigil/client/kafka"
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
  kafkaCmd.PersistentFlags().StringVarP(&config.Servers, "servers", "s", "localhost", "Kafka server addresses (comma separated)")
  kafkaCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 9092, "Kafka server port")
  kafkaCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "", "Username for authentication")
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

  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().StringVarP(&key, "key", "k", "", "Message key")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 10, "Number of times to repeat sending")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().StringVarP(&acks, "acks", "a", "1", "Acknowledgment level (0, 1, -1/all)")
  cmd.Flags().IntVar(&messageLength, "message-length", 0, "Message length, will pad with spaces if necessary")
  cmd.Flags().StringVarP(&compression, "compression", "c", "", "Compression type (gzip, snappy, lz4, zstd)")
  cmd.Flags().StringVar(&headers, "headers", "", "Message headers in format name=value,name2=value2")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagRequired("message")

  return cmd
}

// handleKafkaSend 处理发送消息
func (c *CLI) handleKafkaSend(config *kafka.ServerConfig, topic, message, key string, repeat, interval int, printLog bool, acks string, messageLength int, compression, headers string) error {
  client := kafka.NewClient(config)

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Kafka:", err.Error())
    return nil
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
    fmt.Println("ERROR failed to send message:", err.Error())
    return nil
  }

  fmt.Printf("Successfully sent %d messages to Kafka topic '%s'\n", repeat, topic)
  return nil
}

// setupKafkaReceiveCommand 设置接收消息命令
func (c *CLI) setupKafkaReceiveCommand() *cobra.Command {
  var topic string
  var groupID string
  var offset int64
  var offsetType string
  var timeout int
  var printLog bool
  var maxMessages int

  cmd := &cobra.Command{
    Use:   "receive",
    Short: "Receive messages from Kafka",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("kafkaConfig").(*kafka.ServerConfig)
      return c.handleKafkaReceive(config, topic, groupID, offset, offsetType, timeout, printLog, maxMessages)
    },
  }

  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic to subscribe")
  cmd.Flags().StringVarP(&groupID, "group-id", "g", "default_consumer_group", "Consumer group ID")
  cmd.Flags().Int64VarP(&offset, "offset", "o", 0, "Offset value (only valid if offset-type is 'specific')")
  cmd.Flags().StringVar(&offsetType, "offset-type", "latest", "Offset type (earliest, latest, specific)")
  cmd.Flags().IntVar(&timeout, "timeout", 0, "Consumer timeout in seconds (0 for no timeout)")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().IntVar(&maxMessages, "max-messages", 0, "Maximum number of messages to receive (0 for unlimited)")

  cmd.MarkFlagRequired("topic")

  return cmd
}

// handleKafkaReceive 处理接收消息
func (c *CLI) handleKafkaReceive(config *kafka.ServerConfig, topic, groupID string, offset int64, offsetType string, timeout int, printLog bool, maxMessages int) error {
  client := kafka.NewClient(config)

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to Kafka:", err.Error())
    return nil
  }
  defer client.Close()

  // 设置消费者配置
  consumerConfig := &kafka.ConsumerConfig{
    Topic:       topic,
    GroupID:     groupID,
    Offset:      offset,
    OffsetType:  offsetType,
    Timeout:     timeout,
    PrintLog:    printLog,
    MaxMessages: maxMessages,
  }

  // 接收消息
  if err := client.ReceiveMessage(consumerConfig); err != nil {
    fmt.Println("ERROR failed to receive message:", err.Error())
    return nil
  }

  fmt.Printf("Finished receiving messages from Kafka topic '%s'\n", topic)
  return nil
}

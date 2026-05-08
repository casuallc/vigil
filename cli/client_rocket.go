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
  "os"
  "path/filepath"

  "github.com/apache/rocketmq-client-go/v2/primitive"
  "github.com/casuallc/vigil/client/rocketmq"
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
  rocketCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "127.0.0.1", "RocketMQ server host")
  rocketCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 9876, "RocketMQ server port")
  rocketCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "", "Username for authentication")
  rocketCmd.PersistentFlags().StringVarP(&config.Namespace, "namespace", "n", "", "Namespace")
  rocketCmd.PersistentFlags().StringVar(&config.AccessKey, "access-key", "", "Access Key for authentication")
  rocketCmd.PersistentFlags().StringVar(&config.SecretKey, "secret-key", "", "Secret Key for authentication")

  // 存储配置到上下文
  rocketCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "rocketConfig", &config))
  }

  // 添加子命令
  rocketCmd.AddCommand(c.setupRocketSendCommand())
  rocketCmd.AddCommand(c.setupRocketReceiveCommand())
  rocketCmd.AddCommand(c.setupRocketBatchSendCommand())
  rocketCmd.AddCommand(c.setupRocketTransactionSendCommand())

  return rocketCmd
}

// setupRocketSendCommand 设置发送消息命令
func (c *CLI) setupRocketSendCommand() *cobra.Command {
  var groupName string
  var topic string
  var tags string
  var keys string
  var message string
  var filePath string
  var recursive bool
  var repeat int
  var interval int
  var sendType string
  var delayLevel int
  var printLog bool
  var useMessageTrace bool
  var messageLength int

  cmd := &cobra.Command{
    Use:   "send",
    Short: "Send message to RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("rocketConfig").(*rocketmq.ServerConfig)
      return c.handleRocketSend(config, groupName, topic, tags, keys, message, filePath, recursive, repeat, interval, sendType, delayLevel, printLog, useMessageTrace, messageLength)
    },
  }

  cmd.Flags().StringVarP(&groupName, "group", "g", "default_group", "Producer group name")
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVar(&tags, "tags", "", "Message tags")
  cmd.Flags().StringVarP(&keys, "keys", "k", "", "Message keys")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().StringVarP(&filePath, "file", "f", "", "File or directory path to read message content from")
  cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "Recursively send files in directory")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 1, "Number of times to repeat sending")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
  cmd.Flags().StringVar(&sendType, "send-type", "sync", "Send type: sync or async")
  cmd.Flags().IntVar(&delayLevel, "delay-level", 0, "Delay level for delayed messages")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().BoolVar(&useMessageTrace, "trace", false, "Use message trace")
  cmd.Flags().IntVar(&messageLength, "message-length", 0, "Message length, will pad with spaces if necessary")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagsOneRequired("message", "file")

  return cmd
}

// handleRocketSend 处理发送消息
func (c *CLI) handleRocketSend(config *rocketmq.ServerConfig, groupName string, topic string, tags string, keys string, message string, filePath string, recursive bool, repeat int, interval int, sendType string, delayLevel int, printLog bool, useMessageTrace bool, messageLength int) error {
  client := rocketmq.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to RocketMQ server:", err.Error())
    return nil
  }

  // 文件夹模式：遍历文件夹，每个文件作为一条消息发送
  if filePath != "" {
    info, err := os.Stat(filePath)
    if err != nil {
      fmt.Println("ERROR failed to stat file:", err.Error())
      return nil
    }

    if info.IsDir() {
      sentCount, err := c.sendRocketFilesInDir(client, filePath, recursive, groupName, topic, tags, keys, interval, sendType, delayLevel, printLog, useMessageTrace, messageLength)
      if err != nil {
        fmt.Println("ERROR failed to walk directory:", err.Error())
        return nil
      }
      fmt.Printf("Sent %d messages from directory '%s'\n", sentCount, filePath)
      return nil
    }
  }

  // 确定发送类型
  sendTypeEnum := rocketmq.SyncSend
  if sendType == "async" {
    sendTypeEnum = rocketmq.AsyncSend
  }

  // 创建生产者配置
  producerConfig := &rocketmq.ProducerConfig{
    GroupName:       groupName,
    Topic:           topic,
    Tags:            tags,
    Keys:            keys,
    Message:         message,
    MessageFile:     filePath,
    Repeat:          repeat,
    Interval:        interval,
    SendType:        sendTypeEnum,
    DelayLevel:      delayLevel,
    PrintLog:        printLog,
    UseMessageTrace: useMessageTrace,
    MessageLength:   messageLength,
  }

  // 发送消息
  if err := client.SendMessage(producerConfig); err != nil {
    fmt.Println("ERROR failed to send message:", err.Error())
    return nil
  }

  if filePath != "" {
    fmt.Printf("Successfully sent %d messages from file '%s' to topic %s\n", repeat, filePath, topic)
  } else {
    fmt.Printf("Successfully sent %d messages to topic %s\n", repeat, topic)
  }
  return nil
}

// sendRocketFilesInDir 遍历目录发送文件，每个文件作为一条消息
func (c *CLI) sendRocketFilesInDir(client *rocketmq.Client, dir string, recursive bool, groupName string, topic string, tags string, keys string, interval int, sendType string, delayLevel int, printLog bool, useMessageTrace bool, messageLength int) (int, error) {
  var sentCount int
  sendTypeEnum := rocketmq.SyncSend
  if sendType == "async" {
    sendTypeEnum = rocketmq.AsyncSend
  }
  err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
    if err != nil {
      return err
    }
    if d.IsDir() {
      if path == dir {
        return nil
      }
      if !recursive {
        return filepath.SkipDir
      }
      return nil
    }

    producerConfig := &rocketmq.ProducerConfig{
      GroupName:       groupName,
      Topic:           topic,
      Tags:            tags,
      Keys:            keys,
      MessageFile:     path,
      Repeat:          1,
      Interval:        interval,
      SendType:        sendTypeEnum,
      DelayLevel:      delayLevel,
      PrintLog:        printLog,
      UseMessageTrace: useMessageTrace,
      MessageLength:   messageLength,
    }

    if err := client.SendMessage(producerConfig); err != nil {
      fmt.Printf("ERROR failed to send file %s: %v\n", path, err)
    } else {
      sentCount++
    }
    return nil
  })
  return sentCount, err
}

// setupRocketReceiveCommand 设置接收消息命令
func (c *CLI) setupRocketReceiveCommand() *cobra.Command {
  var groupName string
  var topic string
  var tags string
  var timeout int
  var startConsumePos string
  var consumeTimestamp string
  var consumeType string
  var printLog bool
  var retryCount int
  var useMessageTrace bool

  cmd := &cobra.Command{
    Use:   "receive",
    Short: "Receive messages from RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("rocketConfig").(*rocketmq.ServerConfig)
      return c.handleRocketReceive(config, groupName, topic, tags, timeout, startConsumePos, consumeTimestamp, consumeType, printLog, retryCount, useMessageTrace)
    },
  }

  cmd.Flags().StringVarP(&groupName, "group", "g", "default_consumer_group", "Consumer group name")
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic to subscribe")
  cmd.Flags().StringVar(&tags, "tags", "*", "Message tags filter (use * for all)")
  cmd.Flags().IntVar(&timeout, "timeout", 0, "Consumer timeout in seconds (0 for no timeout)")
  cmd.Flags().StringVar(&startConsumePos, "start-pos", "LAST", "Start consume position: FIRST, LAST, TIMESTAMP")
  cmd.Flags().StringVar(&consumeTimestamp, "timestamp", "", "Consume timestamp in format 20060102150405")
  cmd.Flags().StringVar(&consumeType, "consume-type", "SYNC", "Consume type: SYNC or ASYNC")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().IntVar(&retryCount, "retry-count", 0, "Message retry count")
  cmd.Flags().BoolVar(&useMessageTrace, "trace", false, "Use message trace")

  cmd.MarkFlagRequired("topic")

  return cmd
}

// handleRocketReceive 处理接收消息
func (c *CLI) handleRocketReceive(config *rocketmq.ServerConfig, groupName string, topic string, tags string, timeout int, startConsumePos string, consumeTimestamp string, consumeType string, printLog bool, retryCount int, useMessageTrace bool) error {
  client := rocketmq.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to RocketMQ server:", err.Error())
    return nil
  }

  // 创建消费者配置
  consumerConfig := &rocketmq.ConsumerConfig{
    GroupName:        groupName,
    Topic:            topic,
    Tags:             tags,
    Timeout:          timeout,
    StartConsumePos:  startConsumePos,
    ConsumeTimestamp: consumeTimestamp,
    ConsumeType:      consumeType,
    PrintLog:         printLog,
    RetryCount:       retryCount,
    UseMessageTrace:  useMessageTrace,
    Handler: func(msg *rocketmq.Message) bool {
      fmt.Printf("\nReceived message:\n")
      fmt.Printf("  Topic: %s\n", msg.Topic)
      fmt.Printf("  Tags: %s\n", msg.Tags)
      fmt.Printf("  Keys: %s\n", msg.Keys)
      fmt.Printf("  MsgID: %s\n", msg.MsgID)
      fmt.Printf("  QueueID: %d\n", msg.QueueID)
      fmt.Printf("  Body: %s\n", msg.Body)
      return true
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

// setupRocketBatchSendCommand 设置批量发送消息命令
func (c *CLI) setupRocketBatchSendCommand() *cobra.Command {
  var groupName string
  var topic string
  var tags string
  var keys string
  var message string
  var filePath string
  var repeat int
  var interval int
  var batchSize int
  var printLog bool
  var useMessageTrace bool

  cmd := &cobra.Command{
    Use:   "batch-send",
    Short: "Batch send messages to RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("rocketConfig").(*rocketmq.ServerConfig)
      return c.handleRocketBatchSend(config, groupName, topic, tags, keys, message, filePath, repeat, interval, batchSize, printLog, useMessageTrace)
    },
  }

  cmd.Flags().StringVarP(&groupName, "group", "g", "default_group", "Producer group name")
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVar(&tags, "tags", "", "Message tags")
  cmd.Flags().StringVarP(&keys, "keys", "k", "", "Message keys")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().StringVarP(&filePath, "file", "f", "", "File path to read message content from")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 1, "Number of times to repeat sending")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between batches in milliseconds")
  cmd.Flags().IntVar(&batchSize, "batch-size", 10, "Batch size")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().BoolVar(&useMessageTrace, "trace", false, "Use message trace")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagsOneRequired("message", "file")

  return cmd
}

// handleRocketBatchSend 处理批量发送消息
func (c *CLI) handleRocketBatchSend(config *rocketmq.ServerConfig, groupName string, topic string, tags string, keys string, message string, filePath string, repeat int, interval int, batchSize int, printLog bool, useMessageTrace bool) error {
  client := rocketmq.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to RocketMQ server:", err.Error())
    return nil
  }

  // 创建生产者配置
  producerConfig := &rocketmq.ProducerConfig{
    GroupName:       groupName,
    Topic:           topic,
    Tags:            tags,
    Keys:            keys,
    Message:         message,
    MessageFile:     filePath,
    Repeat:          repeat,
    Interval:        interval,
    BatchSize:       batchSize,
    PrintLog:        printLog,
    UseMessageTrace: useMessageTrace,
  }

  // 发送消息
  if err := client.SendMessage(producerConfig); err != nil {
    fmt.Println("ERROR failed to send batch messages:", err.Error())
    return nil
  }

  if filePath != "" {
    fmt.Printf("Successfully sent %d batch messages from file '%s' to topic %s\n", repeat, filePath, topic)
  } else {
    fmt.Printf("Successfully sent %d batch messages to topic %s\n", repeat, topic)
  }
  return nil
}

// setupRocketTransactionSendCommand 设置事务消息发送命令
func (c *CLI) setupRocketTransactionSendCommand() *cobra.Command {
  var groupName string
  var topic string
  var tags string
  var keys string
  var message string
  var filePath string
  var repeat int
  var interval int
  var printLog bool
  var checkTimes int

  cmd := &cobra.Command{
    Use:   "transaction-send",
    Short: "Send transaction messages to RocketMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("rocketConfig").(*rocketmq.ServerConfig)
      return c.handleRocketTransactionSend(config, groupName, topic, tags, keys, message, filePath, repeat, interval, printLog, checkTimes)
    },
  }

  cmd.Flags().StringVarP(&groupName, "group", "g", "default_transaction_group", "Transaction producer group name")
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "Message topic")
  cmd.Flags().StringVar(&tags, "tags", "", "Message tags")
  cmd.Flags().StringVarP(&keys, "keys", "k", "", "Message keys")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().StringVarP(&filePath, "file", "f", "", "File path to read message content from")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 1, "Number of times to repeat sending")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
  cmd.Flags().IntVar(&checkTimes, "check-times", 3, "Transaction check times")

  cmd.MarkFlagRequired("topic")
  cmd.MarkFlagsOneRequired("message", "file")

  return cmd
}

// handleRocketTransactionSend 处理事务消息发送
func (c *CLI) handleRocketTransactionSend(config *rocketmq.ServerConfig, groupName string, topic string, tags string, keys string, message string, filePath string, repeat int, interval int, printLog bool, checkTimes int) error {
  client := rocketmq.NewClient(config)
  defer client.Close()

  if err := client.Connect(); err != nil {
    fmt.Println("ERROR failed to connect to RocketMQ server:", err.Error())
    return nil
  }

  listener := &simpleTransactionListener{printLog: printLog}
  producerConfig := &rocketmq.ProducerConfig{
    GroupName:   groupName,
    Topic:       topic,
    Tags:        tags,
    Keys:        keys,
    Message:     message,
    MessageFile: filePath,
    Repeat:      repeat,
    Interval:    interval,
    PrintLog:    printLog,
    CheckTimes:  checkTimes,
  }

  if err := client.SendTransactionMessage(producerConfig, listener); err != nil {
    fmt.Println("ERROR failed to send transaction message:", err.Error())
    return nil
  }

  if filePath != "" {
    fmt.Printf("Successfully sent %d transaction messages from file '%s' to topic %s\n", repeat, filePath, topic)
  } else {
    fmt.Printf("Successfully sent %d transaction messages to topic %s\n", repeat, topic)
  }
  return nil
}

// simpleTransactionListener 简单的事务监听器实现
type simpleTransactionListener struct {
  printLog bool
}

// ExecuteLocalTransaction 执行本地事务
func (l *simpleTransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
  // 模拟本地事务执行
  if l.printLog {
    fmt.Printf("Executing local transaction for message: %s\n", msg.TransactionId)
  }
  // 这里返回 COMMIT_MESSAGE 表示事务成功，实际应用中应根据业务逻辑判断
  return primitive.CommitMessageState
}

// CheckLocalTransaction 检查本地事务状态
func (l *simpleTransactionListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
  // 模拟事务回查
  if l.printLog {
    fmt.Printf("Checking local transaction for message: %s\n", msg.MsgId)
  }
  // 这里返回 COMMIT_MESSAGE 表示事务成功，实际应用中应查询事务状态
  return primitive.CommitMessageState
}

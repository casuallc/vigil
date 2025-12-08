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

package rocketmq

import (
  "context"
  "fmt"
  "github.com/apache/rocketmq-client-go/v2"
  "log"
  "os"
  "os/signal"
  "strings"
  "syscall"
  "time"

  "github.com/apache/rocketmq-client-go/v2/consumer"
  "github.com/apache/rocketmq-client-go/v2/primitive"
)

// CreateConsumer 创建消费者
func (c *Client) CreateConsumer(consumeConfig *ConsumerConfig) error {

  // 关闭已存在的消费者
  if c.consumer != nil {
    _ = c.consumer.Shutdown()
  }

  addr := fmt.Sprintf("%s:%d", c.Config.Server, c.Config.Port)

  // 1. 先根据配置确定 ConsumeFromWhere 的值
  var consumeFromWhere consumer.ConsumeFromWhere
  switch strings.ToUpper(consumeConfig.StartConsumePos) {
  case "FIRST":
    consumeFromWhere = consumer.ConsumeFromFirstOffset
  case "LAST":
    consumeFromWhere = consumer.ConsumeFromLastOffset
  case "TIMESTAMP":
    // 使用配置中的时间戳（建议从 config 读取，而不是默认当前时间）
    // 如果 config 没有提供具体时间戳，才用当前时间
    timestamp := consumeConfig.ConsumeTimestamp
    if timestamp == "" {
      timestamp = time.Now().Format("20060102150405")
    }
    consumeFromWhere = consumer.ConsumeFromTimestamp
  default:
    // 默认行为：从最后 offset 开始（或根据业务需求）
    consumeFromWhere = consumer.ConsumeFromLastOffset
  }

  options := []consumer.Option{
    consumer.WithNameServer([]string{addr}),
    consumer.WithGroupName(consumeConfig.GroupName),
    consumer.WithConsumeFromWhere(consumeFromWhere),
    consumer.WithConsumeTimestamp(consumeConfig.ConsumeTimestamp),
  }

  // 添加命名空间配置
  if c.Config.Namespace != "" {
    options = append(options, consumer.WithNamespace(c.Config.Namespace))
  }

  // 添加认证配置
  if c.Config.AccessKey != "" && c.Config.SecretKey != "" {
    options = append(options, consumer.WithCredentials(primitive.Credentials{
      AccessKey: c.Config.AccessKey,
      SecretKey: c.Config.SecretKey,
    }))
  }

  con, err := rocketmq.NewPushConsumer(options...)

  if err != nil {
    return fmt.Errorf("failed to create consumer: %w", err)
  }

  c.consumer = con
  log.Printf("Consumer created with group: %s", consumeConfig.GroupName)
  return nil
}

// ReceiveMessage 接收消息
func (c *Client) ReceiveMessage(config *ConsumerConfig) error {

  if c.consumer == nil {
    // 如果没有消费者，先创建一个
    if err := c.CreateConsumer(config); err != nil {
      return err
    }
  }

  // 默认订阅
  subExpression := "*"
  if config.Tags != "" {
    subExpression = config.Tags
  }

  if err := c.consumer.Subscribe(config.Topic, consumer.MessageSelector{Type: consumer.TAG, Expression: subExpression},
    func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
      return c.handleMessages(msgs, config)
    }); err != nil {
    return fmt.Errorf("failed to subscribe topic: %w", err)
  }

  // 启动消费者
  if err := c.consumer.Start(); err != nil {
    return fmt.Errorf("failed to start consumer: %w", err)
  }

  if config.PrintLog {
    log.Printf("Started consuming messages from topic %s", config.Topic)
  }

  // 设置超时
  if config.Timeout > 0 {
    timer := time.NewTimer(time.Duration(config.Timeout) * time.Second)
    <-timer.C
    if config.PrintLog {
      log.Printf("Consumer timeout after %d seconds", config.Timeout)
    }
  } else {
    // 如果没有设置超时，一直运行直到被中断
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    if config.PrintLog {
      log.Printf("Consumer interrupted")
    }
  }

  return nil
}

// handleMessages 处理接收到的消息
func (c *Client) handleMessages(msgs []*primitive.MessageExt, config *ConsumerConfig) (consumer.ConsumeResult, error) {
  // AI Modified: 记录消费的消息总数
  c.consumedCount += int64(len(msgs))
  
  for _, msg := range msgs {
    rocketMsg := &Message{
      Topic:         msg.Topic,
      Tags:          msg.GetTags(),
      Keys:          msg.GetKeys(),
      Body:          string(msg.Body),
      MsgID:         msg.MsgId,
      QueueID:       int32(msg.Queue.QueueId),
      StoreTime:     msg.StoreTimestamp,
      TransactionID: msg.GetProperty("TRANSACTION_ID"),
    }

    // 处理消息重试
    retryCount := 0
    retryProperty := msg.GetProperty("RETRY_TOPIC")
    if retryProperty != "" {
      // 尝试获取重试次数
      retryCountProperty := msg.GetProperty("REAL_RETRY_TIMES")
      if retryCountProperty != "" {
        fmt.Sscanf(retryCountProperty, "%d", &retryCount)
      }
    }

    var success bool
    if config.Handler != nil {
      success = config.Handler(rocketMsg)
    } else {
      // 默认处理逻辑
      success = true
      if config.PrintLog {
        log.Printf("Received message: Topic=%s, Tags=%s, MsgID=%s, Body=%s",
          msg.Topic, msg.GetTags(), msg.MsgId, string(msg.Body))
      }
    }

    // 如果配置了重试次数，并且当前重试次数小于配置的次数，模拟消费失败
    if config.RetryCount > 0 && retryCount < config.RetryCount {
      if config.PrintLog {
        log.Printf("Simulating consume failure for retry %d/%d, msgID: %s",
          retryCount+1, config.RetryCount, msg.MsgId)
      }
      return consumer.ConsumeRetryLater, nil
    }

    if !success {
      if config.PrintLog {
        log.Printf("Message handling failed, msgID: %s", msg.MsgId)
      }
      return consumer.ConsumeRetryLater, nil
    }
  }

  return consumer.ConsumeSuccess, nil
}

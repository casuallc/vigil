package rocketmq

import (
  "context"
  "fmt"
  "github.com/apache/rocketmq-client-go/v2"
  "log"
  "strings"
  "sync"
  "time"

  "github.com/apache/rocketmq-client-go/v2/primitive"
  "github.com/apache/rocketmq-client-go/v2/producer"
)

// CreateProducer 创建生产者
func (c *Client) CreateProducer(groupName string) error {

  // 关闭已存在的生产者
  if c.producer != nil {
    _ = c.producer.Shutdown()
  }

  addr := fmt.Sprintf("%s:%d", c.Config.Server, c.Config.Port)

  options := []producer.Option{
    producer.WithNameServer([]string{addr}),
    producer.WithGroupName(groupName),
  }

  // 添加命名空间配置
  if c.Config.Namespace != "" {
    options = append(options, producer.WithNamespace(c.Config.Namespace))
  }

  // 添加认证配置
  if c.Config.AccessKey != "" && c.Config.SecretKey != "" {
    options = append(options, producer.WithCredentials(primitive.Credentials{
      AccessKey: c.Config.AccessKey,
      SecretKey: c.Config.SecretKey,
    }))
  }

  p, err := rocketmq.NewProducer(options...)

  if err != nil {
    return fmt.Errorf("failed to create producer: %w", err)
  }

  if err = p.Start(); err != nil {
    return fmt.Errorf("failed to start producer: %w", err)
  }

  c.producer = p
  log.Printf("Producer created with group: %s", groupName)
  return nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(config *ProducerConfig) error {

  if c.producer == nil {
    // 如果没有生产者，先创建一个
    if err := c.CreateProducer(config.GroupName); err != nil {
      return err
    }
  }

  // 处理批量发送
  if config.BatchSize > 1 {
    return c.sendBatchMessages(config)
  }

  ticker := time.NewTicker(time.Duration(config.Interval) * time.Millisecond)
  defer ticker.Stop()

  var wg sync.WaitGroup

  for i := 0; i < config.Repeat; i++ {
    if config.Interval > 0 && i > 0 {
      <-ticker.C
    }

    // 准备消息内容
    messageBody := config.Message
    // 如果指定了消息长度，并且当前消息长度不足，则补全
    if config.MessageLength > 0 && len(messageBody) < config.MessageLength {
      messageBody = messageBody + strings.Repeat(" ", config.MessageLength-len(messageBody))
    }

    msg := primitive.NewMessage(config.Topic, []byte(messageBody))
    msg.WithTag(config.Tags)
    msg.WithKeys(strings.Split(config.Keys, ","))

    // 设置延迟消息
    if config.DelayLevel > 0 {
      msg.WithDelayTimeLevel(config.DelayLevel)
    }

    // 设置定时消息
    if config.DeliverTime != nil {
      delay := time.Until(*config.DeliverTime)
      msg.WithDelayTimeLevel(int(delay.Seconds() / 10))
    }

    wg.Add(1)

    // 根据发送方式发送消息
    if config.SendType == SyncSend {
      // 同步发送
      go func(idx int, message *primitive.Message) {
        defer wg.Done()

        result, err := c.producer.SendSync(c.ctx, message)

        if err != nil {
          log.Printf("Failed to send message %d: %v", idx, err)
        } else if config.PrintLog {
          log.Printf("Message %d sent successfully, msgID: %s, queueID: %d",
            idx, result.MsgID, result.MessageQueue.QueueId)
        }

        // 如果有回调函数，调用回调
        if config.AsyncCallback != nil {
          config.AsyncCallback(result, err)
        }
      }(i, msg)
    } else {
      // 异步发送
      go func(idx int, message *primitive.Message) {
        defer wg.Done()

        callback := func(ctx context.Context, result *primitive.SendResult, err error) {
          if err != nil {
            log.Printf("Failed to send message %d: %v", idx, err)
          } else if config.PrintLog {
            log.Printf("Message %d sent successfully, msgID: %s, queueID: %d",
              idx, result.MsgID, result.MessageQueue.QueueId)
          }

          // 如果有回调函数，调用回调
          if config.AsyncCallback != nil {
            config.AsyncCallback(result, err)
          }
        }

        err := c.producer.SendAsync(c.ctx, callback, message)
        if err != nil {
          fmt.Printf("Failed to send message %d: %v", idx, err)
          return
        }
      }(i, msg)
    }
  }

  wg.Wait()
  c.producedCount += int64(config.Repeat)
  log.Printf("Total messages sent: %d", config.Repeat)
  return nil
}

// sendBatchMessages 批量发送消息
func (c *Client) sendBatchMessages(config *ProducerConfig) error {
  // 准备批量消息
  var batchMessages []*primitive.Message
  messageCount := 0

  for i := 0; i < config.Repeat; i++ {
    // 准备消息内容
    messageBody := config.Message
    // 如果指定了消息长度，并且当前消息长度不足，则补全
    if config.MessageLength > 0 && len(messageBody) < config.MessageLength {
      messageBody = messageBody + strings.Repeat(" ", config.MessageLength-len(messageBody))
    }

    msg := primitive.NewMessage(config.Topic, []byte(messageBody))
    msg.WithTag(config.Tags)
    msg.WithKeys(strings.Split(config.Keys, ","))

    // 设置延迟消息
    if config.DelayLevel > 0 {
      msg.WithDelayTimeLevel(config.DelayLevel)
    }

    // 设置定时消息
    if config.DeliverTime != nil {
      delay := time.Until(*config.DeliverTime)
      msg.WithDelayTimeLevel(int(delay.Seconds() / 10))
    }

    batchMessages = append(batchMessages, msg)
    messageCount++

    // 如果达到批量大小或者是最后一条消息，发送批次
    if messageCount >= config.BatchSize || i == config.Repeat-1 {
      result, err := c.producer.SendSync(c.ctx, batchMessages...)
      if err != nil {
        return fmt.Errorf("failed to send batch message: %w", err)
      } else if config.PrintLog {
        log.Printf("Batch message sent successfully, msgID: %s, queueID: %d",
          result.MsgID, result.MessageQueue.QueueId)
      }

      // 如果有回调函数，调用回调
      if config.AsyncCallback != nil {
        config.AsyncCallback(result, err)
      }

      // 重置批次
      batchMessages = []*primitive.Message{}
      messageCount = 0

      // 间隔时间
      if config.Interval > 0 && i < config.Repeat-1 {
        time.Sleep(time.Duration(config.Interval) * time.Millisecond)
      }
    }
  }

  c.producedCount += int64(config.Repeat)
  log.Printf("Total batch messages sent: %d", config.Repeat)
  return nil
}

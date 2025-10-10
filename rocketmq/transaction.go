package rocketmq

import (
  "fmt"
  "github.com/apache/rocketmq-client-go/v2"
  "log"
  "strings"
  "time"

  "github.com/apache/rocketmq-client-go/v2/primitive"
  "github.com/apache/rocketmq-client-go/v2/producer"
)

// CreateTransactionProducer 创建事务生产者
func (c *Client) CreateTransactionProducer(groupName string, listener TransactionListener) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  // 关闭已存在的事务生产者
  if c.transactionProducer != nil {
    _ = c.transactionProducer.Shutdown()
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

  // 创建事务监听器包装器
  wrapper := &transactionListenerWrapper{
    listener: listener,
  }

  p, err := rocketmq.NewTransactionProducer(wrapper, options...)

  if err != nil {
    return fmt.Errorf("failed to create transaction producer: %w", err)
  }

  if err = p.Start(); err != nil {
    return fmt.Errorf("failed to start transaction producer: %w", err)
  }

  c.transactionProducer = p
  log.Printf("Transaction producer created with group: %s", groupName)
  return nil
}

// transactionListenerWrapper 包装事务监听器接口
type transactionListenerWrapper struct {
  listener TransactionListener
}

// ExecuteLocalTransaction 执行本地事务
func (w *transactionListenerWrapper) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
  return w.listener.ExecuteLocalTransaction(msg)
}

// CheckLocalTransaction 检查本地事务状态
func (w *transactionListenerWrapper) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
  return w.listener.CheckLocalTransaction(msg)
}

// SendTransactionMessage 发送事务消息
func (c *Client) SendTransactionMessage(config *ProducerConfig, listener TransactionListener) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  if c.transactionProducer == nil {
    // 如果没有事务生产者，先创建一个
    if err := c.CreateTransactionProducer(config.GroupName, listener); err != nil {
      return err
    }
  }

  ticker := time.NewTicker(time.Duration(config.Interval) * time.Millisecond)
  defer ticker.Stop()

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

    // 设置自定义属性用于事务回查
    msg.WithProperty("CHECK_TIMES", fmt.Sprintf("%d", config.CheckTimes))

    result, err := c.transactionProducer.SendMessageInTransaction(c.ctx, msg)
    if err != nil {
      return fmt.Errorf("failed to send transaction message: %w", err)
    } else if config.PrintLog {
      log.Printf("Transaction message sent successfully, msgID: %s, transactionID: %s, state: %v",
        result.MsgID, result.TransactionID, result.State)
    }
  }

  if config.PrintLog {
    log.Printf("Total transaction messages sent: %d", config.Repeat)
  }
  return nil
}

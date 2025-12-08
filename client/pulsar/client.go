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

package pulsar

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/common"
  "log"
  "strconv"
  "strings"
  "sync"
  "time"

  "github.com/apache/pulsar-client-go/pulsar"
)

// Client 表示 Pulsar 客户端
type Client struct {
  config         *ServerConfig
  client         pulsar.Client
  producers      map[string]pulsar.Producer
  consumers      map[string]pulsar.Consumer
  mu             sync.Mutex
  ctx            context.Context
  producedCount  int64 // AI Modified: 记录生产的消息总数
  consumedCount  int64 // AI Modified: 记录消费的消息总数
}

// NewClient 创建一个新的 Pulsar 客户端
func NewClient(config *ServerConfig) *Client {
  return &Client{
    config:    config,
    producers: make(map[string]pulsar.Producer),
    consumers: make(map[string]pulsar.Consumer),
    ctx:       context.Background(),
  }
}

// Connect 连接到 Pulsar 服务器
func (c *Client) Connect() error {

  if c.client != nil {
    return nil // 已经连接
  }

  // 创建客户端配置
  clientConfig := pulsar.ClientOptions{
    URL: c.config.URL,
  }

  if c.config.Auth.Type == "" {
    c.config.Auth.Type = "token"
  }

  // 设置认证信息
  if c.config.Auth.Type != "none" {
    switch c.config.Auth.Type {
    case "token":
      clientConfig.Authentication = pulsar.NewAuthenticationToken(c.config.Auth.Token)
    case "tls":
      clientConfig.TLSAllowInsecureConnection = false
      if c.config.Auth.CertFile != "" {
        clientConfig.TLSCertificateFile = c.config.Auth.CertFile
      }
      if c.config.Auth.KeyFile != "" {
        clientConfig.TLSKeyFilePath = c.config.Auth.KeyFile
      }
      if c.config.Auth.CAFile != "" {
        clientConfig.TLSTrustCertsFilePath = c.config.Auth.CAFile
      }
    case "basic":
      // Pulsar Go 客户端不直接支持基本认证，需要通过自定义认证器实现
      // 这里简化处理
      log.Println("Warning: Basic authentication is not directly supported in this implementation")
    default:
      return fmt.Errorf("unsupported authentication type: %s", c.config.Auth.Type)
    }
  }

  // 设置超时时间
  if c.config.Timeout > 0 {
    clientConfig.OperationTimeout = time.Duration(c.config.Timeout) * time.Second
  }

  // 创建 Pulsar 客户端
  client, err := pulsar.NewClient(clientConfig)
  if err != nil {
    return fmt.Errorf("failed to create pulsar client: %v", err)
  }

  c.client = client
  log.Printf("Pulsar client connected to %s", c.config.URL)
  return nil
}

// Close 关闭客户端
func (c *Client) Close() {

  // 关闭所有生产者
  for _, producer := range c.producers {
    producer.Close()
  }
  c.producers = make(map[string]pulsar.Producer)

  // 关闭所有消费者
  for _, consumer := range c.consumers {
    consumer.Close()
  }
  c.consumers = make(map[string]pulsar.Consumer)

  // 关闭客户端
  if c.client != nil {
    c.client.Close()
    c.client = nil
  }
  // AI Modified: 打印消息计数
  log.Printf("Pulsar Client Stats - Produced: %d, Consumed: %d", c.producedCount, c.consumedCount)
  log.Println("Pulsar client closed")
}

// CreateProducer 创建一个生产者
func (c *Client) CreateProducer(config ProducerConfig) (pulsar.Producer, error) {

  // 确保客户端已连接
  if c.client == nil {
    if err := c.Connect(); err != nil {
      return nil, err
    }
  }

  // 检查生产者是否已存在
  if producer, exists := c.producers[config.Topic]; exists {
    return producer, nil
  }

  // 创建生产者配置
  producerConfig := pulsar.ProducerOptions{
    Topic: config.Topic,
  }

  // 设置压缩（如果启用）
  if config.EnableCompression {
    producerConfig.CompressionType = pulsar.LZ4
  }

  // 设置批处理选项
  if config.EnableBatching {
    producerConfig.DisableBatching = false
    if config.BatchingMaxPublishDelay > 0 {
      producerConfig.BatchingMaxPublishDelay = time.Duration(config.BatchingMaxPublishDelay) * time.Millisecond
    }
    if config.BatchingMaxMessages > 0 {
      producerConfig.BatchingMaxMessages = uint(config.BatchingMaxMessages)
    }
  } else {
    producerConfig.DisableBatching = true
  }

  // 创建生产者
  producer, err := c.client.CreateProducer(producerConfig)
  if err != nil {
    return nil, fmt.Errorf("failed to create producer: %v", err)
  }

  // 存储生产者
  c.producers[config.Topic] = producer
  if config.PrintLog {
    log.Printf("Producer created for topic: %s", config.Topic)
  }

  return producer, nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(config ProducerConfig) error {

  // 获取或创建生产者
  producer, err := c.CreateProducer(config)
  if err != nil {
    return err
  }

  // 设置重复次数和间隔
  repeat := config.Repeat
  if repeat <= 0 {
    repeat = 1
  }

  interval := config.Interval

  // 解析消息属性
  properties := make(map[string]string)
  if config.Properties != "" {
    err := parseProperties(config.Properties, properties)
    if err != nil {
      return fmt.Errorf("invalid properties format: %v", err)
    }
  }

  // 创建计时器
  var ticker *time.Ticker
  if interval > 0 {
    ticker = time.NewTicker(time.Duration(interval) * time.Millisecond)
    defer ticker.Stop()
  }

  // 发送消息
  for i := 0; i < repeat; i++ {
    // 如果有间隔且不是第一条消息，等待间隔时间
    if interval > 0 && i > 0 {
      <-ticker.C
    }

    // 准备消息内容，根据长度要求补全
    messageBody := config.Message
    if config.MessageLength > 0 && len(messageBody) < config.MessageLength {
      messageBody = messageBody + strings.Repeat(" ", config.MessageLength-len(messageBody))
    }

    // 创建消息
    message := &pulsar.ProducerMessage{
      Payload:    []byte(messageBody),
      Properties: properties,
    }

    // 设置消息键
    if config.Key != "" {
      message.Key = config.Key
    }

    // 设置延迟消息
    if config.DelayTime > 0 {
      // Pulsar 支持通过消息属性设置延迟消息
      message.Properties["DELAY_TIME"] = strconv.Itoa(config.DelayTime)
    }

    // 设置定时消息
    if config.DeliverTime != nil {
      // 计算从现在到投递时间的延迟
      delay := time.Until(*config.DeliverTime)
      message.Properties["DELIVERY_TIME"] = strconv.Itoa(int(delay.Milliseconds()))
    }

    // 设置消息压缩
    if config.EnableCompression {
      // 在 Pulsar Go 客户端中，压缩是在生产者级别设置的，不是在消息级别
      // 因此这里只是记录日志，实际的压缩设置需要在 CreateProducer 方法中处理
      log.Println("Message compression is enabled (set at producer level)")
    }

    // 发送消息
    ctx := c.ctx
    if config.SendTimeout > 0 {
      var cancel context.CancelFunc
      ctx, cancel = context.WithTimeout(ctx, time.Duration(config.SendTimeout)*time.Millisecond)
      defer cancel()
    }

    // 发送消息并等待确认
    msgId, err := producer.Send(ctx, message)
    if err != nil {
      return fmt.Errorf("failed to send message %d: %v", i+1, err)
    }

    if config.PrintLog {
      log.Printf("Message %d sent successfully, message ID: %v", i+1, msgId)
    }
  }

  c.producedCount += int64(config.Repeat)
  log.Printf("Total messages sent: %d", config.Repeat)

  return nil
}

// parseProperties 解析消息属性字符串
func parseProperties(propsStr string, props map[string]string) error {
  array := common.ParsePropertyArray(propsStr)
  if array == nil {
    return nil
  }
  for _, pair := range array {
    key := pair[0]
    value := pair[1]
    props[key] = value
  }
  return nil
}

// CreateConsumer 创建一个消费者
func (c *Client) CreateConsumer(config ConsumerConfig) (pulsar.Consumer, error) {

  // 确保客户端已连接
  if c.client == nil {
    if err := c.Connect(); err != nil {
      return nil, err
    }
  }

  // 生成消费者ID
  consumerID := fmt.Sprintf("%s-%s", config.Topic, config.Subscription)

  // 检查消费者是否已存在
  if consumer, exists := c.consumers[consumerID]; exists {
    return consumer, nil
  }

  // 创建消费者配置
  consumerConfig := pulsar.ConsumerOptions{
    Topic:            config.Topic,
    SubscriptionName: config.Subscription,
  }

  // 设置订阅类型
  switch config.SubscriptionType {
  case "Shared":
    consumerConfig.Type = pulsar.Shared
  case "Failover":
    consumerConfig.Type = pulsar.Failover
  case "Key_Shared":
    consumerConfig.Type = pulsar.KeyShared
  default:
    consumerConfig.Type = pulsar.Exclusive
  }

  // 设置初始位置
  if config.InitialPosition == "Earliest" {
    consumerConfig.SubscriptionInitialPosition = pulsar.SubscriptionPositionEarliest
  } else {
    consumerConfig.SubscriptionInitialPosition = pulsar.SubscriptionPositionLatest
  }

  // 创建消费者
  consumer, err := c.client.Subscribe(consumerConfig)
  if err != nil {
    return nil, fmt.Errorf("failed to create consumer: %v", err)
  }

  // 存储消费者
  c.consumers[consumerID] = consumer
  log.Printf("Consumer created for topic: %s, subscription: %s", config.Topic, config.Subscription)

  return consumer, nil
}

// ReceiveMessage 接收消息
func (c *Client) ReceiveMessage(config ConsumerConfig, handler func(*Message) bool) error {

  // 获取或创建消费者
  consumer, err := c.CreateConsumer(config)
  if err != nil {
    return err
  }

  // 创建上下文
  ctx := c.ctx

  // 创建超时控制（如果需要）
  var cancel context.CancelFunc
  if config.MessageTimeout > 0 {
    ctx, cancel = context.WithTimeout(ctx, time.Duration(config.MessageTimeout)*time.Second)
    defer cancel()
  }

  // 使用单独的 goroutine 接收消息
  go func() {
    // 接收消息循环
    for {
      select {
      case <-ctx.Done():
        log.Printf("Consumer stopped, reason: %v", ctx.Err())
        return
      default:
        // 创建接收上下文
        receiveCtx := context.Background()
        if config.ReceiveTimeout > 0 {
          var receiveCancel context.CancelFunc
          receiveCtx, receiveCancel = context.WithTimeout(receiveCtx, time.Duration(config.ReceiveTimeout)*time.Millisecond)
          defer receiveCancel()
        }

        // 接收消息
        msg, err := consumer.Receive(receiveCtx)
        if err != nil {
          if ctx.Err() == nil {
            log.Printf("Error receiving message: %v", err)
          }
          continue
        }

        // 转换为我们的消息结构
        message := &Message{
          Topic:       msg.Topic(),
          Key:         msg.Key(),
          Payload:     msg.Payload(),
          MessageID:   msg.ID().String(),
          PublishTime: msg.PublishTime(),
          EventTime:   msg.EventTime(),
          Properties:  msg.Properties(),
        }

        // AI Modified: 记录消费的消息总数
        c.consumedCount++
        
        // 调用处理函数
        continueProcessing := handler(message)
        if !continueProcessing {
          log.Println("Consumer stopped by handler")
          return
        }

        // 确认消息
        if config.AutoAck {
          if err := consumer.Ack(msg); err != nil {
            log.Printf("Failed to acknowledge message: %v", err)
          }
        }
      }
    }
  }()

  // 等待上下文完成
  <-ctx.Done()
  return ctx.Err()
}

// AcknowledgeMessage 确认消息
func (c *Client) AcknowledgeMessage(consumer pulsar.Consumer, msgID string) error {
  // 注意：这里简化了实现，实际上需要将字符串类型的 msgID 转换为 pulsar.MessageID
  // 完整实现需要处理消息ID的序列化和反序列化
  log.Println("Warning: AcknowledgeMessage is not fully implemented")
  return nil
}

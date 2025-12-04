package kafka

import (
  "context"
  "errors"
  "fmt"
  "github.com/casuallc/vigil/common"
  "log"
  "strings"
  "sync"
  "time"

  "github.com/IBM/sarama"
)

// Client 定义Kafka客户端
type Client struct {
  producer      sarama.SyncProducer
  consumer      sarama.Consumer
  consumerGroup sarama.ConsumerGroup
  client        sarama.Client
  config        *ServerConfig
  mu            sync.Mutex
  ctx           context.Context
  producedCount int64 // AI Modified: 记录生产的消息总数
  consumedCount int64 // AI Modified: 记录消费的消息总数
}

// NewClient 创建新的Kafka客户端
func NewClient(config *ServerConfig) *Client {
  return &Client{
    config: config,
    ctx:    context.Background(),
  }
}

// Connect 连接到Kafka服务器
func (c *Client) Connect() error {

  config := sarama.NewConfig()
  config.Version = sarama.V2_0_0_0 // 使用兼容的Kafka版本
  config.ClientID = "vigil-cli"

  // 设置连接超时
  if c.config.Timeout > 0 {
    config.Net.DialTimeout = time.Duration(c.config.Timeout) * time.Second
  }

  // 配置SASL认证
  if c.config.User != "" && c.config.Password != "" {
    config.Net.SASL.Enable = true
    config.Net.SASL.User = c.config.User
    config.Net.SASL.Password = c.config.Password

    // 设置SASL机制
    if c.config.SASLMechanism != "" {
      switch c.config.SASLMechanism {
      case "PLAIN":
        config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
      case "SCRAM-SHA-256":
        config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
      case "SCRAM-SHA-512":
        config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
      }
    }
  }

  // 解析服务器地址
  serverAddrs := strings.Split(c.config.Servers, ",")
  if len(serverAddrs) == 0 {
    return fmt.Errorf("no Kafka servers specified")
  }

  // 如果指定了端口，构建完整的服务器地址
  if c.config.Port > 0 {
    for i, addr := range serverAddrs {
      serverAddrs[i] = fmt.Sprintf("%s:%d", addr, c.config.Port)
    }
  }

  // 创建客户端
  client, err := sarama.NewClient(serverAddrs, config)
  if err != nil {
    return fmt.Errorf("failed to create Kafka client: %w", err)
  }

  c.client = client
  log.Printf("Connected to Kafka servers: %s", strings.Join(serverAddrs, ","))
  return nil
}

// Close 关闭客户端连接
func (c *Client) Close() {

  if c.producer != nil {
    _ = c.producer.Close()
  }

  if c.consumer != nil {
    _ = c.consumer.Close()
  }

  if c.client != nil {
    _ = c.client.Close()
  }
  // AI Modified: 打印消息计数
  log.Printf("Kafka Client Stats - Produced: %d, Consumed: %d", c.producedCount, c.consumedCount)
}

// CreateProducer 创建生产者
func (c *Client) CreateProducer(config *ProducerConfig) error {
  // 关闭已存在的生产者
  if c.producer != nil {
    _ = c.producer.Close()
  }

  // 获取当前客户端的配置副本
  producerConfig := c.client.Config()
  producerConfig.Producer.Return.Successes = true

  // 设置acks
  if config.Acks != "" {
    switch config.Acks {
    case "0":
      producerConfig.Producer.RequiredAcks = sarama.NoResponse
    case "1":
      producerConfig.Producer.RequiredAcks = sarama.WaitForLocal
    case "all":
    case "-1":
      producerConfig.Producer.RequiredAcks = sarama.WaitForAll
    }
  }

  // 设置压缩
  if config.Compression != "" {
    switch config.Compression {
    case "gzip":
      producerConfig.Producer.Compression = sarama.CompressionGZIP
    case "snappy":
      producerConfig.Producer.Compression = sarama.CompressionSnappy
    case "lz4":
      producerConfig.Producer.Compression = sarama.CompressionLZ4
    case "zstd":
      producerConfig.Producer.Compression = sarama.CompressionZSTD
    }
  }

  // 创建同步生产者
  syncProducer, err := sarama.NewSyncProducerFromClient(c.client)
  if err != nil {
    return fmt.Errorf("failed to create producer: %w", err)
  }

  c.producer = syncProducer
  log.Printf("Kafka producer created")
  return nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(config *ProducerConfig) error {
  if c.producer == nil {
    // 如果没有生产者，先创建一个
    if err := c.CreateProducer(config); err != nil {
      return err
    }
  }

  ticker := time.NewTicker(time.Duration(config.Interval) * time.Millisecond)
  defer ticker.Stop()

  var wg sync.WaitGroup

  for i := 0; i < config.Repeat; i++ {
    if config.Interval > 0 && i > 0 {
      <-ticker.C
    }

    wg.Add(1)
    go func(idx int) {
      defer wg.Done()

      // 如果指定了消息长度，并且当前消息长度不足，则补全
      messageContent := config.Message
      if config.MessageLength > 0 && len(messageContent) < config.MessageLength {
        messageContent = messageContent + strings.Repeat(" ", config.MessageLength-len(messageContent))
      }

      // 准备消息
      message := &sarama.ProducerMessage{
        Topic: config.Topic,
        Value: sarama.StringEncoder(messageContent),
      }

      // 设置消息键
      if config.Key != "" {
        message.Key = sarama.StringEncoder(config.Key)
      }

      // 设置消息headers（如果有的话）
      headers := common.ParsePropertyArray(config.Headers)
      if headers != nil {
        for _, kv := range headers {
          message.Headers = append(message.Headers, sarama.RecordHeader{
            Key:   []byte(kv[0]),
            Value: []byte(kv[1]),
          })
        }
      }

      // 发送消息
      partition, offset, err := c.producer.SendMessage(message)
      if err != nil {
        log.Printf("Failed to send message %d: %v", idx, err)
      } else if config.PrintLog {
        log.Printf("Message %d sent successfully, topic: %s, partition: %d, offset: %d, size: %d bytes",
          idx, config.Topic, partition, offset, len(messageContent))
      }
    }(i)
  }

  wg.Wait()
  c.producedCount += int64(config.Repeat)
  log.Printf("Total messages sent: %d", config.Repeat)
  return nil
}

// CreateConsumer 创建消费者
func (c *Client) CreateConsumer(consumerConfig *ConsumerConfig) error {

  // 关闭已存在的消费者
  if c.consumer != nil {
    _ = c.consumer.Close()
  }
  if c.consumerGroup != nil {
    _ = c.consumerGroup.Close()
  }

  // 创建消费者
  config := sarama.NewConfig()
  config.Consumer.Offsets.Initial = sarama.OffsetOldest

  consumer, err := sarama.NewConsumerFromClient(c.client)
  if err != nil {
    return fmt.Errorf("failed to create consumer: %w", err)
  }

  // 创建 ConsumerGroup
  consumerGroup, err := sarama.NewConsumerGroupFromClient(consumerConfig.GroupID, c.client)
  if err != nil {
    panic(err)
  }

  c.consumer = consumer
  c.consumerGroup = consumerGroup
  log.Printf("Kafka consumer created")
  return nil
}

func (h *kafkaGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
  return nil
}

func (h *kafkaGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
  return nil
}

func (h *kafkaGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
  for msg := range claim.Messages() {
    // 处理消息
    if h.config.PrintLog {
      log.Printf("Received message: topic=%s, partition=%d, offset=%d, key=%s, value=%s",
        msg.Topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))
    }

    // 标记消息已处理
    session.MarkMessage(msg, "")

    // 增加消息计数
    h.mu.Lock()
    h.messageCount++
    currentCount := h.messageCount
    h.mu.Unlock()

    // AI Modified: 更新client的消费计数
    h.client.consumedCount++

    // 检查是否已经达到最大消息数
    if h.config.MaxMessages > 0 && currentCount >= h.config.MaxMessages {
      return nil
    }
  }
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

  // 定义消费者组处理器
  handler := &kafkaGroupHandler{
    config:       config,
    messageCount: 0,
    client:       c, // AI Modified: 传递client指针
  }

  // 设置上下文
  ctx, cancel := context.WithCancel(c.ctx)
  if config.Timeout > 0 {
    var cancelTimeout context.CancelFunc
    ctx, cancelTimeout = context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
    defer cancelTimeout()
  }
  defer cancel()

  // 启动消费循环
  for {
    if err := c.consumerGroup.Consume(ctx, []string{config.Topic}, handler); err != nil {
      if !errors.Is(err, sarama.ErrClosedConsumerGroup) && !errors.Is(err, context.Canceled) {
        return fmt.Errorf("error from consumer: %w", err)
      }
      break
    }

    // 检查是否已经达到最大消息数
    if config.MaxMessages > 0 && handler.messageCount >= config.MaxMessages {
      break
    }

    // 检查上下文是否已取消
    if ctx.Err() != nil {
      break
    }
  }

  log.Printf("Close consumer group '%s' from topic '%s'", config.GroupID, config.Topic)
  return nil
}

// getOffset 根据配置获取偏移量
func (config *ConsumerConfig) getOffset() int64 {
  if config.OffsetType == "earliest" {
    return sarama.OffsetOldest
  } else if config.OffsetType == "latest" {
    return sarama.OffsetNewest
  } else if config.Offset > 0 {
    return config.Offset
  }
  return sarama.OffsetNewest
}

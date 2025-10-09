package rocketmq

import (
  "context"
  "fmt"
  "log"
  "os"
  "os/signal"
  "sync"
  "syscall"
  "time"

  "github.com/apache/rocketmq-client-go/v2"
  "github.com/apache/rocketmq-client-go/v2/consumer"
  "github.com/apache/rocketmq-client-go/v2/primitive"
  "github.com/apache/rocketmq-client-go/v2/producer"
)

// Client 定义RocketMQ客户端

type Client struct {
  producer rocketmq.Producer
  consumer rocketmq.PushConsumer
  Config   *ServerConfig
  mu       sync.Mutex
  ctx      context.Context
}

// NewClient 创建新的RocketMQ客户端
func NewClient(config *ServerConfig) *Client {
  return &Client{
    Config: config,
    ctx:    context.Background(),
  }
}

// Connect 连接到RocketMQ服务器
func (c *Client) Connect() error {
  c.mu.Lock()
  defer c.mu.Unlock()

  // 这里不需要实际连接，因为RocketMQ客户端在创建时才会连接
  log.Printf("RocketMQ client configured for server %s:%d", c.Config.Server, c.Config.Port)
  return nil
}

// Close 关闭客户端连接
func (c *Client) Close() {
  c.mu.Lock()
  defer c.mu.Unlock()

  if c.producer != nil {
    _ = c.producer.Shutdown()
  }

  if c.consumer != nil {
    _ = c.consumer.Shutdown()
  }
}

// CreateProducer 创建生产者
func (c *Client) CreateProducer(groupName string) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  // 关闭已存在的生产者
  if c.producer != nil {
    _ = c.producer.Shutdown()
  }

  addr := fmt.Sprintf("%s:%d", c.Config.Server, c.Config.Port)

  p, err := rocketmq.NewProducer(
    producer.WithNameServer([]string{addr}),
    producer.WithGroupName(groupName),
  )

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

// CreateConsumer 创建消费者
func (c *Client) CreateConsumer(groupName string) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  // 关闭已存在的消费者
  if c.consumer != nil {
    _ = c.consumer.Shutdown()
  }

  addr := fmt.Sprintf("%s:%d", c.Config.Server, c.Config.Port)

  con, err := rocketmq.NewPushConsumer(
    consumer.WithNameServer([]string{addr}),
    consumer.WithGroupName(groupName),
  )

  if err != nil {
    return fmt.Errorf("failed to create consumer: %w", err)
  }

  c.consumer = con
  log.Printf("Consumer created with group: %s", groupName)
  return nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(config *ProducerConfig) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  if c.producer == nil {
    // 如果没有生产者，先创建一个
    if err := c.CreateProducer(config.GroupName); err != nil {
      return err
    }
  }

  ticker := time.NewTicker(time.Duration(config.Interval) * time.Millisecond)
  defer ticker.Stop()

  for i := 0; i < config.Repeat; i++ {
    if config.Interval > 0 && i > 0 {
      <-ticker.C
    }

    msg := primitive.NewMessage(config.Topic, []byte(config.Message))
    msg.WithTag(config.Tags)
    msg.WithKeys([]string{config.Keys})

    result, err := c.producer.SendSync(c.ctx, msg)
    if err != nil {
      return fmt.Errorf("failed to send message: %w", err)
    }

    log.Printf("Message sent successfully, msgID: %s, queueID: %d", result.MsgID, result.MessageQueue.QueueId)
  }

  log.Printf("Total messages sent: %d", config.Repeat)
  return nil
}

// ReceiveMessage 接收消息
func (c *Client) ReceiveMessage(config *ConsumerConfig) error {
  c.mu.Lock()
  defer c.mu.Unlock()

  if c.consumer == nil {
    // 如果没有消费者，先创建一个
    if err := c.CreateConsumer(config.GroupName); err != nil {
      return err
    }
  }

  // 订阅主题
  subExpression := "*"
  if config.Tags != "" {
    subExpression = config.Tags
  }

  if err := c.consumer.Subscribe(config.Topic, consumer.MessageSelector{Type: consumer.TAG, Expression: subExpression}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
    for _, msg := range msgs {
      rocketMsg := &Message{
        Topic:     msg.Topic,
        Tags:      msg.GetTags(),
        Keys:      msg.GetKeys(),
        Body:      string(msg.Body),
        MsgID:     msg.MsgId,
        QueueID:   int32(msg.Queue.QueueId),
        StoreTime: msg.StoreTimestamp,
      }

      if config.Handler != nil {
        config.Handler(rocketMsg)
      } else {
        log.Printf("Received message: Topic=%s, Tags=%s, MsgID=%s, Body=%s",
          msg.Topic, msg.GetTags(), msg.MsgId, string(msg.Body))
      }
    }
    return consumer.ConsumeSuccess, nil
  }); err != nil {
    return fmt.Errorf("failed to subscribe topic: %w", err)
  }

  // 启动消费者
  if err := c.consumer.Start(); err != nil {
    return fmt.Errorf("failed to start consumer: %w", err)
  }

  log.Printf("Started consuming messages from topic %s", config.Topic)

  // 设置超时
  if config.Timeout > 0 {
    timer := time.NewTimer(time.Duration(config.Timeout) * time.Second)
    <-timer.C
    log.Printf("Consumer timeout after %d seconds", config.Timeout)
  } else {
    // 如果没有设置超时，一直运行直到被中断
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Printf("Consumer interrupted")
  }

  return nil
}

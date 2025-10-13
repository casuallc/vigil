package rocketmq

import (
  "context"
  "github.com/apache/rocketmq-client-go/v2"
  "log"
  "sync"
)

// Client 定义RocketMQ客户端
type Client struct {
  producer            rocketmq.Producer
  consumer            rocketmq.PushConsumer
  transactionProducer rocketmq.TransactionProducer
  Config              *ServerConfig
  mu                  sync.Mutex
  ctx                 context.Context
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

  // 这里不需要实际连接，因为RocketMQ客户端在创建时才会连接
  log.Printf("RocketMQ client configured for server %s:%d", c.Config.Server, c.Config.Port)
  return nil
}

// Close 关闭客户端连接
func (c *Client) Close() {

  if c.producer != nil {
    _ = c.producer.Shutdown()
  }

  if c.consumer != nil {
    _ = c.consumer.Shutdown()
  }

  if c.transactionProducer != nil {
    _ = c.transactionProducer.Shutdown()
  }
}

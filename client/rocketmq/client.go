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
  "github.com/apache/rocketmq-client-go/v2"
  "log"
)

// Client 定义RocketMQ客户端
type Client struct {
  producer            rocketmq.Producer
  consumer            rocketmq.PushConsumer
  transactionProducer rocketmq.TransactionProducer
  Config              *ServerConfig
  ctx                 context.Context
  producedCount       int64 // AI Modified: 记录生产的消息总数
  consumedCount       int64 // AI Modified: 记录消费的消息总数
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

  // AI Modified: 打印消息计数
  log.Printf("RocketMQ Client Stats - Produced: %d, Consumed: %d", c.producedCount, c.consumedCount)
}

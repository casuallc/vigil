package rabbitmq

import (
  amqp "github.com/rabbitmq/amqp091-go"
)

type ServerConfig struct {
  Server   string
  Port     int
  Vhost    string
  User     string
  Password string
}

type ExchangeConfig struct {
  Name       string
  Type       string
  Durable    bool
  AutoDelete bool
}

type QueueConfig struct {
  Name       string
  Passive    bool
  Durable    bool
  AutoDelete bool
  Exclusive  bool
  Args       amqp.Table
}

type BindConfig struct {
  Queue      string
  Exchange   string
  RoutingKey string
  Arguments  amqp.Table
}

type PublishConfig struct {
  Print      bool
  Exchange   string
  RoutingKey string
  Interval   int        // 时间间隔；秒
  Message    string     // 消息内容
  Repeat     int        // 重复次数
  RateLimit  int        // 发送速率
  Headers    amqp.Table // 消息头
}

type ConsumeConfig struct {
  Queue    string
  Consumer string
  AutoAck  bool
  Timeout  int
  Handler  func(msg amqp.Delivery)
}

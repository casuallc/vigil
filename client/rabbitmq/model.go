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
  PrintLog   bool
  Exchange   string
  RoutingKey string
  Interval   int        // 时间间隔；毫秒
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

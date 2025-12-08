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

package mqtt

import (
  "time"
)

// ServerConfig 定义MQTT服务器配置
type ServerConfig struct {
  Server     string
  Port       int
  User       string
  Password   string
  ClientID   string
  CleanStart bool
  KeepAlive  int
  Timeout    int
}

// PublishConfig 定义发布消息配置

type PublishConfig struct {
  Topic    string
  QoS      int
  Message  string
  Repeat   int
  Interval int  // 时间间隔（毫秒）
  Retained bool // 是否保留消息
  PrintLog bool // 是否打印发送日志
}

// SubscribeConfig 定义订阅消息配置

type SubscribeConfig struct {
  Topic    string
  QoS      int
  Timeout  int                     // 超时时间（秒）
  Handler  func(msg *Message) bool // 处理函数，返回true表示处理成功，false表示处理失败
  PrintLog bool                    // 是否打印接收日志
}

// Message 定义消息结构

type Message struct {
  Topic      string
  QoS        int
  Retained   bool
  Payload    string
  MessageID  uint16
  ReceivedAt time.Time
}

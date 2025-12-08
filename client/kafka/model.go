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

package kafka

import (
  "sync"
  "time"
)

// ServerConfig 定义Kafka服务器配置
type ServerConfig struct {
  Servers       string
  Port          int
  User          string
  Password      string
  SASLMechanism string
  SASLProtocol  string
  Timeout       int
}

// ProducerConfig 定义生产者配置
type ProducerConfig struct {
  Topic         string
  Message       string
  Key           string
  Repeat        int
  Interval      int
  PrintLog      bool
  Acks          string
  MessageLength int
  Compression   string
  Headers       string // 添加Headers字段，格式为name=value,name2=value2
}

// ConsumerConfig 定义消费者配置
type ConsumerConfig struct {
  Topic       string
  GroupID     string
  Offset      int64
  OffsetType  string
  Timeout     int
  PrintLog    bool
  MaxMessages int
}

// Message 定义消息结构
type Message struct {
  Topic     string
  Key       string
  Value     string
  Partition int32
  Offset    int64
  Timestamp time.Time
  Headers   map[string]string
}

type kafkaGroupHandler struct {
  config       *ConsumerConfig
  messageCount int
  mu           sync.Mutex
  client       *Client // AI Modified: 添加指向Client的指针，用于更新消费计数
}

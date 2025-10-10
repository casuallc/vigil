package kafka

import (
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
  Topic          string
  GroupID        string
  Partition      int32
  Offset         int64
  OffsetType     string
  Timeout        int
  PrintLog       bool
  MaxMessages    int
  CommitInterval int
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

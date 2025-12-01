package pulsar

import (
  "time"
)

// ServerConfig 表示 Pulsar 服务器配置
type ServerConfig struct {
  URL     string     // 服务 URL，格式为 pulsar://host:port 或 pulsar+ssl://host:port
  Auth    AuthConfig // 认证配置
  Timeout int        // 连接超时时间（秒）
}

// AuthConfig 表示 Pulsar 认证配置
type AuthConfig struct {
  Type     string // 认证类型（如 "token", "tls", "none"）
  Token    string // Token 值
  Username string // 用户名
  Password string // 密码
  CertFile string // TLS 证书路径
  KeyFile  string // TLS 密钥路径
  CAFile   string // CA 证书路径
}

// ProducerConfig 表示 Pulsar 生产者配置
type ProducerConfig struct {
  Topic                   string     // 主题名称
  Message                 string     // 消息内容
  Key                     string     // 消息键
  SendTimeout             int        // 发送超时时间（毫秒）
  EnableBatching          bool       // 是否启用确认
  BatchingMaxPublishDelay int        // 批处理最大延迟（毫秒）
  BatchingMaxMessages     int        // 批处理最大消息数量
  MessageLength           int        // 消息长度（不够则自动补全）
  Repeat                  int        // 重复发送次数
  Interval                int        // 消息发送间隔（毫秒）
  PrintLog                bool       // 是否打印发送日志
  DelayTime               int        // 延迟消息的延迟时间（毫秒）
  DeliverTime             *time.Time // 定时消息的投递时间
  EnableCompression       bool       // 是否启用消息压缩
  Properties              string     // 消息属性（格式：key=val,key=val）
}

// ConsumerConfig 表示 Pulsar 消费者配置
type ConsumerConfig struct {
  Topic            string // 主题名称
  Subscription     string // 订阅名称
  SubscriptionType string // 订阅类型（如 "Exclusive", "Shared", "Failover"）
  ReceiveTimeout   int    // 接收超时时间（毫秒）
  MessageTimeout   int    // 消息处理超时时间（秒）
  InitialPosition  string // 初始位置（"Earliest" 或 "Latest"）
  AutoAck          bool   // 是否自动确认消息
}

// Message 表示 Pulsar 消息
type Message struct {
  Topic       string            // 主题名称
  Key         string            // 消息键
  Payload     []byte            // 消息负载
  MessageID   string            // 消息 ID
  PublishTime time.Time         // 发布时间
  EventTime   time.Time         // 事件时间
  Properties  map[string]string // 属性
}

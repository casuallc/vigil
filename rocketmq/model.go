package rocketmq

import (
  "github.com/apache/rocketmq-client-go/v2/primitive"
  "time"
)

// SendType 发送方式枚举
type SendType int

const (
  SyncSend  SendType = iota // 同步发送
  AsyncSend                 // 异步发送
)

// ServerConfig 定义RocketMQ服务器配置
type ServerConfig struct {
  Server    string
  Port      int
  User      string
  Password  string
  Namespace string // 命名空间
  AccessKey string // AK
  SecretKey string // SK
}

// ProducerConfig 定义生产者配置
type ProducerConfig struct {
  GroupName       string
  Topic           string
  Tags            string
  Keys            string
  Message         string
  MessageLength   int      // 消息长度（不够则自动补全）
  Repeat          int      // 重复次数
  Interval        int      // 时间间隔（毫秒）
  SendType        SendType // 发送方式（同步或异步）
  BatchSize       int      // 批量发送大小
  DelayLevel      int      // 延迟消息级别
  ShardingKey     string   // 顺序消息分片键
  UseMessageTrace bool     // 是否使用消息轨迹
  PrintLog        bool     // 是否打印发送日志

  // 定时消息参数
  DeliverTime *time.Time // 定时发送时间

  // 异步发送回调
  AsyncCallback func(result *primitive.SendResult, err error)

  // 事务消息参数
  CheckTimes int // 事务回查次数
}

// ConsumerConfig 定义消费者配置
type ConsumerConfig struct {
  GroupName        string
  Topic            string
  Tags             string
  Timeout          int                     // 超时时间（秒）
  Handler          func(msg *Message) bool // 处理函数，返回true表示处理成功，false表示处理失败
  StartConsumePos  string                  // 开始消费位置：FIRST、LAST、TIMESTAMP
  ConsumeTimestamp string                  // 消费时间戳，格式：20060102150405
  ConsumeType      string                  // 接收方式：SYNC、ASYNC
  PrintLog         bool                    // 是否打印接收日志
  RetryCount       int                     // 消息重试次数
  UseMessageTrace  bool                    // 是否使用消息轨迹

  // 事务消息参数
  CheckTimes int // 事务回查次数
}

// Message 定义消息结构
type Message struct {
  Topic     string
  Tags      string
  Keys      string
  Body      string
  MsgID     string
  QueueID   int32
  StoreTime int64

  // 事务消息相关
  TransactionID string
}

// TransactionListener 事务监听器接口
type TransactionListener interface {
  // ExecuteLocalTransaction 执行本地事务
  ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState
  // CheckLocalTransaction 检查本地事务状态
  CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState
}

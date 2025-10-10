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
	Topic      string
	QoS        int
	Message    string
	Repeat     int
	Interval   int        // 时间间隔（毫秒）
	Retained   bool       // 是否保留消息
	PrintLog   bool       // 是否打印发送日志
}

// SubscribeConfig 定义订阅消息配置

type SubscribeConfig struct {
	Topic      string
	QoS        int
	Timeout    int        // 超时时间（秒）
	Handler    func(msg *Message) bool // 处理函数，返回true表示处理成功，false表示处理失败
	PrintLog   bool       // 是否打印接收日志
}

// Message 定义消息结构

type Message struct {
	Topic     string
	QoS       int
	Retained  bool
	Payload   string
	MessageID uint16
	ReceivedAt time.Time
}
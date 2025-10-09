package rocketmq

// ServerConfig 定义RocketMQ服务器配置
type ServerConfig struct {
	Server   string
	Port     int
	User     string
	Password string
}

// ProducerConfig 定义生产者配置
type ProducerConfig struct {
	GroupName string
	Topic     string
	Tags      string
	Keys      string
	Message   string
	Repeat    int        // 重复次数
	Interval  int        // 时间间隔（毫秒）
}

// ConsumerConfig 定义消费者配置
type ConsumerConfig struct {
	GroupName string
	Topic     string
	Tags      string
	Timeout   int        // 超时时间（秒）
	Handler   func(msg *Message)
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
}
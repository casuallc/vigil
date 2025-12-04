package mqtt

import (
  "context"
  "fmt"
  "log"
  "os"
  "os/signal"
  "sync"
  "syscall"
  "time"

  mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client 定义MQTT客户端
type Client struct {
  client        mqtt.Client
  Config        *ServerConfig
  mu            sync.Mutex
  ctx           context.Context
  cancel        context.CancelFunc
  producedCount int64 // AI Modified: 记录生产的消息总数
  consumedCount int64 // AI Modified: 记录消费的消息总数
}

// NewClient 创建新的MQTT客户端
func NewClient(config *ServerConfig) *Client {
  ctx, cancel := context.WithCancel(context.Background())
  return &Client{
    Config: config,
    ctx:    ctx,
    cancel: cancel,
  }
}

// Connect 连接到MQTT服务器
func (c *Client) Connect() error {

  // 创建客户端选项
  opts := mqtt.NewClientOptions()

  // 设置服务器地址
  broker := fmt.Sprintf("tcp://%s:%d", c.Config.Server, c.Config.Port)
  opts.AddBroker(broker)

  // 设置客户端ID
  if c.Config.ClientID != "" {
    opts.SetClientID(c.Config.ClientID)
  } else {
    // 生成默认的客户端ID
    opts.SetClientID(fmt.Sprintf("mqtt-client-%d", time.Now().UnixNano()))
  }

  // 设置认证信息
  if c.Config.User != "" && c.Config.Password != "" {
    opts.SetUsername(c.Config.User)
    opts.SetPassword(c.Config.Password)
  }

  // 设置CleanStart
  opts.SetCleanSession(c.Config.CleanStart)

  // 设置KeepAlive
  if c.Config.KeepAlive > 0 {
    opts.SetKeepAlive(time.Duration(c.Config.KeepAlive) * time.Second)
  } else {
    opts.SetKeepAlive(60 * time.Second)
  }

  // 设置超时时间
  if c.Config.Timeout > 0 {
    opts.SetConnectTimeout(time.Duration(c.Config.Timeout) * time.Second)
  } else {
    opts.SetConnectTimeout(30 * time.Second)
  }

  // 设置连接回调
  opts.SetOnConnectHandler(func(client mqtt.Client) {
    log.Printf("Connected to MQTT server %s:%d", c.Config.Server, c.Config.Port)
  })

  // 设置连接丢失回调
  opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
    log.Printf("Connection to MQTT server lost: %v", err)
  })

  // 创建客户端
  c.client = mqtt.NewClient(opts)

  // 连接服务器
  if token := c.client.Connect(); token.Wait() && token.Error() != nil {
    return fmt.Errorf("failed to connect to MQTT server: %w", token.Error())
  }

  return nil
}

// Close 关闭客户端连接
func (c *Client) Close() {

  c.cancel()
  if c.client != nil && c.client.IsConnected() {
    c.client.Disconnect(250)
  }
  // AI Modified: 打印消息计数
  log.Printf("MQTT Client Stats - Produced: %d, Consumed: %d", c.producedCount, c.consumedCount)
}

// PublishMessage 发布消息
func (c *Client) PublishMessage(config *PublishConfig) error {

  if c.client == nil || !c.client.IsConnected() {
    return fmt.Errorf("MQTT client is not connected")
  }

  // 设置默认值
  if config.Repeat <= 0 {
    config.Repeat = 1
  }

  if config.Interval <= 0 {
    config.Interval = 1000
  }

  if config.QoS < 0 || config.QoS > 2 {
    config.QoS = 0
  }

  ticker := time.NewTicker(time.Duration(config.Interval) * time.Millisecond)
  defer ticker.Stop()

  var wg sync.WaitGroup

  for i := 0; i < config.Repeat; i++ {
    if i > 0 {
      select {
      case <-ticker.C:
      case <-c.ctx.Done():
        return nil
      }
    }

    wg.Add(1)
    go func(idx int) {
      defer wg.Done()

      token := c.client.Publish(config.Topic, byte(config.QoS), config.Retained, config.Message)
      token.Wait()

      if token.Error() != nil {
        log.Printf("Failed to publish message %d: %v", idx, token.Error())
      } else if config.PrintLog {
        log.Printf("Message %d published successfully to topic '%s'", idx, config.Topic)
      }
    }(i)
  }

  wg.Wait()
  c.producedCount += int64(config.Repeat)
  log.Printf("Total messages published: %d", config.Repeat)
  return nil
}

// SubscribeMessage 订阅消息
func (c *Client) SubscribeMessage(config *SubscribeConfig) error {

  if c.client == nil || !c.client.IsConnected() {
    return fmt.Errorf("MQTT client is not connected")
  }

  // 设置默认值
  if config.QoS < 0 || config.QoS > 2 {
    config.QoS = 0
  }

  // 创建消息处理函数
  messageHandler := func(client mqtt.Client, msg mqtt.Message) {
    // AI Modified: 记录消费的消息总数
    c.consumedCount++
    
    mqttMsg := &Message{
      Topic:      msg.Topic(),
      QoS:        int(msg.Qos()),
      Retained:   msg.Retained(),
      Payload:    string(msg.Payload()),
      MessageID:  msg.MessageID(),
      ReceivedAt: time.Now(),
    }

    var success bool
    if config.Handler != nil {
      success = config.Handler(mqttMsg)
    } else {
      // 默认处理逻辑
      success = true
      if config.PrintLog {
        log.Printf("Received message: Topic=%s, QoS=%d, Retained=%v, Payload=%s",
          mqttMsg.Topic, mqttMsg.QoS, mqttMsg.Retained, mqttMsg.Payload)
      }
    }

    // 如果处理成功，自动确认消息
    if success {
      msg.Ack()
    }
  }

  // 订阅主题
  token := c.client.Subscribe(config.Topic, byte(config.QoS), messageHandler)
  token.Wait()

  if token.Error() != nil {
    return fmt.Errorf("failed to subscribe to topic '%s': %w", config.Topic, token.Error())
  }

  log.Printf("Subscribed to topic '%s' with QoS %d", config.Topic, config.QoS)

  // 设置超时
  if config.Timeout > 0 {
    timer := time.NewTimer(time.Duration(config.Timeout) * time.Second)
    select {
    case <-timer.C:
      log.Printf("Subscription timeout after %d seconds", config.Timeout)
    case <-c.ctx.Done():
      log.Printf("Subscription canceled")
    }
  } else {
    // 如果没有设置超时，一直运行直到被中断
    log.Printf("Subscribed and waiting for messages. Press Ctrl+C to stop...")

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    select {
    case <-sigChan:
      log.Printf("Subscription interrupted by signal")
    case <-c.ctx.Done():
      log.Printf("Subscription canceled")
    }
  }

  // 取消订阅
  token = c.client.Unsubscribe(config.Topic)
  token.Wait()

  log.Printf("Unsubscribed from topic '%s'", config.Topic)

  return nil
}

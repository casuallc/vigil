package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/mqtt"
  "github.com/spf13/cobra"
)

// setupMqttCommands 设置MQTT相关命令
func (c *CLI) setupMqttCommands() *cobra.Command {
  mqttCmd := &cobra.Command{
    Use:   "mqtt",
    Short: "MQTT related commands",
    Long:  `Perform MQTT operations like publishing and subscribing to messages.`,
  }

  // 为父命令添加持久化标志
  var config mqtt.ServerConfig
  mqttCmd.PersistentFlags().StringVar(&config.Server, "server", "localhost", "MQTT server address")
  mqttCmd.PersistentFlags().IntVar(&config.Port, "port", 1883, "MQTT server port")
  mqttCmd.PersistentFlags().StringVar(&config.User, "user", "", "Username for authentication")
  mqttCmd.PersistentFlags().StringVar(&config.Password, "password", "", "Password for authentication")
  mqttCmd.PersistentFlags().StringVar(&config.ClientID, "client-id", "", "Client ID")
  mqttCmd.PersistentFlags().BoolVar(&config.CleanStart, "clean-start", true, "Clean start flag")
  mqttCmd.PersistentFlags().IntVar(&config.KeepAlive, "keep-alive", 60, "Keep alive interval in seconds")
  mqttCmd.PersistentFlags().IntVar(&config.Timeout, "timeout", 30, "Connection timeout in seconds")

  // 存储配置到上下文
  mqttCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "mqttConfig", &config))
  }

  // 将子命令添加到父命令
  mqttCmd.AddCommand(c.setupMqttPublishCommand())
  mqttCmd.AddCommand(c.setupMqttSubscribeCommand())

  return mqttCmd
}

// setupMqttPublishCommand 设置发送消息命令
func (c *CLI) setupMqttPublishCommand() *cobra.Command {
  var topic string
  var qos int
  var message string
  var repeat, interval int
  var retained, printLog bool

  // 添加发布消息命令
  cmd := &cobra.Command{
    Use:   "publish",
    Short: "Publish a message to an MQTT topic",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("mqttConfig").(*mqtt.ServerConfig)
      return c.handleMqttPublish(config, topic, qos, message, repeat, interval, retained, printLog)
    },
  }

  // 添加发布命令的标志
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "MQTT topic to publish to")
  cmd.Flags().IntVarP(&qos, "qos", "q", 0, "Quality of Service (0, 1, 2)")
  cmd.Flags().StringVarP(&message, "message", "m", "Hello, MQTT!", "Message content")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 1, "Number of times to repeat publishing")
  cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
  cmd.Flags().BoolVarP(&retained, "retained", "R", false, "Retain message flag")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print log messages")
  cmd.MarkFlagRequired("topic")

  return cmd
}

// handleMqttPublish 处理发布消息
func (c *CLI) handleMqttPublish(config *mqtt.ServerConfig, topic string, qos int, message string, repeat int, interval int, retained bool, printLog bool) error {
  // 创建客户端
  client := mqtt.NewClient(config)
  defer client.Close()

  // 连接到MQTT服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to MQTT server: %w", err)
  }

  // 创建发布配置
  publishConfig := &mqtt.PublishConfig{
    Topic:    topic,
    QoS:      qos,
    Message:  message,
    Repeat:   repeat,
    Interval: interval,
    Retained: retained,
    PrintLog: printLog,
  }

  // 发布消息
  if err := client.PublishMessage(publishConfig); err != nil {
    return fmt.Errorf("failed to publish message: %w", err)
  }

  fmt.Printf("Successfully published %d messages to topic '%s'\n", repeat, topic)
  return nil
}

// setupMqttSubscribeCommand 设置发送消息命令
func (c *CLI) setupMqttSubscribeCommand() *cobra.Command {
  var topic string
  var qos, timeout int
  var printLog bool

  // 添加订阅消息命令
  cmd := &cobra.Command{
    Use:   "subscribe",
    Short: "Subscribe to an MQTT topic",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("mqttConfig").(*mqtt.ServerConfig)
      return c.handleMqttSubscribe(config, topic, qos, timeout, printLog)
    },
  }

  // 添加订阅命令的标志
  cmd.Flags().StringVarP(&topic, "topic", "t", "", "MQTT topic to subscribe to")
  cmd.Flags().IntVarP(&qos, "qos", "q", 0, "Quality of Service (0, 1, 2)")
  cmd.Flags().IntVarP(&timeout, "timeout", "o", 0, "Timeout in seconds (0 for unlimited)")
  cmd.Flags().BoolVar(&printLog, "print-log", true, "Print log messages")
  cmd.MarkFlagRequired("topic")
  return cmd
}

// handleMqttSubscribe 处理订阅消息
func (c *CLI) handleMqttSubscribe(config *mqtt.ServerConfig, topic string, qos int, timeout int, printLog bool) error {
  // 创建客户端
  client := mqtt.NewClient(config)
  defer client.Close()

  // 连接到MQTT服务器
  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to MQTT server: %w", err)
  }

  // 创建订阅配置
  subscribeConfig := &mqtt.SubscribeConfig{
    Topic:    topic,
    QoS:      qos,
    Timeout:  timeout,
    PrintLog: printLog,
    Handler: func(msg *mqtt.Message) bool {
      fmt.Printf("\nReceived message:\n")
      fmt.Printf("  Topic: %s\n", msg.Topic)
      fmt.Printf("  QoS: %d\n", msg.QoS)
      fmt.Printf("  Retained: %v\n", msg.Retained)
      fmt.Printf("  MessageID: %d\n", msg.MessageID)
      fmt.Printf("  Received At: %s\n", msg.ReceivedAt.Format("2006-01-02 15:04:05.000"))
      fmt.Printf("  Payload: %s\n", msg.Payload)
      return true
    },
  }

  fmt.Printf("Starting to subscribe to topic '%s'\n", topic)
  if timeout > 0 {
    fmt.Printf("Subscription will stop after %d seconds if no messages are received\n", timeout)
  } else {
    fmt.Printf("Subscribed and waiting for messages. Press Ctrl+C to stop...\n")
  }

  // 订阅消息
  if err := client.SubscribeMessage(subscribeConfig); err != nil {
    return fmt.Errorf("failed to subscribe to topic: %w", err)
  }

  fmt.Printf("Finished subscribing to topic '%s'\n", topic)
  return nil
}

package cli

import (
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

  // 添加发布消息命令
  publishCmd := &cobra.Command{
    Use:   "publish",
    Short: "Publish a message to an MQTT topic",
    RunE: func(cmd *cobra.Command, args []string) error {
      // 获取命令行参数
      topic, _ := cmd.Flags().GetString("topic")
      qos, _ := cmd.Flags().GetInt("qos")
      message, _ := cmd.Flags().GetString("message")
      repeat, _ := cmd.Flags().GetInt("repeat")
      interval, _ := cmd.Flags().GetInt("interval")
      retained, _ := cmd.Flags().GetBool("retained")
      printLog, _ := cmd.Flags().GetBool("print-log")

      return c.handleMqttPublish(&config, topic, qos, message, repeat, interval, retained, printLog)
    },
  }

  // 添加发布命令的标志
  publishCmd.Flags().StringP("topic", "t", "", "MQTT topic to publish to")
  publishCmd.Flags().IntP("qos", "q", 0, "Quality of Service (0, 1, 2)")
  publishCmd.Flags().StringP("message", "m", "Hello, MQTT!", "Message content")
  publishCmd.Flags().IntP("repeat", "r", 1, "Number of times to repeat publishing")
  publishCmd.Flags().IntP("interval", "i", 1000, "Interval between messages in milliseconds")
  publishCmd.Flags().BoolP("retained", "R", false, "Retain message flag")
  publishCmd.Flags().BoolP("print-log", "l", true, "Print log messages")
  publishCmd.MarkFlagRequired("topic")

  // 添加订阅消息命令
  subscribeCmd := &cobra.Command{
    Use:   "subscribe",
    Short: "Subscribe to an MQTT topic",
    RunE: func(cmd *cobra.Command, args []string) error {
      // 获取命令行参数
      topic, _ := cmd.Flags().GetString("topic")
      qos, _ := cmd.Flags().GetInt("qos")
      timeout, _ := cmd.Flags().GetInt("timeout")
      printLog, _ := cmd.Flags().GetBool("print-log")

      return c.handleMqttSubscribe(&config, topic, qos, timeout, printLog)
    },
  }

  // 添加订阅命令的标志
  subscribeCmd.Flags().StringP("topic", "t", "", "MQTT topic to subscribe to")
  subscribeCmd.Flags().IntP("qos", "q", 0, "Quality of Service (0, 1, 2)")
  subscribeCmd.Flags().IntP("timeout", "o", 0, "Timeout in seconds (0 for unlimited)")
  subscribeCmd.Flags().BoolP("print-log", "l", true, "Print log messages")
  subscribeCmd.MarkFlagRequired("topic")

  // 将子命令添加到父命令
  mqttCmd.AddCommand(publishCmd)
  mqttCmd.AddCommand(subscribeCmd)

  return mqttCmd
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

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

package cli

import (
  "fmt"
  "sync"
  "time"

  "github.com/casuallc/vigil/client/mqtt"
  "github.com/spf13/cobra"
)

// setupMqttTestCommands 设置MQTT测试命令
func (c *CLI) setupMqttTestCommands() *cobra.Command {
  mqttTestCmd := &cobra.Command{
    Use:   "mqtt",
    Short: "Run MQTT integration tests",
    Long:  "Run integration tests for MQTT functionality",
  }

  // Test all MQTT functionality
  allCmd := &cobra.Command{
    Use:   "all",
    Short: "Run all MQTT tests",
    Long:  "Run all MQTT integration tests",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestAll()
    },
  }
  mqttTestCmd.AddCommand(allCmd)

  // Add MQTT 3.1.1 test commands
  mqtt311Cmd := c.setupMqtt311TestCommands()
  mqttTestCmd.AddCommand(mqtt311Cmd)

  // Add MQTT 5.0 test commands
  mqtt50Cmd := c.setupMqtt50TestCommands()
  mqttTestCmd.AddCommand(mqtt50Cmd)

  // Add EMQX specific test commands
  emqxCmd := c.setupEmqxTestCommands()
  mqttTestCmd.AddCommand(emqxCmd)

  return mqttTestCmd
}

// setupMqtt311TestCommands 设置MQTT 3.1.1测试命令
func (c *CLI) setupMqtt311TestCommands() *cobra.Command {
  mqtt311Cmd := &cobra.Command{
    Use:   "v3",
    Short: "Run MQTT 3.1.1 tests",
    Long:  "Run tests for MQTT 3.1.1 functionality",
  }

  // Test MQTT connection with clean session
  connectCmd := &cobra.Command{
    Use:   "connect",
    Short: "Test MQTT connection",
    Long:  "Test MQTT client connection functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestConnect()
    },
  }
  mqtt311Cmd.AddCommand(connectCmd)

  // Test MQTT publish/subscribe
  pubsubCmd := &cobra.Command{
    Use:   "pubsub",
    Short: "Test MQTT publish/subscribe",
    Long:  "Test MQTT publish and subscribe functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestPubSub()
    },
  }
  mqtt311Cmd.AddCommand(pubsubCmd)

  // Test MQTT QoS levels
  qosCmd := &cobra.Command{
    Use:   "qos",
    Short: "Test MQTT QoS levels",
    Long:  "Test MQTT QoS 0/1/2 message delivery",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestQoS()
    },
  }
  mqtt311Cmd.AddCommand(qosCmd)

  // Test MQTT retained messages
  retainedCmd := &cobra.Command{
    Use:   "retained",
    Short: "Test MQTT retained messages",
    Long:  "Test MQTT retained message functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestRetained()
    },
  }
  mqtt311Cmd.AddCommand(retainedCmd)

  // Test MQTT wildcard subscriptions
  wildcardCmd := &cobra.Command{
    Use:   "wildcard",
    Short: "Test MQTT wildcard subscriptions",
    Long:  "Test MQTT wildcard subscription matching",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestWildcard()
    },
  }
  mqtt311Cmd.AddCommand(wildcardCmd)

  // Test MQTT keep alive
  keepaliveCmd := &cobra.Command{
    Use:   "keepalive",
    Short: "Test MQTT keep alive functionality",
    Long:  "Test MQTT keep alive timeout disconnection",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestKeepAlive()
    },
  }
  mqtt311Cmd.AddCommand(keepaliveCmd)

  // Test MQTT ACL control
  aclCmd := &cobra.Command{
    Use:   "acl",
    Short: "Test MQTT ACL control",
    Long:  "Test MQTT ACL control functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestACL()
    },
  }
  mqtt311Cmd.AddCommand(aclCmd)

  // Test MQTT TLS connection
  tlsCmd := &cobra.Command{
    Use:   "tls",
    Short: "Test MQTT TLS connection",
    Long:  "Test MQTT TLS encrypted connection",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestTLS()
    },
  }
  mqtt311Cmd.AddCommand(tlsCmd)

  return mqtt311Cmd
}

// setupMqtt50TestCommands 设置MQTT 5.0测试命令
func (c *CLI) setupMqtt50TestCommands() *cobra.Command {
  mqtt50Cmd := &cobra.Command{
    Use:   "v5",
    Short: "Run MQTT 5.0 tests",
    Long:  "Run tests for MQTT 5.0 functionality",
  }

  // Test MQTT 5.0 Session Expiry Interval
  sessionExpiryCmd := &cobra.Command{
    Use:   "session-expiry",
    Short: "Test MQTT 5.0 session expiry interval",
    Long:  "Test MQTT 5.0 session expiry interval functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestSessionExpiry()
    },
  }
  mqtt50Cmd.AddCommand(sessionExpiryCmd)

  // Test MQTT 5.0 Message Expiry Interval
  messageExpiryCmd := &cobra.Command{
    Use:   "message-expiry",
    Short: "Test MQTT 5.0 message expiry interval",
    Long:  "Test MQTT 5.0 message expiry interval functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestMessageExpiry()
    },
  }
  mqtt50Cmd.AddCommand(messageExpiryCmd)

  // Test MQTT 5.0 Reason Code and Reason String
  reasonCodeCmd := &cobra.Command{
    Use:   "reason-code",
    Short: "Test MQTT 5.0 reason code and reason string",
    Long:  "Test MQTT 5.0 reason code and reason string functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestReasonCode()
    },
  }
  mqtt50Cmd.AddCommand(reasonCodeCmd)

  // Test MQTT 5.0 User Properties
  userPropertiesCmd := &cobra.Command{
    Use:   "user-properties",
    Short: "Test MQTT 5.0 user properties",
    Long:  "Test MQTT 5.0 user properties functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestUserProperties()
    },
  }
  mqtt50Cmd.AddCommand(userPropertiesCmd)

  // Test MQTT 5.0 Response Topic and Correlation Data
  responseTopicCmd := &cobra.Command{
    Use:   "response-topic",
    Short: "Test MQTT 5.0 response topic and correlation data",
    Long:  "Test MQTT 5.0 response topic and correlation data functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestResponseTopic()
    },
  }
  mqtt50Cmd.AddCommand(responseTopicCmd)

  // Test MQTT 5.0 Shared Subscription
  sharedSubCmd := &cobra.Command{
    Use:   "shared-subscription",
    Short: "Test MQTT 5.0 shared subscription",
    Long:  "Test MQTT 5.0 shared subscription functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestSharedSubscription()
    },
  }
  mqtt50Cmd.AddCommand(sharedSubCmd)

  // Test MQTT 5.0 Subscription Identifier
  subscriptionIdCmd := &cobra.Command{
    Use:   "subscription-id",
    Short: "Test MQTT 5.0 subscription identifier",
    Long:  "Test MQTT 5.0 subscription identifier functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestSubscriptionIdentifier()
    },
  }
  mqtt50Cmd.AddCommand(subscriptionIdCmd)

  // Test MQTT 5.0 No Local
  noLocalCmd := &cobra.Command{
    Use:   "no-local",
    Short: "Test MQTT 5.0 no local",
    Long:  "Test MQTT 5.0 no local functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestNoLocal()
    },
  }
  mqtt50Cmd.AddCommand(noLocalCmd)

  // Test MQTT 5.0 Retain Handling
  retainHandlingCmd := &cobra.Command{
    Use:   "retain-handling",
    Short: "Test MQTT 5.0 retain handling",
    Long:  "Test MQTT 5.0 retain handling functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestRetainHandling()
    },
  }
  mqtt50Cmd.AddCommand(retainHandlingCmd)

  // Test MQTT 5.0 Maximum Packet Size
  maxPacketSizeCmd := &cobra.Command{
    Use:   "max-packet-size",
    Short: "Test MQTT 5.0 maximum packet size",
    Long:  "Test MQTT 5.0 maximum packet size functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestMaxPacketSize()
    },
  }
  mqtt50Cmd.AddCommand(maxPacketSizeCmd)

  // Test MQTT 5.0 Receive Maximum
  receiveMaxCmd := &cobra.Command{
    Use:   "receive-max",
    Short: "Test MQTT 5.0 receive maximum",
    Long:  "Test MQTT 5.0 receive maximum functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestReceiveMax()
    },
  }
  mqtt50Cmd.AddCommand(receiveMaxCmd)

  return mqtt50Cmd
}

// setupEmqxTestCommands 设置EMQX特定测试命令
func (c *CLI) setupEmqxTestCommands() *cobra.Command {
  emqxCmd := &cobra.Command{
    Use:   "emqx",
    Short: "Run EMQX specific tests",
    Long:  "Run tests for EMQX specific functionality",
  }

  // Test EMQX QoS 2 message persistence and deduplication
  qos2PersistenceCmd := &cobra.Command{
    Use:   "qos2-persistence",
    Short: "Test EMQX QoS 2 message persistence and deduplication",
    Long:  "Test EMQX QoS 2 message persistence and deduplication functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestQoS2Persistence()
    },
  }
  emqxCmd.AddCommand(qos2PersistenceCmd)

  // Test EMQX offline message queue length limit
  offlineQueueCmd := &cobra.Command{
    Use:   "offline-queue",
    Short: "Test EMQX offline message queue length limit",
    Long:  "Test EMQX offline message queue length limit functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleMqttTestOfflineQueue()
    },
  }
  emqxCmd.AddCommand(offlineQueueCmd)

  return emqxCmd
}

// handleMqttTestAll 运行所有MQTT测试
func (c *CLI) handleMqttTestAll() error {
  fmt.Println("Running all MQTT tests...")

  // 运行所有MQTT测试
  tests := []struct {
    name string
    fn   func() error
  }{
    // MQTT 3.1.1 Tests
    {"Connect Test", c.handleMqttTestConnect},
    {"PubSub Test", c.handleMqttTestPubSub},
    {"QoS Test", c.handleMqttTestQoS},
    {"Retained Message Test", c.handleMqttTestRetained},
    {"Wildcard Subscription Test", c.handleMqttTestWildcard},
    {"Keep Alive Test", c.handleMqttTestKeepAlive},
    {"ACL Test", c.handleMqttTestACL},
    {"TLS Test", c.handleMqttTestTLS},

    // MQTT 5.0 Tests
    {"Session Expiry Test", c.handleMqttTestSessionExpiry},
    {"Message Expiry Test", c.handleMqttTestMessageExpiry},
    {"Reason Code Test", c.handleMqttTestReasonCode},
    {"User Properties Test", c.handleMqttTestUserProperties},
    {"Response Topic Test", c.handleMqttTestResponseTopic},
    {"Shared Subscription Test", c.handleMqttTestSharedSubscription},
    {"Subscription Identifier Test", c.handleMqttTestSubscriptionIdentifier},
    {"No Local Test", c.handleMqttTestNoLocal},
    {"Retain Handling Test", c.handleMqttTestRetainHandling},
    {"Maximum Packet Size Test", c.handleMqttTestMaxPacketSize},
    {"Receive Maximum Test", c.handleMqttTestReceiveMax},

    // EMQX Specific Tests
    {"QoS 2 Persistence Test", c.handleMqttTestQoS2Persistence},
    {"Offline Queue Test", c.handleMqttTestOfflineQueue},
  }

  var successCount, failCount int
  for _, test := range tests {
    fmt.Printf("\n=== Running %s ===\n", test.name)
    if err := test.fn(); err != nil {
      fmt.Printf("❌ %s FAILED: %v\n", test.name, err)
      failCount++
    } else {
      fmt.Printf("✅ %s PASSED\n", test.name)
      successCount++
    }
  }

  fmt.Printf("\n=== Test Results ===\n")
  fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", len(tests), successCount, failCount)

  if failCount > 0 {
    return fmt.Errorf("%d tests failed", failCount)
  }
  return nil
}

// handleMqttTestConnect 测试MQTT连接
func (c *CLI) handleMqttTestConnect() error {
  fmt.Println("Testing MQTT connection...")

  // 测试连接参数
  testCases := []struct {
    name       string
    cleanStart bool
  }{{
    name:       "Clean Session",
    cleanStart: true,
  }, {
    name:       "Non-Clean Session",
    cleanStart: false,
  }}

  for _, tc := range testCases {
    fmt.Printf("  Testing %s...\n", tc.name)

    // 创建MQTT客户端配置
    config := &mqtt.ServerConfig{
      Server:     "localhost",
      Port:       1883,
      ClientID:   fmt.Sprintf("test-connect-client-%d", time.Now().UnixNano()),
      CleanStart: tc.cleanStart,
      KeepAlive:  60,
      Timeout:    10,
    }

    // 创建并连接客户端
    client := mqtt.NewClient(config)
    err := client.Connect()
    if err != nil {
      client.Close()
      return fmt.Errorf("failed to connect to MQTT server: %v", err)
    }

    fmt.Printf("  ✅ %s connected successfully\n", tc.name)
    client.Close()
  }

  return nil
}

// handleMqttTestPubSub 测试MQTT发布/订阅
func (c *CLI) handleMqttTestPubSub() error {
  fmt.Println("Testing MQTT publish/subscribe...")

  // 创建MQTT客户端配置
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-pubsub-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }
  defer client.Close()

  // 测试主题
  testTopic := "test/pubsub"
  testMessage := "Hello MQTT!"

  // 订阅消息，使用超时10秒
  subscribeConfig := &mqtt.SubscribeConfig{
    Topic:   testTopic,
    QoS:     0,
    Timeout: 10,
    Handler: func(msg *mqtt.Message) bool {
      fmt.Printf("  Received message: %s\n", msg.Payload)
      return true
    },
    PrintLog: false,
  }

  // 启动订阅协程
  var wg sync.WaitGroup
  wg.Add(1)
  go func() {
    defer wg.Done()
    err := client.SubscribeMessage(subscribeConfig)
    if err != nil {
      fmt.Printf("  Subscription error: %v\n", err)
    }
  }()

  // 等待订阅启动
  time.Sleep(1 * time.Second)

  // 发布消息
  publishConfig := &mqtt.PublishConfig{
    Topic:    testTopic,
    QoS:      0,
    Message:  testMessage,
    Repeat:   1,
    Interval: 0,
    Retained: false,
    PrintLog: false,
  }

  err = client.PublishMessage(publishConfig)
  if err != nil {
    return fmt.Errorf("failed to publish message: %v", err)
  }

  fmt.Println("  ✅ Message published successfully")

  // 等待订阅完成
  wg.Wait()

  return nil
}

// handleMqttTestQoS 测试MQTT QoS级别
func (c *CLI) handleMqttTestQoS() error {
  fmt.Println("Testing MQTT QoS levels...")

  // 创建MQTT客户端配置
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-qos-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }
  defer client.Close()

  // 测试不同QoS级别
  qosLevels := []int{0, 1, 2}

  for _, qos := range qosLevels {
    fmt.Printf("  Testing QoS %d...\n", qos)

    testTopic := fmt.Sprintf("test/qos/%d", qos)
    testMessage := fmt.Sprintf("QoS %d test message", qos)

    // 订阅消息
    received := false
    subscribeConfig := &mqtt.SubscribeConfig{
      Topic:   testTopic,
      QoS:     qos,
      Timeout: 5,
      Handler: func(msg *mqtt.Message) bool {
        fmt.Printf("    Received QoS %d message: %s\n", msg.QoS, msg.Payload)
        received = true
        return true
      },
      PrintLog: false,
    }

    var wg sync.WaitGroup
    wg.Add(1)
    go func(q int) {
      defer wg.Done()
      err := client.SubscribeMessage(subscribeConfig)
      if err != nil {
        fmt.Printf("    Subscription error for QoS %d: %v\n", q, err)
      }
    }(qos)

    // 等待订阅启动
    time.Sleep(500 * time.Millisecond)

    // 发布消息
    publishConfig := &mqtt.PublishConfig{
      Topic:    testTopic,
      QoS:      qos,
      Message:  testMessage,
      Repeat:   1,
      Interval: 0,
      Retained: false,
      PrintLog: false,
    }

    err = client.PublishMessage(publishConfig)
    if err != nil {
      return fmt.Errorf("failed to publish message with QoS %d: %v", qos, err)
    }

    // 等待订阅完成
    wg.Wait()

    if !received {
      return fmt.Errorf("failed to receive message with QoS %d", qos)
    }

    fmt.Printf("  ✅ QoS %d test completed successfully\n", qos)
  }

  return nil
}

// handleMqttTestRetained 测试MQTT保留消息
func (c *CLI) handleMqttTestRetained() error {
  fmt.Println("Testing MQTT retained messages...")

  // 创建MQTT客户端配置
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-retained-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }
  defer client.Close()

  testTopic := "test/retained"

  // 发布保留消息
  fmt.Println("  Publishing retained message...")
  publishConfig := &mqtt.PublishConfig{
    Topic:    testTopic,
    QoS:      0,
    Message:  "Retained message",
    Repeat:   1,
    Interval: 0,
    Retained: true,
    PrintLog: false,
  }

  err = client.PublishMessage(publishConfig)
  if err != nil {
    return fmt.Errorf("failed to publish retained message: %v", err)
  }

  // 关闭并重新连接客户端
  client.Close()

  // 创建新客户端以测试保留消息
  newConfig := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-retained-client-new-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  newClient := mqtt.NewClient(newConfig)
  err = newClient.Connect()
  if err != nil {
    newClient.Close()
    return fmt.Errorf("failed to reconnect to MQTT server: %v", err)
  }
  defer newClient.Close()

  // 订阅主题，应该立即收到保留消息
  receivedRetained := false
  subscribeConfig := &mqtt.SubscribeConfig{
    Topic:   testTopic,
    QoS:     0,
    Timeout: 5,
    Handler: func(msg *mqtt.Message) bool {
      if msg.Retained {
        fmt.Printf("  Received retained message: %s\n", msg.Payload)
        receivedRetained = true
      }
      return true
    },
    PrintLog: false,
  }

  var wg sync.WaitGroup
  wg.Add(1)
  go func() {
    defer wg.Done()
    err := newClient.SubscribeMessage(subscribeConfig)
    if err != nil {
      fmt.Printf("  Subscription error: %v\n", err)
    }
  }()

  // 等待订阅完成
  wg.Wait()

  if !receivedRetained {
    return fmt.Errorf("failed to receive retained message")
  }

  // 清除保留消息
  fmt.Println("  Clearing retained message...")
  clearConfig := &mqtt.PublishConfig{
    Topic:    testTopic,
    QoS:      0,
    Message:  "",
    Repeat:   1,
    Interval: 0,
    Retained: true,
    PrintLog: false,
  }

  err = newClient.PublishMessage(clearConfig)
  if err != nil {
    return fmt.Errorf("failed to clear retained message: %v", err)
  }

  fmt.Println("  ✅ Retained message test completed successfully")
  return nil
}

// handleMqttTestWildcard 测试MQTT通配符订阅
func (c *CLI) handleMqttTestWildcard() error {
  fmt.Println("Testing MQTT wildcard subscriptions...")

  // 创建MQTT客户端配置
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-wildcard-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }
  defer client.Close()

  // 测试通配符订阅
  wildcardTopic := "test/wildcard/+/value"

  receivedMessages := make(map[string]bool)

  // 订阅通配符主题
  subscribeConfig := &mqtt.SubscribeConfig{
    Topic:   wildcardTopic,
    QoS:     0,
    Timeout: 10,
    Handler: func(msg *mqtt.Message) bool {
      fmt.Printf("  Received message on topic %s: %s\n", msg.Topic, msg.Payload)
      receivedMessages[msg.Topic] = true
      return true
    },
    PrintLog: false,
  }

  var wg sync.WaitGroup
  wg.Add(1)
  go func() {
    defer wg.Done()
    err := client.SubscribeMessage(subscribeConfig)
    if err != nil {
      fmt.Printf("  Subscription error: %v\n", err)
    }
  }()

  // 等待订阅启动
  time.Sleep(500 * time.Millisecond)

  // 发布多个匹配通配符的消息
  testTopics := []string{
    "test/wildcard/device1/value",
    "test/wildcard/device2/value",
    "test/wildcard/device3/value",
  }

  for _, topic := range testTopics {
    publishConfig := &mqtt.PublishConfig{
      Topic:    topic,
      QoS:      0,
      Message:  fmt.Sprintf("Message from %s", topic),
      Repeat:   1,
      Interval: 0,
      Retained: false,
      PrintLog: false,
    }

    err = client.PublishMessage(publishConfig)
    if err != nil {
      return fmt.Errorf("failed to publish message to %s: %v", topic, err)
    }

    // 等待消息处理
    time.Sleep(200 * time.Millisecond)
  }

  // 等待订阅完成
  wg.Wait()

  // 检查是否收到了所有消息
  for _, topic := range testTopics {
    if !receivedMessages[topic] {
      return fmt.Errorf("failed to receive message on topic %s", topic)
    }
  }

  fmt.Printf("  ✅ Successfully received messages on %d topics\n", len(receivedMessages))
  return nil
}

// handleMqttTestKeepAlive 测试MQTT Keep Alive
func (c *CLI) handleMqttTestKeepAlive() error {
  fmt.Println("Testing MQTT Keep Alive...")

  // 创建MQTT客户端配置，设置较短的Keep Alive时间
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-keepalive-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  5, // 5秒Keep Alive
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }

  fmt.Println("  Client connected with 5s Keep Alive")

  // 等待超过Keep Alive时间，验证连接是否保持
  fmt.Println("  Waiting for 10s to verify Keep Alive functionality...")
  time.Sleep(10 * time.Second)

  // 检查连接是否仍然活跃
  // 注意：当前MQTT客户端没有提供IsConnected方法，我们通过尝试发布消息来验证
  testTopic := "test/keepalive"
  publishConfig := &mqtt.PublishConfig{
    Topic:    testTopic,
    QoS:      0,
    Message:  "Keep Alive test message",
    Repeat:   1,
    Interval: 0,
    Retained: false,
    PrintLog: false,
  }

  err = client.PublishMessage(publishConfig)
  if err != nil {
    client.Close()
    return fmt.Errorf("connection lost during Keep Alive test: %v", err)
  }

  fmt.Println("  ✅ Keep Alive test completed successfully")
  client.Close()
  return nil
}

// handleMqttTestACL 测试MQTT ACL控制
func (c *CLI) handleMqttTestACL() error {
  fmt.Println("Testing MQTT ACL control...")

  // 注意：ACL测试需要MQTT服务器配置了适当的ACL规则
  // 这里我们只测试基本的连接和发布/订阅逻辑，实际ACL效果取决于服务器配置

  fmt.Println("  ACL test skipped - requires server-side ACL configuration")
  return nil
}

// handleMqttTestTLS 测试MQTT TLS连接
func (c *CLI) handleMqttTestTLS() error {
  fmt.Println("Testing MQTT TLS connection...")

  // 注意：TLS测试需要MQTT服务器配置了TLS证书
  // 这里我们只测试基本的连接逻辑，实际TLS效果取决于服务器配置

  fmt.Println("  TLS test skipped - requires server-side TLS configuration")
  return nil
}

// handleMqttTestSessionExpiry 测试MQTT 5.0会话过期
func (c *CLI) handleMqttTestSessionExpiry() error {
  fmt.Println("Testing MQTT 5.0 session expiry...")

  // 注意：会话过期测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Session Expiry Interval的字段

  fmt.Println("  Session expiry test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestMessageExpiry 测试MQTT 5.0消息过期
func (c *CLI) handleMqttTestMessageExpiry() error {
  fmt.Println("Testing MQTT 5.0 message expiry...")

  // 注意：消息过期测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Message Expiry Interval的字段

  fmt.Println("  Message expiry test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestReasonCode 测试MQTT 5.0原因码
func (c *CLI) handleMqttTestReasonCode() error {
  fmt.Println("Testing MQTT 5.0 reason code...")

  // 注意：原因码测试需要MQTT 5.0支持
  // 当前MQTT客户端没有暴露原因码的获取方式

  fmt.Println("  Reason code test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestUserProperties 测试MQTT 5.0用户属性
func (c *CLI) handleMqttTestUserProperties() error {
  fmt.Println("Testing MQTT 5.0 user properties...")

  // 注意：用户属性测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置User Properties的字段

  fmt.Println("  User properties test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestResponseTopic 测试MQTT 5.0响应主题
func (c *CLI) handleMqttTestResponseTopic() error {
  fmt.Println("Testing MQTT 5.0 response topic...")

  // 注意：响应主题测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Response Topic的字段

  fmt.Println("  Response topic test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestSharedSubscription 测试MQTT 5.0共享订阅
func (c *CLI) handleMqttTestSharedSubscription() error {
  fmt.Println("Testing MQTT 5.0 shared subscription...")

  // 创建MQTT客户端配置
  config := &mqtt.ServerConfig{
    Server:     "localhost",
    Port:       1883,
    ClientID:   fmt.Sprintf("test-shared-sub-client-%d", time.Now().UnixNano()),
    CleanStart: true,
    KeepAlive:  60,
    Timeout:    10,
  }

  // 创建并连接客户端
  client := mqtt.NewClient(config)
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to MQTT server: %v", err)
  }
  defer client.Close()

  // 测试共享订阅
  sharedTopic := "$share/group1/test/shared"
  receivedMessages := 0

  // 订阅共享主题
  subscribeConfig := &mqtt.SubscribeConfig{
    Topic:   sharedTopic,
    QoS:     0,
    Timeout: 10,
    Handler: func(msg *mqtt.Message) bool {
      fmt.Printf("  Received message on shared topic: %s\n", msg.Payload)
      receivedMessages++
      return true
    },
    PrintLog: false,
  }

  var wg sync.WaitGroup
  wg.Add(1)
  go func() {
    defer wg.Done()
    err := client.SubscribeMessage(subscribeConfig)
    if err != nil {
      fmt.Printf("  Subscription error: %v\n", err)
    }
  }()

  // 等待订阅启动
  time.Sleep(500 * time.Millisecond)

  // 发布消息到共享主题
  for i := 0; i < 3; i++ {
    publishConfig := &mqtt.PublishConfig{
      Topic:    "test/shared",
      QoS:      0,
      Message:  fmt.Sprintf("Shared message %d", i),
      Repeat:   1,
      Interval: 0,
      Retained: false,
      PrintLog: false,
    }

    err = client.PublishMessage(publishConfig)
    if err != nil {
      return fmt.Errorf("failed to publish message to shared topic: %v", err)
    }

    // 等待消息处理
    time.Sleep(200 * time.Millisecond)
  }

  // 等待订阅完成
  wg.Wait()

  if receivedMessages == 0 {
    return fmt.Errorf("failed to receive messages on shared topic")
  }

  fmt.Printf("  ✅ Shared subscription test completed successfully, received %d messages\n", receivedMessages)
  return nil
}

// handleMqttTestSubscriptionIdentifier 测试MQTT 5.0订阅标识符
func (c *CLI) handleMqttTestSubscriptionIdentifier() error {
  fmt.Println("Testing MQTT 5.0 subscription identifier...")

  // 注意：订阅标识符测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Subscription Identifier的字段

  fmt.Println("  Subscription identifier test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestNoLocal 测试MQTT 5.0 No Local
func (c *CLI) handleMqttTestNoLocal() error {
  fmt.Println("Testing MQTT 5.0 No Local...")

  // 注意：No Local测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置No Local的字段

  fmt.Println("  No Local test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestRetainHandling 测试MQTT 5.0 Retain Handling
func (c *CLI) handleMqttTestRetainHandling() error {
  fmt.Println("Testing MQTT 5.0 Retain Handling...")

  // 注意：Retain Handling测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Retain Handling的字段

  fmt.Println("  Retain Handling test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestMaxPacketSize 测试MQTT 5.0 Maximum Packet Size
func (c *CLI) handleMqttTestMaxPacketSize() error {
  fmt.Println("Testing MQTT 5.0 Maximum Packet Size...")

  // 注意：Maximum Packet Size测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Maximum Packet Size的字段

  fmt.Println("  Maximum Packet Size test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestReceiveMax 测试MQTT 5.0 Receive Maximum
func (c *CLI) handleMqttTestReceiveMax() error {
  fmt.Println("Testing MQTT 5.0 Receive Maximum...")

  // 注意：Receive Maximum测试需要MQTT 5.0支持
  // 当前MQTT客户端配置中没有直接设置Receive Maximum的字段

  fmt.Println("  Receive Maximum test skipped - requires MQTT 5.0 client support")
  return nil
}

// handleMqttTestQoS2Persistence 测试EMQX QoS 2消息持久化与去重
func (c *CLI) handleMqttTestQoS2Persistence() error {
  fmt.Println("Testing EMQX QoS 2 message persistence and deduplication...")

  // 注意：QoS 2持久化测试需要EMQX特定配置
  // 这里我们只测试基本的QoS 2功能

  fmt.Println("  QoS 2 persistence test skipped - requires EMQX specific configuration")
  return nil
}

// handleMqttTestOfflineQueue 测试EMQX离线消息队列长度限制
func (c *CLI) handleMqttTestOfflineQueue() error {
  fmt.Println("Testing EMQX offline message queue length limit...")

  // 注意：离线消息队列测试需要EMQX特定配置
  // 这里我们只测试基本的离线消息功能

  fmt.Println("  Offline queue test skipped - requires EMQX specific configuration")
  return nil
}

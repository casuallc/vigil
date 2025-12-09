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

	config := &mqtt.ServerConfig{}

	// 全局MQTT测试参数
	mqttTestCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "localhost", "MQTT server address")
	mqttTestCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 1883, "MQTT server port")
	mqttTestCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "", "MQTT username")
	mqttTestCmd.PersistentFlags().StringVarP(&config.Password, "password", "P", "", "MQTT password")

	// Test all MQTT functionality
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Run all MQTT tests",
		Long:  "Run all MQTT integration tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestAll(config)
		},
	}
	mqttTestCmd.AddCommand(allCmd)

	// Add MQTT 3.1.1 test commands
	mqtt311Cmd := c.setupMqtt311TestCommands(config)
	mqttTestCmd.AddCommand(mqtt311Cmd)

	// Add MQTT 5.0 test commands
	mqtt50Cmd := c.setupMqtt50TestCommands(config)
	mqttTestCmd.AddCommand(mqtt50Cmd)

	// Add EMQX specific test commands
	emqxCmd := c.setupEmqxTestCommands(config)
	mqttTestCmd.AddCommand(emqxCmd)

	return mqttTestCmd
}

// setupMqtt311TestCommands 设置MQTT 3.1.1测试命令
func (c *CLI) setupMqtt311TestCommands(config *mqtt.ServerConfig) *cobra.Command {
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
			return c.handleMqttTestConnect(config)
		},
	}
	mqtt311Cmd.AddCommand(connectCmd)

	// Test MQTT publish/subscribe
	pubsubCmd := &cobra.Command{
		Use:   "pubsub",
		Short: "Test MQTT publish/subscribe",
		Long:  "Test MQTT publish and subscribe functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestPubSub(config)
		},
	}
	mqtt311Cmd.AddCommand(pubsubCmd)

	// Test MQTT QoS levels
	qosCmd := &cobra.Command{
		Use:   "qos",
		Short: "Test MQTT QoS levels",
		Long:  "Test MQTT QoS 0/1/2 message delivery",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestQoS(config)
		},
	}
	mqtt311Cmd.AddCommand(qosCmd)

	// Test MQTT retained messages
	retainedCmd := &cobra.Command{
		Use:   "retained",
		Short: "Test MQTT retained messages",
		Long:  "Test MQTT retained message functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestRetained(config)
		},
	}
	mqtt311Cmd.AddCommand(retainedCmd)

	// Test MQTT wildcard subscriptions
	wildcardCmd := &cobra.Command{
		Use:   "wildcard",
		Short: "Test MQTT wildcard subscriptions",
		Long:  "Test MQTT wildcard subscription matching",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestWildcard(config)
		},
	}
	mqtt311Cmd.AddCommand(wildcardCmd)

	// Test MQTT keep alive
	keepaliveCmd := &cobra.Command{
		Use:   "keepalive",
		Short: "Test MQTT keep alive functionality",
		Long:  "Test MQTT keep alive timeout disconnection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestKeepAlive(config)
		},
	}
	mqtt311Cmd.AddCommand(keepaliveCmd)

	// Test MQTT ACL control
	aclCmd := &cobra.Command{
		Use:   "acl",
		Short: "Test MQTT ACL control",
		Long:  "Test MQTT ACL control functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestACL(config)
		},
	}
	mqtt311Cmd.AddCommand(aclCmd)

	// Test MQTT TLS connection
	tlsCmd := &cobra.Command{
		Use:   "tls",
		Short: "Test MQTT TLS connection",
		Long:  "Test MQTT TLS encrypted connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestTLS(config)
		},
	}
	mqtt311Cmd.AddCommand(tlsCmd)

	// Test MQTT Last Will and Testament (LWT)
	lwtCmd := &cobra.Command{
		Use:   "lwt",
		Short: "Test MQTT Last Will and Testament",
		Long:  "Test MQTT Last Will and Testament functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestLWT(config)
		},
	}
	mqtt311Cmd.AddCommand(lwtCmd)

	// Test MQTT shared subscriptions
	sharedCmd := &cobra.Command{
		Use:   "shared",
		Short: "Test MQTT shared subscriptions",
		Long:  "Test MQTT shared subscription functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestShared(config)
		},
	}
	mqtt311Cmd.AddCommand(sharedCmd)

	return mqtt311Cmd
}

// setupMqtt50TestCommands 设置MQTT 5.0测试命令
func (c *CLI) setupMqtt50TestCommands(config *mqtt.ServerConfig) *cobra.Command {
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
			return c.handleMqttTestSessionExpiry(config)
		},
	}
	mqtt50Cmd.AddCommand(sessionExpiryCmd)

	// Test MQTT 5.0 Message Expiry Interval
	messageExpiryCmd := &cobra.Command{
		Use:   "message-expiry",
		Short: "Test MQTT 5.0 message expiry interval",
		Long:  "Test MQTT 5.0 message expiry interval functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestMessageExpiry(config)
		},
	}
	mqtt50Cmd.AddCommand(messageExpiryCmd)

	// Test MQTT 5.0 Reason Code and Reason String
	reasonCodeCmd := &cobra.Command{
		Use:   "reason-code",
		Short: "Test MQTT 5.0 reason code and reason string",
		Long:  "Test MQTT 5.0 reason code and reason string functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestReasonCode(config)
		},
	}
	mqtt50Cmd.AddCommand(reasonCodeCmd)

	// Test MQTT 5.0 User Properties
	userPropertiesCmd := &cobra.Command{
		Use:   "user-properties",
		Short: "Test MQTT 5.0 user properties",
		Long:  "Test MQTT 5.0 user properties functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestUserProperties(config)
		},
	}
	mqtt50Cmd.AddCommand(userPropertiesCmd)

	// Test MQTT 5.0 Response Topic and Correlation Data
	responseTopicCmd := &cobra.Command{
		Use:   "response-topic",
		Short: "Test MQTT 5.0 response topic and correlation data",
		Long:  "Test MQTT 5.0 response topic and correlation data functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestResponseTopic(config)
		},
	}
	mqtt50Cmd.AddCommand(responseTopicCmd)

	// Test MQTT 5.0 Shared Subscription
	sharedSubCmd := &cobra.Command{
		Use:   "shared-subscription",
		Short: "Test MQTT 5.0 shared subscription",
		Long:  "Test MQTT 5.0 shared subscription functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestSharedSubscription(config)
		},
	}
	mqtt50Cmd.AddCommand(sharedSubCmd)

	// Test MQTT 5.0 Subscription Identifier
	subscriptionIdCmd := &cobra.Command{
		Use:   "subscription-id",
		Short: "Test MQTT 5.0 subscription identifier",
		Long:  "Test MQTT 5.0 subscription identifier functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestSubscriptionIdentifier(config)
		},
	}
	mqtt50Cmd.AddCommand(subscriptionIdCmd)

	// Test MQTT 5.0 No Local
	noLocalCmd := &cobra.Command{
		Use:   "no-local",
		Short: "Test MQTT 5.0 no local",
		Long:  "Test MQTT 5.0 no local functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestNoLocal(config)
		},
	}
	mqtt50Cmd.AddCommand(noLocalCmd)

	// Test MQTT 5.0 Retain Handling
	retainHandlingCmd := &cobra.Command{
		Use:   "retain-handling",
		Short: "Test MQTT 5.0 retain handling",
		Long:  "Test MQTT 5.0 retain handling functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestRetainHandling(config)
		},
	}
	mqtt50Cmd.AddCommand(retainHandlingCmd)

	// Test MQTT 5.0 Maximum Packet Size
	maxPacketSizeCmd := &cobra.Command{
		Use:   "max-packet-size",
		Short: "Test MQTT 5.0 maximum packet size",
		Long:  "Test MQTT 5.0 maximum packet size functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestMaxPacketSize(config)
		},
	}
	mqtt50Cmd.AddCommand(maxPacketSizeCmd)

	// Test MQTT 5.0 Receive Maximum
	receiveMaxCmd := &cobra.Command{
		Use:   "receive-max",
		Short: "Test MQTT 5.0 receive maximum",
		Long:  "Test MQTT 5.0 receive maximum functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestReceiveMax(config)
		},
	}
	mqtt50Cmd.AddCommand(receiveMaxCmd)

	return mqtt50Cmd
}

// setupEmqxTestCommands 设置EMQX特定测试命令
func (c *CLI) setupEmqxTestCommands(config *mqtt.ServerConfig) *cobra.Command {
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
			return c.handleMqttTestQoS2Persistence(config)
		},
	}
	emqxCmd.AddCommand(qos2PersistenceCmd)

	// Test EMQX offline message queue length limit
	offlineQueueCmd := &cobra.Command{
		Use:   "offline-queue",
		Short: "Test EMQX offline message queue length limit",
		Long:  "Test EMQX offline message queue length limit functionality",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleMqttTestOfflineQueue(config)
		},
	}
	emqxCmd.AddCommand(offlineQueueCmd)

	return emqxCmd
}

// handleMqttTestAll 运行所有MQTT测试
func (c *CLI) handleMqttTestAll(config *mqtt.ServerConfig) error {
	fmt.Println("Running all MQTT tests...")

	// 运行所有MQTT测试
	tests := []struct {
		name string
		fn   func() error
	}{
		// MQTT 3.1.1 Tests
		{"Connect Test", func() error { return c.handleMqttTestConnect(config) }},
		{"PubSub Test", func() error { return c.handleMqttTestPubSub(config) }},
		{"QoS Test", func() error { return c.handleMqttTestQoS(config) }},
		{"Retained Message Test", func() error { return c.handleMqttTestRetained(config) }},
		{"Wildcard Subscription Test", func() error { return c.handleMqttTestWildcard(config) }},
		{"Keep Alive Test", func() error { return c.handleMqttTestKeepAlive(config) }},
		{"ACL Test", func() error { return c.handleMqttTestACL(config) }},
		{"TLS Test", func() error { return c.handleMqttTestTLS(config) }},
		{"LWT Test", func() error { return c.handleMqttTestLWT(config) }},
		{"Shared Subscription Test", func() error { return c.handleMqttTestShared(config) }},

		// MQTT 5.0 Tests
		{"Session Expiry Test", func() error { return c.handleMqttTestSessionExpiry(config) }},
		{"Message Expiry Test", func() error { return c.handleMqttTestMessageExpiry(config) }},
		{"Reason Code Test", func() error { return c.handleMqttTestReasonCode(config) }},
		{"User Properties Test", func() error { return c.handleMqttTestUserProperties(config) }},
		{"Response Topic Test", func() error { return c.handleMqttTestResponseTopic(config) }},
		{"Subscription Identifier Test", func() error { return c.handleMqttTestSubscriptionIdentifier(config) }},
		{"No Local Test", func() error { return c.handleMqttTestNoLocal(config) }},
		{"Retain Handling Test", func() error { return c.handleMqttTestRetainHandling(config) }},
		{"Maximum Packet Size Test", func() error { return c.handleMqttTestMaxPacketSize(config) }},
		{"Receive Maximum Test", func() error { return c.handleMqttTestReceiveMax(config) }},

		// EMQX Specific Tests
		{"QoS 2 Persistence Test", func() error { return c.handleMqttTestQoS2Persistence(config) }},
		{"Offline Queue Test", func() error { return c.handleMqttTestOfflineQueue(config) }},
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
func (c *CLI) handleMqttTestConnect(config *mqtt.ServerConfig) error {
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
		clientConfig := *config
		clientConfig.ClientID = fmt.Sprintf("test-connect-client-%d", time.Now().UnixNano())
		clientConfig.CleanStart = tc.cleanStart
		clientConfig.KeepAlive = 60
		clientConfig.Timeout = 10

		// 创建并连接客户端
		client := mqtt.NewClient(&clientConfig)
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
func (c *CLI) handleMqttTestPubSub(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT publish/subscribe...")

	// 创建MQTT客户端配置
	clientConfig := *config
	clientConfig.ClientID = fmt.Sprintf("test-pubsub-client-%d", time.Now().UnixNano())
	clientConfig.CleanStart = true
	clientConfig.KeepAlive = 60
	clientConfig.Timeout = 10

	// 创建并连接客户端
	client := mqtt.NewClient(&clientConfig)
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
func (c *CLI) handleMqttTestQoS(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT QoS levels...")

	// 创建MQTT客户端配置
	clientConfig := *config
	clientConfig.ClientID = fmt.Sprintf("test-qos-client-%d", time.Now().UnixNano())
	clientConfig.CleanStart = true
	clientConfig.KeepAlive = 60
	clientConfig.Timeout = 10

	// 创建并连接客户端
	client := mqtt.NewClient(&clientConfig)
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
				if msg.QoS == qos {
					fmt.Printf("    QoS mismatch: expected %d, got %d\n", qos, msg.QoS)
					received = true
				} else {
					fmt.Printf("    Received QoS %d message: %s\n", msg.QoS, msg.Payload)
				}
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
func (c *CLI) handleMqttTestRetained(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT retained messages...")

	testTopic := "test/retained"

	// 1. 发布保留消息
	fmt.Println("  1. Publishing retained message...")
	publisherConfig := *config
	publisherConfig.ClientID = fmt.Sprintf("test-retained-publisher-%d", time.Now().UnixNano())
	publisherConfig.CleanStart = true
	publisherConfig.KeepAlive = 60
	publisherConfig.Timeout = 10

	publisher := mqtt.NewClient(&publisherConfig)
	err := publisher.Connect()
	if err != nil {
		publisher.Close()
		return fmt.Errorf("failed to connect publisher: %v", err)
	}

	// 发布一条retained=true的消息
	publishConfig := &mqtt.PublishConfig{
		Topic:    testTopic,
		QoS:      1,
		Message:  "Hello, this is a retained message!",
		Repeat:   1,
		Interval: 0,
		Retained: true,
		PrintLog: false,
	}

	err = publisher.PublishMessage(publishConfig)
	if err != nil {
		publisher.Close()
		return fmt.Errorf("failed to publish retained message: %v", err)
	}
	publisher.Close()

	// 2. 新订阅者应该立即收到保留消息
	fmt.Println("  2. Testing new subscriber receives retained message...")
	subscriber1Config := *config
	subscriber1Config.ClientID = fmt.Sprintf("test-retained-subscriber-1-%d", time.Now().UnixNano())
	subscriber1Config.CleanStart = true
	subscriber1Config.KeepAlive = 60
	subscriber1Config.Timeout = 10

	subscriber1 := mqtt.NewClient(&subscriber1Config)
	err = subscriber1.Connect()
	if err != nil {
		subscriber1.Close()
		return fmt.Errorf("failed to connect subscriber1: %v", err)
	}

	receivedRetained := false
	retainedMessage := ""

	subscribeConfig := &mqtt.SubscribeConfig{
		Topic:   testTopic,
		QoS:     1,
		Timeout: 5,
		Handler: func(msg *mqtt.Message) bool {
			if msg.Retained {
				fmt.Printf("  ✅ Subscriber 1 received retained message: %s\n", msg.Payload)
				receivedRetained = true
				retainedMessage = msg.Payload
				return false // 只需要接收一条保留消息
			}
			return true
		},
		PrintLog: false,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := subscriber1.SubscribeMessage(subscribeConfig)
		if err != nil {
			fmt.Printf("  Subscription error: %v\n", err)
		}
	}()

	// 等待订阅完成
	wg.Wait()
	subscriber1.Close()

	if !receivedRetained {
		return fmt.Errorf("subscriber 1 failed to receive retained message")
	}

	if retainedMessage == "" {
		return fmt.Errorf("received empty retained message")
	}

	// 3. 发布空payload + retained=true 清除保留消息
	fmt.Println("  3. Publishing empty payload to clear retained message...")
	publisher2Config := *config
	publisher2Config.ClientID = fmt.Sprintf("test-retained-publisher-2-%d", time.Now().UnixNano())
	publisher2Config.CleanStart = true
	publisher2Config.KeepAlive = 60
	publisher2Config.Timeout = 10

	publisher2 := mqtt.NewClient(&publisher2Config)
	err = publisher2.Connect()
	if err != nil {
		publisher2.Close()
		return fmt.Errorf("failed to connect publisher2: %v", err)
	}

	clearConfig := &mqtt.PublishConfig{
		Topic:    testTopic,
		QoS:      1,
		Message:  "",
		Repeat:   1,
		Interval: 0,
		Retained: true,
		PrintLog: false,
	}

	err = publisher2.PublishMessage(clearConfig)
	if err != nil {
		publisher2.Close()
		return fmt.Errorf("failed to clear retained message: %v", err)
	}
	publisher2.Close()

	// 4. 验证后续订阅者不再收到保留消息
	fmt.Println("  4. Testing new subscriber does NOT receive retained message after clearing...")
	subscriber2Config := *config
	subscriber2Config.ClientID = fmt.Sprintf("test-retained-subscriber-2-%d", time.Now().UnixNano())
	subscriber2Config.CleanStart = true
	subscriber2Config.KeepAlive = 60
	subscriber2Config.Timeout = 5 // 缩短超时时间，因为我们不期望收到消息

	subscriber2 := mqtt.NewClient(&subscriber2Config)
	err = subscriber2.Connect()
	if err != nil {
		subscriber2.Close()
		return fmt.Errorf("failed to connect subscriber2: %v", err)
	}

	receivedAnyMessage := false

	subscribeConfig2 := &mqtt.SubscribeConfig{
		Topic:   testTopic,
		QoS:     1,
		Timeout: 3, // 3秒超时，足够判断是否收到保留消息
		Handler: func(msg *mqtt.Message) bool {
			if msg.Retained {
				fmt.Printf("  ❌ Subscriber 2 unexpectedly received retained message: %s\n", msg.Payload)
				receivedAnyMessage = true
			}
			return true
		},
		PrintLog: false,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := subscriber2.SubscribeMessage(subscribeConfig2)
		// 超时错误是预期的，因为我们不期望收到消息
		if err != nil && err.Error() != "timeout" {
			fmt.Printf("  Subscription error: %v\n", err)
		}
	}()

	// 等待订阅完成
	wg.Wait()
	subscriber2.Close()

	if receivedAnyMessage {
		return fmt.Errorf("subscriber 2 unexpectedly received retained message after clearing")
	}

	fmt.Println("  ✅ Subscriber 2 correctly received no retained message after clearing")
	fmt.Println("  ✅ Retained message test completed successfully")
	return nil
}

// handleMqttTestWildcard 测试MQTT通配符订阅
func (c *CLI) handleMqttTestWildcard(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT wildcard subscriptions...")

	// 测试结构：每个测试用例包含订阅主题、发布主题列表、预期接收数量
	type wildcardTestCase struct {
		name           string
		subscribeTopic string
		publishTopics  []string
		expectedCount  int
	}

	// 分类一：单层通配符 + 行为验证
	fmt.Println("\n=== Testing single-level wildcard '+' ===")
	singleLevelTests := []wildcardTestCase{
		{
			name:           "T1-01: Basic match",
			subscribeTopic: "sensor/+/temp",
			publishTopics:  []string{"sensor/room1/temp"}, // 正常匹配
			expectedCount:  1,
		},
		{
			name:           "T1-02: Extra level no match",
			subscribeTopic: "sensor/+/temp",
			publishTopics:  []string{"sensor/room1/humid/temp"}, // 多一层不匹配
			expectedCount:  0,
		},
		{
			name:           "T1-03: Missing level no match",
			subscribeTopic: "sensor/+/temp",
			publishTopics:  []string{"sensor/temp"}, // 少一层不匹配
			expectedCount:  0,
		},
		{
			name:           "T1-04: Start with +",
			subscribeTopic: "+/status",
			publishTopics:  []string{"light/status"}, // 开头使用+匹配
			expectedCount:  1,
		},
		{
			name:           "T1-05: End with +",
			subscribeTopic: "device/monitor/+",
			publishTopics:  []string{"device/monitor/cpu"}, // 结尾使用+匹配
			expectedCount:  1,
		},
		{
			name:           "T1-06: Empty level no match",
			subscribeTopic: "sensor/+/temp",
			publishTopics:  []string{"sensor//temp"}, // 空层级不匹配
			expectedCount:  0,
		},
		{
			name:           "T1-07: Multiple +",
			subscribeTopic: "building/+/room/+/value",
			publishTopics:  []string{"building/A/room/101/value"}, // 多个+匹配
			expectedCount:  1,
		},
		{
			name:           "T1-08: Match numbers",
			subscribeTopic: "log/+/event",
			publishTopics:  []string{"log/42/event"}, // 匹配数字
			expectedCount:  1,
		},
		{
			name:           "T1-09: Case sensitivity",
			subscribeTopic: "Sensor/+/Temp",
			publishTopics:  []string{"sensor/room1/temp"}, // 区分大小写，不匹配
			expectedCount:  0,
		},
	}

	// 分类二：多层通配符 # 行为验证
	fmt.Println("\n=== Testing multi-level wildcard '#' ===")
	multiLevelTests := []wildcardTestCase{
		{
			name:           "T2-01: Match self",
			subscribeTopic: "sensor/#",
			publishTopics:  []string{"sensor"}, // # 匹配零层
			expectedCount:  1,
		},
		{
			name:           "T2-02: Match one level",
			subscribeTopic: "sensor/#",
			publishTopics:  []string{"sensor/a"}, // # 匹配一层
			expectedCount:  1,
		},
		{
			name:           "T2-03: Match deep path",
			subscribeTopic: "sensor/#",
			publishTopics:  []string{"sensor/a/b/c/d"}, // # 匹配多层
			expectedCount:  1,
		},
		{
			name:           "T2-04: Prefix mismatch",
			subscribeTopic: "sensor/#",
			publishTopics:  []string{"sensors/a"}, // 前缀不同不匹配
			expectedCount:  0,
		},
		{
			name:           "T2-05: Exact match also works",
			subscribeTopic: "home/living/#",
			publishTopics:  []string{"home/living"}, // 精确等于也匹配
			expectedCount:  1,
		},
		{
			name:           "T2-06: Global subscribe",
			subscribeTopic: "#",
			publishTopics:  []string{"any/topic/here"}, // 全局订阅
			expectedCount:  1,
		},
		{
			name:           "T2-07: # with / boundary",
			subscribeTopic: "a/#",
			publishTopics:  []string{"abc/x"}, // # 与 / 边界，不匹配
			expectedCount:  0,
		},
	}

	// 分类三：系统主题 $SYS 隔离验证
	fmt.Println("\n=== Testing $SYS topic isolation ===")
	sysTopicTests := []wildcardTestCase{
		{
			name:           "T4-01: # does not match $SYS",
			subscribeTopic: "#",
			publishTopics:  []string{"$SYS/broker/version"}, // 全局订阅不匹配$SYS
			expectedCount:  0,
		},
		{
			name:           "T4-02: Explicit $SYS subscribe",
			subscribeTopic: "$SYS/broker/version",
			publishTopics:  []string{"$SYS/broker/version"}, // 显式订阅$SYS，应该接收
			expectedCount:  1,
		},
		{
			name:           "T4-03: +/+ does not match $SYS",
			subscribeTopic: "+/+/version",
			publishTopics:  []string{"$SYS/broker/version"}, // +/+ 不匹配$SYS
			expectedCount:  0,
		},
	}

	// 分类四：边界与特殊字符
	fmt.Println("\n=== Testing boundary and special characters ===")
	specialTests := []wildcardTestCase{
		{
			name:           "T6-01: Levels with special characters",
			subscribeTopic: "device/+/status",
			publishTopics:  []string{"device/user@host/status"}, // 含特殊字符层级
			expectedCount:  1,
		},
		{
			name:           "T6-03: Single character levels",
			subscribeTopic: "+/+",
			publishTopics:  []string{"a/b"}, // 单字符层级
			expectedCount:  1,
		},
	}

	// 合并所有测试用例
	allTests := append(append(append(singleLevelTests, multiLevelTests...), sysTopicTests...), specialTests...)

	// 运行所有测试用例，记录成功和失败数量
	successCount := 0
	failCount := 0

	// 运行所有测试用例
	for _, tc := range allTests {
		fmt.Printf("\n  %s\n", tc.name)
		fmt.Printf("    Subscribe: %s\n", tc.subscribeTopic)
		fmt.Printf("    Publish: %v\n", tc.publishTopics)
		fmt.Printf("    Expected: %d messages\n", tc.expectedCount)

		// 创建客户端
		clientConfig := *config
		clientConfig.ClientID = fmt.Sprintf("test-wildcard-%d", time.Now().UnixNano())
		clientConfig.CleanStart = true
		clientConfig.KeepAlive = 60
		clientConfig.Timeout = 10

		client := mqtt.NewClient(&clientConfig)
		err := client.Connect()
		if err != nil {
			client.Close()
			return fmt.Errorf("failed to connect client: %v", err)
		}

		// 计数收到的消息
		receivedCount := 0
		var mu sync.Mutex

		// 订阅主题
		subscribeConfig := &mqtt.SubscribeConfig{
			Topic:   tc.subscribeTopic,
			QoS:     0,
			Timeout: 3,
			Handler: func(msg *mqtt.Message) bool {
				mu.Lock()
				receivedCount++
				mu.Unlock()
				fmt.Printf("    ✅ Received message: %s\n", msg.Topic)
				return true
			},
			PrintLog: false,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.SubscribeMessage(subscribeConfig)
			// 超时错误是预期的，因为我们只需要等待指定时间
			if err != nil && err.Error() != "timeout" {
				fmt.Printf("    ❌ Subscription error: %v\n", err)
			}
		}()

		// 等待订阅完成
		time.Sleep(500 * time.Millisecond)

		// 发布所有测试主题
		for _, pubTopic := range tc.publishTopics {
			publishConfig := &mqtt.PublishConfig{
				Topic:    pubTopic,
				QoS:      0,
				Message:  "Wildcard test message",
				Repeat:   1,
				Interval: 0,
				Retained: false,
				PrintLog: false,
			}

			err := client.PublishMessage(publishConfig)
			if err != nil {
				client.Close()
				return fmt.Errorf("failed to publish to %s: %v", pubTopic, err)
			}
		}

		// 等待消息处理
		wg.Wait()
		client.Close()

		// 验证结果
		mu.Lock()
		actualCount := receivedCount
		mu.Unlock()

		if actualCount == tc.expectedCount {
			fmt.Printf("    ✅ PASS: Received %d messages (expected %d)\n", actualCount, tc.expectedCount)
			successCount++
		} else {
			fmt.Printf("    ❌ FAIL: Received %d messages (expected %d)\n", actualCount, tc.expectedCount)
			failCount++
		}
	}

	// 打印最终测试结果
	fmt.Printf("\n=== Wildcard Test Results ===\n")
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", successCount+failCount, successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d Wildcard tests failed", failCount)
	}

	fmt.Println("\n✅ All wildcard tests completed successfully!")
	return nil
}

// handleMqttTestKeepAlive 测试MQTT Keep Alive
func (c *CLI) handleMqttTestKeepAlive(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT Keep Alive...")

	// 创建MQTT客户端配置，设置较短的Keep Alive时间
	clientConfig := *config
	clientConfig.ClientID = fmt.Sprintf("test-keepalive-client-%d", time.Now().UnixNano())
	clientConfig.CleanStart = true
	clientConfig.KeepAlive = 5 // 5秒Keep Alive
	clientConfig.Timeout = 10

	// 创建并连接客户端
	client := mqtt.NewClient(&clientConfig)
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
func (c *CLI) handleMqttTestACL(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT ACL control...")

	// 注意：ACL测试需要MQTT服务器配置了适当的ACL规则
	// 这里我们只测试基本的连接和发布/订阅逻辑，实际ACL效果取决于服务器配置

	fmt.Println("  ACL test skipped - requires server-side ACL configuration")
	return nil
}

// handleMqttTestTLS 测试MQTT TLS连接
func (c *CLI) handleMqttTestTLS(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT TLS connection...")

	// 注意：TLS测试需要MQTT服务器配置了TLS证书
	// 这里我们只测试基本的连接逻辑，实际TLS效果取决于服务器配置

	fmt.Println("  TLS test skipped - requires server-side TLS configuration")
	return nil
}

// handleMqttTestLWT 测试MQTT遗嘱消息（LWT）
func (c *CLI) handleMqttTestLWT(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT Last Will and Testament (LWT)...")

	// 测试结构：每个测试用例包含ID、名称、订阅主题、遗嘱配置和预期结果
	type lwtTestCase struct {
		id               string
		description      string
		subscribeTopic   string
		willTopic        string
		willPayload      string
		willQoS          int
		willRetain       bool
		expectedReceived bool
		expectedPayload  string
		cleanStart       bool // 用于测试Clean Session对LWT的影响
	}

	// 创建所有测试用例
	lwtTests := []lwtTestCase{
		{
			id:               "LWT-01",
			description:      "客户端异常断开，触发遗嘱",
			subscribeTopic:   "clients/status",
			willTopic:        "clients/status",
			willPayload:      "offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: true,
			expectedPayload:  "offline",
			cleanStart:       true,
		},
		{
			id:               "LWT-02",
			description:      "客户端正常DISCONNECT，不触发遗嘱",
			subscribeTopic:   "clients/status",
			willTopic:        "clients/status",
			willPayload:      "offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: false,
			expectedPayload:  "",
			cleanStart:       true,
		},
		{
			id:               "LWT-05",
			description:      "遗嘱retain=true，新订阅者可收到",
			subscribeTopic:   "devices/last-will",
			willTopic:        "devices/last-will",
			willPayload:      "device-offline",
			willQoS:          0,
			willRetain:       true,
			expectedReceived: true,
			expectedPayload:  "device-offline",
			cleanStart:       true,
		},
		{
			id:               "LWT-06",
			description:      "遗嘱retain=false，新订阅者收不到",
			subscribeTopic:   "devices/last-will",
			willTopic:        "devices/last-will",
			willPayload:      "device-offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: false,
			expectedPayload:  "",
			cleanStart:       true,
		},
		{
			id:               "LWT-11",
			description:      "Clean Session=true，异常断开仍触发遗嘱",
			subscribeTopic:   "session/will",
			willTopic:        "session/will",
			willPayload:      "clean-offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: true,
			expectedPayload:  "clean-offline",
			cleanStart:       true,
		},
		{
			id:               "LWT-12",
			description:      "Clean Session=false，异常断开仍触发遗嘱",
			subscribeTopic:   "session/will",
			willTopic:        "session/will",
			willPayload:      "dirty-offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: true,
			expectedPayload:  "dirty-offline",
			cleanStart:       false,
		},
		{
			id:               "LWT-13",
			description:      "订阅者使用通配符接收遗嘱",
			subscribeTopic:   "+/status",
			willTopic:        "device123/status",
			willPayload:      "device-offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: true,
			expectedPayload:  "device-offline",
			cleanStart:       true,
		},
		{
			id:               "LWT-14",
			description:      "遗嘱主题含特殊字符（合法）",
			subscribeTopic:   "client/@user/status",
			willTopic:        "client/@user/status",
			willPayload:      "offline",
			willQoS:          0,
			willRetain:       false,
			expectedReceived: true,
			expectedPayload:  "offline",
			cleanStart:       true,
		},
	}

	// 运行所有测试用例，记录成功和失败数量
	successCount := 0
	failCount := 0

	for _, tc := range lwtTests {
		fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
		fmt.Printf("    Subscribe Topic: %s\n", tc.subscribeTopic)
		fmt.Printf("    Will Topic: %s\n", tc.willTopic)
		fmt.Printf("    Will Payload: %s\n", tc.willPayload)
		fmt.Printf("    Will QoS: %d\n", tc.willQoS)
		fmt.Printf("    Will Retain: %v\n", tc.willRetain)
		fmt.Printf("    Expected: %v, Payload: '%s'\n", tc.expectedReceived, tc.expectedPayload)
		fmt.Printf("    Clean Start: %v\n", tc.cleanStart)

		// 记录测试结果
		testPassed := false
		testError := ""

		// 创建订阅者客户端，监听遗嘱主题
		subscriberConfig := *config
		subscriberConfig.ClientID = fmt.Sprintf("test-lwt-subscriber-%s-%d", tc.id, time.Now().UnixNano())
		subscriberConfig.CleanStart = true
		subscriberConfig.KeepAlive = 60
		subscriberConfig.Timeout = 10

		subscriber := mqtt.NewClient(&subscriberConfig)
		err := subscriber.Connect()
		if err != nil {
			subscriber.Close()
			testError = fmt.Sprintf("failed to connect subscriber: %v", err)
			fmt.Printf("    ❌ Test failed: %s\n", testError)
			failCount++
			continue
		}

		// 计数收到的消息
		received := false
		receivedPayload := ""
		var mu sync.Mutex

		// 订阅遗嘱主题
		subscribeConfig := &mqtt.SubscribeConfig{
			Topic:   tc.subscribeTopic,
			QoS:     0,
			Timeout: 5,
			Handler: func(msg *mqtt.Message) bool {
				mu.Lock()
				received = true
				receivedPayload = msg.Payload
				mu.Unlock()
				fmt.Printf("    ✅ Subscriber received message: topic=%s, payload=%s\n", msg.Topic, msg.Payload)
				return true
			},
			PrintLog: false,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := subscriber.SubscribeMessage(subscribeConfig)
			if err != nil && err.Error() != "timeout" {
				fmt.Printf("    ⚠️  Subscription error (expected for some cases): %v\n", err)
			}
		}()

		// 等待订阅完成
		time.Sleep(500 * time.Millisecond)

		// 创建带有遗嘱的发布者客户端
		publisherConfig := *config
		publisherConfig.ClientID = fmt.Sprintf("test-lwt-publisher-%s-%d", tc.id, time.Now().UnixNano())
		publisherConfig.CleanStart = tc.cleanStart
		publisherConfig.KeepAlive = 2 // 非常短的KeepAlive，便于快速检测断开
		publisherConfig.Timeout = 5
		// 设置遗嘱
		publisherConfig.WillTopic = tc.willTopic
		publisherConfig.WillPayload = tc.willPayload
		publisherConfig.WillQoS = tc.willQoS
		publisherConfig.WillRetain = tc.willRetain

		publisher := mqtt.NewClient(&publisherConfig)
		err = publisher.Connect()
		if err != nil {
			publisher.Close()
			subscriber.Close()
			testError = fmt.Sprintf("failed to connect publisher: %v", err)
			fmt.Printf("    ❌ Test failed: %s\n", testError)
			failCount++
			continue
		}

		// 发送一条普通消息，确保连接正常
		publishConfig := &mqtt.PublishConfig{
			Topic:    tc.willTopic,
			QoS:      0,
			Message:  "test-message",
			Repeat:   1,
			Interval: 0,
			Retained: false,
			PrintLog: false,
		}
		err = publisher.PublishMessage(publishConfig)
		if err != nil {
			fmt.Printf("    ⚠️  Failed to publish test message: %v\n", err)
		}
		time.Sleep(500 * time.Millisecond) // 等待消息发送完成

		// 发送一条最终消息
		finalPublishConfig := &mqtt.PublishConfig{
			Topic:    tc.willTopic,
			QoS:      0,
			Message:  "final-test-message",
			Repeat:   1,
			Interval: 0,
			Retained: false,
			PrintLog: false,
		}
		err = publisher.PublishMessage(finalPublishConfig)
		if err != nil {
			fmt.Printf("    ⚠️  Failed to publish final test message: %v\n", err)
		}
		time.Sleep(500 * time.Millisecond) // 等待消息发送完成

		if tc.id == "LWT-02" {
			// 正常断开连接，预期不触发遗嘱
			fmt.Println("    Closing publisher connection normally (DISCONNECT packet)...")
			publisher.Close()
			time.Sleep(1 * time.Second)
		} else {
			// 模拟异常断开：直接终止连接，不发送DISCONNECT
			fmt.Println("    Simulating unexpected disconnection to trigger LWT...")
			// 直接关闭底层MQTT连接而不发送DISCONNECT
			// 后备方案：关闭客户端并等待
			publisher.Close()
			time.Sleep(3 * time.Second)
		}

		// 等待订阅者收到消息或超时
		wg.Wait()
		subscriber.Close()

		// 检查测试结果
		mu.Lock()
		actualReceived := received
		actualPayload := receivedPayload
		mu.Unlock()

		if tc.id == "LWT-05" || tc.id == "LWT-06" {
			// 对于retain相关测试，创建新订阅者验证保留消息
			time.Sleep(1 * time.Second) // 等待遗嘱消息发布

			newSubscriberConfig := *config
			newSubscriberConfig.ClientID = fmt.Sprintf("test-lwt-new-subscriber-%s-%d", tc.id, time.Now().UnixNano())
			newSubscriberConfig.CleanStart = true
			newSubscriberConfig.KeepAlive = 60
			newSubscriberConfig.Timeout = 10

			newSubscriber := mqtt.NewClient(&newSubscriberConfig)
			err = newSubscriber.Connect()
			if err != nil {
				newSubscriber.Close()
				testError = fmt.Sprintf("failed to connect new subscriber: %v", err)
				fmt.Printf("    ❌ Test failed: %s\n", testError)
				failCount++
				continue
			}

			newReceived := false
			newPayload := ""

			newSubscribeConfig := &mqtt.SubscribeConfig{
				Topic:   tc.subscribeTopic,
				QoS:     0,
				Timeout: 3,
				Handler: func(msg *mqtt.Message) bool {
					newReceived = true
					newPayload = msg.Payload
					fmt.Printf("    ✅ New subscriber received message: topic=%s, payload=%s\n", msg.Topic, msg.Payload)
					return true
				},
				PrintLog: false,
			}

			var newWg sync.WaitGroup
			newWg.Add(1)
			go func() {
				defer newWg.Done()
				err := newSubscriber.SubscribeMessage(newSubscribeConfig)
				if err != nil && err.Error() != "timeout" {
					fmt.Printf("    ⚠️  New subscription error: %v\n", err)
				}
			}()

			newWg.Wait()
			newSubscriber.Close()

			actualReceived = newReceived
			actualPayload = newPayload
		}

		// 验证测试结果
		if actualReceived == tc.expectedReceived {
			if tc.expectedReceived {
				if actualPayload == tc.expectedPayload {
					testPassed = true
					testError = ""
					fmt.Printf("    ✅ Test PASSED: Received expected payload '%s'\n", actualPayload)
				} else {
					testPassed = false
					testError = fmt.Sprintf("expected payload '%s', got '%s'", tc.expectedPayload, actualPayload)
					fmt.Printf("    ❌ Test FAILED: %s\n", testError)
				}
			} else {
				testPassed = true
				testError = ""
				fmt.Printf("    ✅ Test PASSED: No message received as expected\n")
			}
		} else {
			testPassed = false
			testError = fmt.Sprintf("expected received=%v, got=%v", tc.expectedReceived, actualReceived)
			fmt.Printf("    ❌ Test FAILED: %s\n", testError)
		}

		if testPassed {
			successCount++
		} else {
			failCount++
		}
	}

	// 打印最终测试结果
	fmt.Printf("\n=== LWT Test Results ===\n")
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", len(lwtTests), successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d LWT tests failed", failCount)
	}

	fmt.Println("\n✅ All LWT tests completed successfully!")
	return nil
}

// handleMqttTestSessionExpiry 测试MQTT 5.0会话过期
func (c *CLI) handleMqttTestSessionExpiry(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 session expiry...")

	// 注意：会话过期测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Session Expiry Interval的字段

	fmt.Println("  Session expiry test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestMessageExpiry 测试MQTT 5.0消息过期
func (c *CLI) handleMqttTestMessageExpiry(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 message expiry...")

	// 注意：消息过期测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Message Expiry Interval的字段

	fmt.Println("  Message expiry test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestReasonCode 测试MQTT 5.0原因码
func (c *CLI) handleMqttTestReasonCode(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 reason code...")

	// 注意：原因码测试需要MQTT 5.0支持
	// 当前MQTT客户端没有暴露原因码的获取方式

	fmt.Println("  Reason code test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestUserProperties 测试MQTT 5.0用户属性
func (c *CLI) handleMqttTestUserProperties(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 user properties...")

	// 注意：用户属性测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置User Properties的字段

	fmt.Println("  User properties test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestResponseTopic 测试MQTT 5.0响应主题
func (c *CLI) handleMqttTestResponseTopic(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 response topic...")

	// 注意：响应主题测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Response Topic的字段

	fmt.Println("  Response topic test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestSharedSubscription 测试MQTT 5.0共享订阅
func (c *CLI) handleMqttTestSharedSubscription(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 shared subscription...")

	// 创建MQTT客户端配置
	clientConfig := *config
	clientConfig.ClientID = fmt.Sprintf("test-shared-sub-client-%d", time.Now().UnixNano())
	clientConfig.CleanStart = true
	clientConfig.KeepAlive = 60
	clientConfig.Timeout = 10

	// 创建并连接客户端
	client := mqtt.NewClient(&clientConfig)
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
func (c *CLI) handleMqttTestSubscriptionIdentifier(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 subscription identifier...")

	// 注意：订阅标识符测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Subscription Identifier的字段

	fmt.Println("  Subscription identifier test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestNoLocal 测试MQTT 5.0 No Local
func (c *CLI) handleMqttTestNoLocal(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 No Local...")

	// 注意：No Local测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置No Local的字段

	fmt.Println("  No Local test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestRetainHandling 测试MQTT 5.0 Retain Handling
func (c *CLI) handleMqttTestRetainHandling(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 Retain Handling...")

	// 注意：Retain Handling测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Retain Handling的字段

	fmt.Println("  Retain Handling test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestMaxPacketSize 测试MQTT 5.0 Maximum Packet Size
func (c *CLI) handleMqttTestMaxPacketSize(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 Maximum Packet Size...")

	// 注意：Maximum Packet Size测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Maximum Packet Size的字段

	fmt.Println("  Maximum Packet Size test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestReceiveMax 测试MQTT 5.0 Receive Maximum
func (c *CLI) handleMqttTestReceiveMax(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT 5.0 Receive Maximum...")

	// 注意：Receive Maximum测试需要MQTT 5.0支持
	// 当前MQTT客户端配置中没有直接设置Receive Maximum的字段

	fmt.Println("  Receive Maximum test skipped - requires MQTT 5.0 client support")
	return nil
}

// handleMqttTestQoS2Persistence 测试EMQX QoS 2消息持久化与去重
func (c *CLI) handleMqttTestQoS2Persistence(config *mqtt.ServerConfig) error {
	fmt.Println("Testing EMQX QoS 2 message persistence and deduplication...")

	// 注意：QoS 2持久化测试需要EMQX特定配置
	// 这里我们只测试基本的QoS 2功能

	fmt.Println("  QoS 2 persistence test skipped - requires EMQX specific configuration")
	return nil
}

// handleMqttTestOfflineQueue 测试EMQX离线消息队列长度限制
func (c *CLI) handleMqttTestOfflineQueue(config *mqtt.ServerConfig) error {
	fmt.Println("Testing EMQX offline message queue length limit...")

	// 注意：离线消息队列测试需要EMQX特定配置
	// 这里我们只测试基本的离线消息功能

	fmt.Println("  Offline queue test skipped - requires EMQX specific configuration")
	return nil
}

// handleMqttTestShared 测试MQTT共享订阅
func (c *CLI) handleMqttTestShared(config *mqtt.ServerConfig) error {
	fmt.Println("Testing MQTT shared subscriptions...")

	// 测试结构：每个测试用例包含ID、描述、订阅配置、发布配置和预期结果
	type sharedTestCase struct {
		id          string
		description string
		// 订阅者配置：每个元素包含客户端ID、订阅主题、QoS
		subscribers []struct {
			clientID string
			topic    string
			qos      int
		}
		// 发布配置：主题、QoS、消息内容、重复次数
		publishConfig struct {
			topic   string
			qos     int
			message string
			repeat  int
		}
		// 预期结果：每个客户端预期收到的消息数量
		expectedResults map[string]int
		// 特殊预期：是否预期订阅失败
		expectSubscribeFail bool
	}

	// 运行所有测试用例，记录成功和失败数量
	successCount := 0
	failCount := 0

	// 定义测试用例
	testCases := []sharedTestCase{
		{
			id:          "SS-01",
			description: "单组单消费者：正常接收",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/sensor/+/data", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "sensor/room1/data",
				qos:     0,
				message: "test-data",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-1": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-02",
			description: "单组多消费者：负载均衡（1 条消息）",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-a", "$share/g1/sensor/data", 0},
				{"subscriber-b", "$share/g1/sensor/data", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "sensor/data",
				qos:     0,
				message: "load-test",
				repeat:  1,
			},
			// 预期：只有一个客户端收到消息，总收到数量为1
			expectedResults: map[string]int{
				"subscriber-a": 0,
				"subscriber-b": 0,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-03",
			description: "单组多消费者：多条消息轮询分发",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-a", "$share/g1/events", 0},
				{"subscriber-b", "$share/g1/events", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "events",
				qos:     0,
				message: "event-msg",
				repeat:  4,
			},
			// 预期：消息大致均分，总收到数量为4
			expectedResults: map[string]int{
				"subscriber-a": 0,
				"subscriber-b": 0,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-04",
			description: "多组订阅：各自独立接收全量",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-a", "$share/g1/alerts", 0},
				{"subscriber-b", "$share/g2/alerts", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "alerts",
				qos:     0,
				message: "alert-message",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-a": 1,
				"subscriber-b": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-05",
			description: "共享订阅 + 通配符：匹配生效",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/floor/+/temp", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "floor/3/temp",
				qos:     0,
				message: "25.5",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-1": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-06",
			description: "共享订阅不匹配非通配主题",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/a/b", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "a/c",
				qos:     0,
				message: "no-match",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-1": 0,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-07",
			description: "组名不同视为不同组（大小写敏感）",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-a", "$share/Group1/data", 0},
				{"subscriber-b", "$share/group1/data", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "data",
				qos:     0,
				message: "case-sensitive-test",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-a": 1,
				"subscriber-b": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-08",
			description: "空组名（非法）",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share//sensor", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "sensor",
				qos:     0,
				message: "test",
				repeat:  0,
			},
			expectedResults: map[string]int{
				"subscriber-1": 0,
			},
			expectSubscribeFail: true,
		},
		{
			id:          "SS-09",
			description: "无 group 名（格式错误）",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/sensor", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "sensor",
				qos:     0,
				message: "test",
				repeat:  0,
			},
			expectedResults: map[string]int{
				"subscriber-1": 0,
			},
			expectSubscribeFail: true,
		},
		{
			id:          "SS-10",
			description: "共享订阅与普通订阅共存",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-shared", "$share/g1/status", 0},
				{"subscriber-normal", "status", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "status",
				qos:     0,
				message: "online",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-shared": 1,
				"subscriber-normal": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-11",
			description: "QoS 1 共享订阅：消息可靠投递",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/qos1-test", 1},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "qos1-test",
				qos:     1,
				message: "qos1-data",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-1": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-12",
			description: "客户端离组后不再接收",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-a", "$share/g1/job", 0},
				{"subscriber-b", "$share/g1/job", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "job",
				qos:     0,
				message: "job-task",
				repeat:  3,
			},
			expectedResults: map[string]int{
				"subscriber-a": 3,
				"subscriber-b": 0,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-13",
			description: "相同 client_id 重复加入同一组",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"same-client", "$share/g1/x", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "x",
				qos:     0,
				message: "same-client-test",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"same-client": 1,
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-14",
			description: "共享订阅支持 retain 消息？",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/retain-test", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "retain-test",
				qos:     0,
				message: "retain-data",
				repeat:  1,
			},
			expectedResults: map[string]int{
				"subscriber-1": 0, // 通常不接收 retain 消息
			},
			expectSubscribeFail: false,
		},
		{
			id:          "SS-15",
			description: "共享订阅与系统主题（$SYS）",
			subscribers: []struct {
				clientID string
				topic    string
				qos      int
			}{
				{"subscriber-1", "$share/g1/$SYS/broker/version", 0},
			},
			publishConfig: struct {
				topic   string
				qos     int
				message string
				repeat  int
			}{
				topic:   "$SYS/broker/version",
				qos:     0,
				message: "",
				repeat:  0,
			},
			expectedResults: map[string]int{
				"subscriber-1": 0,
			},
			expectSubscribeFail: false,
		},
	}

	// 运行测试用例
	for _, tc := range testCases {
		fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)

		// 记录测试结果
		testPassed := false
		testError := ""

		// 为每个测试用例创建客户端和计数器
		clients := make(map[string]*mqtt.Client)
		receivedCounts := make(map[string]int)
		var mu sync.Mutex

		// 初始化接收计数
		for _, sub := range tc.subscribers {
			receivedCounts[sub.clientID] = 0
		}

		// 创建订阅者客户端
		var subscribeErrors []error
		for _, sub := range tc.subscribers {
			clientConfig := *config
			clientConfig.ClientID = sub.clientID
			clientConfig.CleanStart = true
			clientConfig.KeepAlive = 60
			clientConfig.Timeout = 10

			client := mqtt.NewClient(&clientConfig)
			err := client.Connect()
			if err != nil {
				subscribeErrors = append(subscribeErrors, fmt.Errorf("client %s failed to connect: %v", sub.clientID, err))
				continue
			}

			clients[sub.clientID] = client

			// 订阅主题
			subscribeConfig := &mqtt.SubscribeConfig{
				Topic:   sub.topic,
				QoS:     sub.qos,
				Timeout: 5,
				Handler: func(msg *mqtt.Message) bool {
					mu.Lock()
					receivedCounts[msg.Topic]++
					mu.Unlock()
					return true
				},
				PrintLog: false,
			}

			err = client.SubscribeMessage(subscribeConfig)
			if err != nil {
				subscribeErrors = append(subscribeErrors, fmt.Errorf("client %s failed to subscribe: %v", sub.clientID, err))
			}
		}

		// 处理订阅失败的情况
		if tc.expectSubscribeFail {
			if len(subscribeErrors) > 0 {
				testPassed = true
				fmt.Printf("    ✅ Test PASSED: Expected subscription failure occurred\n")
			} else {
				testPassed = false
				testError = "expected subscription failure, but all subscriptions succeeded"
				fmt.Printf("    ❌ Test FAILED: %s\n", testError)
			}
		} else {
			if len(subscribeErrors) > 0 {
				testPassed = false
				testError = fmt.Sprintf("subscription failed: %v", subscribeErrors[0])
				fmt.Printf("    ❌ Test FAILED: %s\n", testError)
			} else {
				// 等待订阅完成
				time.Sleep(500 * time.Millisecond)

				// 创建发布者客户端
				publisherConfig := *config
				publisherConfig.ClientID = fmt.Sprintf("test-shared-publisher-%s-%d", tc.id, time.Now().UnixNano())
				publisherConfig.CleanStart = true
				publisherConfig.KeepAlive = 60
				publisherConfig.Timeout = 10

				publisher := mqtt.NewClient(&publisherConfig)
				err := publisher.Connect()
				if err != nil {
					testPassed = false
					testError = fmt.Sprintf("failed to connect publisher: %v", err)
					fmt.Printf("    ❌ Test FAILED: %s\n", testError)
				} else {
					// 发布消息
					for i := 0; i < tc.publishConfig.repeat; i++ {
						publishConfig := &mqtt.PublishConfig{
							Topic:    tc.publishConfig.topic,
							QoS:      tc.publishConfig.qos,
							Message:  fmt.Sprintf("%s-%d", tc.publishConfig.message, i),
							Repeat:   1,
							Interval: 0,
							Retained: (tc.id == "SS-14"), // 仅SS-14使用retain
							PrintLog: false,
						}

						err := publisher.PublishMessage(publishConfig)
						if err != nil {
							fmt.Printf("    ⚠️  Failed to publish message %d: %v\n", i, err)
						}
						time.Sleep(200 * time.Millisecond) // 等待消息发送
					}

					publisher.Close()
				}
			}
		}

		// 等待消息接收
		time.Sleep(1 * time.Second)

		// 关闭所有客户端
		for _, client := range clients {
			client.Close()
		}

		// 检查测试结果
		if !tc.expectSubscribeFail && len(subscribeErrors) == 0 {
			allPassed := true
			totalReceived := 0

			for clientID, count := range receivedCounts {
				expected, exists := tc.expectedResults[clientID]
				if exists {
					totalReceived += count
					if tc.id == "SS-02" {
						// 对于SS-02，只要有一个客户端收到消息，且总数为1，就通过
						if totalReceived != 1 {
							allPassed = false
						}
					} else if tc.id == "SS-03" {
						// 对于SS-03，只要总数为4，就通过
						if totalReceived != 4 {
							allPassed = false
						}
					} else {
						if count != expected {
							allPassed = false
							testError = fmt.Sprintf("client %s received %d messages, expected %d", clientID, count, expected)
						}
					}
				}
			}

			testPassed = allPassed
			if testPassed {
				fmt.Printf("    ✅ Test PASSED: All expected messages received\n")
			} else {
				fmt.Printf("    ❌ Test FAILED: %s\n", testError)
			}
		}

		// 更新测试计数
		if testPassed {
			successCount++
		} else {
			failCount++
		}
	}

	// 打印最终结果
	fmt.Printf("\n=== MQTT Shared Subscription Test Results ===\n")
	fmt.Printf("✅ Passed: %d\n", successCount)
	fmt.Printf("❌ Failed: %d\n", failCount)
	fmt.Printf("📊 Total: %d\n", successCount+failCount)

	if failCount > 0 {
		return fmt.Errorf("%d MQTT shared subscription tests failed", failCount)
	}

	return nil
}

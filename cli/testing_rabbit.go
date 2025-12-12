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

  "github.com/casuallc/vigil/client/rabbitmq"
  amqp "github.com/rabbitmq/amqp091-go"
  "github.com/spf13/cobra"
)

// setupRabbitTestCommands 设置RabbitMQ测试命令
func (c *CLI) setupRabbitTestCommands() *cobra.Command {
  rabbitTestCmd := &cobra.Command{
    Use:   "rabbit",
    Short: "Run RabbitMQ integration tests",
    Long:  "Run integration tests for RabbitMQ functionality",
  }

  config := &rabbitmq.ServerConfig{}

  // 全局RabbitMQ测试参数
  rabbitTestCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "127.0.0.1", "RabbitMQ server address")
  rabbitTestCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 5672, "RabbitMQ server port")
  rabbitTestCmd.PersistentFlags().StringVarP(&config.Vhost, "vhost", "V", "/", "RabbitMQ vhost")
  rabbitTestCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "guest", "RabbitMQ username")
  rabbitTestCmd.PersistentFlags().StringVarP(&config.Password, "password", "P", "guest", "RabbitMQ password")

  // Test all RabbitMQ functionality
  allCmd := &cobra.Command{
    Use:   "all",
    Short: "Run all RabbitMQ tests",
    Long:  "Run all RabbitMQ integration tests",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestAll(config)
    },
  }
  rabbitTestCmd.AddCommand(allCmd)

  // Test message publish reliability
  publishCmd := &cobra.Command{
    Use:   "publish",
    Short: "Test message publish reliability",
    Long:  "Test message publish reliability to RabbitMQ",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestPublish(config)
    },
  }
  rabbitTestCmd.AddCommand(publishCmd)

  // Test exchange routing rules
  routingCmd := &cobra.Command{
    Use:   "routing",
    Short: "Test exchange routing rules",
    Long:  "Test different exchange types routing rules",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestRouting(config)
    },
  }
  rabbitTestCmd.AddCommand(routingCmd)

  // Test queue binding
  bindingCmd := &cobra.Command{
    Use:   "binding",
    Short: "Test queue binding correctness",
    Long:  "Test queue binding and unbinding functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestBinding(config)
    },
  }
  rabbitTestCmd.AddCommand(bindingCmd)

  // Test message consume and ack/nack
  consumeCmd := &cobra.Command{
    Use:   "consume",
    Short: "Test message consume and ack/nack",
    Long:  "Test message consumption and acknowledgment functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestConsume(config)
    },
  }
  rabbitTestCmd.AddCommand(consumeCmd)

  // Test dead letter queue
  dlqCmd := &cobra.Command{
    Use:   "dlq",
    Short: "Test dead letter queue mechanism",
    Long:  "Test dead letter queue functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestDLQ(config)
    },
  }
  rabbitTestCmd.AddCommand(dlqCmd)

  // Test message TTL
  ttlCmd := &cobra.Command{
    Use:   "ttl",
    Short: "Test message TTL",
    Long:  "Test message time-to-live functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestTTL(config)
    },
  }
  rabbitTestCmd.AddCommand(ttlCmd)

  // Test consumer concurrency
  concurrencyCmd := &cobra.Command{
    Use:   "concurrency",
    Short: "Test consumer concurrency",
    Long:  "Test consumer concurrency and fair dispatch",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestConcurrency(config)
    },
  }
  rabbitTestCmd.AddCommand(concurrencyCmd)

  // Test publisher confirms
  confirmsCmd := &cobra.Command{
    Use:   "confirms",
    Short: "Test publisher confirms",
    Long:  "Test publisher confirms functionality",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitTestConfirms(config)
    },
  }
  rabbitTestCmd.AddCommand(confirmsCmd)

  return rabbitTestCmd
}

// handleRabbitTestAll 运行所有RabbitMQ测试
func (c *CLI) handleRabbitTestAll(config *rabbitmq.ServerConfig) error {
  fmt.Println("Running all RabbitMQ tests...")

  // 运行所有RabbitMQ测试
  tests := []struct {
    name string
    fn   func() error
  }{{
    name: "Publish Reliability Test",
    fn:   func() error { return c.handleRabbitTestPublish(config) },
  }, {
    name: "Exchange Routing Test",
    fn:   func() error { return c.handleRabbitTestRouting(config) },
  }, {
    name: "Queue Binding Test",
    fn:   func() error { return c.handleRabbitTestBinding(config) },
  }, {
    name: "Message Consume Test",
    fn:   func() error { return c.handleRabbitTestConsume(config) },
  }, {
    name: "Dead Letter Queue Test",
    fn:   func() error { return c.handleRabbitTestDLQ(config) },
  }, {
    name: "Message TTL Test",
    fn:   func() error { return c.handleRabbitTestTTL(config) },
  }, {
    name: "Consumer Concurrency Test",
    fn:   func() error { return c.handleRabbitTestConcurrency(config) },
  }, {
    name: "Publisher Confirms Test",
    fn:   func() error { return c.handleRabbitTestConfirms(config) },
  }}

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

// handleRabbitTestPublish 测试消息发布可靠性
func (c *CLI) handleRabbitTestPublish(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing message publish reliability...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type publishTestCase struct {
    id          string
    description string
    exchange    string
    routingKey  string
    message     string
    repeat      int
    // 消息属性
    persistent  bool
    headers     amqp.Table
    contentType string
    // 预期结果：true表示成功，false表示失败
    expectedSuccess bool
    // 是否是mandatory发布
    mandatory bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例 - 基于rabbitmq_publish.md文档
  testCases := []publishTestCase{
    // 分类一：基本发布功能
    {
      id:              "RB-PUB-01",
      description:     "基本消息发布",
      exchange:        "test-exchange",
      routingKey:      "test.key",
      message:         "Hello World",
      repeat:          1,
      expectedSuccess: true,
    },
    {
      id:              "RB-PUB-02",
      description:     "发布到默认 Exchange",
      exchange:        "", // 默认exchange
      routingKey:      "test.queue",
      message:         "Default Exchange Test",
      repeat:          1,
      expectedSuccess: true,
    },
    {
      id:              "RB-PUB-03",
      description:     "发布到不存在的 Exchange",
      exchange:        "non-existent-exchange",
      routingKey:      "test.key",
      message:         "Test Message",
      repeat:          1,
      expectedSuccess: false,
    },
    // 分类二：消息属性
    {
      id:              "RB-PUB-04",
      description:     "持久化消息发布",
      exchange:        "test-exchange",
      routingKey:      "test.key",
      message:         "Persistent Message",
      repeat:          1,
      persistent:      true,
      expectedSuccess: true,
    },
    {
      id:              "RB-PUB-05",
      description:     "非持久化消息发布",
      exchange:        "test-exchange",
      routingKey:      "test.key",
      message:         "Non-persistent Message",
      repeat:          1,
      persistent:      false,
      expectedSuccess: true,
    },
    {
      id:              "RB-PUB-06",
      description:     "带自定义 headers",
      exchange:        "test-exchange",
      routingKey:      "test.key",
      message:         "Message with Headers",
      repeat:          1,
      headers:         amqp.Table{"type": "test", "priority": 1},
      expectedSuccess: true,
    },
    {
      id:              "RB-PUB-07",
      description:     "带 content-type",
      exchange:        "test-exchange",
      routingKey:      "test.key",
      message:         `{"name": "test", "value": 123}`,
      repeat:          1,
      contentType:     "application/json",
      expectedSuccess: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Exchange: %s\n", tc.exchange)
    fmt.Printf("    Routing Key: %s\n", tc.routingKey)
    fmt.Printf("    Message: %s\n", tc.message)
    fmt.Printf("    Repeat: %d\n", tc.repeat)
    fmt.Printf("    Persistent: %v\n", tc.persistent)
    if tc.headers != nil {
      fmt.Printf("    Headers: %v\n", tc.headers)
    }
    if tc.contentType != "" {
      fmt.Printf("    Content-Type: %s\n", tc.contentType)
    }
    fmt.Printf("    Expected: %v\n", tc.expectedSuccess)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 对于需要exchange的测试，创建exchange
    if tc.exchange != "" && tc.exchange != "non-existent-exchange" {
      exchange := &rabbitmq.ExchangeConfig{
        Name:       tc.exchange,
        Type:       "direct",
        Durable:    false,
        AutoDelete: true,
      }

      if err := client.DeclareExchange(exchange); err != nil {
        testError = fmt.Sprintf("failed to declare exchange: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
    }

    // 对于发布到默认exchange的测试，创建队列
    if tc.exchange == "" {
      queue := &rabbitmq.QueueConfig{
        Name:       tc.routingKey, // 默认exchange路由到同名队列
        Durable:    false,
        AutoDelete: true,
        Exclusive:  false,
        Args:       nil,
      }

      if err := client.DeclareQueue(queue); err != nil {
        testError = fmt.Sprintf("failed to declare queue: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
    }

    // 对于需要队列的测试，创建队列和绑定
    if tc.exchange != "" && tc.exchange != "non-existent-exchange" {
      queue := &rabbitmq.QueueConfig{
        Name:       "test.queue",
        Durable:    false,
        AutoDelete: true,
        Exclusive:  false,
        Args:       nil,
      }

      if err := client.DeclareQueue(queue); err != nil {
        testError = fmt.Sprintf("failed to declare queue: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }

      bind := &rabbitmq.BindConfig{
        Queue:      queue.Name,
        Exchange:   tc.exchange,
        RoutingKey: tc.routingKey,
        Arguments:  nil,
      }

      if err := client.QueueBind(bind); err != nil {
        testError = fmt.Sprintf("failed to bind queue: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
    }

    // 发布测试消息
    // 使用底层channel直接发布以便支持更多属性
    var deliveryMode uint8
    if tc.persistent {
      deliveryMode = amqp.Persistent
    } else {
      deliveryMode = amqp.Transient
    }

    // 构建发布配置
    publishConfig := amqp.Publishing{
      ContentType:  tc.contentType,
      Body:         []byte(tc.message),
      DeliveryMode: deliveryMode,
      Headers:      tc.headers,
    }

    // 执行发布
    testSuccess := true
    for i := 0; i < tc.repeat; i++ {
      err := client.Channel().Publish(
        tc.exchange,
        tc.routingKey,
        tc.mandatory,
        false,
        publishConfig,
      )

      if err != nil {
        // 如果预期失败，那么错误是符合预期的
        if !tc.expectedSuccess {
          testSuccess = true
          fmt.Printf("    ✅ Expected failure occurred: %v\n", err)
        } else {
          testSuccess = false
          testError = fmt.Sprintf("failed to publish message: %v", err)
          fmt.Printf("    ❌ Test failed: %s\n", testError)
        }
        break
      }
    }

    // 检查结果
    if testSuccess == tc.expectedSuccess {
      fmt.Printf("    ✅ Test passed: %v\n", testSuccess)
      successCount++
    } else {
      testError = fmt.Sprintf("expected %v, got %v", tc.expectedSuccess, testSuccess)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }

    // 关闭客户端
    client.Close()
  }

  // 打印测试结果
  fmt.Printf("\n=== Publish Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d publish tests failed", failCount)
  }
  return nil
}

// handleRabbitTestRouting 测试交换器路由规则
func (c *CLI) handleRabbitTestRouting(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing exchange routing rules...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type routingTestCase struct {
    id             string
    description    string
    exchangeConfig struct {
      name string
      typ  string
    }
    queueName      string
    bindingKey     string
    routingKey     string
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []routingTestCase{
    {
      id:          "RB-ROUT-01",
      description: "Direct Exchange 精确匹配",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-direct",
        typ:  "direct",
      },
      queueName:      "queue1",
      bindingKey:     "key1",
      routingKey:     "key1",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-02",
      description: "不匹配路由",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-direct",
        typ:  "direct",
      },
      queueName:      "queue1",
      bindingKey:     "key1",
      routingKey:     "key2",
      expectedResult: false,
    },
    {
      id:          "RB-ROUT-03",
      description: "多队列匹配相同键",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-direct",
        typ:  "direct",
      },
      queueName:      "queue1",
      bindingKey:     "shared-key",
      routingKey:     "shared-key",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-04",
      description: "Topic Exchange 精确主题匹配",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-topic",
        typ:  "topic",
      },
      queueName:      "temp-queue",
      bindingKey:     "sensor.temp.room1",
      routingKey:     "sensor.temp.room1",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-05",
      description: "单层通配符匹配",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-topic",
        typ:  "topic",
      },
      queueName:      "temp-queue",
      bindingKey:     "sensor.temp.*",
      routingKey:     "sensor.temp.room2",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-06",
      description: "多层通配符匹配",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-topic",
        typ:  "topic",
      },
      queueName:      "all-sensors",
      bindingKey:     "sensor.#",
      routingKey:     "sensor.temp.room1.floor2",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-07",
      description: "多层通配符不匹配",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-topic",
        typ:  "topic",
      },
      queueName:      "other-sensors",
      bindingKey:     "other.#",
      routingKey:     "sensor.temp.room1",
      expectedResult: false,
    },
    {
      id:          "RB-ROUT-09",
      description: "广播到所有绑定队列",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-fanout",
        typ:  "fanout",
      },
      queueName:      "queue1",
      bindingKey:     "any-binding",
      routingKey:     "any-key",
      expectedResult: true,
    },
    {
      id:          "RB-ROUT-10",
      description: "忽略路由键",
      exchangeConfig: struct {
        name string
        typ  string
      }{
        name: "test-fanout",
        typ:  "fanout",
      },
      queueName:      "queue1",
      bindingKey:     "binding-key",
      routingKey:     "",
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Exchange: %s (%s)\n", tc.exchangeConfig.name, tc.exchangeConfig.typ)
    fmt.Printf("    Queue: %s\n", tc.queueName)
    fmt.Printf("    Binding Key: %s\n", tc.bindingKey)
    fmt.Printf("    Routing Key: %s\n", tc.routingKey)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 声明测试交换器和队列
    exchange := &rabbitmq.ExchangeConfig{
      Name:       tc.exchangeConfig.name,
      Type:       tc.exchangeConfig.typ,
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      client.Close()
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       tc.queueName,
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      client.Close()
      failCount++
      continue
    }

    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: tc.bindingKey,
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      client.Close()
      failCount++
      continue
    }

    // 对于多队列匹配的测试，额外绑定另一个队列
    var queue2Name string
    if tc.id == "RB-ROUT-03" {
      queue2Name = "queue2"
      queue2 := &rabbitmq.QueueConfig{
        Name:       queue2Name,
        Durable:    false,
        AutoDelete: true,
        Exclusive:  false,
        Args:       nil,
      }
      if err := client.DeclareQueue(queue2); err != nil {
        testError = fmt.Sprintf("failed to declare queue2: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
      bind2 := &rabbitmq.BindConfig{
        Queue:      queue2Name,
        Exchange:   exchange.Name,
        RoutingKey: "shared-key",
        Arguments:  nil,
      }
      if err := client.QueueBind(bind2); err != nil {
        testError = fmt.Sprintf("failed to bind queue2: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
    }

    // 发布测试消息
    publishConfig := &rabbitmq.PublishConfig{
      Exchange:   exchange.Name,
      RoutingKey: tc.routingKey,
      Message:    "Routing Test Message",
      Repeat:     1,
      Interval:   0,
      PrintLog:   false,
    }

    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish message: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      client.Close()
      failCount++
      continue
    }

    // 消费消息验证
    var mu sync.Mutex
    received := false

    consumeConfig := &rabbitmq.ConsumeConfig{
      Queue:    queue.Name,
      Consumer: fmt.Sprintf("test-routing-consumer-%s", tc.id),
      AutoAck:  true,
      Timeout:  5,
      Handler: func(msg amqp.Delivery) {
        mu.Lock()
        received = true
        mu.Unlock()
        fmt.Printf("    ✅ Received message: %s\n", msg.Body)
      },
    }

    if err := client.ConsumeMessage(consumeConfig); err != nil {
      // 超时错误是预期的，因为有些测试应该收不到消息
      if err.Error() != "timeout" {
        testError = fmt.Sprintf("failed to consume message: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        client.Close()
        failCount++
        continue
      }
    }

    // 对于多队列匹配的测试，检查第二个队列是否也收到消息
    if tc.id == "RB-ROUT-03" {
      var receivedQueue2 bool
      consumeConfig2 := &rabbitmq.ConsumeConfig{
        Queue:    queue2Name,
        Consumer: "test-routing-consumer-queue2",
        AutoAck:  true,
        Timeout:  5,
        Handler: func(msg amqp.Delivery) {
          receivedQueue2 = true
          fmt.Printf("    ✅ Queue2 received message: %s\n", msg.Body)
        },
      }
      if err := client.ConsumeMessage(consumeConfig2); err != nil && err.Error() != "timeout" {
        fmt.Printf("    ⚠️  Failed to consume from queue2: %v\n", err)
      }
      // 如果两个队列都收到消息，才认为测试通过
      mu.Lock()
      actualReceived := received && receivedQueue2
      mu.Unlock()
      if actualReceived == tc.expectedResult {
        fmt.Printf("    ✅ Test passed: Expected %v, got %v\n", tc.expectedResult, actualReceived)
        successCount++
      } else {
        testError = fmt.Sprintf("expected both queues to receive messages, got queue1: %v, queue2: %v", received, receivedQueue2)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        failCount++
      }
    } else {
      // 普通测试用例
      mu.Lock()
      actualReceived := received
      mu.Unlock()
      if actualReceived == tc.expectedResult {
        fmt.Printf("    ✅ Test passed: Expected %v, got %v\n", tc.expectedResult, actualReceived)
        successCount++
      } else {
        testError = fmt.Sprintf("expected %v, got %v", tc.expectedResult, actualReceived)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        failCount++
      }
    }

    // 关闭客户端
    client.Close()
  }

  // 打印测试结果
  fmt.Printf("\n=== Routing Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d routing tests failed", failCount)
  }
  return nil
}

// handleRabbitTestBinding 测试队列绑定
func (c *CLI) handleRabbitTestBinding(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing queue binding correctness...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type bindingTestCase struct {
    id             string
    description    string
    exchangeName   string
    queueName      string
    bindingKey     string
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []bindingTestCase{
    {
      id:             "RB-BIND-01",
      description:    "队列绑定到交换器",
      exchangeName:   "test-binding-exchange",
      queueName:      "test-binding-queue",
      bindingKey:     "test-binding-key",
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Exchange: %s\n", tc.exchangeName)
    fmt.Printf("    Queue: %s\n", tc.queueName)
    fmt.Printf("    Binding Key: %s\n", tc.bindingKey)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明测试交换器和队列
    exchange := &rabbitmq.ExchangeConfig{
      Name:       tc.exchangeName,
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       tc.queueName,
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 绑定队列
    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: tc.bindingKey,
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    fmt.Printf("    ✅ Test passed: Queue bound successfully\n")
    successCount++

    // 解绑队列
    if err := client.QueueUnBind(bind); err != nil {
      fmt.Printf("    ⚠️  Failed to unbind queue: %v\n", err)
    } else {
      fmt.Printf("    ✅ Queue unbound successfully\n")
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== Binding Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d binding tests failed", failCount)
  }
  return nil
}

// handleRabbitTestConsume 测试消息消费和确认
func (c *CLI) handleRabbitTestConsume(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing message consume and ack/nack...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type consumeTestCase struct {
    id             string
    description    string
    autoAck        bool
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []consumeTestCase{
    {
      id:             "RB-CONS-01",
      description:    "自动确认消费",
      autoAck:        true,
      expectedResult: true,
    },
    {
      id:             "RB-CONS-02",
      description:    "手动确认消费",
      autoAck:        false,
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    AutoAck: %v\n", tc.autoAck)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明测试交换器和队列
    exchange := &rabbitmq.ExchangeConfig{
      Name:       "test-consume-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       "test-consume-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: "test-consume-key",
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 发布测试消息
    publishConfig := &rabbitmq.PublishConfig{
      Exchange:   exchange.Name,
      RoutingKey: "test-consume-key",
      Message:    "Test consume message",
      Repeat:     1,
      Interval:   0,
      PrintLog:   false,
    }

    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish message: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 消费测试消息
    consumed := false
    consumeConfig := &rabbitmq.ConsumeConfig{
      Queue:    queue.Name,
      Consumer: "test-consumer",
      AutoAck:  tc.autoAck,
      Timeout:  5,
      Handler: func(msg amqp.Delivery) {
        fmt.Printf("    ✅ Received message: %s\n", msg.Body)
        consumed = true
        if !tc.autoAck {
          if err := msg.Ack(false); err != nil {
            fmt.Printf("    ❌ Failed to acknowledge message: %v\n", err)
          }
        }
      },
    }

    if err := client.ConsumeMessage(consumeConfig); err != nil {
      testError = fmt.Sprintf("failed to consume message: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    if consumed == tc.expectedResult {
      fmt.Printf("    ✅ Test passed: Expected %v, got %v\n", tc.expectedResult, consumed)
      successCount++
    } else {
      testError = fmt.Sprintf("expected %v, got %v", tc.expectedResult, consumed)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== Consume Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d consume tests failed", failCount)
  }
  return nil
}

// handleRabbitTestDLQ 测试死信队列
func (c *CLI) handleRabbitTestDLQ(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing dead letter queue mechanism...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type dlqTestCase struct {
    id             string
    description    string
    triggerType    string
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []dlqTestCase{
    {
      id:             "RB-DLQ-01",
      description:    "消息被拒绝进入DLQ",
      triggerType:    "reject",
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Trigger: %s\n", tc.triggerType)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明死信交换器和队列
    dlxExchange := &rabbitmq.ExchangeConfig{
      Name:       "test-dlx-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(dlxExchange); err != nil {
      testError = fmt.Sprintf("failed to declare DLX exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    dlqQueue := &rabbitmq.QueueConfig{
      Name:       "test-dlq-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(dlqQueue); err != nil {
      testError = fmt.Sprintf("failed to declare DLQ queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    dlqBind := &rabbitmq.BindConfig{
      Queue:      dlqQueue.Name,
      Exchange:   dlxExchange.Name,
      RoutingKey: "#",
      Arguments:  nil,
    }

    if err := client.QueueBind(dlqBind); err != nil {
      testError = fmt.Sprintf("failed to bind DLQ queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 声明主交换器和队列，配置死信交换器
    mainExchange := &rabbitmq.ExchangeConfig{
      Name:       "test-main-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(mainExchange); err != nil {
      testError = fmt.Sprintf("failed to declare main exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    mainQueue := &rabbitmq.QueueConfig{
      Name:       "test-main-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args: amqp.Table{
        "x-dead-letter-exchange": dlxExchange.Name,
      },
    }

    if err := client.DeclareQueue(mainQueue); err != nil {
      testError = fmt.Sprintf("failed to declare main queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    mainBind := &rabbitmq.BindConfig{
      Queue:      mainQueue.Name,
      Exchange:   mainExchange.Name,
      RoutingKey: "test-dlq-key",
      Arguments:  nil,
    }

    if err := client.QueueBind(mainBind); err != nil {
      testError = fmt.Sprintf("failed to bind main queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 发布测试消息
    publishConfig := &rabbitmq.PublishConfig{
      Exchange:   mainExchange.Name,
      RoutingKey: "test-dlq-key",
      Message:    "Test DLQ message",
      Repeat:     1,
      Interval:   0,
      PrintLog:   false,
    }

    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish message: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 消费消息并拒绝，使其进入死信队列
    consumeConfig := &rabbitmq.ConsumeConfig{
      Queue:    mainQueue.Name,
      Consumer: "test-dlq-consumer",
      AutoAck:  false,
      Timeout:  5,
      Handler: func(msg amqp.Delivery) {
        fmt.Printf("    ✅ Received message on main queue: %s\n", msg.Body)
        // 拒绝消息，使其进入死信队列
        if err := msg.Nack(false, false); err != nil {
          fmt.Printf("    ❌ Failed to reject message: %v\n", err)
        }
      },
    }

    if err := client.ConsumeMessage(consumeConfig); err != nil {
      testError = fmt.Sprintf("failed to consume message from main queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 检查死信队列中是否有消息
    dlqConsumed := false
    dlqConsumeConfig := &rabbitmq.ConsumeConfig{
      Queue:    dlqQueue.Name,
      Consumer: "test-dlq-dlq-consumer",
      AutoAck:  true,
      Timeout:  5,
      Handler: func(msg amqp.Delivery) {
        fmt.Printf("    ✅ Received message on DLQ: %s\n", msg.Body)
        dlqConsumed = true
      },
    }

    if err := client.ConsumeMessage(dlqConsumeConfig); err != nil {
      testError = fmt.Sprintf("failed to consume message from DLQ: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    if dlqConsumed == tc.expectedResult {
      fmt.Printf("    ✅ Test passed: Expected %v, got %v\n", tc.expectedResult, dlqConsumed)
      successCount++
    } else {
      testError = fmt.Sprintf("expected %v, got %v", tc.expectedResult, dlqConsumed)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== DLQ Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d DLQ tests failed", failCount)
  }
  return nil
}

// handleRabbitTestTTL 测试消息TTL
func (c *CLI) handleRabbitTestTTL(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing message TTL...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type ttlTestCase struct {
    id             string
    description    string
    ttl            int
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []ttlTestCase{
    {
      id:             "RB-TTL-01",
      description:    "消息TTL过期",
      ttl:            1000,
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    TTL: %d ms\n", tc.ttl)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明测试交换器和队列，配置TTL
    exchange := &rabbitmq.ExchangeConfig{
      Name:       "test-ttl-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       "test-ttl-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args: amqp.Table{
        "x-message-ttl": tc.ttl,
      },
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: "test-ttl-key",
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 发布测试消息
    publishConfig := &rabbitmq.PublishConfig{
      Exchange:   exchange.Name,
      RoutingKey: "test-ttl-key",
      Message:    "Test TTL message",
      Repeat:     1,
      Interval:   0,
      PrintLog:   false,
    }

    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish message: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 立即消费，应该能收到消息
    fmt.Println("    ✅ Checking message before TTL...")
    consumedBefore := false
    consumeConfig := &rabbitmq.ConsumeConfig{
      Queue:    queue.Name,
      Consumer: "test-ttl-consumer-before",
      AutoAck:  true,
      Timeout:  2,
      Handler: func(msg amqp.Delivery) {
        fmt.Printf("        ✅ Received message: %s\n", msg.Body)
        consumedBefore = true
      },
    }

    if err := client.ConsumeMessage(consumeConfig); err != nil {
      testError = fmt.Sprintf("failed to consume message before TTL: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 再次发布消息，等待TTL过期
    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish message for TTL test: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 等待TTL过期
    fmt.Printf("    ✅ Waiting for TTL (%d ms) to expire...\n", tc.ttl)
    time.Sleep(time.Duration(tc.ttl+500) * time.Millisecond)

    // 再次消费，应该收不到消息
    fmt.Println("    ✅ Checking message after TTL...")
    consumedAfter := false
    consumeConfigAfter := &rabbitmq.ConsumeConfig{
      Queue:    queue.Name,
      Consumer: "test-ttl-consumer-after",
      AutoAck:  true,
      Timeout:  2,
      Handler: func(msg amqp.Delivery) {
        fmt.Printf("        ❌ Unexpectedly received message after TTL: %s\n", msg.Body)
        consumedAfter = true
      },
    }

    if err := client.ConsumeMessage(consumeConfigAfter); err != nil {
      // 超时是预期的，因为消息应该已经过期
      if err.Error() != "timeout" {
        testError = fmt.Sprintf("failed to consume message after TTL: %v", err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        failCount++
        continue
      }
    }

    if consumedBefore && !consumedAfter {
      fmt.Printf("    ✅ Test passed: Message expired as expected\n")
      successCount++
    } else {
      testError = fmt.Sprintf("expected message to expire after TTL, got consumedBefore: %v, consumedAfter: %v", consumedBefore, consumedAfter)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== TTL Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d TTL tests failed", failCount)
  }
  return nil
}

// handleRabbitTestConcurrency 测试消费者并发
func (c *CLI) handleRabbitTestConcurrency(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing consumer concurrency...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type concurrencyTestCase struct {
    id             string
    description    string
    consumerCount  int
    messageCount   int
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []concurrencyTestCase{
    {
      id:             "RB-CONCUR-01",
      description:    "多消费者并发消费",
      consumerCount:  2,
      messageCount:   4,
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Consumer Count: %d\n", tc.consumerCount)
    fmt.Printf("    Message Count: %d\n", tc.messageCount)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明测试交换器和队列
    exchange := &rabbitmq.ExchangeConfig{
      Name:       "test-concurrency-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       "test-concurrency-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: "test-concurrency-key",
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 启动多个消费者
    var mu sync.Mutex
    receivedCount := 0
    doneCh := make(chan struct{})

    for i := 0; i < tc.consumerCount; i++ {
      go func(consumerID int) {
        consumerConfig := &rabbitmq.ConsumeConfig{
          Queue:    queue.Name,
          Consumer: fmt.Sprintf("test-concurrency-consumer-%d", consumerID),
          AutoAck:  true,
          Timeout:  10,
          Handler: func(msg amqp.Delivery) {
            mu.Lock()
            receivedCount++
            mu.Unlock()
            fmt.Printf("    ✅ Consumer %d received message: %s\n", consumerID, msg.Body)
          },
        }

        if err := client.ConsumeMessage(consumerConfig); err != nil {
          // 超时错误是预期的
          if err.Error() != "timeout" {
            fmt.Printf("    ⚠️  Consumer %d failed: %v\n", consumerID, err)
          }
        }

        doneCh <- struct{}{}
      }(i)
    }

    // 发布多个测试消息
    publishConfig := &rabbitmq.PublishConfig{
      Exchange:   exchange.Name,
      RoutingKey: "test-concurrency-key",
      Message:    "Test concurrency message",
      Repeat:     tc.messageCount,
      Interval:   0,
      PrintLog:   false,
    }

    if err := client.PublishMessage(publishConfig); err != nil {
      testError = fmt.Sprintf("failed to publish messages: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 等待所有消费者完成
    for i := 0; i < tc.consumerCount; i++ {
      <-doneCh
    }

    if receivedCount == tc.messageCount {
      fmt.Printf("    ✅ Test passed: All %d messages were consumed\n", receivedCount)
      successCount++
    } else {
      testError = fmt.Sprintf("expected %d messages to be consumed, got %d", tc.messageCount, receivedCount)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== Concurrency Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d concurrency tests failed", failCount)
  }
  return nil
}

// handleRabbitTestConfirms 测试发布者确认
func (c *CLI) handleRabbitTestConfirms(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing publisher confirms...")

  // 测试结构：每个测试用例包含ID、描述、配置和预期结果
  type confirmsTestCase struct {
    id             string
    description    string
    messageCount   int
    expectedResult bool
  }

  // 运行所有测试用例，记录成功和失败数量
  successCount := 0
  failCount := 0

  // 定义测试用例
  testCases := []confirmsTestCase{
    {
      id:             "RB-CONF-01",
      description:    "发布者确认测试",
      messageCount:   5,
      expectedResult: true,
    },
  }

  // 运行测试用例
  for _, tc := range testCases {
    fmt.Printf("\n=== %s: %s ===\n", tc.id, tc.description)
    fmt.Printf("    Message Count: %d\n", tc.messageCount)
    fmt.Printf("    Expected: %v\n", tc.expectedResult)

    // 记录测试结果
    testError := ""

    // 创建并连接客户端
    client := &rabbitmq.RabbitClient{Config: config}
    err := client.Connect()
    if err != nil {
      testError = fmt.Sprintf("failed to connect to RabbitMQ server: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }
    defer client.Close()

    // 声明测试交换器和队列
    exchange := &rabbitmq.ExchangeConfig{
      Name:       "test-confirms-exchange",
      Type:       "direct",
      Durable:    false,
      AutoDelete: true,
    }

    if err := client.DeclareExchange(exchange); err != nil {
      testError = fmt.Sprintf("failed to declare exchange: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    queue := &rabbitmq.QueueConfig{
      Name:       "test-confirms-queue",
      Durable:    false,
      AutoDelete: true,
      Exclusive:  false,
      Args:       nil,
    }

    if err := client.DeclareQueue(queue); err != nil {
      testError = fmt.Sprintf("failed to declare queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    bind := &rabbitmq.BindConfig{
      Queue:      queue.Name,
      Exchange:   exchange.Name,
      RoutingKey: "test-confirms-key",
      Arguments:  nil,
    }

    if err := client.QueueBind(bind); err != nil {
      testError = fmt.Sprintf("failed to bind queue: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 启用发布者确认
    if err := client.Channel().Confirm(false); err != nil {
      testError = fmt.Sprintf("failed to enable publisher confirms: %v", err)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
      continue
    }

    // 监听发布者确认
    confirms := make(chan amqp.Confirmation, tc.messageCount)
    client.Channel().NotifyPublish(confirms)

    // 发布测试消息
    for i := 0; i < tc.messageCount; i++ {
      err := client.Channel().Publish(
        exchange.Name,
        "test-confirms-key",
        false,
        false,
        amqp.Publishing{
          Body: []byte(fmt.Sprintf("Test confirms message %d", i)),
        },
      )

      if err != nil {
        testError = fmt.Sprintf("failed to publish message %d: %v", i, err)
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        failCount++
        continue
      }
    }

    // 等待确认
    confirmCount := 0
    timeout := time.After(5 * time.Second)

    for i := 0; i < tc.messageCount; i++ {
      select {
      case confirm := <-confirms:
        if confirm.Ack {
          confirmCount++
          fmt.Printf("    ✅ Message confirmed: %d\n", i+1)
        } else {
          fmt.Printf("    ❌ Message not confirmed: %d\n", i+1)
        }
      case <-timeout:
        testError = "timeout waiting for publisher confirms"
        fmt.Printf("    ❌ Test failed: %s\n", testError)
        goto endConfirmLoop
      }
    }

  endConfirmLoop:
    if confirmCount == tc.messageCount {
      fmt.Printf("    ✅ Test passed: All %d messages confirmed\n", confirmCount)
      successCount++
    } else {
      testError = fmt.Sprintf("expected %d confirms, got %d", tc.messageCount, confirmCount)
      fmt.Printf("    ❌ Test failed: %s\n", testError)
      failCount++
    }
  }

  // 打印测试结果
  fmt.Printf("\n=== Publisher Confirms Test Results ===\n")
  fmt.Printf("✅ Passed: %d\n", successCount)
  fmt.Printf("❌ Failed: %d\n", failCount)
  fmt.Printf("📊 Total: %d\n", successCount+failCount)

  if failCount > 0 {
    return fmt.Errorf("%d publisher confirms tests failed", failCount)
  }
  return nil
}

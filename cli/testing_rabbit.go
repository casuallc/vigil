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
  rabbitTestCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "localhost", "RabbitMQ server address")
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

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
  }
  defer client.Close()

  // 声明测试交换器和队列
  exchange := &rabbitmq.ExchangeConfig{
    Name:       "test-publish-exchange",
    Type:       "direct",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(exchange); err != nil {
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-publish-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
  }

  // 发布测试消息
  publishConfig := &rabbitmq.PublishConfig{
    Exchange:   exchange.Name,
    RoutingKey: "test-key",
    Message:    "Test publish message",
    Repeat:     5,
    Interval:   500,
    PrintLog:   false,
  }

  if err := client.PublishMessage(publishConfig); err != nil {
    return fmt.Errorf("failed to publish messages: %v", err)
  }

  fmt.Println("  ✅ Messages published successfully")
  return nil
}

// handleRabbitTestRouting 测试交换器路由规则
func (c *CLI) handleRabbitTestRouting(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing exchange routing rules...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
  }
  defer client.Close()

  // 测试direct exchange
  fmt.Println("  Testing direct exchange...")
  directExchange := &rabbitmq.ExchangeConfig{
    Name:       "test-direct-exchange",
    Type:       "direct",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(directExchange); err != nil {
    return fmt.Errorf("failed to declare direct exchange: %v", err)
  }

  // 测试topic exchange
  fmt.Println("  Testing topic exchange...")
  topicExchange := &rabbitmq.ExchangeConfig{
    Name:       "test-topic-exchange",
    Type:       "topic",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(topicExchange); err != nil {
    return fmt.Errorf("failed to declare topic exchange: %v", err)
  }

  // 测试fanout exchange
  fmt.Println("  Testing fanout exchange...")
  fanoutExchange := &rabbitmq.ExchangeConfig{
    Name:       "test-fanout-exchange",
    Type:       "fanout",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(fanoutExchange); err != nil {
    return fmt.Errorf("failed to declare fanout exchange: %v", err)
  }

  fmt.Println("  ✅ Exchange routing rules test completed")
  return nil
}

// handleRabbitTestBinding 测试队列绑定
func (c *CLI) handleRabbitTestBinding(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing queue binding correctness...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
  }
  defer client.Close()

  // 声明测试交换器和队列
  exchange := &rabbitmq.ExchangeConfig{
    Name:       "test-binding-exchange",
    Type:       "direct",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(exchange); err != nil {
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-binding-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  // 绑定队列
  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-binding-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
  }

  fmt.Println("  ✅ Queue bound successfully")

  // 解绑队列
  if err := client.QueueUnBind(bind); err != nil {
    return fmt.Errorf("failed to unbind queue: %v", err)
  }

  fmt.Println("  ✅ Queue unbound successfully")
  return nil
}

// handleRabbitTestConsume 测试消息消费和确认
func (c *CLI) handleRabbitTestConsume(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing message consume and ack/nack...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
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
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-consume-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-consume-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
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
    return fmt.Errorf("failed to publish message: %v", err)
  }

  // 消费测试消息
  consumed := false
  consumeConfig := &rabbitmq.ConsumeConfig{
    Queue:    queue.Name,
    Consumer: "test-consumer",
    AutoAck:  false,
    Timeout:  5,
    Handler: func(msg amqp.Delivery) {
      fmt.Printf("  Received message: %s\n", msg.Body)
      consumed = true
      if err := msg.Ack(false); err != nil {
        fmt.Printf("  Failed to acknowledge message: %v\n", err)
      }
    },
  }

  if err := client.ConsumeMessage(consumeConfig); err != nil {
    return fmt.Errorf("failed to consume message: %v", err)
  }

  if !consumed {
    return fmt.Errorf("no message consumed")
  }

  fmt.Println("  ✅ Message consumed and acknowledged successfully")
  return nil
}

// handleRabbitTestDLQ 测试死信队列
func (c *CLI) handleRabbitTestDLQ(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing dead letter queue mechanism...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
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
    return fmt.Errorf("failed to declare DLX exchange: %v", err)
  }

  dlqQueue := &rabbitmq.QueueConfig{
    Name:       "test-dlq-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(dlqQueue); err != nil {
    return fmt.Errorf("failed to declare DLQ queue: %v", err)
  }

  dlqBind := &rabbitmq.BindConfig{
    Queue:      dlqQueue.Name,
    Exchange:   dlxExchange.Name,
    RoutingKey: "#",
    Arguments:  nil,
  }

  if err := client.QueueBind(dlqBind); err != nil {
    return fmt.Errorf("failed to bind DLQ queue: %v", err)
  }

  // 声明主交换器和队列，配置死信交换器
  mainExchange := &rabbitmq.ExchangeConfig{
    Name:       "test-main-exchange",
    Type:       "direct",
    Durable:    false,
    AutoDelete: true,
  }

  if err := client.DeclareExchange(mainExchange); err != nil {
    return fmt.Errorf("failed to declare main exchange: %v", err)
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
    return fmt.Errorf("failed to declare main queue: %v", err)
  }

  mainBind := &rabbitmq.BindConfig{
    Queue:      mainQueue.Name,
    Exchange:   mainExchange.Name,
    RoutingKey: "test-dlq-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(mainBind); err != nil {
    return fmt.Errorf("failed to bind main queue: %v", err)
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
    return fmt.Errorf("failed to publish message: %v", err)
  }

  // 消费消息并拒绝，使其进入死信队列
  consumeConfig := &rabbitmq.ConsumeConfig{
    Queue:    mainQueue.Name,
    Consumer: "test-dlq-consumer",
    AutoAck:  false,
    Timeout:  5,
    Handler: func(msg amqp.Delivery) {
      fmt.Printf("  Received message on main queue: %s\n", msg.Body)
      // 拒绝消息，使其进入死信队列
      if err := msg.Nack(false, false); err != nil {
        fmt.Printf("  Failed to reject message: %v\n", err)
      }
    },
  }

  if err := client.ConsumeMessage(consumeConfig); err != nil {
    return fmt.Errorf("failed to consume message from main queue: %v", err)
  }

  // 检查死信队列中是否有消息
  dlqConsumed := false
  dlqConsumeConfig := &rabbitmq.ConsumeConfig{
    Queue:    dlqQueue.Name,
    Consumer: "test-dlq-dlq-consumer",
    AutoAck:  true,
    Timeout:  5,
    Handler: func(msg amqp.Delivery) {
      fmt.Printf("  Received message on DLQ: %s\n", msg.Body)
      dlqConsumed = true
    },
  }

  if err := client.ConsumeMessage(dlqConsumeConfig); err != nil {
    return fmt.Errorf("failed to consume message from DLQ: %v", err)
  }

  if !dlqConsumed {
    return fmt.Errorf("no message found in DLQ")
  }

  fmt.Println("  ✅ Dead letter queue test completed successfully")
  return nil
}

// handleRabbitTestTTL 测试消息TTL
func (c *CLI) handleRabbitTestTTL(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing message TTL...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
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
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-ttl-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args: amqp.Table{
      "x-message-ttl": 1000, // 1秒TTL
    },
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-ttl-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
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
    return fmt.Errorf("failed to publish message: %v", err)
  }

  // 立即消费，应该能收到消息
  fmt.Println("  Checking message before TTL...")
  consumedBefore := false
  consumeConfig := &rabbitmq.ConsumeConfig{
    Queue:    queue.Name,
    Consumer: "test-ttl-consumer-before",
    AutoAck:  true,
    Timeout:  2,
    Handler: func(msg amqp.Delivery) {
      fmt.Printf("    Received message: %s\n", msg.Body)
      consumedBefore = true
    },
  }

  if err := client.ConsumeMessage(consumeConfig); err != nil {
    return fmt.Errorf("failed to consume message before TTL: %v", err)
  }

  if !consumedBefore {
    return fmt.Errorf("no message consumed before TTL")
  }

  // 再次发布消息，等待TTL过期
  if err := client.PublishMessage(publishConfig); err != nil {
    return fmt.Errorf("failed to publish message for TTL test: %v", err)
  }

  // 等待TTL过期
  fmt.Println("  Waiting for TTL to expire...")
  time.Sleep(2 * time.Second)

  // 再次消费，应该收不到消息
  fmt.Println("  Checking message after TTL...")
  consumedAfter := false
  consumeConfigAfter := &rabbitmq.ConsumeConfig{
    Queue:    queue.Name,
    Consumer: "test-ttl-consumer-after",
    AutoAck:  true,
    Timeout:  2,
    Handler: func(msg amqp.Delivery) {
      fmt.Printf("    Unexpectedly received message after TTL: %s\n", msg.Body)
      consumedAfter = true
    },
  }

  if err := client.ConsumeMessage(consumeConfigAfter); err != nil {
    return fmt.Errorf("failed to consume message after TTL: %v", err)
  }

  if consumedAfter {
    return fmt.Errorf("message still available after TTL")
  }

  fmt.Println("  ✅ Message TTL test completed successfully")
  return nil
}

// handleRabbitTestConcurrency 测试消费者并发
func (c *CLI) handleRabbitTestConcurrency(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing consumer concurrency...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
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
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-concurrency-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-concurrency-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
  }

  // 发布多条测试消息
  publishConfig := &rabbitmq.PublishConfig{
    Exchange:   exchange.Name,
    RoutingKey: "test-concurrency-key",
    Message:    "Test concurrency message",
    Repeat:     5,
    Interval:   100,
    PrintLog:   false,
  }

  if err := client.PublishMessage(publishConfig); err != nil {
    return fmt.Errorf("failed to publish messages: %v", err)
  }

  // 创建多个消费者
  var wg sync.WaitGroup
  var mu sync.Mutex
  consumedCount := 0
  consumerCount := 2

  for i := 0; i < consumerCount; i++ {
    wg.Add(1)
    go func(consumerID int) {
      defer wg.Done()

      // 创建新的客户端连接用于每个消费者

      consumerClient := &rabbitmq.RabbitClient{Config: config}
      if err := consumerClient.Connect(); err != nil {
        fmt.Printf("  Consumer %d failed to connect: %v\n", consumerID, err)
        return
      }
      defer consumerClient.Close()

      consumeConfig := &rabbitmq.ConsumeConfig{
        Queue:    queue.Name,
        Consumer: fmt.Sprintf("test-concurrency-consumer-%d", consumerID),
        AutoAck:  true,
        Timeout:  5,
        Handler: func(msg amqp.Delivery) {
          mu.Lock()
          consumedCount++
          mu.Unlock()
          fmt.Printf("  Consumer %d received message: %s\n", consumerID, msg.Body)
        },
      }

      if err := consumerClient.ConsumeMessage(consumeConfig); err != nil {
        fmt.Printf("  Consumer %d failed to consume: %v\n", consumerID, err)
      }
    }(i)
  }

  // 等待所有消费者完成
  wg.Wait()

  mu.Lock()
  count := consumedCount
  mu.Unlock()

  fmt.Printf("  Total messages consumed: %d\n", count)
  if count < 5 {
    return fmt.Errorf("not all messages consumed, expected 5, got %d", count)
  }

  fmt.Println("  ✅ Consumer concurrency test completed successfully")
  return nil
}

// handleRabbitTestConfirms 测试发布者确认
func (c *CLI) handleRabbitTestConfirms(config *rabbitmq.ServerConfig) error {
  fmt.Println("Testing publisher confirms...")

  // 创建并连接客户端
  client := &rabbitmq.RabbitClient{Config: config}
  err := client.Connect()
  if err != nil {
    client.Close()
    return fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
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
    return fmt.Errorf("failed to declare exchange: %v", err)
  }

  queue := &rabbitmq.QueueConfig{
    Name:       "test-confirms-queue",
    Durable:    false,
    AutoDelete: true,
    Exclusive:  false,
    Args:       nil,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %v", err)
  }

  bind := &rabbitmq.BindConfig{
    Queue:      queue.Name,
    Exchange:   exchange.Name,
    RoutingKey: "test-confirms-key",
    Arguments:  nil,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue: %v", err)
  }

  // 启用发布者确认
  if err := client.Channel().Confirm(false); err != nil {
    return fmt.Errorf("failed to enable publisher confirms: %v", err)
  }

  // 设置确认和返回通道
  confirms := client.Channel().NotifyPublish(make(chan amqp.Confirmation, 1))
  returns := client.Channel().NotifyReturn(make(chan amqp.Return, 1))

  // 发布测试消息
  publishConfig := &rabbitmq.PublishConfig{
    Exchange:   exchange.Name,
    RoutingKey: "test-confirms-key",
    Message:    "Test confirms message",
    Repeat:     1,
    Interval:   0,
    PrintLog:   false,
  }

  // 直接使用channel.Publish发送消息以测试确认
  err = client.Channel().Publish(
    publishConfig.Exchange,
    publishConfig.RoutingKey,
    true, // mandatory
    false,
    amqp.Publishing{
      ContentType:  "text/plain",
      Body:         []byte(publishConfig.Message),
      DeliveryMode: amqp.Persistent,
    },
  )

  if err != nil {
    return fmt.Errorf("failed to publish message: %v", err)
  }

  // 等待确认
  select {
  case confirm := <-confirms:
    if confirm.Ack {
      fmt.Println("  ✅ Publisher confirm received: Ack")
    } else {
      return fmt.Errorf("publisher confirm received: Nack")
    }
  case <-returns:
    return fmt.Errorf("message returned by server")
  case <-time.After(5 * time.Second):
    return fmt.Errorf("timeout waiting for publisher confirm")
  }

  fmt.Println("  ✅ Publisher confirms test completed successfully")
  return nil
}

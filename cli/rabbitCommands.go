package cli

import (
  "context"
  "fmt"
  "github.com/casuallc/vigil/rabbitmq"
  "github.com/spf13/cobra"
  "strings"
)

// setupRabbitCommands 设置RabbitMQ相关命令
func (c *CLI) setupRabbitCommands() *cobra.Command {
  rabbitCmd := &cobra.Command{
    Use:   "rabbitmq",
    Short: "RabbitMQ related commands",
    Long:  `Perform RabbitMQ operations like declaring exchanges/queues, publishing messages, etc.`,
  }

  // 为父命令添加持久化标志
  var config rabbitmq.ServerConfig
  rabbitCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "localhost", "RabbitMQ server host")
  rabbitCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 5672, "RabbitMQ server port")
  rabbitCmd.PersistentFlags().StringVarP(&config.Vhost, "vhost", "v", "/", "RabbitMQ virtual host")
  rabbitCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "guest", "RabbitMQ username")
  rabbitCmd.PersistentFlags().StringVarP(&config.Password, "password", "P", "guest", "RabbitMQ password")

  // 存储配置到上下文
  rabbitCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
    cmd.SetContext(context.WithValue(cmd.Context(), "rabbitConfig", &config))
  }

  // 添加子命令
  rabbitCmd.AddCommand(c.setupRabbitDeclareExchangeCommand())
  rabbitCmd.AddCommand(c.setupRabbitDeleteExchangeCommand())
  rabbitCmd.AddCommand(c.setupRabbitDeclareQueueCommand())
  rabbitCmd.AddCommand(c.setupRabbitDeleteQueueCommand())
  rabbitCmd.AddCommand(c.setupRabbitQueueBindCommand())
  rabbitCmd.AddCommand(c.setupRabbitQueueUnbindCommand())
  rabbitCmd.AddCommand(c.setupRabbitPublishCommand())
  rabbitCmd.AddCommand(c.setupRabbitConsumeCommand())

  return rabbitCmd
}

// setupRabbitDeclareExchangeCommand 设置声明交换机命令
func (c *CLI) setupRabbitDeclareExchangeCommand() *cobra.Command {
  var exchangeName, exchangeType string
  var durable, autoDelete bool

  cmd := &cobra.Command{
    Use:   "declare-exchange",
    Short: "Declare a RabbitMQ exchange",
    RunE: func(cmd *cobra.Command, args []string) error {
      config := cmd.Context().Value("rabbitConfig").(*rabbitmq.ServerConfig)
      return c.handleRabbitDeclareExchange(exchangeName, exchangeType, durable, autoDelete, config)
    },
  }

  cmd.Flags().StringVarP(&exchangeName, "name", "n", "", "Exchange name")
  cmd.Flags().StringVarP(&exchangeType, "type", "t", "direct", "Exchange type")
  cmd.Flags().BoolVarP(&durable, "durable", "d", false, "Exchange will survive broker restart")
  cmd.Flags().BoolVarP(&autoDelete, "auto-delete", "a", false, "Exchange will be deleted")
  cmd.MarkFlagRequired("name")

  return cmd
}

// 修改handleRabbitDeclareExchange函数签名
func (c *CLI) handleRabbitDeclareExchange(name, typ string, durable, autoDelete bool, config *rabbitmq.ServerConfig) error {
  client := &rabbitmq.RabbitClient{
    Config: config,
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  exchange := &rabbitmq.ExchangeConfig{
    Name:       name,
    Type:       typ,
    Durable:    durable,
    AutoDelete: autoDelete,
  }

  if err := client.DeclareExchange(exchange); err != nil {
    return fmt.Errorf("failed to declare exchange: %w", err)
  }

  fmt.Printf("Exchange '%s' of type '%s' declared successfully\n", name, typ)
  return nil
}

// setupRabbitDeleteExchangeCommand 设置删除交换机命令
func (c *CLI) setupRabbitDeleteExchangeCommand() *cobra.Command {
  var exchangeName string
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "delete-exchange",
    Short: "Delete a RabbitMQ exchange",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitDeleteExchange(exchangeName, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&exchangeName, "name", "n", "", "Exchange name")
  cmd.MarkFlagRequired("name")

  return cmd
}

// handleRabbitDeleteExchange 处理删除交换机命令
func (c *CLI) handleRabbitDeleteExchange(name string, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  if err := client.DeleteExchange(name); err != nil {
    return fmt.Errorf("failed to delete exchange: %w", err)
  }

  fmt.Printf("Exchange '%s' deleted successfully\n", name)
  return nil
}

// setupRabbitDeclareQueueCommand 设置声明队列命令
func (c *CLI) setupRabbitDeclareQueueCommand() *cobra.Command {
  var queueName string
  var durable, autoDelete, exclusive bool
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "declare-queue",
    Short: "Declare a RabbitMQ queue",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitDeclareQueue(queueName, durable, autoDelete, exclusive, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&queueName, "name", "n", "", "Queue name")
  cmd.Flags().BoolVarP(&durable, "durable", "d", false, "Queue will survive broker restart")
  cmd.Flags().BoolVarP(&autoDelete, "auto-delete", "a", false, "Queue will be deleted when last consumer unsubscribes")
  cmd.Flags().BoolVarP(&exclusive, "exclusive", "e", false, "Queue can only be accessed by the current connection")
  cmd.MarkFlagRequired("name")

  return cmd
}

// handleRabbitDeclareQueue 处理声明队列命令
func (c *CLI) handleRabbitDeclareQueue(name string, durable, autoDelete, exclusive bool, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  queue := &rabbitmq.QueueConfig{
    Name:       name,
    Durable:    durable,
    AutoDelete: autoDelete,
    Exclusive:  exclusive,
  }

  if err := client.DeclareQueue(queue); err != nil {
    return fmt.Errorf("failed to declare queue: %w", err)
  }

  fmt.Printf("Queue '%s' declared successfully\n", name)
  return nil
}

// setupRabbitDeleteQueueCommand 设置删除队列命令
func (c *CLI) setupRabbitDeleteQueueCommand() *cobra.Command {
  var queueName string
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "delete-queue",
    Short: "Delete a RabbitMQ queue",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitDeleteQueue(queueName, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&queueName, "name", "n", "", "Queue name")
  cmd.MarkFlagRequired("name")

  return cmd
}

// handleRabbitDeleteQueue 处理删除队列命令
func (c *CLI) handleRabbitDeleteQueue(name string, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  if err := client.DeleteQueue(name); err != nil {
    return fmt.Errorf("failed to delete queue: %w", err)
  }

  fmt.Printf("Queue '%s' deleted successfully\n", name)
  return nil
}

// setupRabbitQueueBindCommand 设置队列绑定命令
func (c *CLI) setupRabbitQueueBindCommand() *cobra.Command {
  var queueName, exchangeName, routingKey string
  var args string
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "queue-bind",
    Short: "Bind a queue to an exchange",
    RunE: func(cmd *cobra.Command, args []string) error {
      var bindArgs map[string]interface{}
      // 解析绑定参数
      if args[0] != "" {
        bindArgs = parseBindArgs(args[0])
      }
      return c.handleRabbitQueueBind(queueName, exchangeName, routingKey, bindArgs, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue name")
  cmd.Flags().StringVarP(&exchangeName, "exchange", "e", "", "Exchange name")
  cmd.Flags().StringVarP(&routingKey, "routing-key", "r", "", "Routing key")
  cmd.Flags().StringVarP(&args, "args", "a", "", "Additional binding arguments in format key1=value1,key2=value2")
  cmd.MarkFlagRequired("queue")
  cmd.MarkFlagRequired("exchange")

  return cmd
}

// handleRabbitQueueBind 处理队列绑定命令
func (c *CLI) handleRabbitQueueBind(queue, exchange, routingKey string, args map[string]interface{}, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  bind := &rabbitmq.BindConfig{
    Queue:      queue,
    Exchange:   exchange,
    RoutingKey: routingKey,
    Arguments:  args,
  }

  if err := client.QueueBind(bind); err != nil {
    return fmt.Errorf("failed to bind queue to exchange: %w", err)
  }

  fmt.Printf("Queue '%s' bound to exchange '%s' with routing key '%s'\n", queue, exchange, routingKey)
  return nil
}

// setupRabbitQueueUnbindCommand 设置队列解绑命令
func (c *CLI) setupRabbitQueueUnbindCommand() *cobra.Command {
  var queueName, exchangeName, routingKey string
  var args string
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "queue-unbind",
    Short: "Unbind a queue from an exchange",
    RunE: func(cmd *cobra.Command, args []string) error {
      var bindArgs map[string]interface{}
      // 解析绑定参数
      if args[0] != "" {
        bindArgs = parseBindArgs(args[0])
      }
      return c.handleRabbitQueueUnbind(queueName, exchangeName, routingKey, bindArgs, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue name")
  cmd.Flags().StringVarP(&exchangeName, "exchange", "e", "", "Exchange name")
  cmd.Flags().StringVarP(&routingKey, "routing-key", "r", "", "Routing key")
  cmd.Flags().StringVarP(&args, "args", "a", "", "Binding arguments in format key1=value1,key2=value2")
  cmd.MarkFlagRequired("queue")
  cmd.MarkFlagRequired("exchange")

  return cmd
}

// handleRabbitQueueUnbind 处理队列解绑命令
func (c *CLI) handleRabbitQueueUnbind(queue, exchange, routingKey string, args map[string]interface{}, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  bind := &rabbitmq.BindConfig{
    Queue:      queue,
    Exchange:   exchange,
    RoutingKey: routingKey,
    Arguments:  args,
  }

  if err := client.QueueUnBind(bind); err != nil {
    return fmt.Errorf("failed to unbind queue from exchange: %w", err)
  }

  fmt.Printf("Queue '%s' unbound from exchange '%s'\n", queue, exchange)
  return nil
}

// setupRabbitPublishCommand 设置发布消息命令
func (c *CLI) setupRabbitPublishCommand() *cobra.Command {
  var printLog bool
  var exchangeName, routingKey, message string
  var interval, repeat, rateLimit int
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "publish",
    Short: "Publish a message to an exchange",
    RunE: func(cmd *cobra.Command, args []string) error {
      client := &rabbitmq.RabbitClient{
        Config: &rabbitmq.ServerConfig{
          Server:   server,
          Port:     port,
          Vhost:    vhost,
          User:     user,
          Password: password,
        },
      }

      config := &rabbitmq.PublishConfig{
        PrintLog:   printLog,
        Exchange:   exchangeName,
        RoutingKey: routingKey,
        Message:    message,
        Interval:   interval,
        Repeat:     repeat,
        RateLimit:  rateLimit,
      }
      return c.handleRabbitPublish(client, config)
    },
  }

  cmd.Flags().BoolVar(&printLog, "print-log", true, "PrintLog log")
  cmd.Flags().StringVarP(&exchangeName, "exchange", "e", "", "Exchange name")
  cmd.Flags().StringVarP(&routingKey, "routing-key", "r", "", "Routing key")
  cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
  cmd.Flags().IntVarP(&interval, "interval", "t", 1000, "Time interval in milliseconds between messages")
  cmd.Flags().IntVarP(&repeat, "repeat", "r", 10, "Number of times to repeat sending the message")
  cmd.Flags().IntVarP(&rateLimit, "rate-limit", "l", 0, "Send rate limit")
  cmd.MarkFlagRequired("message")

  return cmd
}

// handleRabbitPublish 处理发布消息命令
func (c *CLI) handleRabbitPublish(client *rabbitmq.RabbitClient, config *rabbitmq.PublishConfig) error {

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  if err := client.PublishMessage(config); err != nil {
    return fmt.Errorf("failed to publish message: %w", err)
  }

  fmt.Printf("Message published to exchange '%s' with routing key '%s'\n", config.Exchange, config.RoutingKey)
  return nil
}

// setupRabbitConsumeCommand 设置消费消息命令
func (c *CLI) setupRabbitConsumeCommand() *cobra.Command {
  var queueName, consumer string
  var autoAck bool
  var timeout int
  var server string
  var port int
  var vhost, user, password string

  cmd := &cobra.Command{
    Use:   "consume",
    Short: "Consume messages from a queue",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleRabbitConsume(queueName, consumer, autoAck, timeout, server, port, vhost, user, password)
    },
  }

  cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue name")
  cmd.Flags().StringVarP(&consumer, "consumer", "c", "", "Consumer name")
  cmd.Flags().BoolVarP(&autoAck, "auto-ack", "a", true, "Auto acknowledge messages")
  cmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "Timeout in seconds for waiting messages")
  cmd.MarkFlagRequired("queue")

  return cmd
}

// handleRabbitConsume 处理消费消息命令
func (c *CLI) handleRabbitConsume(queue, consumer string, autoAck bool, timeout int, server string, port int, vhost, user, password string) error {
  client := &rabbitmq.RabbitClient{
    Config: &rabbitmq.ServerConfig{
      Server:   server,
      Port:     port,
      Vhost:    vhost,
      User:     user,
      Password: password,
    },
  }

  if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
  }
  defer client.Close()

  consume := &rabbitmq.ConsumeConfig{
    Queue:    queue,
    Consumer: consumer,
    AutoAck:  autoAck,
    Timeout:  timeout,
  }

  fmt.Printf("Starting to consume messages from queue '%s'\n", queue)
  fmt.Printf("Press Ctrl+C to stop...\n")

  if err := client.ConsumeMessage(consume); err != nil {
    return fmt.Errorf("error consuming messages: %w", err)
  }

  return nil
}

// parseBindArgs 解析绑定参数
func parseBindArgs(argsStr string) map[string]interface{} {
  args := make(map[string]interface{})
  pairs := strings.Split(argsStr, ",")

  for _, pair := range pairs {
    kv := strings.SplitN(pair, "=", 2)
    if len(kv) == 2 {
      key := strings.TrimSpace(kv[0])
      value := strings.TrimSpace(kv[1])
      args[key] = value
    }
  }

  return args
}

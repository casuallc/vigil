package rabbitmq

import (
  "fmt"
  amqp "github.com/rabbitmq/amqp091-go"
  "log"
  "sync"
  "time"
)

type RabbitClient struct {
  conn    *amqp.Connection
  channel *amqp.Channel
  Config  *RabbitMQServerConfig
  mu      sync.Mutex
}

func (r *RabbitClient) Connect() error {
  url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
    r.Config.User,
    r.Config.Password,
    r.Config.Server,
    r.Config.Port,
    r.Config.Vhost,
  )

  conn, err := amqp.Dial(url)
  if err != nil {
    return fmt.Errorf("can not connect rabbitmq: %w", err)
  }
  r.conn = conn

  ch, err := conn.Channel()
  if err != nil {
    r.Close()
    return fmt.Errorf("can not open channel: %w", err)
  }
  r.channel = ch

  log.Printf("Connected to rabbitmq server %s:%d", r.Config.Server, r.Config.Port)
  return nil
}

func (r *RabbitClient) Close() {
  if r.channel != nil {
    _ = r.channel.Close()
  }
  if r.conn != nil {
    _ = r.conn.Close()
  }
}

func (r *RabbitClient) DeclareExchange(exchange *RabbitMQExchangeConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if err := r.channel.ExchangeDeclare(
    exchange.Name,
    exchange.Type,
    exchange.Durable,
    exchange.AutoDelete,
    false,
    false,
    nil,
  ); err != nil {
    return fmt.Errorf("failed to declare exchange %w", err)
  }

  log.Printf("Declared exchange %s", exchange.Name)
  return nil
}

func (r *RabbitClient) DeleteExchange(name string) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if err := r.channel.ExchangeDelete(
    name,
    false,
    false,
  ); err != nil {
    return fmt.Errorf("failed to delete exchange %w", err)
  }

  log.Printf("Delete exchange %s", name)
  return nil
}

func (r *RabbitClient) DeclareQueue(queue *RabbitMQQueueConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if _, err := r.channel.QueueDeclare(
    queue.Name,
    queue.Durable,
    queue.AutoDelete,
    queue.Exclusive,
    false,
    queue.Args,
  ); err != nil {
    return fmt.Errorf("failed to declare queue %w", err)
  }

  log.Printf("Declared queue %s", queue.Name)
  return nil
}

func (r *RabbitClient) DeleteQueue(name string) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if _, err := r.channel.QueueDelete(
    name,
    false,
    false,
    false,
  ); err != nil {
    return fmt.Errorf("failed to delete queue %w", err)
  }

  log.Printf("Delete queue %s", name)
  return nil
}

func (r *RabbitClient) QueueBind(bind *RabbitMQBindConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if err := r.channel.QueueBind(
    bind.Queue,
    bind.RoutingKey,
    bind.Exchange,
    false,
    bind.Arguments,
  ); err != nil {
    return fmt.Errorf("failed to bind queue %w", err)
  }

  log.Printf("Bind queue %s to exchange %s", bind.Queue, bind.Exchange)
  return nil
}

func (r *RabbitClient) QueueUnBind(bind *RabbitMQBindConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if err := r.channel.QueueUnbind(
    bind.Queue,
    bind.RoutingKey,
    bind.Exchange,
    bind.Arguments,
  ); err != nil {
    return fmt.Errorf("failed to unbind queue %w", err)
  }

  log.Printf("Unbind queue %s to exchange %s", bind.Queue, bind.Exchange)
  return nil
}

// PublishMessage 发送消息
func (r *RabbitClient) PublishMessage(publish *RabbitMQPublishConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  ticker := time.NewTicker(time.Duration(publish.Interval) * time.Millisecond)
  defer ticker.Stop()

  for i := 0; i < publish.Repeat; i++ {
    select {
    case <-ticker.C:
    }

    err := r.channel.Publish(
      publish.Exchange,
      publish.RoutingKey,
      false,
      false,
      amqp.Publishing{
        Headers:      publish.Headers,
        ContentType:  "text/plain",
        Body:         []byte(publish.Message),
        DeliveryMode: amqp.Persistent,
      })

    if err != nil {
      return fmt.Errorf("failed to publish message %w", err)
    }

    log.Printf("Publish message %s", publish.Message)
  }

  log.Printf("Publish message count: %d", publish.Repeat)
  return nil
}

// ConsumeMessage 接收消息
func (r *RabbitClient) ConsumeMessage(consume *RabbitMQConsumeConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if consume.Timeout == 0 {
    consume.Timeout = 10
  }

  msgs, err := r.channel.Consume(
    consume.Queue,
    consume.Consumer,
    consume.AutoAck,
    false,
    false,
    false,
    nil,
  )
  if err != nil {
    return fmt.Errorf("failed to consume message %w", err)
  }

  log.Printf("Starting consume message from %s", consume.Queue)

  idleTimer := time.NewTimer(time.Duration(consume.Timeout) * time.Second)
  // 确保 timer 可以停止，避免资源泄漏
  defer func() {
    if !idleTimer.Stop() {
      select {
      case <-idleTimer.C:
      default:
      }
    }
  }()

  for {
    select {
    case msg, ok := <-msgs:
      if !ok {
        return fmt.Errorf("message channel closed unexpectedly")
      }

      // 重置空闲计时器（收到消息，说明未超时）
      if !idleTimer.Stop() {
        select {
        case <-idleTimer.C:
        default:
        }
      }
      idleTimer.Reset(time.Duration(consume.Timeout) * time.Second)

      // 处理消息
      if consume.Handler != nil {
        consume.Handler(msg)
      } else {
        log.Printf("Received a message: %s", msg.Body)
        if !consume.AutoAck {
          if err := msg.Ack(false); err != nil {
            log.Printf("Failed to acknowledge message: %v", err)
          }
        }
      }

    case <-idleTimer.C:
      log.Printf("No messages received for %d seconds, exiting consumer for queue: %s",
        consume.Timeout, consume.Queue)
      return nil // 正常退出，也可以返回特定错误
    }
  }
}

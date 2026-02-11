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

package rabbitmq

import (
  "fmt"
  "log"
  "sync"
  "time"

  amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
  conn          *amqp.Connection
  channel       *amqp.Channel
  Config        *ServerConfig
  mu            sync.Mutex
  producedCount int64 // AI Modified: 记录生产的消息总数
  consumedCount int64 // AI Modified: 记录消费的消息总数
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

// Channel returns the underlying AMQP channel
func (r *RabbitClient) Channel() *amqp.Channel {
  return r.channel
}

func (r *RabbitClient) Close() {
  if r.channel != nil {
    _ = r.channel.Close()
  }
  if r.conn != nil {
    _ = r.conn.Close()
  }
  // AI Modified: 打印消息计数
  log.Printf("RabbitMQ Client Stats - Produced: %d, Consumed: %d", r.producedCount, r.consumedCount)
}

func (r *RabbitClient) DeclareExchange(exchange *ExchangeConfig) error {
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

func (r *RabbitClient) DeclareQueue(queue *QueueConfig) error {
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

func (r *RabbitClient) QueueBind(bind *BindConfig) error {
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

func (r *RabbitClient) QueueUnBind(bind *BindConfig) error {
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
func (r *RabbitClient) PublishMessage(publish *PublishConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if publish.Interval == 0 {
    publish.Interval = 1000
  }
  if publish.Repeat == 0 {
    publish.Repeat = 1
  }

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

    if publish.PrintLog {
      log.Printf("Publish message: %s", publish.Message)
    }
  }

  r.producedCount += int64(publish.Repeat)
  log.Printf("Total messages sent: %d", publish.Repeat)
  return nil
}

// ConsumeMessage 接收消息
func (r *RabbitClient) ConsumeMessage(consume *ConsumeConfig) error {
  r.mu.Lock()
  defer r.mu.Unlock()

  if consume.Timeout == 0 {
    consume.Timeout = 10
  }
  if err := r.channel.Qos(1, 0, false); err != nil {
    log.Fatal(err)
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

      // AI Modified: 记录消费的消息总数
      r.consumedCount++

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

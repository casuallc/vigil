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

package activemq

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
)

// Client defines the ActiveMQ STOMP client
type Client struct {
	conn          *stomp.Conn
	config        *ServerConfig
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	producedCount int64
	consumedCount int64
}

// NewClient creates a new ActiveMQ client
func NewClient(config *ServerConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Connect connects to the ActiveMQ server via STOMP
func (c *Client) Connect() error {
	addr := net.JoinHostPort(c.config.Server, fmt.Sprintf("%d", c.config.Port))

	opts := []func(*stomp.Conn) error{}

	// Authentication
	if c.config.User != "" && c.config.Password != "" {
		opts = append(opts, stomp.ConnOpt.Login(c.config.User, c.config.Password))
	}

	// VHost
	if c.config.VHost != "" {
		opts = append(opts, stomp.ConnOpt.Host(c.config.VHost))
	}

	// Heartbeat
	if c.config.HeartBeat > 0 {
		opts = append(opts, stomp.ConnOpt.HeartBeat(
			time.Duration(c.config.HeartBeat)*time.Second,
			time.Duration(c.config.HeartBeat)*time.Second,
		))
	}

	// Connection timeout
	if c.config.Timeout > 0 {
		dialer := &net.Dialer{Timeout: time.Duration(c.config.Timeout) * time.Second}
		netConn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to ActiveMQ %s: %w", addr, err)
		}
		conn, err := stomp.Connect(netConn, opts...)
		if err != nil {
			_ = netConn.Close()
			return fmt.Errorf("failed to establish STOMP connection: %w", err)
		}
		c.conn = conn
	} else {
		netConn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to ActiveMQ %s: %w", addr, err)
		}
		conn, err := stomp.Connect(netConn, opts...)
		if err != nil {
			_ = netConn.Close()
			return fmt.Errorf("failed to establish STOMP connection: %w", err)
		}
		c.conn = conn
	}

	log.Printf("Connected to ActiveMQ server %s", addr)
	return nil
}

// Close closes the client connection
func (c *Client) Close() {
	c.cancel()
	if c.conn != nil {
		_ = c.conn.Disconnect()
	}
	log.Printf("ActiveMQ Client Stats - Produced: %d, Consumed: %d", c.producedCount, c.consumedCount)
}

// SendMessage sends a message to the destination
func (c *Client) SendMessage(config *ProducerConfig) error {
	if c.conn == nil {
		return fmt.Errorf("ActiveMQ client is not connected")
	}

	// Set defaults
	if config.Repeat <= 0 {
		config.Repeat = 1
	}
	if config.Interval <= 0 {
		config.Interval = 1000
	}

	var body []byte
	var err error
	if config.MessageFile != "" {
		body, err = os.ReadFile(config.MessageFile)
		if err != nil {
			return fmt.Errorf("failed to read message file %s: %w", config.MessageFile, err)
		}
	}

	// Parse headers
	var sendOpts []func(*frame.Frame) error
	headers := common.ParsePropertyArray(config.Headers)
	if headers != nil {
		for _, kv := range headers {
			sendOpts = append(sendOpts, stomp.SendOpt.Header(kv[0], kv[1]))
		}
	}

	// Set persistent delivery
	if config.Persistent {
		sendOpts = append(sendOpts, stomp.SendOpt.Header("persistent", "true"))
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

			var messageContent string
			if config.MessageFile != "" {
				messageContent = string(body)
			} else {
				messageContent = config.Message
			}

			err := c.conn.Send(config.Destination, "text/plain", []byte(messageContent), sendOpts...)
			if err != nil {
				log.Printf("Failed to send message %d: %v", idx, err)
			} else if config.PrintLog {
				log.Printf("Message %d sent successfully to destination '%s', size: %d bytes",
					idx, config.Destination, len(messageContent))
			}
		}(i)
	}

	wg.Wait()
	c.producedCount += int64(config.Repeat)
	log.Printf("Total messages sent: %d", config.Repeat)
	return nil
}

// ReceiveMessage receives messages from the destination
func (c *Client) ReceiveMessage(config *ConsumerConfig) error {
	if c.conn == nil {
		return fmt.Errorf("ActiveMQ client is not connected")
	}

	// Set defaults
	if config.AckMode == "" {
		config.AckMode = "auto"
	}

	// Determine ack mode
	var ackMode stomp.AckMode
	switch strings.ToLower(config.AckMode) {
	case "client":
		ackMode = stomp.AckClient
	case "client-individual":
		ackMode = stomp.AckClientIndividual
	default:
		ackMode = stomp.AckAuto
	}

	var subOpts []func(*frame.Frame) error

	// Durable subscription
	if config.Durable && config.SubscriptionName != "" {
		subOpts = append(subOpts, stomp.SubscribeOpt.Header("durable-subscription-name", config.SubscriptionName))
	}

	sub, err := c.conn.Subscribe(config.Destination, ackMode, subOpts...)
	if err != nil {
		return fmt.Errorf("failed to subscribe to destination '%s': %w", config.Destination, err)
	}
	defer sub.Unsubscribe()

	log.Printf("Subscribed to destination '%s' with ack mode '%s'", config.Destination, config.AckMode)

	messageCount := 0
	done := make(chan struct{})

	// Handle timeout
	if config.Timeout > 0 {
		go func() {
			timer := time.NewTimer(time.Duration(config.Timeout) * time.Second)
			select {
			case <-timer.C:
				log.Printf("Subscription timeout after %d seconds", config.Timeout)
				close(done)
			case <-c.ctx.Done():
				close(done)
			}
		}()
	} else {
		// Wait for interrupt signal
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-sigChan:
				log.Printf("Subscription interrupted by signal")
			case <-c.ctx.Done():
				log.Printf("Subscription canceled")
			}
			close(done)
		}()
	}

	for {
		select {
		case msg := <-sub.C:
			if msg == nil {
				log.Printf("Subscription closed")
				return nil
			}

			c.consumedCount++
			messageCount++

			if config.PrintLog {
				log.Printf("Received message %d: destination=%s, message-id=%s, body=%s",
					messageCount, config.Destination, msg.Header.Get("message-id"), string(msg.Body))
			}

			// Manual ack for client mode
			if config.AckMode == "client" || config.AckMode == "client-individual" {
				if err := c.conn.Ack(msg); err != nil {
					log.Printf("Failed to ack message: %v", err)
				}
			}

			// Check max messages
			if config.MaxMessages > 0 && messageCount >= config.MaxMessages {
				log.Printf("Reached maximum message count: %d", config.MaxMessages)
				return nil
			}

		case <-done:
			log.Printf("Finished receiving %d messages from destination '%s'", messageCount, config.Destination)
			return nil
		case <-c.ctx.Done():
			log.Printf("Subscription canceled")
			return nil
		}
	}
}

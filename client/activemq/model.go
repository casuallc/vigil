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
	"time"
)

// ServerConfig defines ActiveMQ server configuration
type ServerConfig struct {
	Server    string
	Port      int
	User      string
	Password  string
	Timeout   int
	VHost     string
	HeartBeat int
}

// ProducerConfig defines message sending configuration
type ProducerConfig struct {
	Destination string
	Message     string
	MessageFile string
	Repeat      int
	Interval    int
	PrintLog    bool
	Headers     string
	Persistent  bool
}

// ConsumerConfig defines message receiving configuration
type ConsumerConfig struct {
	Destination      string
	Timeout          int
	PrintLog         bool
	MaxMessages      int
	Durable          bool
	SubscriptionName string
	AckMode          string
}

// Message defines the message structure
type Message struct {
	Destination string
	Body        string
	Headers     map[string]string
	MessageID   string
	Timestamp   time.Time
}

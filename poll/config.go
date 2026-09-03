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

// Package poll implements the outbound-only task pulling mode: bbx polls
// one or more independent upstream services over HTTP long-polling,
// executes tasks locally and posts results back.
package poll

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that unmarshals from YAML strings like "25s",
// or plain integers interpreted as seconds.
type Duration time.Duration

// Std returns the value as a standard time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("invalid duration value: %v", value.Value)
	}
	if s == "" {
		*d = 0
		return nil
	}
	if v, err := time.ParseDuration(s); err == nil {
		*d = Duration(v)
		return nil
	}
	// Plain numbers are interpreted as seconds.
	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}
	return fmt.Errorf("invalid duration %q", s)
}

// MarshalYAML implements yaml.Marshaler.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// TLSConfig configures the TLS verification behavior of one upstream.
type TLSConfig struct {
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CAFile             string `yaml:"ca_file"`
}

// Defaults holds poll parameters shared by all upstreams.
type Defaults struct {
	LongPollWait         Duration  `yaml:"long_poll_wait"`
	BusyInterval         Duration  `yaml:"busy_interval"`
	IdleBackoffMax       Duration  `yaml:"idle_backoff_max"`
	BusyToIdleEmptyPolls int       `yaml:"busy_to_idle_empty_polls"`
	TaskTimeout          Duration  `yaml:"task_timeout"`
	MaxTopics            int       `yaml:"max_topics"`
	QueueBuffer          int       `yaml:"queue_buffer"`
	TopicIdleTTL         Duration  `yaml:"topic_idle_ttl"`
	TLS                  TLSConfig `yaml:"tls"`
}

// UpstreamAuth holds per-upstream Basic Auth credentials.
type UpstreamAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// UpstreamConfig configures one independent upstream service.
type UpstreamConfig struct {
	Name      string       `yaml:"name"`
	Endpoint  string       `yaml:"endpoint"`
	Auth      UpstreamAuth `yaml:"auth"`
	TLS       *TLSConfig   `yaml:"tls"`
	AllowHTTP bool         `yaml:"allow_http"` // plaintext HTTP must be explicitly allowed
}

// PollConfig is the root configuration of the poll-mode agent.
type PollConfig struct {
	Enabled   bool             `yaml:"enabled"`
	AgentID   string           `yaml:"agent_id"` // defaults to hostname
	Defaults  Defaults         `yaml:"defaults"`
	Upstreams []UpstreamConfig `yaml:"upstreams"`
}

// Built-in default values applied when the config leaves fields unset.
const (
	defaultLongPollWait         = 25 * time.Second
	defaultBusyInterval         = 500 * time.Millisecond
	defaultIdleBackoffMax       = 10 * time.Second
	defaultBusyToIdleEmptyPolls = 3
	defaultTaskTimeout          = 120 * time.Second
	defaultMaxTopics            = 32
	defaultQueueBuffer          = 64
	defaultTopicIdleTTL         = 10 * time.Minute

	// drainTimeout bounds how long workers keep executing queued tasks
	// during shutdown before NACK-ing the rest.
	drainTimeout = 10 * time.Second

	// dedupCacheSize is how many recently completed task ids are
	// remembered per upstream for idempotent re-ack.
	dedupCacheSize = 1024
)

// applyDefaults fills zero-valued Default fields with built-in defaults.
func (c *PollConfig) applyDefaults() {
	d := &c.Defaults
	if d.LongPollWait == 0 {
		d.LongPollWait = Duration(defaultLongPollWait)
	}
	if d.BusyInterval == 0 {
		d.BusyInterval = Duration(defaultBusyInterval)
	}
	if d.IdleBackoffMax == 0 {
		d.IdleBackoffMax = Duration(defaultIdleBackoffMax)
	}
	if d.BusyToIdleEmptyPolls <= 0 {
		d.BusyToIdleEmptyPolls = defaultBusyToIdleEmptyPolls
	}
	if d.TaskTimeout == 0 {
		d.TaskTimeout = Duration(defaultTaskTimeout)
	}
	if d.MaxTopics <= 0 {
		d.MaxTopics = defaultMaxTopics
	}
	if d.QueueBuffer <= 0 {
		d.QueueBuffer = defaultQueueBuffer
	}
	if d.TopicIdleTTL == 0 {
		d.TopicIdleTTL = Duration(defaultTopicIdleTTL)
	}
}

// validate checks the configuration for actionable errors.
func (c *PollConfig) validate() error {
	if len(c.Upstreams) == 0 {
		return fmt.Errorf("poll: no upstreams configured")
	}
	seen := make(map[string]bool)
	for i, u := range c.Upstreams {
		if u.Endpoint == "" {
			return fmt.Errorf("poll: upstream #%d has no endpoint", i)
		}
		if u.Name == "" {
			u.Name = fmt.Sprintf("upstream-%d", i)
			c.Upstreams[i].Name = u.Name
		}
		if seen[u.Name] {
			return fmt.Errorf("poll: duplicate upstream name %q", u.Name)
		}
		seen[u.Name] = true
		if isPlainHTTP(u.Endpoint) && !u.AllowHTTP {
			return fmt.Errorf("poll: upstream %q uses plaintext http; set allow_http: true to permit it", u.Name)
		}
	}
	return nil
}

func isPlainHTTP(endpoint string) bool {
	return len(endpoint) >= 7 && endpoint[:7] == "http://"
}

// tlsConfigFor returns the effective TLS config for an upstream,
// falling back to the shared defaults.
func (c *PollConfig) tlsConfigFor(u *UpstreamConfig) TLSConfig {
	if u.TLS != nil {
		return *u.TLS
	}
	return c.Defaults.TLS
}

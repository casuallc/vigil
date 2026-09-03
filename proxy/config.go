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

// Package proxy implements HTTP reverse proxy instances for bbx-server:
// static instances from config.yaml, dynamic instances managed over the
// REST API (persisted in SQLite), and the tunnel core used by poll-mode
// proxy_session tasks.
package proxy

// TLSConfig configures optional TLS termination of one proxy listener.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertPath string `yaml:"cert_path" json:"cert_path"`
	KeyPath  string `yaml:"key_path" json:"key_path"`
}

// InstanceConfig describes one reverse proxy instance.
type InstanceConfig struct {
	Name         string            `yaml:"name" json:"name"`
	Listen       string            `yaml:"listen" json:"listen"`                     // ":8080" / "127.0.0.1:8080"
	Target       string            `yaml:"target" json:"target"`                     // "http://10.0.0.5:9000"
	Whitelist    []string          `yaml:"whitelist" json:"whitelist"`               // empty = only the target host itself
	AllowPrivate bool              `yaml:"allow_private" json:"allow_private"`       // default false: SSRF guard
	TLS          TLSConfig         `yaml:"tls" json:"tls"`
	MaxBodyMB    int64             `yaml:"max_body_mb" json:"max_body_mb"`           // 0 = unlimited request body
	HeaderSet    map[string]string `yaml:"header_set" json:"header_set"`             // extra headers injected upstream
}

// TunnelConfig gates poll-mode proxy_session tunnel tasks.
// The allowed targets are a local policy: the upstream service cannot
// broaden them. An empty AllowedTargets list disables tunneling entirely.
type TunnelConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	AllowedTargets []string `yaml:"allowed_targets" json:"allowed_targets"`
	MaxSessions    int      `yaml:"max_sessions" json:"max_sessions"`        // default 8
	MaxDurationSec int      `yaml:"max_duration_sec" json:"max_duration_sec"` // default 3600
	MaxBodyMB      int64    `yaml:"max_body_mb" json:"max_body_mb"`
}

// ProxyConfig is the root configuration of the proxy feature.
type ProxyConfig struct {
	Enabled   bool             `yaml:"enabled" json:"enabled"`
	Instances []InstanceConfig `yaml:"instances" json:"instances"` // origin=config static instances
	Tunnel    TunnelConfig     `yaml:"tunnel" json:"tunnel"`
}

// Built-in defaults for the tunnel configuration.
const (
	defaultMaxSessions    = 8
	defaultMaxDurationSec = 3600
)

// applyDefaults fills zero-valued tunnel fields with built-in defaults.
func (c *TunnelConfig) applyDefaults() {
	if c.MaxSessions <= 0 {
		c.MaxSessions = defaultMaxSessions
	}
	if c.MaxDurationSec <= 0 {
		c.MaxDurationSec = defaultMaxDurationSec
	}
}

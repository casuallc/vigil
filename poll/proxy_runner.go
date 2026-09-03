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

package poll

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

// SessionLimits bounds one proxy_session tunnel.
type SessionLimits struct {
	MaxDuration time.Duration
	MaxBodyMB   int64
}

// SessionStats summarizes one finished proxy_session tunnel. It is sent
// back as the task's ack result.
type SessionStats struct {
	Requests   int64  `json:"requests"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	DurationMs int64  `json:"duration_ms"`
	EndReason  string `json:"end_reason"` // closed | max_duration | shutdown | error
}

// ProxyRunner executes a proxy_session tunnel over an already-dialed
// WebSocket connection. It is implemented by proxy.TunnelCore and injected
// through Options so the poll package never imports the proxy package.
// A nil runner makes proxy_session tasks fail explicitly.
type ProxyRunner interface {
	RunProxySession(ctx context.Context, conn *websocket.Conn, target string, limits SessionLimits) (SessionStats, error)
}

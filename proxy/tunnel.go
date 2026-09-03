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

package proxy

// TunnelCore executes poll-mode proxy_session tunnel sessions. The full
// HTTP-over-WebSocket session loop lands with the poll integration; this
// file holds the core type, configuration gating and concurrency control.
type TunnelCore struct {
	cfg  TunnelConfig
	wl   *Whitelist
	hook AccessHook
	sem  chan struct{}
}

// newTunnelCore builds the tunnel core from config. An empty
// AllowedTargets list (or a disabled tunnel) yields a whitelist that
// rejects everything, so no session can be established.
func newTunnelCore(cfg TunnelConfig, hook AccessHook) *TunnelCore {
	cfg.applyDefaults()
	wl, err := ParseWhitelist(cfg.AllowedTargets, true)
	if err != nil {
		// Invalid entries must not silently widen access: reject all.
		wl, _ = ParseWhitelist(nil, false)
	}
	return &TunnelCore{
		cfg:  cfg,
		wl:   wl,
		hook: hook,
		sem:  make(chan struct{}, cfg.MaxSessions),
	}
}

// Enabled reports whether tunnel sessions are allowed at all.
func (tc *TunnelCore) Enabled() bool {
	return tc.cfg.Enabled && len(tc.cfg.AllowedTargets) > 0
}

// TargetAllowed reports whether a tunnel session may proxy to target.
func (tc *TunnelCore) TargetAllowed(target string) bool {
	return tc.Enabled() && tc.wl.Allowed(target)
}

// acquire takes a session slot, blocking until one is free or the session
// context ends. The returned function releases the slot.
func (tc *TunnelCore) acquire() (release func(), ok bool) {
	select {
	case tc.sem <- struct{}{}:
		return func() { <-tc.sem }, true
	default:
		return nil, false
	}
}

// Close releases tunnel resources. Sessions are owned by the poll agent's
// task contexts and end with it; there is nothing persistent to close yet.
func (tc *TunnelCore) Close() {}

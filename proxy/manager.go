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

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// Origins of an instance definition.
const (
	OriginConfig = "config"
	OriginAPI    = "api"
)

// ErrConfigOrigin is returned when an API caller tries to delete an
// instance that is defined in config.yaml. The API layer maps it to 409.
var ErrConfigOrigin = errors.New("proxy: instance is defined in config.yaml; edit the config file instead")

// ErrNotFound is returned when an instance name does not exist.
var ErrNotFound = errors.New("proxy: instance not found")

// AccessRecord describes one proxied request (or a rejected one).
type AccessRecord struct {
	Instance   string `json:"instance"`
	ClientIP   string `json:"client_ip"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	BytesOut   int    `json:"bytes_out"`
	DurationMs int64  `json:"duration_ms"`
	Via        string `json:"via"` // "listen" | "forward" | "tunnel"
	Denied     bool   `json:"denied,omitempty"`
}

// AccessHook observes every proxied request. The api layer fans this out
// to the audit logger (denied only) and the per-instance access log file.
type AccessHook func(rec AccessRecord)

// Manager owns all proxy instances: static ones from config.yaml and
// dynamic ones persisted in SQLite.
type Manager struct {
	cfg  *ProxyConfig
	st   *store
	hook AccessHook
	auth AuthFunc // validates forward-proxy client credentials

	mu   sync.RWMutex
	inst map[string]*Instance

	tunnel *TunnelCore // phase 2: poll proxy_session support
}

// NewManager opens the store and prepares the manager. Call Recover to
// actually start listeners.
func NewManager(cfg *ProxyConfig, dbPath string, hook AccessHook) (*Manager, error) {
	st, err := newStore(dbPath)
	if err != nil {
		return nil, err
	}
	cfg.Tunnel.applyDefaults()
	m := &Manager{
		cfg:  cfg,
		st:   st,
		hook: hook,
		inst: make(map[string]*Instance),
	}
	m.tunnel = newTunnelCore(cfg.Tunnel, hook)
	return m, nil
}

// SetAuthFunc installs the credential validator used by forward-mode
// instances. Call it before Recover so recovered instances see it too.
func (m *Manager) SetAuthFunc(fn AuthFunc) { m.auth = fn }

// Recover starts config-defined instances and restarts API-created
// instances whose desired state is "running". A single failing instance
// is logged and skipped so it never blocks the others.
func (m *Manager) Recover(ctx context.Context) error {
	stored, err := m.st.loadAll()
	if err != nil {
		return fmt.Errorf("proxy: failed to load instances: %w", err)
	}

	// 1) Config-defined instances win over same-named DB records.
	seen := make(map[string]bool)
	for _, cfg := range m.cfg.Instances {
		seen[cfg.Name] = true
		if err := m.st.upsert(cfg, OriginConfig, desiredRunning); err != nil {
			log.Printf("proxy: failed to persist config instance %q: %v", cfg.Name, err)
		}
		m.startInstance(cfg, OriginConfig)
	}

	// 2) API-created instances with desired_state=running.
	for _, si := range stored {
		if seen[si.cfg.Name] || si.origin != OriginAPI || si.desired != desiredRunning {
			continue
		}
		m.startInstance(si.cfg, OriginAPI)
	}
	return nil
}

// startInstance creates (replacing any existing entry) and starts one
// instance. Failures are logged, not propagated.
func (m *Manager) startInstance(cfg InstanceConfig, origin string) {
	inst, err := NewInstance(cfg, origin, m.hook, m.auth)
	if err != nil {
		log.Printf("proxy: instance %q: %v", cfg.Name, err)
		return
	}
	m.mu.Lock()
	old := m.inst[cfg.Name]
	m.inst[cfg.Name] = inst
	m.mu.Unlock()
	if old != nil {
		_ = old.Stop(context.Background())
	}
	if err := inst.Start(context.Background()); err != nil {
		log.Printf("proxy: %v", err)
		return
	}
	log.Printf("proxy: instance %q listening on %s%s", cfg.Name, cfg.Listen, targetSuffix(cfg))
}

// targetSuffix renders the " -> target" log suffix (empty for forward mode).
func targetSuffix(cfg InstanceConfig) string {
	if cfg.Target == "" {
		return " (mode=" + cfg.Mode + ")"
	}
	return " -> " + cfg.Target
}

// Create registers a new API-managed instance and optionally starts it.
func (m *Manager) Create(cfg InstanceConfig, autostart bool) error {
	m.mu.RLock()
	_, exists := m.inst[cfg.Name]
	m.mu.RUnlock()
	if exists {
		return fmt.Errorf("proxy: instance %q already exists", cfg.Name)
	}
	// Validate eagerly so bad configs never reach the store.
	if _, err := NewInstance(cfg, OriginAPI, nil, m.auth); err != nil {
		return err
	}

	desired := desiredStopped
	if autostart {
		desired = desiredRunning
	}
	if err := m.st.upsert(cfg, OriginAPI, desired); err != nil {
		return err
	}
	if autostart {
		inst, _ := NewInstance(cfg, OriginAPI, m.hook, m.auth)
		m.mu.Lock()
		m.inst[cfg.Name] = inst
		m.mu.Unlock()
		if err := inst.Start(context.Background()); err != nil {
			_ = m.st.setDesired(cfg.Name, desiredStopped)
			return err
		}
		log.Printf("proxy: instance %q listening on %s%s", cfg.Name, cfg.Listen, targetSuffix(cfg))
	} else {
		inst, _ := NewInstance(cfg, OriginAPI, m.hook, m.auth)
		m.mu.Lock()
		m.inst[cfg.Name] = inst
		m.mu.Unlock()
	}
	return nil
}

// Update replaces an instance's config. A running instance is restarted
// with the new config; origin=config instances keep their origin.
func (m *Manager) Update(name string, cfg InstanceConfig) error {
	m.mu.RLock()
	old, exists := m.inst[name]
	m.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}
	cfg.Name = name

	inst, err := NewInstance(cfg, old.Origin(), m.hook, m.auth)
	if err != nil {
		return err
	}
	if err := m.st.upsert(cfg, old.Origin(), desiredStopped); err != nil {
		return err
	}

	wasRunning := old.Status().State == StateRunning
	m.mu.Lock()
	m.inst[name] = inst
	m.mu.Unlock()
	if err := old.Stop(context.Background()); err != nil {
		log.Printf("proxy: instance %q: stop during update: %v", name, err)
	}
	if wasRunning {
		if err := inst.Start(context.Background()); err != nil {
			return err
		}
		_ = m.st.setDesired(name, desiredRunning)
	}
	return nil
}

// Delete removes an API-managed instance. Config-defined instances
// cannot be deleted through the API.
func (m *Manager) Delete(name string) error {
	m.mu.RLock()
	inst, exists := m.inst[name]
	m.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}
	if inst.Origin() == OriginConfig {
		return ErrConfigOrigin
	}
	if err := inst.Stop(context.Background()); err != nil {
		log.Printf("proxy: instance %q: stop during delete: %v", name, err)
	}
	m.mu.Lock()
	delete(m.inst, name)
	m.mu.Unlock()
	return m.st.delete(name)
}

// Start launches a stopped instance.
func (m *Manager) Start(name string) error {
	m.mu.RLock()
	inst, exists := m.inst[name]
	m.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}
	if err := inst.Start(context.Background()); err != nil {
		return err
	}
	return m.st.setDesired(name, desiredRunning)
}

// Stop halts a running instance.
func (m *Manager) Stop(name string) error {
	m.mu.RLock()
	inst, exists := m.inst[name]
	m.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}
	if err := inst.Stop(context.Background()); err != nil {
		return err
	}
	return m.st.setDesired(name, desiredStopped)
}

// Get returns the status of one instance.
func (m *Manager) Get(name string) (InstanceStatus, error) {
	m.mu.RLock()
	inst, exists := m.inst[name]
	m.mu.RUnlock()
	if !exists {
		return InstanceStatus{}, ErrNotFound
	}
	return inst.Status(), nil
}

// List returns the status of every instance.
func (m *Manager) List() []InstanceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]InstanceStatus, 0, len(m.inst))
	for _, inst := range m.inst {
		out = append(out, inst.Status())
	}
	return out
}

// Shutdown stops every listener and closes the store.
func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	instances := make([]*Instance, 0, len(m.inst))
	for _, inst := range m.inst {
		instances = append(instances, inst)
	}
	m.mu.Unlock()
	for _, inst := range instances {
		if err := inst.Stop(ctx); err != nil {
			log.Printf("proxy: instance %q: shutdown: %v", inst.Name(), err)
		}
	}
	if m.tunnel != nil {
		m.tunnel.Close()
	}
	_ = m.st.close()
}

// Tunnel returns the tunnel core used by poll-mode proxy_session tasks.
func (m *Manager) Tunnel() *TunnelCore { return m.tunnel }

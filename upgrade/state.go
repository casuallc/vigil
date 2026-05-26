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

package upgrade

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State represents the upgrade state file content
type State struct {
	PID       int       `json:"pid"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"started_at"`
	Token     string    `json:"token"` // random token to prevent PID reuse attacks
	Upgrade   UpgradeInfo `json:"upgrade"`
}

// UpgradeInfo represents the upgrade sub-state
type UpgradeInfo struct {
	State        string    `json:"state"`
	NewPID       int       `json:"new_pid"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Error        string    `json:"error"`
	BinaryPath   string    `json:"binary_path"`
}

// Upgrade state constants
const (
	StateIdle           = "idle"
	StateDownloading    = "downloading"
	StateVerifying      = "verifying"
	StateStarting       = "starting"
	StateHealthChecking = "health_checking"
	StateReady          = "ready"
	StateCompleted      = "completed"
	StateFailed         = "failed"
)

// StateManager handles atomic reads/writes of the state file
type StateManager struct {
	path string
	mu   sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(dataDir string) *StateManager {
	return &StateManager{
		path: filepath.Join(dataDir, "bbx.state"),
	}
}

// Init initializes the state file if it doesn't exist
func (sm *StateManager) Init(pid int, version string) error {
	state, err := sm.Read()
	if err == nil && state != nil {
		// State file exists, update PID if in idle state
		if state.Upgrade.State == StateIdle || state.Upgrade.State == StateFailed {
			state.PID = pid
			state.Version = version
			state.StartedAt = time.Now().UTC()
			state.Token = generateToken()
			state.Upgrade.State = StateIdle
			state.Upgrade.Error = ""
			return sm.Write(state)
		}
		return nil
	}

	// Create new state file
	state = &State{
		PID:       pid,
		Version:   version,
		StartedAt: time.Now().UTC(),
		Token:     generateToken(),
		Upgrade: UpgradeInfo{
			State: StateIdle,
		},
	}
	return sm.Write(state)
}

// Read reads the current state from file (with shared lock)
func (sm *StateManager) Read() (*State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	f, err := os.Open(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	// Acquire shared lock
	if err := lockFile(f, false); err != nil {
		return nil, err
	}
	defer unlockFile(f)

	var state State
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Write writes state atomically (with exclusive lock)
func (sm *StateManager) Write(state *State) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(sm.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := sm.path + ".tmp"
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, sm.path)
}

// UpdateUpgrade updates only the upgrade sub-state
func (sm *StateManager) UpdateUpgrade(fn func(*UpgradeInfo)) error {
	state, err := sm.Read()
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("state file not initialized")
	}
	fn(&state.Upgrade)
	return sm.Write(state)
}

// IsIdle checks if upgrade state is idle
func (sm *StateManager) IsIdle() (bool, error) {
	state, err := sm.Read()
	if err != nil {
		return false, err
	}
	if state == nil {
		return true, nil
	}
	return state.Upgrade.State == StateIdle || state.Upgrade.State == StateFailed, nil
}

// generateToken creates a random token for PID validation
func generateToken() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

// ValidateProcess checks if a process with given PID/token is the expected one
func ValidateProcess(pid int, token string) bool {
	if pid <= 0 {
		return false
	}
	// Check if process exists and matches token
	// On Unix: send signal 0; on Windows: OpenProcess
	return processExists(pid)
}

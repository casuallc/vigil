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
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Env vars used for upgrade communication
const (
	EnvUpgradeMode     = "BBX_UPGRADE"
	EnvOldPID          = "BBX_OLD_PID"
	EnvConfigPath      = "BBX_CONFIG_PATH"
	EnvInternalAddr    = "BBX_INTERNAL_ADDR"
	EnvNewBinaryPath   = "BBX_NEW_BINARY_PATH"
	EnvOldToken        = "BBX_OLD_TOKEN"
)

// Upgrade orchestrates the graceful upgrade process
type Upgrade struct {
	stateManager  *StateManager
	binaryManager *BinaryManager
	dataDir       string
}

// NewUpgrade creates a new upgrade orchestrator
func NewUpgrade(dataDir string) *Upgrade {
	return &Upgrade{
		stateManager:  NewStateManager(dataDir),
		binaryManager: NewBinaryManager(dataDir),
		dataDir:       dataDir,
	}
}

// StateManager returns the state manager
func (u *Upgrade) StateManager() *StateManager {
	return u.stateManager
}

// BinaryManager returns the binary manager
func (u *Upgrade) BinaryManager() *BinaryManager {
	return u.binaryManager
}

// IsUpgradeMode checks if current process is running in upgrade mode (new process)
func IsUpgradeMode() bool {
	return os.Getenv(EnvUpgradeMode) == "1"
}

// GetOldPID returns the old process PID from env
func GetOldPID() int {
	var pid int
	fmt.Sscanf(os.Getenv(EnvOldPID), "%d", &pid)
	return pid
}

// GetOldToken returns the old process token from env
func GetOldToken() string {
	return os.Getenv(EnvOldToken)
}

// GetConfigPath returns the config path from env
func GetConfigPath() string {
	path := os.Getenv(EnvConfigPath)
	if path == "" {
		return "conf/config.yaml"
	}
	return path
}

// GetInternalAddr returns the internal address from env
func GetInternalAddr() string {
	addr := os.Getenv(EnvInternalAddr)
	if addr == "" {
		return ":57576"
	}
	return addr
}

// GetNewBinaryPath returns the new binary path from env
func GetNewBinaryPath() string {
	path := os.Getenv(EnvNewBinaryPath)
	if path == "" {
		return filepath.Join("data", "bbx-server.new")
	}
	return path
}

// StartUpgrade initiates the upgrade process from the old process
func (u *Upgrade) StartUpgrade(binaryPath string, r io.Reader, checksum string) error {
	// 1. Check if already upgrading
	isIdle, err := u.stateManager.IsIdle()
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if !isIdle {
		state, _ := u.stateManager.Read()
		if state != nil {
			return fmt.Errorf("upgrade already in progress: %s", state.Upgrade.State)
		}
		return fmt.Errorf("upgrade already in progress")
	}

	// 2. Mark downloading
	if err := u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateDownloading
		ui.StartedAt = time.Now().UTC()
		ui.Error = ""
	}); err != nil {
		return err
	}

	// 3. Save binary
	var savedPath string
	if r != nil {
		savedPath, err = u.binaryManager.SaveFromReader(r)
	} else if binaryPath != "" {
		savedPath = binaryPath
	} else {
		u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "no binary provided"
		})
		return fmt.Errorf("no binary provided")
	}
	if err != nil {
		u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = err.Error()
		})
		return fmt.Errorf("failed to save binary: %w", err)
	}

	// 4. Mark verifying
	if err := u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateVerifying
		ui.BinaryPath = savedPath
	}); err != nil {
		return err
	}

	// 5. Verify binary
	if err := u.binaryManager.VerifyExecutable(savedPath); err != nil {
		u.binaryManager.Cleanup()
		u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "verification failed: " + err.Error()
		})
		return fmt.Errorf("binary verification failed: %w", err)
	}

	if checksum != "" {
		if err := u.binaryManager.VerifyChecksum(savedPath, checksum); err != nil {
			u.binaryManager.Cleanup()
			u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
				ui.State = StateFailed
				ui.Error = "checksum failed: " + err.Error()
			})
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 6. Mark starting and fork new process
	if err := u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateStarting
	}); err != nil {
		return err
	}

	oldState, err := u.stateManager.Read()
	if err != nil || oldState == nil {
		u.binaryManager.Cleanup()
		u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "failed to read state before fork"
		})
		return fmt.Errorf("failed to read state before fork")
	}

	// Prepare env for new process
	configPath := os.Getenv("BBX_CONFIG_PATH")
	if configPath == "" {
		configPath = "conf/config.yaml"
	}

	newEnv := append(os.Environ(),
		fmt.Sprintf("%s=1", EnvUpgradeMode),
		fmt.Sprintf("%s=%d", EnvOldPID, oldState.PID),
		fmt.Sprintf("%s=%s", EnvConfigPath, configPath),
		fmt.Sprintf("%s=%s", EnvInternalAddr, ":57576"),
		fmt.Sprintf("%s=%s", EnvNewBinaryPath, savedPath),
		fmt.Sprintf("%s=%s", EnvOldToken, oldState.Token),
	)

	cmd := exec.Command(savedPath)
	cmd.Env = newEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(savedPath)

	if err := cmd.Start(); err != nil {
		u.binaryManager.Cleanup()
		u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "fork failed: " + err.Error()
		})
		return fmt.Errorf("failed to start new process: %w", err)
	}

	newPID := cmd.Process.Pid
	log.Printf("[upgrade] New process started with PID %d", newPID)

	// 7. Mark health checking
	if err := u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateHealthChecking
		ui.NewPID = newPID
	}); err != nil {
		return err
	}

	// 8. Start monitoring in background
	go u.monitorUpgrade(newPID, oldState.PID, oldState.Token)

	return nil
}

// monitorUpgrade monitors the upgrade process from the old process
func (u *Upgrade) monitorUpgrade(newPID, oldPID int, oldToken string) {
	// 60s total timeout for entire upgrade
	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			log.Printf("[upgrade] Timeout reached, killing new process %d", newPID)
			if processExists(newPID) {
				process, _ := os.FindProcess(newPID)
				if process != nil {
					process.Kill()
				}
			}
			u.binaryManager.Cleanup()
			u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
				ui.State = StateFailed
				ui.Error = "upgrade timed out after 60s"
			})
			return

		case <-ticker.C:
			state, err := u.stateManager.Read()
			if err != nil {
				continue
			}
			if state == nil {
				continue
			}

			switch state.Upgrade.State {
			case StateReady:
				// New process is healthy, start draining
				log.Printf("[upgrade] New process %d is ready, starting drain", newPID)
				// The actual draining is handled by the caller (server.Stop() and exit)
				return

			case StateCompleted:
				// Handoff complete, old process should exit
				log.Printf("[upgrade] Handoff completed, old process exiting")
				return

			case StateFailed:
				log.Printf("[upgrade] Upgrade failed: %s", state.Upgrade.Error)
				if processExists(newPID) {
					process, _ := os.FindProcess(newPID)
					if process != nil {
						process.Kill()
					}
				}
				u.binaryManager.Cleanup()
				return

			default:
				// Still in progress, check if new process is alive
				if !processExists(newPID) {
					log.Printf("[upgrade] New process %d died unexpectedly", newPID)
					u.binaryManager.Cleanup()
					u.stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
						ui.State = StateFailed
						ui.Error = "new process exited unexpectedly"
					})
					return
				}
			}
		}
	}
}

// RunNewProcess handles the new process side of the upgrade
func RunNewProcess(startServer func(string) error, stopFunc func()) error {
	if !IsUpgradeMode() {
		return fmt.Errorf("not in upgrade mode")
	}

	oldPID := GetOldPID()
	oldToken := GetOldToken()
	internalAddr := GetInternalAddr()

	log.Printf("[upgrade] Running as new process, old PID=%d", oldPID)

	// 1. Read current state
	dataDir := "data"
	stateManager := NewStateManager(dataDir)

	state, err := stateManager.Read()
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("state file not found")
	}

	// 2. Validate we are the expected new process
	if state.Upgrade.NewPID != os.Getpid() {
		return fmt.Errorf("PID mismatch: expected new_pid=%d, actual=%d", state.Upgrade.NewPID, os.Getpid())
	}

	// 3. Start internal HTTP server
	log.Printf("[upgrade] Starting internal server on %s", internalAddr)

	// Use a minimal mux just for health
	mux := http.NewServeMux()
	server := &http.Server{Addr: internalAddr, Handler: mux}

	listener, err := net.Listen("tcp", internalAddr)
	if err != nil {
		stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "failed to bind internal address: " + err.Error()
		})
		return fmt.Errorf("failed to bind internal address: %w", err)
	}

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	go server.Serve(listener)
	defer server.Close()

	// 4. Start the actual server components (this blocks until ready)
	log.Printf("[upgrade] Initializing server components...")
	if err := startServer(internalAddr); err != nil {
		stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "server init failed: " + err.Error()
		})
		return fmt.Errorf("server initialization failed: %w", err)
	}

	// 5. Self health check
	hc := NewHealthChecker(internalAddr)
	if err := hc.WaitForReady(5 * time.Second); err != nil {
		stopFunc()
		stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "health check failed: " + err.Error()
		})
		return fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("[upgrade] Health check passed")

	// 6. Mark ready
	if err := stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateReady
	}); err != nil {
		stopFunc()
		return fmt.Errorf("failed to update state: %w", err)
	}

	// 7. Wait for old process to release port
	log.Printf("[upgrade] Waiting for old process to release port...")
	if err := waitForPortRelease(oldPID, oldToken, internalAddr); err != nil {
		log.Printf("[upgrade] Warning: %v", err)
	}

	// 8. Bind main port (with retry)
	mainAddr := ":57575"
	if err := bindMainPort(mainAddr, 10); err != nil {
		stopFunc()
		stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
			ui.State = StateFailed
			ui.Error = "failed to bind main port: " + err.Error()
		})
		return fmt.Errorf("failed to bind main port: %w", err)
	}

	// 9. Mark completed
	if err := stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateCompleted
		ui.CompletedAt = time.Now().UTC()
	}); err != nil {
		log.Printf("[upgrade] Warning: failed to mark completed: %v", err)
	}

	// 10. Wait for old process to exit
	log.Printf("[upgrade] Waiting for old process %d to exit...", oldPID)
	waitForProcessExit(oldPID, 30*time.Second)

	// 11. Replace binary (Windows: old process exited, file lock released)
	newBinaryPath := GetNewBinaryPath()
	exePath, _ := os.Executable()
	if exePath != "" && newBinaryPath != "" {
		bm := NewBinaryManager(dataDir)
		if err := bm.Replace(exePath); err != nil {
			log.Printf("[upgrade] Warning: failed to replace binary: %v", err)
		} else {
			log.Printf("[upgrade] Binary replaced successfully")
		}
	}

	// 12. Reset state to idle
	if err := stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateIdle
		ui.NewPID = 0
		ui.StartedAt = time.Time{}
		ui.CompletedAt = time.Time{}
		ui.Error = ""
		ui.BinaryPath = ""
	}); err != nil {
		log.Printf("[upgrade] Warning: failed to reset state: %v", err)
	}

	log.Printf("[upgrade] Upgrade complete, now serving on main port")
	return nil
}

// waitForPortRelease waits for the old process to stop listening
func waitForPortRelease(oldPID int, oldToken, internalAddr string) error {
	// Poll until old process exits or timeout
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if !ValidateProcess(oldPID, oldToken) {
			return nil // old process exited
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for old process to exit")
}

// bindMainPort attempts to bind the main port with retries
func bindMainPort(addr string, maxRetries int) error {
	delay := time.Second
	for i := 0; i < maxRetries; i++ {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2
			if delay > 8*time.Second {
				delay = 8 * time.Second
			}
		}
	}
	return fmt.Errorf("failed to bind %s after %d retries", addr, maxRetries)
}

// waitForProcessExit waits for a process to exit with timeout
func waitForProcessExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// EnableWAL enables SQLite WAL mode on a database connection
func EnableWAL(db interface{ Exec(string, ...interface{}) (interface{}, error) }) error {
	_, err := db.Exec("PRAGMA journal_mode=WAL;")
	return err
}

// RunAsNewProcess is a helper that wraps the full new-process flow.
// It should be called from main() when IsUpgradeMode() returns true.
func RunAsNewProcess(ctx context.Context, internalAddr string, startServer func() error, stopFunc func()) error {
	log.Printf("[upgrade] Running in upgrade mode")

	oldPID := GetOldPID()
	dataDir := "data"
	stateManager := NewStateManager(dataDir)

	// Mark ready after health check
	// First, the caller should start the internal server and do health check
	// Then call this to update state and wait for handoff

	// Update state to ready
	if err := stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateReady
	}); err != nil {
		return fmt.Errorf("failed to mark ready: %w", err)
	}

	// Wait for old process to exit
	waitForProcessExit(oldPID, 30*time.Second)

	// Replace binary
	newBinaryPath := GetNewBinaryPath()
	exePath, _ := os.Executable()
	if exePath != "" && newBinaryPath != "" {
		bm := NewBinaryManager(dataDir)
		if err := bm.Replace(exePath); err != nil {
			log.Printf("[upgrade] Warning: failed to replace binary: %v", err)
		}
	}

	// Reset state
	stateManager.UpdateUpgrade(func(ui *UpgradeInfo) {
		ui.State = StateIdle
		ui.NewPID = 0
		ui.StartedAt = time.Time{}
		ui.CompletedAt = time.Time{}
		ui.Error = ""
		ui.BinaryPath = ""
	})

	return nil
}

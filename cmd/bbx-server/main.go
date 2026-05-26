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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/casuallc/vigil/api"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/proc"
	"github.com/casuallc/vigil/upgrade"
	"github.com/casuallc/vigil/version"
)

func main() {
	// Parse command line arguments
	var (
		configPath  string
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "", "Config file path")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Show version information if requested
	if showVersion {
		fmt.Println(version.GetVersionInfo())
		return
	}

	// Set default config file path
	if configPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to get executable path: %v", err)
		}
		configPath = filepath.Join(filepath.Dir(exePath), "./conf/config.yaml")
	}

	// Load config file
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config file: %v", err)
		return
	}

	// Common initialization
	processManager, processStore := initProcessManager()
	server := api.NewServerWithManager(cfg, processManager, configPath)

	// Check if running in upgrade mode (new process)
	if upgrade.IsUpgradeMode() {
		runAsUpgradeProcess(server, processManager, processStore)
		return
	}

	// Normal mode: initialize state file
	stateManager := upgrade.NewStateManager("data")
	if err := stateManager.Init(os.Getpid(), version.Version); err != nil {
		log.Printf("Warning: failed to initialize state file: %v", err)
	}

	// Start the server
	runNormalServer(server, processManager, processStore)
}

// initProcessManager initializes the process manager and store
func initProcessManager() (*proc.Manager, *proc.ProcessStore) {
	processManager := proc.NewManager()
	dbPath := "data/vigil.db"
	processStore, err := proc.NewProcessStore(dbPath)
	if err != nil {
		log.Printf("Warning: failed to create process store: %v", err)
		return processManager, nil
	}

	processManager.SetStore(processStore)
	if err := processStore.LoadManagedProcesses(processManager); err != nil {
		log.Printf("Warning: failed to load managed processes: %v", err)
	}

	return processManager, processStore
}

// runAsUpgradeProcess handles the new process side of graceful upgrade
func runAsUpgradeProcess(server *api.Server, processManager *proc.Manager, processStore *proc.ProcessStore) {
	log.Printf("[upgrade] Running in upgrade mode")

	oldPID := upgrade.GetOldPID()
	internalAddr := upgrade.GetInternalAddr()
	dataDir := "data"

	stateManager := upgrade.NewStateManager(dataDir)

	// 1. Read current state and validate
	state, err := stateManager.Read()
	if err != nil || state == nil {
		log.Fatalf("[upgrade] Failed to read state: %v", err)
	}

	if state.Upgrade.NewPID != os.Getpid() {
		log.Fatalf("[upgrade] PID mismatch: expected new_pid=%d, actual=%d", state.Upgrade.NewPID, os.Getpid())
	}

	// 2. Start internal HTTP server for health checks
	internalMux := http.NewServeMux()
	internalMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	internalServer := &http.Server{Addr: internalAddr, Handler: internalMux}
	go func() {
		log.Printf("[upgrade] Internal health server starting on %s", internalAddr)
		if err := internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[upgrade] Internal server error: %v", err)
		}
	}()
	defer internalServer.Close()

	// 3. Mark health_checking
	if err := stateManager.UpdateUpgrade(func(ui *upgrade.UpgradeInfo) {
		ui.State = upgrade.StateHealthChecking
	}); err != nil {
		log.Fatalf("[upgrade] Failed to update state: %v", err)
	}

	// 4. Self health check
	hc := upgrade.NewHealthChecker(internalAddr)
	if err := hc.WaitForReady(5 * time.Second); err != nil {
		stateManager.UpdateUpgrade(func(ui *upgrade.UpgradeInfo) {
			ui.State = upgrade.StateFailed
			ui.Error = "health check failed: " + err.Error()
		})
		log.Fatalf("[upgrade] Health check failed: %v", err)
	}
	log.Printf("[upgrade] Health check passed")

	// 5. Mark ready
	if err := stateManager.UpdateUpgrade(func(ui *upgrade.UpgradeInfo) {
		ui.State = upgrade.StateReady
	}); err != nil {
		log.Fatalf("[upgrade] Failed to mark ready: %v", err)
	}

	// 6. Wait for old process to exit
	log.Printf("[upgrade] Waiting for old process %d to exit...", oldPID)
	waitForProcessExit(oldPID, 30*time.Second)

	// 7. Replace binary (Windows: old process exited, file lock released)
	go func() {
		newBinaryPath := upgrade.GetNewBinaryPath()
		exePath, err := os.Executable()
		if err != nil || exePath == "" || newBinaryPath == "" {
			return
		}
		bm := upgrade.NewBinaryManager(dataDir)
		if err := bm.Replace(exePath); err != nil {
			log.Printf("[upgrade] Warning: failed to replace binary: %v", err)
		} else {
			log.Printf("[upgrade] Binary replaced successfully")
		}
	}()

	// 8. Reset upgrade state to idle
	if err := stateManager.UpdateUpgrade(func(ui *upgrade.UpgradeInfo) {
		ui.State = upgrade.StateIdle
		ui.NewPID = 0
		ui.StartedAt = time.Time{}
		ui.CompletedAt = time.Time{}
		ui.Error = ""
		ui.BinaryPath = ""
	}); err != nil {
		log.Printf("[upgrade] Warning: failed to reset state: %v", err)
	}

	log.Printf("[upgrade] Handoff complete, starting main server")

	// 9. Start main server (blocks)
	startAndWait(server, processManager, processStore)
}

// runNormalServer runs the server in normal mode
func runNormalServer(server *api.Server, processManager *proc.Manager, processStore *proc.ProcessStore) {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for termination signal or server error
	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down...", sig)

		// Save managed processes before shutdown
		if err := processManager.SaveManagedProcesses(""); err != nil {
			log.Printf("Warning: failed to save managed processes during shutdown: %v", err)
		}

		// Close storage
		if processStore != nil {
			processStore.Close()
		}

		// Graceful shutdown
		time.Sleep(1 * time.Second)
	}
}

// startAndWait starts the server and waits for shutdown signal
func startAndWait(server *api.Server, processManager *proc.Manager, processStore *proc.ProcessStore) {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for termination signal or server error
	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down...", sig)

		// Save managed processes before shutdown
		if err := processManager.SaveManagedProcesses(""); err != nil {
			log.Printf("Warning: failed to save managed processes during shutdown: %v", err)
		}

		// Close storage
		if processStore != nil {
			processStore.Close()
		}

		// Graceful shutdown
		time.Sleep(1 * time.Second)
	}
}

// waitForProcessExit waits for a process to exit with timeout
func waitForProcessExit(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !upgrade.ValidateProcess(pid, "") {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Printf("[upgrade] Timeout waiting for process %d to exit", pid)
}

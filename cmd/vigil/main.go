package main

import (
  "flag"
  "github.com/casuallc/vigil/api"
  "github.com/casuallc/vigil/config"
  "log"
  "os"
  "os/signal"
  "path/filepath"
  "syscall"
)

func main() {
  // Parse command line arguments
  var (
    configPath string
    apiAddr    string
    foreground bool
  )

  flag.StringVar(&configPath, "config", "", "Config file path")
  flag.StringVar(&apiAddr, "addr", ":8080", "API server address")
  flag.BoolVar(&foreground, "foreground", false, "Run in foreground mode")
  flag.Parse()

  // Set default config file path
  if configPath == "" {
    // Get executable directory
    exePath, err := os.Executable()
    if err != nil {
      log.Fatalf("Failed to get executable path: %v", err)
    }
    configPath = filepath.Join(filepath.Dir(exePath), "config.yaml")
  }

  // Load config file
  cfg, err := config.LoadConfig(configPath)
  if err != nil {
    log.Printf("Failed to load config file, using default config: %v", err)
    cfg = config.DefaultConfig()
  }

  // If not running in foreground, daemonize the process
  if !foreground {
    // Here we would typically daemonize the process
    // For simplicity, we'll just log that we would do this
    log.Println("Starting in daemon mode (implementation simplified)")
  }

  // Create and start the API server
  server := api.NewServer(cfg)

  // Setup signal handling
  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

  // Start the server in a goroutine
  serverErr := make(chan error, 1)
  go func() {
    serverErr <- server.Start(apiAddr)
  }()

  // Wait for termination signal or server error
  select {
  case err := <-serverErr:
    log.Fatalf("Server error: %v", err)
  case sig := <-sigChan:
    log.Printf("Received signal %s, shutting down...", sig)
    // Here we would implement graceful shutdown
  }
}

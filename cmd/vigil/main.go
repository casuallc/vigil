package main

import (
  "flag"
  "github.com/casuallc/vigil/api"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/process/core"
  "log"
  "os"
  "os/signal"
  "path/filepath"
  "syscall"
  "time"
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

  // 创建进程管理器
  processManager := core.NewManager()

  // 加载已保存的进程信息
  ProcessesFilePath := "proc/managed_processes.yaml"
  if err := processManager.LoadManagedProcesses(ProcessesFilePath); err != nil {
    log.Printf("Warning: failed to load managed processes: %v", err)
  }

  // If not running in foreground, daemonize the proc
  if !foreground {
    // Here we would typically daemonize the proc
    // For simplicity, we'll just log that we would do this
    log.Println("Starting in daemon mode (implementation simplified)")
  }

  // Create and start the API server with the loaded proc manager
  server := api.NewServerWithManager(cfg, processManager)

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

    // 在关闭前保存进程信息
    if err := processManager.SaveManagedProcesses(ProcessesFilePath); err != nil {
      log.Printf("Warning: failed to save managed processes during shutdown: %v", err)
    }

    // 这里应该实现优雅关闭
    time.Sleep(1 * time.Second) // 给保存操作一点时间
  }
}

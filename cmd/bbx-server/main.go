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
  "github.com/casuallc/vigil/api"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/proc"
  "github.com/casuallc/vigil/version"
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
    // Get executable directory
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

  // 创建进程管理器
  processManager := proc.NewManager()

  // 创建进程存储管理器（使用 SQLite）
  dbPath := "data/vigil.db"
  processStore, err := proc.NewProcessStore(dbPath)
  if err != nil {
    log.Printf("Warning: failed to create process store: %v", err)
  } else {
    // 设置管理器的存储
    processManager.SetStore(processStore)

    // 从数据库加载进程信息
    if err := processStore.LoadManagedProcesses(processManager); err != nil {
      log.Printf("Warning: failed to load managed processes: %v", err)
    }
  }

  // Create and start the API server with the loaded proc manager
  server := api.NewServerWithManager(cfg, processManager)

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

    // 在关闭前保存进程信息
    if err := processManager.SaveManagedProcesses(""); err != nil {
      log.Printf("Warning: failed to save managed processes during shutdown: %v", err)
    }

    // 关闭存储
    if processStore != nil {
      processStore.Close()
    }

    // 优雅关闭
    time.Sleep(1 * time.Second)
  }
}

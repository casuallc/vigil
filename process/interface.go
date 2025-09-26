package process

import (
  "github.com/casuallc/vigil/config"
)

// ProcManager is the core interface for process management
type ProcManager interface {
  // ScanProcesses Scan processes
  ScanProcesses(query string) ([]ManagedProcess, error)
  // ManageProcess Manage a process
  ManageProcess(process ManagedProcess) error
  // StartProcess Start a process
  StartProcess(name string) error
  // StopProcess Stop a process
  StopProcess(name string) error
  // RestartProcess Restart a process
  RestartProcess(name string) error
  // GetProcessStatus Get process status
  GetProcessStatus(name string) (ManagedProcess, error)
  // ListManagedProcesses Get all managed processes
  ListManagedProcesses() ([]ManagedProcess, error)
  // MonitorProcess Monitor process resources
  MonitorProcess(name string) (ResourceStats, error)
  // UpdateProcessConfig Update process configuration
  UpdateProcessConfig(name string, config config.AppConfig) error
}

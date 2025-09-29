package process

import (
  "github.com/casuallc/vigil/config"
)

// ProcManager is the core interface for process management
type ProcManager interface {
  // ScanProcesses Scan processes
  ScanProcesses(query string) ([]ManagedProcess, error)
  // ManageProcess Manage a process
  CreateProcess(process ManagedProcess) error
  // StartProcess Start a process
  StartProcess(namespace, name string) error
  // StopProcess Stop a process
  StopProcess(namespace, name string) error
  // RestartProcess Restart a process
  RestartProcess(namespace, name string) error
  // GetProcessStatus Get process status
  GetProcessStatus(namespace, name string) (ManagedProcess, error)
  // ListManagedProcesses Get all managed processes
  ListManagedProcesses(namespace string) ([]ManagedProcess, error)
  // MonitorProcess Monitor process resources
  MonitorProcess(namespace, name string) (*ResourceStats, error)
  // UpdateProcessConfig Update process configuration
  UpdateProcessConfig(namespace, name string, config config.AppConfig) error
  // DeleteProcess Delete a managed process
  DeleteProcess(namespace, name string) error
}

package process

import (
  "time"

  "github.com/casuallc/vigil/config"
)

// Manager is the implementation of ProcManager
type Manager struct {
  processes map[string]*ManagedProcess
}

// Status represents the status of a process
type Status string

const (
  StatusRunning  Status = "running"
  StatusStopped  Status = "stopped"
  StatusStarting Status = "starting"
  StatusStopping Status = "stopping"
  StatusFailed   Status = "failed"
  StatusUnknown  Status = "unknown"
)

// PortInfo 定义端口监听信息结构体
type PortInfo struct {
  Port     int    `json:"port" yaml:"port"`         // 端口号
  Protocol string `json:"protocol" yaml:"protocol"` // 协议类型（TCP/UDP）
  Address  string `json:"address" yaml:"address"`   // 绑定的IP地址
}

// ResourceStats represents process resource usage statistics
type ResourceStats struct {
  CPUUsage       float64    `json:"cpu_usage" yaml:"cpu_usage"`             // CPU usage (percentage)
  MemoryUsage    uint64     `json:"memory_usage" yaml:"memory_usage"`       // Memory usage (bytes)
  DiskIO         uint64     `json:"disk_io" yaml:"disk_io"`                 // Disk IO (bytes)
  NetworkIO      uint64     `json:"network_io" yaml:"network_io"`           // Network IO (bytes)
  ListeningPorts []PortInfo `json:"listening_ports" yaml:"listening_ports"` // 监听端口信息
}

// RestartPolicy represents the restart policy for a process
type RestartPolicy string

const (
  RestartPolicyAlways    RestartPolicy = "always"     // 总是重启
  RestartPolicyOnFailure RestartPolicy = "on-failure" // 失败时重启
  RestartPolicyNever     RestartPolicy = "never"      // 从不重启
  RestartPolicyOnSuccess RestartPolicy = "on-success" // 成功时重启
)

// EnvironmentVariable 定义环境变量的结构体
type EnvironmentVariable struct {
  Name  string `json:"name" yaml:"name"`   // 环境变量名称
  Value string `json:"value" yaml:"value"` // 环境变量值
}

// CommandConfig 定义命令配置结构体，包含命令内容和超时时间
type CommandConfig struct {
  Command string        `json:"command" yaml:"command"` // 命令内容
  Args    []string      `json:"args" yaml:"args"`       // 命令参数
  Timeout time.Duration `json:"timeout" yaml:"timeout"` // 命令执行超时时间
}

// ManagedProcess represents a managed process
type ManagedProcess struct {
  ID              string                `json:"id" yaml:"id"`
  Name            string                `json:"name" yaml:"name"`
  PID             int                   `json:"pid" yaml:"pid"`
  Status          Status                `json:"status" yaml:"status"`
  Command         CommandConfig         `json:"command_config" yaml:"command_config"`
  Env             []EnvironmentVariable `json:"env" yaml:"env"`
  WorkingDir      string                `json:"working_dir" yaml:"working_dir"`
  StartTime       time.Time             `json:"start_time" yaml:"start_time"`
  LastExitCode    int                   `json:"last_exit_code" yaml:"last_exit_code"`
  RestartCount    int                   `json:"restart_count" yaml:"restart_count"`
  Config          config.AppConfig      `json:"config" yaml:"config"`
  Stats           ResourceStats         `json:"stats" yaml:"stats"`
  StartCommand    CommandConfig         `json:"start_command_config" yaml:"start_command_config"`
  StopCommand     CommandConfig         `json:"stop_command_config" yaml:"stop_command_config"`
  RestartPolicy   RestartPolicy         `json:"restart_policy" yaml:"restart_policy"`
  LogDir          string                `json:"log_dir" yaml:"log_dir"`
  MaxRestarts     int                   `json:"max_restarts" yaml:"max_restarts"`         // 最大重启次数
  RestartInterval time.Duration         `json:"restart_interval" yaml:"restart_interval"` // 重启时间间隔
  User            string                `json:"user" yaml:"user"`                         // 进程所属用户
  UserGroup       string                `json:"user_group" yaml:"user_group"`             // 进程所属用户组
}

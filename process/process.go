package process

import (
	"errors"
	"fmt"
	"time"

	"github.com/casuallc/vigil/config"
)

// ProcessStatus represents the status of a process

type ProcessStatus string

const (
	StatusRunning  ProcessStatus = "running"
	StatusStopped  ProcessStatus = "stopped"
	StatusStarting ProcessStatus = "starting"
	StatusStopping ProcessStatus = "stopping"
	StatusFailed   ProcessStatus = "failed"
	StatusUnknown  ProcessStatus = "unknown"
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

// 定义重启策略枚举类型

type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "always"     // 总是重启
	RestartPolicyOnFailure RestartPolicy = "on-failure" // 失败时重启
	RestartPolicyNever     RestartPolicy = "never"      // 从不重启
	RestartPolicyOnSuccess RestartPolicy = "on-success" // 成功时重启
)

// 首先定义环境变量的结构体

type EnvironmentVariable struct {
	Name  string `json:"name" yaml:"name"`   // 环境变量名称
	Value string `json:"value" yaml:"value"` // 环境变量值
}

// 定义命令配置结构体，包含命令内容和超时时间

type CommandConfig struct {
	Command string        `json:"command" yaml:"command"` // 命令内容
	Args    []string      `json:"args" yaml:"args"`       // 命令参数
	Timeout time.Duration `json:"timeout" yaml:"timeout"` // 命令执行超时时间
}

// ManagedProcess represents a managed process

type ManagedProcess struct {
	ID     string        `json:"id" yaml:"id"`
	Name   string        `json:"name" yaml:"name"`
	PID    int           `json:"pid" yaml:"pid"`
	Status ProcessStatus `json:"status" yaml:"status"`
	// 使用 CommandConfig 结构体替代原来的 Command 和 Args
	Command CommandConfig `json:"command_config" yaml:"command_config"`
	// 将环境变量从字符串数组改为结构体数组，便于明确指定名称和值
	Env          []EnvironmentVariable `json:"env" yaml:"env"`
	WorkingDir   string                `json:"working_dir" yaml:"working_dir"`
	StartTime    time.Time             `json:"start_time" yaml:"start_time"`
	LastExitCode int                   `json:"last_exit_code" yaml:"last_exit_code"`
	RestartCount int                   `json:"restart_count" yaml:"restart_count"`
	Config       config.AppConfig      `json:"config" yaml:"config"`
	Stats        ResourceStats         `json:"stats" yaml:"stats"`
	// 使用 CommandConfig 结构体替代原来的 StartCommand 和 StopCommand
	StartCommand    CommandConfig `json:"start_command_config" yaml:"start_command_config"`
	StopCommand     CommandConfig `json:"stop_command_config" yaml:"stop_command_config"`
	RestartPolicy   RestartPolicy `json:"restart_policy" yaml:"restart_policy"`
	LogDir          string        `json:"log_dir" yaml:"log_dir"`
	MaxRestarts     int           `json:"max_restarts" yaml:"max_restarts"`         // 最大重启次数
	RestartInterval time.Duration `json:"restart_interval" yaml:"restart_interval"` // 重启时间间隔
}

// ProcessManager is the core interface for process management

type ProcessManager interface {
	// Scan processes
	ScanProcesses(query string) ([]ManagedProcess, error)
	// Manage a process
	ManageProcess(process ManagedProcess) error
	// Start a process
	StartProcess(name string) error
	// Stop a process
	StopProcess(name string) error
	// Restart a process
	RestartProcess(name string) error
	// Get process status
	GetProcessStatus(name string) (ManagedProcess, error)
	// Get all managed processes
	ListManagedProcesses() ([]ManagedProcess, error)
	// Monitor process resources
	MonitorProcess(name string) (ResourceStats, error)
	// Update process configuration
	UpdateProcessConfig(name string, config config.AppConfig) error
}

// Manager is the implementation of ProcessManager

type Manager struct {
	processes map[string]*ManagedProcess
}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*ManagedProcess),
	}
}

// GetProcessStatus implements ProcessManager interface
func (m *Manager) GetProcessStatus(name string) (ManagedProcess, error) {
	process, exists := m.processes[name]
	if !exists {
		return ManagedProcess{}, errors.New(fmt.Sprintf("Process %s is not managed", name))
	}
	return *process, nil
}

// ListManagedProcesses implements ProcessManager interface
func (m *Manager) ListManagedProcesses() ([]ManagedProcess, error) {
	result := make([]ManagedProcess, 0, len(m.processes))
	for _, p := range m.processes {
		result = append(result, *p)
	}
	return result, nil
}

// MonitorProcess implements ProcessManager interface
func (m *Manager) MonitorProcess(name string) (ResourceStats, error) {
	// Implementation continues
	return ResourceStats{}, nil
}

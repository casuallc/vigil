package process

import (
  "time"

  "github.com/casuallc/vigil/config"
)

// Manager is the implementation of ProcManager
type Manager struct {
  processes map[string]*ManagedProcess
}

// ManagedProcess 是对一个进程的完整声明式描述，包含 Spec（期望状态）和 Status（实际状态）
type ManagedProcess struct {
  // Metadata 包含进程的元信息，如名称、标签、注解等
  Metadata Metadata `json:"metadata" yaml:"metadata"`

  // Spec 定义了进程的期望状态（用户配置）
  Spec Spec `json:"spec" yaml:"spec"`

  // Status 描述进程的当前运行状态（由系统填充）
  Status Status `json:"status" yaml:"status"`
}

// Metadata 包含进程的标识和元数据，参考 Kubernetes ObjectMeta
type Metadata struct {

  // 唯一标识
  ID string `json:"id" yaml:"id"`

  // 命名空间
  Namespace string `json:"namespace" yaml:"namespace"`

  // Name 是进程的唯一标识（在 namespace 范围内）
  Name string `json:"name" yaml:"name"`

  // Labels 是键值对标签，用于分组、筛选
  Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

  // Annotations 是非标识性元数据，用于存储额外信息（如构建信息、描述等）
  Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`

  // CreationTimestamp 表示该进程配置的创建时间
  CreationTimestamp time.Time `json:"creation_timestamp,omitempty" yaml:"creation_timestamp,omitempty"`
}

// Spec 定义进程的期望状态（用户可配置部分）
type Spec struct {
  // Exec 定义如何启动进程
  Exec Exec `json:"exec" yaml:"exec"`

  // WorkingDir 是进程的工作目录
  WorkingDir string `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`

  // User 和 UserGroup 指定运行用户
  User      string `json:"user,omitempty" yaml:"user,omitempty"`
  UserGroup string `json:"user_group,omitempty" yaml:"user_group,omitempty"`

  // Env 是环境变量列表
  Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

  // Log 配置日志输出
  Log LogConfig `json:"log,omitempty" yaml:"log,omitempty"`

  // RestartPolicy 控制重启行为
  RestartPolicy RestartPolicy `json:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`

  // MaxRestarts 是最大重启次数（配合 RestartPolicy 使用）
  MaxRestarts int32 `json:"max_restarts,omitempty" yaml:"max_restarts,omitempty"`

  // RestartInterval 是重启间隔（例如 5s）
  RestartInterval time.Duration `json:"restart_interval,omitempty" yaml:"restart_interval,omitempty"`

  // Lifecycle 钩子（可选）
  Lifecycle *Lifecycle `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`

  // HealthCheck 健康检查配置（可选）
  HealthCheck *HealthCheck `json:"health_check,omitempty" yaml:"health_check,omitempty"`

  // Resources 资源限制（预留，可后续实现）
  Resources *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`

  // AppConfig 是 Vigil 特有的应用配置
  Config config.AppConfig `json:"config,omitempty" yaml:"config,omitempty"`

  // CheckAlive 用于识别进程的脚本，返回0表示匹配成功
  CheckAlive *CommandConfig `json:"check_alive,omitempty" yaml:"check_alive,omitempty"`
}

// Exec 定义如何执行进程
type Exec struct {
  // Command 是可执行文件路径
  Command string `json:"command" yaml:"command"`

  // Args 是命令行参数
  Args []string `json:"args,omitempty" yaml:"args,omitempty"`

  // StopCommand
  StopCommand *CommandConfig `json:"stop_command" yaml:"stop_command"`
}

// EnvVar 表示一个环境变量
type EnvVar struct {
  Name  string `json:"name" yaml:"name"`
  Value string `json:"value" yaml:"value"`
}

// LogConfig 日志配置
type LogConfig struct {
  // Dir 是日志目录
  Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`

  // Stdout 和 Stderr 可分别重定向（可选）
  StdoutFile string `json:"stdout_file,omitempty" yaml:"stdout_file,omitempty"`
  StderrFile string `json:"stderr_file,omitempty" yaml:"stderr_file,omitempty"`
}

// Lifecycle 定义进程生命周期钩子
type Lifecycle struct {
  // PreStart 在进程启动前执行的命令
  PreStart *CommandConfig `json:"pre_start,omitempty" yaml:"pre_start,omitempty"`

  // PostStop 在进程停止后执行的命令
  PostStop *CommandConfig `json:"post_stop,omitempty" yaml:"post_stop,omitempty"`
}

// HealthCheck 健康检查配置
type HealthCheck struct {
  // Exec 执行命令检查健康状态
  Exec *CommandConfig `json:"exec,omitempty" yaml:"exec,omitempty"`

  // TCP 检查指定端口是否可连接
  TCP *TCPProbe `json:"tcp,omitempty" yaml:"tcp,omitempty"`

  // HTTP 检查 HTTP 端点（可选）
  HTTP *HTTPProbe `json:"http,omitempty" yaml:"http,omitempty"`

  // 初始延迟（启动后多久开始检查）
  InitialDelaySeconds int32 `json:"initial_delay_seconds,omitempty" yaml:"initial_delay_seconds,omitempty"`

  // 检查间隔
  PeriodSeconds int32 `json:"period_seconds,omitempty" yaml:"period_seconds,omitempty"`

  // 超时时间
  TimeoutSeconds int32 `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`

  // 失败阈值（连续失败多少次才认为不健康）
  FailureThreshold int32 `json:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty"`
}

// TCPProbe TCP 健康检查
type TCPProbe struct {
  Port int `json:"port" yaml:"port"`
}

// HTTPProbe HTTP 健康检查
type HTTPProbe struct {
  Port int    `json:"port" yaml:"port"`
  Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// CommandConfig 定义一个可执行命令及其超时
type CommandConfig struct {
  Command string        `json:"command" yaml:"command"`
  Args    []string      `json:"args,omitempty" yaml:"args,omitempty"`
  Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// ResourceRequirements 资源请求与限制（预留）
type ResourceRequirements struct {
  // Requests 是期望的资源
  Requests ResourceList `json:"requests,omitempty" yaml:"requests,omitempty"`

  // Limits 是资源上限
  Limits ResourceList `json:"limits,omitempty" yaml:"limits,omitempty"`
}

// ResourceList 是资源名到数量的映射（如 "cpu": "500m", "memory": "128Mi"）
type ResourceList map[string]string

// Status 表示进程的当前运行状态（由系统维护）
type Status struct {
  // Phase 是高层次状态（Running, Failed, Succeeded, Unknown）
  Phase Phase `json:"phase" yaml:"phase"`

  // Conditions 是详细状态条件列表（类似 K8s PodConditions）
  Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

  // PID 是当前进程 ID
  PID int `json:"pid,omitempty" yaml:"pid,omitempty"`

  // StartTime 是本次启动时间
  StartTime *time.Time `json:"start_time,omitempty" yaml:"start_time,omitempty"`

  // LastTerminationInfo 包含上次退出的信息
  LastTerminationInfo *TerminationInfo `json:"last_termination_info,omitempty" yaml:"last_termination_info,omitempty"`

  // RestartCount 是重启次数
  RestartCount int32 `json:"restart_count,omitempty" yaml:"restart_count,omitempty"`

  // ResourceStats 是资源使用统计
  ResourceStats *ResourceStats `json:"resource_stats,omitempty" yaml:"resource_stats,omitempty"`
}

// Phase 高层次状态
type Phase string

const (
  PhasePending  Phase = "Pending"
  PhaseRunning  Phase = "Running"
  PhaseFailed   Phase = "Failed"
  PhaseStopping Phase = "Stopping"
  PhaseStopped  Phase = "Stopped"
  PhaseUnknown  Phase = "Unknown"
)

// Condition 表示一个状态条件
type Condition struct {
  // Type 是条件类型（如 Ready, Started, Healthy）
  Type string `json:"type" yaml:"type"`

  // Status 是状态（True, False, Unknown）
  Status ConditionStatus `json:"status" yaml:"status"`

  // LastTransitionTime 上次状态变更时间
  LastTransitionTime time.Time `json:"last_transition_time,omitempty" yaml:"last_transition_time,omitempty"`

  // Reason 是状态变更的简短原因
  Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`

  // Message 是人类可读的详细信息
  Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type ConditionStatus string

const (
  ConditionTrue    ConditionStatus = "True"
  ConditionFalse   ConditionStatus = "False"
  ConditionUnknown ConditionStatus = "Unknown"
)

// TerminationInfo 记录进程退出信息
type TerminationInfo struct {
  ExitCode   int       `json:"exit_code" yaml:"exit_code"`
  Signal     int       `json:"signal,omitempty" yaml:"signal,omitempty"`
  FinishedAt time.Time `json:"finished_at" yaml:"finished_at"`
  Message    string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// ResourceStats 表示资源使用情况
type ResourceStats struct {
  CPUUsage       float64    `json:"cpu_usage" yaml:"cpu_usage"`       // 百分比
  MemoryUsage    uint64     `json:"memory_usage" yaml:"memory_usage"` // 字节
  DiskIO         uint64     `json:"disk_io" yaml:"disk_io"`           // 字节
  NetworkIO      uint64     `json:"network_io" yaml:"network_io"`     // 字节
  ListeningPorts []PortInfo `json:"listening_ports,omitempty" yaml:"listening_ports,omitempty"`
}

// PortInfo 端口监听信息
type PortInfo struct {
  Port     int    `json:"port" yaml:"port"`
  Protocol string `json:"protocol" yaml:"protocol"` // "TCP" or "UDP"
  Address  string `json:"address" yaml:"address"`   // e.g. "0.0.0.0"
}

// RestartPolicy 重启策略
type RestartPolicy string

const (
  RestartPolicyAlways    RestartPolicy = "Always"
  RestartPolicyOnFailure RestartPolicy = "OnFailure"
  RestartPolicyNever     RestartPolicy = "Never"
  RestartPolicyOnSuccess RestartPolicy = "OnSuccess"
)

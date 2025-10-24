package proc

import (
  "time"

  "github.com/casuallc/vigil/config"
)

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

  // 目录挂载配置（仅在 Linux 下有效，类似 docker volume）
  Mounts []Mount `json:"mounts,omitempty" yaml:"mounts,omitempty"`

  // User 和 UserGroup 指定运行用户
  User      string `json:"user,omitempty" yaml:"user,omitempty"`
  UserGroup string `json:"user_group,omitempty" yaml:"user_group,omitempty"`

  // Env 是环境变量列表
  Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

  // Log 配置日志输出
  Log LogConfig `json:"log,omitempty" yaml:"log,omitempty"`

  // RestartPolicy 控制重启行为
  RestartPolicy RestartPolicy `json:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`

  // RestartInterval 是重启间隔（例如 5s）
  RestartInterval time.Duration `json:"restart_interval,omitempty" yaml:"restart_interval,omitempty"`

  // Lifecycle 钩子（可选）
  Lifecycle *Lifecycle `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`

  // HealthCheck 健康检查配置（可选）
  HealthCheck *HealthCheck `json:"health_check,omitempty" yaml:"health_check,omitempty"`

  // AppConfig 是 Vigil 特有的应用配置
  Config config.AppConfig `json:"config,omitempty" yaml:"config,omitempty"`

  // CheckAlive 用于识别进程的脚本，返回0表示匹配成功
  CheckAlive *CommandConfig `json:"check_alive,omitempty" yaml:"check_alive,omitempty"`

  // Resources 资源限制（预留，可后续实现）
  Resources *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
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

// Mount 定义目录挂载映射（Linux 绑定挂载、tmpfs、命名卷）
type Mount struct {
  // Type: bind|tmpfs|named，默认 bind
  Type string `json:"type,omitempty" yaml:"type,omitempty"`
  // 唯一挂载标识（由用户指定）
  ID string `json:"id,omitempty" yaml:"id,omitempty"`
  // 源目录（bind 必填；named 由 name 决定；tmpfs 不需要）
  Source string `json:"source,omitempty" yaml:"source,omitempty"`
  // 目标目录
  Target string `json:"target" yaml:"target"`
  // 命名卷名称（Type=named 时使用）
  Name string `json:"name,omitempty" yaml:"name,omitempty"`
  // 只读
  ReadOnly bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`
  // 递归绑定（--rbind）
  Recursive bool `json:"recursive,omitempty" yaml:"recursive,omitempty"`
  // 传播选项：如 rshared、rprivate、rslave
  Propagation string `json:"propagation,omitempty" yaml:"propagation,omitempty"`
  // 若目标不存在是否创建
  CreateTarget bool `json:"create_target,omitempty" yaml:"create_target,omitempty"`
  // 目标目录权限（创建时应用，如 0755；字符串形式以便 YAML 表示）
  Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`
  // 所有者 UID/GID（创建目标时应用）
  UID int `json:"uid,omitempty" yaml:"uid,omitempty"`
  GID int `json:"gid,omitempty" yaml:"gid,omitempty"`
  // tmpfs 大小（MB），Type=tmpfs 时可用
  TmpfsSizeMB int `json:"tmpfs_size_mb,omitempty" yaml:"tmpfs_size_mb,omitempty"`
  // 额外选项（保留）
  Options []string `json:"options,omitempty" yaml:"options,omitempty"`
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

  // PID 是当前进程 ID
  PID int `json:"pid,omitempty" yaml:"pid,omitempty"`

  // Conditions 是详细状态条件列表（类似 K8s PodConditions）
  Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

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
  CPUUsage         float64 `json:"cpu_usage" yaml:"cpu_usage"`
  CPUUsageHuman    string  `json:"cpu_usage_human,omitempty" yaml:"cpu_usage_human,omitempty"`
  MemoryUsage      uint64  `json:"memory_usage" yaml:"memory_usage"`
  MemoryUsageHuman string  `json:"memory_usage_human,omitempty" yaml:"memory_usage_human,omitempty"`

  // 系统内存补充
  MemoryTotal       uint64  `json:"memory_total,omitempty" yaml:"memory_total,omitempty"`
  MemoryUsedPercent float64 `json:"memory_used_percent,omitempty" yaml:"memory_used_percent,omitempty"`

  // 新增：可用内存
  MemoryAvailable uint64 `json:"memory_available,omitempty" yaml:"memory_available,omitempty"`
  // 新增：Swap 使用统计
  SwapTotal uint64 `json:"swap_total,omitempty" yaml:"swap_total,omitempty"`
  SwapUsed  uint64 `json:"swap_used,omitempty" yaml:"swap_used,omitempty"`
  SwapFree  uint64 `json:"swap_free,omitempty" yaml:"swap_free,omitempty"`
  // 新增：内存压力（PSI）
  MemoryPressure PressureStallInfo `json:"memory_pressure,omitempty" yaml:"memory_pressure,omitempty"`

  // 系统磁盘/网络（保留）
  DiskIO         uint64     `json:"disk_io" yaml:"disk_io"`
  DiskIOHuman    string     `json:"disk_io_human,omitempty" yaml:"disk_io_human,omitempty"`
  NetworkIO      uint64     `json:"network_io" yaml:"network_io"`
  NetworkIOHuman string     `json:"network_io_human,omitempty" yaml:"network_io_human,omitempty"`
  ListeningPorts []PortInfo `json:"listening_ports,omitempty" yaml:"listening_ports,omitempty"`

  // 系统磁盘使用与设备IO（保留）
  DiskUsages    []DiskUsageInfo `json:"disk_usages,omitempty" yaml:"disk_usages,omitempty"`
  DiskIODevices []DiskIOInfo    `json:"disk_io_devices,omitempty" yaml:"disk_io_devices,omitempty"`
  Load          LoadAvg         `json:"load,omitempty" yaml:"load,omitempty"`
  FD            FDCheck         `json:"fd,omitempty" yaml:"fd,omitempty"`
  KernelParams  []KernelParam   `json:"kernel_params,omitempty" yaml:"kernel_params,omitempty"`

  // CPU 相关指标
  CPUAffinity   []int   `json:"cpu_affinity,omitempty" yaml:"cpu_affinity,omitempty"`
  CPUUserTime   float64 `json:"cpu_user_time,omitempty" yaml:"cpu_user_time,omitempty"`     // 秒
  CPUSystemTime float64 `json:"cpu_system_time,omitempty" yaml:"cpu_system_time,omitempty"` // 秒
  CPUTotalTime  float64 `json:"cpu_total_time,omitempty" yaml:"cpu_total_time,omitempty"`   // 秒

  // 内存相关指标
  MemoryVMS    uint64 `json:"memory_vms,omitempty" yaml:"memory_vms,omitempty"`       // 虚拟内存
  MemoryRSS    uint64 `json:"memory_rss,omitempty" yaml:"memory_rss,omitempty"`       // 常驻内存
  MemoryShared uint64 `json:"memory_shared,omitempty" yaml:"memory_shared,omitempty"` // 共享内存
  MemoryHeap   uint64 `json:"memory_heap,omitempty" yaml:"memory_heap,omitempty"`     // 堆内存（smaps）

  // I/O 相关指标（进程级）
  IOReadBytes   uint64 `json:"io_read_bytes,omitempty" yaml:"io_read_bytes,omitempty"`
  IOWriteBytes  uint64 `json:"io_write_bytes,omitempty" yaml:"io_write_bytes,omitempty"`
  IOReadCount   uint64 `json:"io_read_count,omitempty" yaml:"io_read_count,omitempty"`
  IOWriteCount  uint64 `json:"io_write_count,omitempty" yaml:"io_write_count,omitempty"`
  IOReadTimeMS  uint64 `json:"io_read_time_ms,omitempty" yaml:"io_read_time_ms,omitempty"`
  IOWriteTimeMS uint64 `json:"io_write_time_ms,omitempty" yaml:"io_write_time_ms,omitempty"`
  OpenFDs       int32  `json:"open_fds,omitempty" yaml:"open_fds,omitempty"`

  // 进程状态与调度
  ProcessStatus          string `json:"process_status,omitempty" yaml:"process_status,omitempty"`
  ThreadCount            int32  `json:"thread_count,omitempty" yaml:"thread_count,omitempty"`
  CtxSwitchesVoluntary   int64  `json:"ctx_switches_voluntary,omitempty" yaml:"ctx_switches_voluntary,omitempty"`
  CtxSwitchesInvoluntary int64  `json:"ctx_switches_involuntary,omitempty" yaml:"ctx_switches_involuntary,omitempty"`
  SchedulerPolicy        string `json:"scheduler_policy,omitempty" yaml:"scheduler_policy,omitempty"`
  SchedulerPriority      int32  `json:"scheduler_priority,omitempty" yaml:"scheduler_priority,omitempty"`
  Nice                   int32  `json:"nice,omitempty" yaml:"nice,omitempty"`

  // 文件与网络资源（进程聚合）
  OpenFilesCount          int `json:"open_files_count,omitempty" yaml:"open_files_count,omitempty"`
  NetworkConnectionsCount int `json:"network_connections_count,omitempty" yaml:"network_connections_count,omitempty"`

  // 新增：网络（系统级聚合）
  NetRxBytesPerSec float64 `json:"net_rx_bytes_per_sec,omitempty" yaml:"net_rx_bytes_per_sec,omitempty"`
  NetTxBytesPerSec float64 `json:"net_tx_bytes_per_sec,omitempty" yaml:"net_tx_bytes_per_sec,omitempty"`
  NetRxPackets     uint64  `json:"net_rx_packets,omitempty" yaml:"net_rx_packets,omitempty"`
  NetTxPackets     uint64  `json:"net_tx_packets,omitempty" yaml:"net_tx_packets,omitempty"`
  NetRxErrors      uint64  `json:"net_rx_errors,omitempty" yaml:"net_rx_errors,omitempty"`
  NetTxErrors      uint64  `json:"net_tx_errors,omitempty" yaml:"net_tx_errors,omitempty"`
  NetRxDropped     uint64  `json:"net_rx_dropped,omitempty" yaml:"net_rx_dropped,omitempty"`
  NetTxDropped     uint64  `json:"net_tx_dropped,omitempty" yaml:"net_tx_dropped,omitempty"`
  // TCP 状态计数
  TCPStateCounts map[string]int `json:"tcp_state_counts,omitempty" yaml:"tcp_state_counts,omitempty"`

  // 稳定性与生命周期
  StartTime *time.Time `json:"start_time,omitempty" yaml:"start_time,omitempty"`
  ParentPID int        `json:"parent_pid,omitempty" yaml:"parent_pid,omitempty"`
  ExitCode  int        `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`

  // 新增：系统运行时间（秒）
  SystemUptimeSeconds float64 `json:"system_uptime_seconds,omitempty" yaml:"system_uptime_seconds,omitempty"`
}

// SetFormattedValues 设置所有格式化的字段值
func (rs *ResourceStats) SetFormattedValues() {
  rs.CPUUsageHuman = FormatCPUUsage(rs.CPUUsage)
  rs.MemoryUsageHuman = FormatBytes(rs.MemoryUsage)
  rs.DiskIOHuman = FormatBytes(rs.DiskIO)
  rs.NetworkIOHuman = FormatBytes(rs.NetworkIO)
}

// 新增：磁盘使用
type DiskUsageInfo struct {
  Device      string  `json:"device" yaml:"device"`
  Mountpoint  string  `json:"mountpoint" yaml:"mountpoint"`
  Fstype      string  `json:"fstype,omitempty" yaml:"fstype,omitempty"`
  Total       uint64  `json:"total" yaml:"total"`
  Used        uint64  `json:"used" yaml:"used"`
  Free        uint64  `json:"free" yaml:"free"`
  UsedPercent float64 `json:"used_percent" yaml:"used_percent"`
  // 新增：Inode 使用
  InodesTotal       uint64  `json:"inodes_total,omitempty" yaml:"inodes_total,omitempty"`
  InodesUsed        uint64  `json:"inodes_used,omitempty" yaml:"inodes_used,omitempty"`
  InodesFree        uint64  `json:"inodes_free,omitempty" yaml:"inodes_free,omitempty"`
  InodesUsedPercent float64 `json:"inodes_used_percent,omitempty" yaml:"inodes_used_percent,omitempty"`
}

// 新增：磁盘IO（每设备）
type DiskIOInfo struct {
  Device     string `json:"device" yaml:"device"`
  ReadBytes  uint64 `json:"read_bytes" yaml:"read_bytes"`
  WriteBytes uint64 `json:"write_bytes" yaml:"write_bytes"`
  ReadCount  uint64 `json:"read_count,omitempty" yaml:"read_count,omitempty"`
  WriteCount uint64 `json:"write_count,omitempty" yaml:"write_count,omitempty"`
  // 新增：读写耗时（ms）
  ReadTimeMS  uint64 `json:"read_time_ms,omitempty" yaml:"read_time_ms,omitempty"`
  WriteTimeMS uint64 `json:"write_time_ms,omitempty" yaml:"write_time_ms,omitempty"`
  // 新增：设备忙碌时间（ms）与利用率（百分比）
  BusyTimeMS         uint64  `json:"busy_time_ms,omitempty" yaml:"busy_time_ms,omitempty"`
  UtilizationPercent float64 `json:"utilization_percent,omitempty" yaml:"utilization_percent,omitempty"`
  // 新增：平均延迟（ms）
  AvgReadLatencyMS  float64 `json:"avg_read_latency_ms,omitempty" yaml:"avg_read_latency_ms,omitempty"`
  AvgWriteLatencyMS float64 `json:"avg_write_latency_ms,omitempty" yaml:"avg_write_latency_ms,omitempty"`
  // 新增：吞吐（估值，B/s）
  ReadThroughputBps  float64 `json:"read_throughput_bps,omitempty" yaml:"read_throughput_bps,omitempty"`
  WriteThroughputBps float64 `json:"write_throughput_bps,omitempty" yaml:"write_throughput_bps,omitempty"`
}

// LoadAvg 新增：系统负载
type LoadAvg struct {
  Load1  float64 `json:"load1" yaml:"load1"`
  Load5  float64 `json:"load5" yaml:"load5"`
  Load15 float64 `json:"load15" yaml:"load15"`
}

// FDCheck 新增：文件描述符检查
type FDCheck struct {
  CurrentAllocated uint64  `json:"current_allocated,omitempty" yaml:"current_allocated,omitempty"`
  InUse            uint64  `json:"in_use,omitempty" yaml:"in_use,omitempty"`
  Max              uint64  `json:"max,omitempty" yaml:"max,omitempty"`
  UsagePercent     float64 `json:"usage_percent,omitempty" yaml:"usage_percent,omitempty"`
}

// KernelParam 新增：内核参数
type KernelParam struct {
  Key   string `json:"key" yaml:"key"`
  Value string `json:"value" yaml:"value"`
}

// PressureStallInfo 新增：压力阻塞信息（PSI）
type PressureStallInfo struct {
  Avg10  float64 `json:"avg10,omitempty" yaml:"avg10,omitempty"`
  Avg60  float64 `json:"avg60,omitempty" yaml:"avg60,omitempty"`
  Avg300 float64 `json:"avg300,omitempty" yaml:"avg300,omitempty"`
  Total  uint64  `json:"total,omitempty" yaml:"total,omitempty"`
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

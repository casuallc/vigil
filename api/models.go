package api

import "time"

type WSMessage struct {
  Type string `json:"type"` // "input" | "resize"
  Data []byte `json:"data,omitempty"`

  Cols int `json:"cols,omitempty"`
  Rows int `json:"rows,omitempty"`
}

// TemplateVariable 表示命令模板变量
type TemplateVariable struct {
  Name    string   `json:"name"`
  Label   string   `json:"label"`
  Type    string   `json:"type"`    // "string", "number", "select"
  Default string   `json:"default"`
  Options []string `json:"options,omitempty"`
}

// CommandTemplate 表示命令模板
type CommandTemplate struct {
  ID          string             `json:"id"`
  Name        string             `json:"name"`
  Description string             `json:"description"`
  Command     string             `json:"command"`
  Variables   []TemplateVariable `json:"variables"`
  Category    string             `json:"category"`
  IsShared    bool               `json:"is_shared"`
  CreatedBy   string             `json:"created_by"`
  CreatedAt   time.Time          `json:"created_at"`
  UpdatedAt   time.Time          `json:"updated_at"`
}

// CommandHistory 表示命令执行历史记录
type CommandHistory struct {
  ID         string    `json:"id"`
  VMName     string    `json:"vm_name"`
  Command    string    `json:"command"`
  ExecutedBy string    `json:"executed_by"`
  ExecutedAt time.Time `json:"executed_at"`
  Status     string    `json:"status"`     // "success", "failed"
  DurationMs int64     `json:"duration_ms"`
  Output     string    `json:"output,omitempty"`
  Error      string    `json:"error,omitempty"`
}

// BatchExecRequest 批量执行命令请求
type BatchExecRequest struct {
  VMNames []string `json:"vm_names"`
  Command string   `json:"command"`
  Timeout int      `json:"timeout"`
  Parallel bool   `json:"parallel"`
}

// BatchExecResult 批量执行命令结果
type BatchExecResult struct {
  VMName     string `json:"vm_name"`
  Status     string `json:"status"` // "success", "failed"
  Output     string `json:"output"`
  Error      string `json:"error,omitempty"`
  DurationMs int64  `json:"duration_ms"`
}

// BatchExecResponse 批量执行命令响应
type BatchExecResponse struct {
  TaskID  string              `json:"task_id"`
  Total   int                 `json:"total"`
  Success int                 `json:"success"`
  Failed  int                 `json:"failed"`
  Results []BatchExecResult   `json:"results"`
}

// VMResourceInfo VM资源信息
type VMResourceInfo struct {
  VMName          string    `json:"vm_name"`
  CPUUsage        float64   `json:"cpu_usage"`
  MemoryUsage     float64   `json:"memory_usage"`
  MemoryTotalGB   float64   `json:"memory_total_gb"`
  MemoryUsedGB    float64   `json:"memory_used_gb"`
  DiskUsage       float64   `json:"disk_usage"`
  DiskTotalGB     float64   `json:"disk_total_gb"`
  DiskUsedGB      float64   `json:"disk_used_gb"`
  LoadAverage     []float64 `json:"load_average"`
  Uptime          string    `json:"uptime"`
  Network         VMNetwork `json:"network"`
  CollectedAt     time.Time `json:"collected_at"`
  Status          string    `json:"status,omitempty"` // "ok", "warning", "error"
  Error           string    `json:"error,omitempty"`
}

// VMNetwork VM网络信息
type VMNetwork struct {
  RXBytesPerSec int64 `json:"rx_bytes_per_sec"`
  TXBytesPerSec int64 `json:"tx_bytes_per_sec"`
}

// FileTransferRequest 跨服务器文件传输请求
type FileTransferRequest struct {
  SourceVM   string `json:"source_vm"`
  SourcePath string `json:"source_path"`
  TargetVM   string `json:"target_vm"`
  TargetPath string `json:"target_path"`
}

// FileTransferResponse 跨服务器文件传输响应
type FileTransferResponse struct {
  Message          string `json:"message"`
  BytesTransferred int64  `json:"bytes_transferred"`
  DurationMs       int64  `json:"duration_ms"`
}

// Schedule 定时任务
type Schedule struct {
  ID            string     `json:"id"`
  Name          string     `json:"name"`
  Description   string     `json:"description"`
  Command       string     `json:"command"`
  VMNames       []string   `json:"vm_names"`
  Cron          string     `json:"cron"`
  Enabled       bool       `json:"enabled"`
  Timeout       int        `json:"timeout"`
  CreatedBy     string     `json:"created_by"`
  CreatedAt     time.Time  `json:"created_at"`
  UpdatedAt     time.Time  `json:"updated_at"`
  LastRunAt     *time.Time `json:"last_run_at,omitempty"`
  LastRunStatus string     `json:"last_run_status,omitempty"`
  NextRunAt     *time.Time `json:"next_run_at,omitempty"`
}

// CreateScheduleRequest 创建定时任务请求
type CreateScheduleRequest struct {
  Name        string   `json:"name"`
  Description string   `json:"description"`
  Command     string   `json:"command"`
  VMNames     []string `json:"vm_names"`
  Cron        string   `json:"cron"`
  Enabled     bool     `json:"enabled"`
  Timeout     int      `json:"timeout"`
}

// UpdateScheduleRequest 更新定时任务请求
type UpdateScheduleRequest struct {
  Name        string   `json:"name"`
  Description string   `json:"description"`
  Command     string   `json:"command"`
  VMNames     []string `json:"vm_names"`
  Cron        string   `json:"cron"`
  Enabled     bool     `json:"enabled"`
  Timeout     int      `json:"timeout"`
}

// ScheduleExecution 定时任务执行记录
type ScheduleExecution struct {
  ID           string                    `json:"id"`
  ScheduleID   string                    `json:"schedule_id"`
  TriggeredAt  time.Time                 `json:"triggered_at"`
  CompletedAt  *time.Time                `json:"completed_at,omitempty"`
  Status       string                    `json:"status"`
  Results      []ScheduleExecutionResult `json:"results"`
}

// ScheduleExecutionResult 单个VM的执行结果
type ScheduleExecutionResult struct {
  VMName   string `json:"vm_name"`
  Status   string `json:"status"`
  Output   string `json:"output"`
  Error    string `json:"error,omitempty"`
  Duration int64  `json:"duration_ms"`
}

// ScheduleExecutionListResponse 执行历史列表响应
type ScheduleExecutionListResponse struct {
  Total int                 `json:"total"`
  Items []ScheduleExecution `json:"items"`
}

// RunScheduleResponse 立即执行任务响应
type RunScheduleResponse struct {
  Message     string `json:"message"`
  ExecutionID string `json:"execution_id"`
}

// ToggleScheduleResponse 启用/禁用任务响应
type ToggleScheduleResponse struct {
  ID      string `json:"id"`
  Enabled bool   `json:"enabled"`
}

// AIGenerateCommandRequest AI生成命令请求
type AIGenerateCommandRequest struct {
  Prompt  string                 `json:"prompt"`
  Context map[string]interface{} `json:"context"`
}

// AIGenerateCommandResponse AI生成命令响应
type AIGenerateCommandResponse struct {
  Command      string                 `json:"command"`
  Explanation  string                 `json:"explanation"`
  Alternatives []AICommandAlternative `json:"alternatives"`
  IsDangerous  bool                   `json:"is_dangerous"`
}

// AICommandAlternative 替代命令
type AICommandAlternative struct {
  Command     string `json:"command"`
  Explanation string `json:"explanation"`
}

// AIExplainCommandRequest AI解释命令请求
type AIExplainCommandRequest struct {
  Command string `json:"command"`
}

// AIExplainCommandResponse AI解释命令响应
type AIExplainCommandResponse struct {
  Explanation string          `json:"explanation"`
  Breakdown   []AICommandPart `json:"breakdown"`
  Warnings    []string        `json:"warnings"`
  IsDangerous bool            `json:"is_dangerous"`
}

// AICommandPart 命令片段解释
type AICommandPart struct {
  Part    string `json:"part"`
  Meaning string `json:"meaning"`
}

// AIFixCommandRequest AI修复命令请求
type AIFixCommandRequest struct {
  Command string `json:"command"`
  Error   string `json:"error"`
}

// AIFixCommandResponse AI修复命令响应
type AIFixCommandResponse struct {
  FixedCommand string `json:"fixed_command"`
  Explanation  string `json:"explanation"`
}

// LicenseInfo represents a license/feature code for a network interface
type LicenseInfo struct {
  Code      string `json:"code"`
  Interface string `json:"interface"`
  IP        string `json:"ip"`
}

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

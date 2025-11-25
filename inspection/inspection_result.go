package inspection

import "time"

const (
  StatusOk    = "ok"
  StatusWarn  = "warn"
  StatusError = "error"

  SeverityOk       = "ok"
  SeverityWarn     = "warn"
  SeverityError    = "error"
  SeverityCritical = "critical"
)

type Result struct {
  ID      string        `yaml:"id" json:"id,omitempty"`
  Version int           `yaml:"version" json:"version"`
  Meta    ResultMeta    `yaml:"meta" json:"meta"`
  Results []CheckResult `yaml:"results" json:"results"`
  Summary SummaryInfo   `yaml:"summary" json:"summary,omitempty"`
}

type MetaInfo struct {
  System           string    `yaml:"system" json:"system"`
  Env              string    `yaml:"env" json:"env"`
  Host             string    `yaml:"host" json:"host"`
  ExecutedAt       time.Time `yaml:"executed_at" json:"executed_at"`
  DurationSeconds  float64   `yaml:"duration_seconds" json:"duration_seconds"`
  InspectorVersion string    `yaml:"inspector_version" json:"inspector_version"`
  Status           string    `yaml:"status" json:"status"` // ok / warn / error
  Summary          string    `yaml:"summary" json:"summary"`
}

// ResultMeta 扩展的元信息结构
type ResultMeta struct {
  System  string    `yaml:"system" json:"system"`
  Host    string    `yaml:"host" json:"host"`
  JobName string    `yaml:"job_name" json:"job_name"`
  Time    time.Time `yaml:"time" json:"time"`
  Status  string    `yaml:"status" json:"status"`
}

type CheckResult struct {
  ID         string      `yaml:"id" json:"id"`
  Name       string      `yaml:"name" json:"name"`
  Type       string      `yaml:"type" json:"type"`
  Value      interface{} `yaml:"value" json:"value"`
  Unit       string      `yaml:"unit,omitempty" json:"unit,omitempty"`
  Status     string      `yaml:"status" json:"status"`     // ok / warn / error
  Severity   string      `yaml:"severity" json:"severity"` // info / warn / error / critical
  Message    string      `yaml:"message" json:"message"`
  DurationMs int64       `yaml:"duration_ms" json:"duration_ms"`
}

type SummaryInfo struct {
  TotalChecks   int       `yaml:"total_checks" json:"total_checks"`
  OK            int       `yaml:"ok" json:"ok"`
  Warn          int       `yaml:"warn" json:"warn"`
  Error         int       `yaml:"error" json:"error"`
  Critical      int       `yaml:"critical" json:"critical"`
  OverallStatus string    `yaml:"overall_status" json:"overall_status"`
  StartedAt     time.Time `yaml:"started_at" json:"started_at"`
  FinishedAt    time.Time `yaml:"finished_at" json:"finished_at"`
}

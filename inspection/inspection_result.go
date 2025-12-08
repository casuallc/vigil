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

package inspection

import "time"

const (
  StatusOk    = "ok"
  StatusError = "error"

  SeverityInfo     = "info"
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
  ID          string      `yaml:"id" json:"id"`
  Name        string      `yaml:"name" json:"name"`
  Type        string      `yaml:"type" json:"type"`
  Value       interface{} `yaml:"value" json:"value"`
  Unit        string      `yaml:"unit,omitempty" json:"unit,omitempty"`
  Status      string      `yaml:"status" json:"status"`     // ok / warn / error
  Severity    string      `yaml:"severity" json:"severity"` // info / warn / error / critical
  Message     string      `yaml:"message" json:"message"`
  DurationMs  int64       `yaml:"duration_ms" json:"duration_ms"`
  Remediation string      `yaml:"remediation,omitempty" json:"remediation,omitempty"` // 修复建议
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

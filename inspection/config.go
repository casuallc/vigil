package inspection

import "time"

// Request 表示巡检请求
type Request struct {
  Version int         `json:"version"`
  Meta    RequestMeta `json:"meta"`
  Checks  []Check     `json:"checks"`
  Env     []string    `json:"env"`
}

// RequestMeta 请求元信息
type RequestMeta struct {
  System  string    `json:"system"`
  Host    string    `json:"host"`
  JobName string    `json:"job_name"`
  Time    time.Time `json:"time"`
}

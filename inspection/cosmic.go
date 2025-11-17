package inspection

import (
  "fmt"
  "gopkg.in/yaml.v3"
  "os"
)

type Node struct {
  Name string `yaml:"name"`
  IP   string `yaml:"ip"`
  Port int    `yaml:"port"`
}

type Env struct {
  Name  string `yaml:"name"`
  Value string `yaml:"value"`
}

type Rule struct {
  Name string `yaml:"name"`
  Path string `yaml:"path"`
}

type Job struct {
  Name    string   `yaml:"name"`
  Targets []string `yaml:"targets"`
  Envs    []Env    `yaml:"envs,omitempty"`
  Rules   []Rule   `yaml:"rules,omitempty"`
}

type CosmicConfig struct {
  Nodes []Node `yaml:"nodes"`
  Jobs  []Job  `yaml:"jobs"`
}

// CosmicJob 表示一个cosmic作业配置
type CosmicJob struct {
  Name   string            `yaml:"name"`
  Host   string            `yaml:"host"`
  Port   int               `yaml:"port"`
  Labels map[string]string `yaml:"labels,omitempty"`
}

// CosmicRequest 表示cosmic巡检请求
type CosmicRequest struct {
  Job  CosmicJob         `json:"job"`
  Envs map[string]string `json:"envs,omitempty"`
}

// CosmicResult 表示cosmic巡检结果
type CosmicResult struct {
  JobName  string        `json:"job_name"`
  Host     string        `json:"host"`
  Port     int           `json:"port"`
  Status   string        `json:"status"`
  Message  string        `json:"message,omitempty"`
  Duration float64       `json:"duration,omitempty"`
  Checks   []CheckResult `json:"checks"`
}

func LoadCosmicConfig(filePath string) (*CosmicConfig, error) {
  data, err := os.ReadFile(filePath)
  if err != nil {
    fmt.Printf("❌  Faied to read file: %s", filePath)
    return nil, err
  }

  var cosmicConfig CosmicConfig
  err = yaml.Unmarshal(data, &cosmicConfig)
  if err != nil {
    fmt.Printf("❌  Failed to parse cosmic config")
    return nil, err
  }
  return &cosmicConfig, nil

}

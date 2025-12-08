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

// RuleConfig 定义规则配置类型（使用inspection_rules.go中的Config）
type RuleConfig Config

// LoadRules 加载规则配置
func LoadRules(filePath string) (*RuleConfig, error) {
  config, err := LoadInspectionConfig(filePath)
  if err != nil {
    return nil, err
  }

  ruleConfig := RuleConfig(*config)
  return &ruleConfig, nil
}

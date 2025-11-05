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

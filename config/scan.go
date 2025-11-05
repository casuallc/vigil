package config

import (
  "gopkg.in/yaml.v3"
  "os"
)

// ScanConfig represents the process scan configuration
type ScanConfig struct {
  Process []ProcessConfig `yaml:"process"`
}

// ProcessConfig represents a process to be scanned
type ProcessConfig struct {
  Name   string        `yaml:"name"`
  Query  string        `yaml:"pidCmd"`
  Labels []LabelConfig `yaml:"labels"`
}

// LabelConfig represents a label for a process
type LabelConfig struct {
  Name  string `yaml:"name"`
  Value string `yaml:"value"`
}

// LoadScanConfig loads scan configuration from file
func LoadScanConfig(filePath string) (*ScanConfig, error) {
  // Read file content
  data, err := os.ReadFile(filePath)
  if err != nil {
    return nil, err
  }

  // Parse YAML
  config := &ScanConfig{}
  if err := yaml.Unmarshal(data, config); err != nil {
    return nil, err
  }

  return config, nil
}

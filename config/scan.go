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
  Query  string        `yaml:"query"`
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

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
  "crypto/rand"
  "encoding/base64"
  "log"
  "os"

  "github.com/casuallc/vigil/common"

  "gopkg.in/yaml.v3"
)

// Config represents the program configuration
type Config struct {
  Log         LogConfig      `yaml:"log"`
  Monitor     MonitorConfig  `yaml:"monitor"`
  Process     ProcConfig     `yaml:"process"`
  Security    SecurityConfig `yaml:"security"`
  HTTPS       HTTPSConfig    `yaml:"https"`
  ManagedApps []AppConfig    `yaml:"managed_apps"`
}

type LogConfig struct {
  Level string `yaml:"level"`
}

type MonitorConfig struct {
  Rate int `yaml:"rate"` // seconds
}

type ProcConfig struct {
  PidFile string `yaml:"pid_file"`
}

type SecurityConfig struct {
  EncryptionKey string `yaml:"encryption_key"`
}

type HTTPSConfig struct {
  Enabled  bool   `yaml:"enabled"`
  CertPath string `yaml:"cert_path"`
  KeyPath  string `yaml:"key_path"`
}

// AppConfig represents the configuration of a managed application
type AppConfig struct {
  Name       string            `yaml:"name"`
  Command    string            `yaml:"command"`
  Args       []string          `yaml:"args,omitempty"`
  Env        map[string]string `yaml:"env,omitempty"`
  WorkingDir string            `yaml:"working_dir,omitempty"`
  Restart    bool              `yaml:"restart"`
  User       string            `yaml:"user,omitempty"`
}

// generateRandomKey generates a random key for encryption
func generateRandomKey(length int) string {
  key := make([]byte, length)
  _, err := rand.Read(key)
  if err != nil {
    // Fallback to a default key if random generation fails
    return "default_encryption_key_change_this_in_production"
  }
  return base64.StdEncoding.EncodeToString(key)
}

// LoadConfig loads configuration from file
func LoadConfig(filePath string) (*Config, error) {
  // Check if file exists
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    log.Printf("Config file %s does not exist", filePath)
    return nil, nil
  }

  // Read file content
  data, err := os.ReadFile(filePath)
  if err != nil {
    return nil, err
  }

  // Parse YAML
  // 直接创建一个空的 Config 结构体，而不是调用 DefaultConfig()
  // 这样可以避免生成新的加密密钥，而是使用文件中已有的密钥
  config := &Config{}
  if err := yaml.Unmarshal(data, config); err != nil {
    return nil, err
  }
  return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(filePath string, config *Config) error {
  // Convert to YAML
  data, err := common.ToYamlString(config)
  if err != nil {
    return err
  }

  // Write to file
  return os.WriteFile(filePath, []byte(data), 0644)
}

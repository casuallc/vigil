package config

import (
  "github.com/casuallc/vigil/common"
  "log"
  "os"

  "gopkg.in/yaml.v3"
)

// Config represents the program configuration
type Config struct {
  LogLevel    string      `yaml:"log_level"`
  MonitorRate int         `yaml:"monitor_rate"` // Monitor frequency (seconds)
  PidFilePath string      `yaml:"pid_file_path"`
  ManagedApps []AppConfig `yaml:"managed_apps"`
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

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
  return &Config{
    LogLevel:    "info",
    MonitorRate: 5,
    PidFilePath: "./vigil.pid",
  }
}

// LoadConfig loads configuration from file
func LoadConfig(filePath string) (*Config, error) {
  // Check if file exists
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    log.Printf("Config file %s does not exist, will create default config", filePath)
    // Create default config file
    defaultConfig := DefaultConfig()
    if err := SaveConfig(filePath, defaultConfig); err != nil {
      return nil, err
    }
    return defaultConfig, nil
  }

  // Read file content
  data, err := os.ReadFile(filePath)
  if err != nil {
    return nil, err
  }

  // Parse YAML
  config := DefaultConfig()
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

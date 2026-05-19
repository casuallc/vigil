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

package cli

import (
  "encoding/json"
  "fmt"
  "os"
  "runtime"

  "github.com/casuallc/vigil/api"
  "github.com/spf13/cobra"
)

type installConfig struct {
  ManagerURL   string `json:"managerUrl"`
  InstallToken string `json:"installToken"`
}

func (c *CLI) setupRegisterCommand() *cobra.Command {
  var configFile string

  cmd := &cobra.Command{
    Use:   "register",
    Short: "Register this BBX agent with the management console",
    Long:  "Register this BBX agent with the remote management console using an install config file",
    RunE: func(cmd *cobra.Command, args []string) error {
      insecure, _ := cmd.Flags().GetBool("insecure")
      return c.handleRegister(configFile, insecure)
    },
  }

  cmd.Flags().StringVarP(&configFile, "file", "f", "/etc/bbx/install-config.json", "Path to install config JSON file")

  return cmd
}

func (c *CLI) handleRegister(configFile string, insecureSkipVerify bool) error {
  data, err := os.ReadFile(configFile)
  if err != nil {
    return fmt.Errorf("failed to read config file: %w", err)
  }

  var cfg installConfig
  if err := json.Unmarshal(data, &cfg); err != nil {
    return fmt.Errorf("failed to parse config file: %w", err)
  }

  if cfg.ManagerURL == "" {
    return fmt.Errorf("managerUrl is required in config file")
  }
  if cfg.InstallToken == "" {
    return fmt.Errorf("installToken is required in config file")
  }

  hostname, err := os.Hostname()
  if err != nil {
    return fmt.Errorf("failed to get hostname: %w", err)
  }

  arch := runtime.GOARCH

  resp, err := api.RegisterBbx(cfg.ManagerURL, cfg.InstallToken, hostname, arch, insecureSkipVerify)
  if err != nil {
    return fmt.Errorf("registration failed: %w", err)
  }

  if resp.Code != 0 {
    return fmt.Errorf("registration failed: %s", resp.Msg)
  }

  fmt.Println("Registration successful")
  return nil
}

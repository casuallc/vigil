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

package inspection_test

import (
  "fmt"
  "github.com/casuallc/vigil/inspection"
  "gopkg.in/yaml.v3"
  "log"
  "os"
  "strconv"
  "testing"
)

func TestInspectionRules(t *testing.T) {
  data, err := os.ReadFile("../conf/cosmic/rules/adcc.yaml")
  if err != nil {
    log.Fatalf("Failed to read config file: %v", err)
  }

  var inspectionConfig inspection.Config
  err = yaml.Unmarshal(data, &inspectionConfig)
  if err != nil {
    log.Fatalf("Failed to parse YAML: %v", err)
  }

  if inspectionConfig.Version != 1 {
    log.Printf("⚠️  Warning: config version is %d, expected 1", inspectionConfig.Version)
  }

  // 校验每个 check
  for _, check := range inspectionConfig.Checks {
    if err := check.Validate(); err != nil {
      log.Fatalf("Config validation failed: %v", err)
    }
  }

  // 编译表达式（可选但推荐）
  if err := inspectionConfig.CompileExpressions(); err != nil {
    log.Fatalf("Expression compilation failed: %v", err)
  }

  fmt.Printf("✅ Config loaded and validated for system: %s\n", inspectionConfig.Meta.System)

  // 示例：打印第一个脚本类检查的命令
  for _, chk := range inspectionConfig.Checks {
    if chk.Type == inspection.TypeScript {
      cmds, _ := chk.GetCommandLines()
      fmt.Printf("   [%s] Command: %v\n", chk.ID, cmds)
      break
    }
  }
}

func TestParseInt(t *testing.T) {
  output := "1"
  val, err := strconv.ParseInt(output, 10, 64)
  if err != nil {
    fmt.Printf("Error parsing int: %v", err)
    return
  }
  fmt.Printf("value: %d", val)
}

package inspection_test

import (
  "fmt"
  "github.com/casuallc/vigil/inspection"
  "gopkg.in/yaml.v3"
  "log"
  "os"
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

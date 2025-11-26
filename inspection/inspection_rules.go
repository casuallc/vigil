package inspection

import (
  "fmt"
  "github.com/expr-lang/expr"
  "gopkg.in/yaml.v3"
  "os"
)

const (
  TypeProcess     = "process"
  TypeLog         = "log"
  TypePerformance = "performance"
  TypeCustom      = "custom"
  TypeCompound    = "compound"
  TypeScript      = "script"
)

var validTypes = map[string]bool{
  TypeProcess:     true,
  TypeLog:         true,
  TypePerformance: true,
  TypeCustom:      true,
  TypeCompound:    true,
  TypeScript:      true,
}

// ParseConfig 解析输出的方式
type ParseConfig struct {
  Kind    string `yaml:"kind,omitempty"`    // int|float|string|regex|regex_list|json|yaml
  Pattern string `yaml:"pattern,omitempty"` // 用于 regex
  Path    string `yaml:"path,omitempty"`    // 用于 json/yaml
}

// Threshold 定义阈值规则
type Threshold struct {
  When     string `yaml:"when"`
  Severity string `yaml:"severity"`
  Message  string `yaml:"message,omitempty"`
  Value    float64
  Operator string
}

// Meta 元信息
type Meta struct {
  System string   `yaml:"system"`
  Owner  string   `yaml:"owner"`
  Tags   []string `yaml:"tags,omitempty"`
}

// Check 单个检查项
type Check struct {
  ID            string       `yaml:"id"`
  Name          string       `yaml:"name"`
  Type          string       `yaml:"type"`              // process | log | performance | custom | compound | script
  Command       interface{}  `yaml:"command,omitempty"` // string or []string
  ScriptPath    string       `yaml:"script_path,omitempty"`
  Expect        interface{}  `yaml:"expect,omitempty"` // string or []string
  Parse         *ParseConfig `yaml:"parse,omitempty"`
  Thresholds    []Threshold  `yaml:"thresholds,omitempty"`
  Compare       string       `yaml:"compare,omitempty"`
  Timeout       int          `yaml:"timeout,omitempty"` // seconds
  Retries       int          `yaml:"retries,omitempty"`
  Severity      string       `yaml:"severity,omitempty"` // default: warn
  Remediation   string       `yaml:"remediation,omitempty"`
  NotifyIfFound bool         `yaml:"notify_if_found,omitempty"` // log type only
  Children      []string     `yaml:"children,omitempty"`        // compound only
  Logic         string       `yaml:"logic,omitempty"`           // compound only
}

// Config 顶层配置
type Config struct {
  Version int     `yaml:"version"`
  Meta    Meta    `yaml:"meta"`
  Checks  []Check `yaml:"checks"`
}

func (c *Check) Validate() error {
  if !validTypes[c.Type] {
    return fmt.Errorf("check '%s': invalid type '%s'", c.ID, c.Type)
  }

  if c.Severity == "" {
    c.Severity = SeverityWarn // 默认 warn
  } else if c.Severity != SeverityInfo && c.Severity != SeverityWarn && c.Severity != SeverityError && c.Severity != SeverityCritical {
    return fmt.Errorf("check '%s': invalid severity '%s'", c.ID, c.Severity)
  }

  // compound 必须有 children 和 logic
  if c.Type == TypeCompound {
    if len(c.Children) == 0 {
      return fmt.Errorf("check '%s': compound type requires 'children'", c.ID)
    }
    if c.Logic == "" {
      return fmt.Errorf("check '%s': compound type requires 'logic'", c.ID)
    }
  }

  // log 类型可选 notify_if_found
  // script/custom 需要 command 或 script_path
  if c.Type == TypeScript || c.Type == TypeCustom {
    if c.Command == nil && c.ScriptPath == "" {
      return fmt.Errorf("check '%s': script/custom type requires 'command' or 'script_path'", c.ID)
    }
  }

  return nil
}

// Helper: 将 interface{} 转为 []string
func toStringSlice(v interface{}) ([]string, error) {
  switch val := v.(type) {
  case string:
    return []string{val}, nil
  case []interface{}:
    result := make([]string, len(val))
    for i, item := range val {
      str, ok := item.(string)
      if !ok {
        return nil, fmt.Errorf("non-string item in command/expect list")
      }
      result[i] = str
    }
    return result, nil
  case []string:
    return val, nil
  case nil:
    return nil, nil
  default:
    return nil, fmt.Errorf("unsupported type for command/expect: %T", v)
  }
}

func (c *Check) GetCommandLines() ([]string, error) {
  return toStringSlice(c.Command)
}

func (c *Check) GetExpectLines() ([]string, error) {
  return toStringSlice(c.Expect)
}

// CompileExpressions 预编译所有表达式，确保语法合法
func (config *Config) CompileExpressions() error {
  env := make(map[string]interface{})
  // 注册所有 check ID 为变量（用于 compound logic）
  for _, chk := range config.Checks {
    env[chk.ID] = 0 // 占位，实际执行时替换
  }

  // 编译 thresholds.when（简化：只校验是否为合法表达式）
  // 注意：when 表达式通常形如 "> 100"，需拼接为 "value > 100"
  // 这里仅做语法检查，完整验证留到执行时
  dummyEnv := map[string]interface{}{"value": 0.0}
  for _, chk := range config.Checks {
    for _, th := range chk.Thresholds {
      // 尝试拼接成 "value <expr>"
      testExpr := "value " + th.When
      _, err := expr.Compile(testExpr, expr.Env(dummyEnv))
      if err != nil {
        return fmt.Errorf("invalid threshold 'when' in check '%s': %v", chk.ID, err)
      }
    }
  }

  return nil
}

// LoadInspectionConfig 加载巡检配置
func LoadInspectionConfig(filePath string) (*Config, error) {
  data, err := os.ReadFile(filePath)
  if err != nil {
    fmt.Printf("❌  Faied to read file: %s", filePath)
    return nil, err
  }

  var inspectionConfig Config
  err = yaml.Unmarshal(data, &inspectionConfig)
  if err != nil {
    fmt.Printf("❌  Failed to parse inspection config")
    return nil, err
  }

  if inspectionConfig.Version != 1 {
    fmt.Printf("⚠️  Warning: config version is %d, expected 1", inspectionConfig.Version)
  }

  // 校验每个 check
  for _, check := range inspectionConfig.Checks {
    if err := check.Validate(); err != nil {
      fmt.Printf("❌  Invalid check: %s", err.Error())
      return nil, err
    }
  }

  return &inspectionConfig, nil
}

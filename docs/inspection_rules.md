下面是**详尽且实用的 YAML 配置标准设计规范**，覆盖字段说明、类型、校验、示例、进阶特性（继承、模板、多步/子任务、脚本插件）、版本兼容、规则表达式、以及解析器应该如何处理这些配置。目标是让你和团队能一致地编写/审查巡检配置，并能最小改动适配不同系统。

# 一、总体目标
+ **配置驱动**：所有巡检指标通过 YAML 配置描述，框架负责执行与分析。
+ **可读可审计**：便于运维/QA 审查、版本管理（git）。
+ **扩展友好**：支持自定义分析、复合条件、多步检查、模板继承。
+ **机器友好**：易于校验（JSON Schema）、转换为标准报告（JSON）。

# 二、顶层结构（建议）
```yaml
# configs/nginx.yaml
version: 1                   # 配置文件版本（用于兼容性）
meta:
  system: nginx
  env: production
  owner: platform-team
  tags: [http, frontend]
  description: "Nginx 基本巡检: 进程、活跃连接、错误日志"
  schedule: "@daily"         # 可选，cron-like 表达式或调度标识

checks:                       # 指标列表（数组）
  - id: process_status        # 唯一 id（必填）
    name: "nginx process"     # 可读名称
    type: process             # 类型：process/log/perf/custom/compound
    command: "ps -ef | grep nginx | grep -v grep"
    expect: ["nginx: master process", "nginx: worker process"] # 匹配任一即 OK
    severity: critical        # severity: info/warn/critical
    remediation: "systemctl restart nginx"
    timeout: 10               # 秒

  - id: active_connections
    name: "active connections"
    type: performance
    command: "curl -s http://127.0.0.1/nginx_status | awk '/Active/ {print $3}'"
    parse:
      kind: int               # 结果解析为整数
    thresholds:
      - when: ">= 1000"
        severity: critical
        message: "连接数过多"
      - when: ">= 500"
        severity: warn
        message: "连接数偏高"
    unit: connections

  - id: error_log_recent
    name: "recent error log"
    type: log
    command: "tail -n 200 /var/log/nginx/error.log"
    parse:
      kind: regex_list
      pattern: "(?i)error|crit|alert|emerg"
    notify_if_found: true
    severity: warn
    max_history: 5

  - id: health_compound
    name: "overall health"
    type: compound
    logic: "ALL_OK"           # 支持 AND/OR/ALL_OK/MAJORITY 或自定义表达式
    children: ["process_status", "active_connections"]
```

# 三、字段详解（必看）
## 顶层字段
+ `version`：整数，配置规范版本，用于兼容性处理。
+ `meta`：元信息（`system`、`env`、`owner`、`tags`、`description`、`schedule` 等）。
+ `checks`：数组，每项为一个检查项（指标）。

## 单个检查项字段
+ `id`（必填，字符串）：检查唯一标识，用于引用（在 compound/依赖中）。
+ `name`（必填，字符串）：可读名称/说明。
+ `type`（必填，字符串）：枚举值 `process | log | performance | custom | compound | script`。
    - `process`：进程/服务存在性检查
    - `log`：日志内容匹配/告警
    - `performance`：数值型、含阈值判断（TPS/延迟/连接数/内存）
    - `custom`：需要自定义 Python/Bash 分析函数（referenced plugin）
    - `compound`：复合指标，由多个子检查组合逻辑得出
    - `script`：执行指定脚本/脚本路径（多步）
+ `command`（可选，字符串或数组）：在目标机器上执行的 shell 命令或命令数组（多步）。
+ `script_path`（可选）：指向可执行脚本文件（相对/绝对），用于 `script` 或 `custom` 类型。
+ `expect`（可选，字符串或数组）：对 `command` 输出的期望匹配（包含/精确匹配）。
+ `parse`（可选，object）：解析输出的方法：
    - `kind`: `int|float|string|regex|regex_list|json|yaml`
    - `pattern`: 对于 regex/regex_list 的正则
    - `path`: 对于 json/yaml 指定的 jsonpath 或 yamlpath
+ `thresholds`（可选，数组）：用于数值类或可比较结果。每个项包含：
    - `when`：表达式字符串，例如 `">= 1000"`, `"> 0.9"`, `"== 0"`。推荐支持复合条件（见表达式部分）。
    - `severity`：`info|warn|error|critical`
    - `message`：提示/说明
+ `compare`（可选）：简写比较符，如 `<, <=, >, >=, ==, !=`（若使用 `thresholds` 可忽略）
+ `timeout`（可选，秒）：执行超时
+ `retries`（可选，整数）：失败重试次数
+ `severity`（可选，默认 `warn`）：当检测失败时的默认严重级别
+ `remediation`（可选，字符串/命令）：推荐的修复步骤或脚本名
+ `notify_if_found`（log 特有，bool）：日志检查中是否发现匹配即触发报警
+ `children`（compound 特有，数组）：列出子检查项 `id` 列表
+ `logic`（compound 特有，字符串/表达式）：`AND | OR | ALL_OK | MAJORITY | custom_expr`（或表达式如 `"process_status && (active_connections < 500)"`）

## meta 字段（推荐）
+ `system`：中间件类型（nginx/redis/kafka）
+ `env`：环境（prod/stage/dev）
+ `owner`：负责团队或人
+ `tags`：用于筛选/仪表盘
+ `schedule`：cron 表达式或调度标识（用于调度器）

# 四、表达式与阈值语法建议
+ `when` 支持简单比较表达式和布尔组合：
    - 简单： `">= 1000"`, `"< 500"`, `"== 0"`
    - 支持百分比/单位： `">= 85%"`（解析到 0.85 或以 unit 判断）
    - 复合： `">= 100 && < 200"` 或 `"value >= 100 and value < 200"`
+ 推荐内部实现：把 `when` 解析为安全表达式（不要直接 eval 未校验输入），或使用小型表达式解析器（如 `asteval`、`expr` 模块或自写解析器）。

# 五、多步检查与脚本支持
当指标需要多步（先获取 token，再调用 API），支持 `steps` 或 `command` 数组：

```yaml
- id: multi_step_example
  name: "multi step check"
  type: script
  steps:
    - name: login
      command: "curl -s -d 'user=foo' http://auth/login | jq -r .token"
      parse: { kind: string }
      save_as: auth_token
    - name: get_metric
      command: "curl -s -H 'Authorization: Bearer {{auth_token}}' http://api/metric | jq -r .value"
      parse: { kind: int }
      thresholds:
        - when: "> 10"
          severity: warn
```

+ `save_as`：把上一步结果存为变量供下一步替换（模板 `{{var}}`）。

# 六、复用/模板/继承（DRY）
支持 `includes` 或 `base`：

```yaml
# base_checks.yaml
checks:
  - id: check_ssh
    name: "ssh alive"
    command: "ss -tnlp | grep :22"
    type: process

# nginx.yaml
includes:
  - base_checks.yaml

checks:
  - id: nginx_specific
    ...
```

或用 `extends` 指定 `base_id` 并 override 少量字段。

# 七、输出与报告约定
配置应定义如何在结果中呈现（便于自动化）：

+ 每个检查返回标准 JSON：

```json
{
  "id": "active_connections",
  "name": "active connections",
  "status": "OK|WARN|CRITICAL|ERROR",
  "value": 123,
  "severity": "warn",
  "message": "连接数偏高",
  "output": "...原始输出...",
  "started_at": "...",
  "duration_ms": 120
}
```

+ `overall` 汇总由 `compound` 或调度器决定。

# 八、安全性与敏感信息
+ **不要在 YAML 中明文存放密码/密钥**。用：
    - 环境变量（`$REDIS_PASSWORD`）或
    - 引用密钥管理（Vault、KMS）令牌（例如在 `meta` 中写 `secrets: { redis_password: vault://path }`）
+ YAML 支持 `env:` 替换或运行时注入。
+ `command` 输出可能包含敏感信息，应在 report 中脱敏或标记为仅可见给特定角色。

# 九、版本管理与向后兼容
+ `version` 字段用于解析器分支：当配置格式升级（v1 -> v2），解析器根据 `version` 采取不同逻辑。
+ 新增字段应向后兼容，弃用字段应保留若干版本并输出警告。

# 十、校验（JSON Schema）
建议提供一个 JSON Schema 用来校验 YAML（示例简略）：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "meta", "checks"],
  "properties": {
    "version": { "type": "integer" },
    "meta": {
      "type": "object",
      "properties": {
        "system": { "type": "string" },
        "env": { "type": "string" }
      }
    },
    "checks": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "name", "type"],
        "properties": {
          "id": { "type": "string" },
          "name": { "type": "string" },
          "type": { "type": "string", "enum": ["process","log","performance","custom","compound","script"] },
          "command": { "anyOf": [{ "type": "string" }, { "type": "array" }] },
          "parse": { "type": "object" },
          "thresholds": { "type": "array" }
        }
      }
    }
  }
}
```

（实际 schema 可更细化：`thresholds` 项的结构、`parse.kind` 的 enum 等）

# 十一、示例（完整：Redis）
```yaml
version: 1
meta:
  system: redis
  env: prod
  owner: db-team
  description: "Redis 基本巡检"
checks:
  - id: redis_process
    name: "redis process"
    type: process
    command: "ps -ef | grep redis-server | grep -v grep"
    expect: "redis-server"
    severity: critical
    remediation: "systemctl restart redis"

  - id: redis_connected_clients
    name: "connected clients"
    type: performance
    command: "redis-cli INFO clients | grep connected_clients | cut -d: -f2"
    parse: { kind: int }
    thresholds:
      - when: ">= 10000"
        severity: critical
        message: "客户端连接数异常"
      - when: ">= 5000"
        severity: warn
        message: "客户端连接数偏高"

  - id: redis_aof_enabled
    name: "aof enabled"
    type: custom
    script_path: "plugins/redis_aof_check.py"
    severity: warn
```

# 十二、解析器/框架对配置的处理建议（伪代码）
```plain
load_yaml(file)
validate_against_schema(yaml)
for check in yaml.checks:
  start_timer()
  if check.type == "script" and check.steps:
    for step in check.steps:
      cmd = render_template(step.command, context)
      output = run_shell(cmd, timeout=step.timeout)
      parse_result = parse_output(output, step.parse)
      if step.save_as: context[step.save_as] = parse_result
  else:
    cmd = render_template(check.command, context)
    output = run_shell(cmd, timeout=check.timeout)
    parsed = parse_output(output, check.parse)
    status, msg = evaluate(parsed, check.thresholds, check.expect, check.compare)
  collect_result(id=check.id, value=parsed, status=status, message=msg, raw=output)
save_report(results)
```

+ `render_template`：将 `{{var}}` 替换为上下文变量（注意安全/注入）
+ `parse_output`：依据 `parse.kind` 做类型解析（int、json、regex）
+ `evaluate`：应用 `thresholds`（按优先级高→低），或 `expect` 的包含匹配
+ 重试/超时逻辑在 `run_shell` 内实现。

---

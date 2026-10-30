# 命令执行 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/exec | POST | 执行命令 |
| /api/v2/exec | POST | 执行命令（返回退出码与输出） |

---

## POST /api/exec

**功能描述**：执行命令

**请求参数**：
- 请求体：
  - `command`：命令内容
  - `env`：环境变量数组

**请求体示例**：
```json
{
  "command": "ls -la",
  "env": ["key=value"]
}
```

**响应格式**（成功，HTTP 200，纯文本，stdout 与 stderr 合并）：
```json
"command output"
```

**响应格式**（失败，HTTP 500，JSON）：
```json
{
  "error": "Command execution failed: exit status 127, output: /bin/bash: line 1: ll: command not found"
}
```

注意：v1 通过 HTTP 状态码区分成败，退出码与非零退出时的输出混在 `error` 文本中；如需结构化的退出码与 stdout/stderr 分离，请使用 `/api/v2/exec`。

---

## POST /api/v2/exec

**功能描述**：执行命令，以 JSON 返回退出码与输出。无论命令成功或失败均返回 HTTP 200，通过 `exit_code` 判断执行结果。

**请求参数**：
- 请求体：
  - `command`：命令内容
  - `env`：环境变量数组

**请求体示例**：
```json
{
  "command": "ls /nonexistent",
  "env": ["key=value"]
}
```

**响应字段**：
- `exit_code`：退出码。`0` 表示成功；非零为命令实际退出码；`-1` 表示命令无法启动（如 bash 不存在）
- `stdout`：标准输出（去除首尾空白）
- `stderr`：标准错误（去除首尾空白）
- `error`：可选，仅当命令无法启动（`exit_code` 为 `-1`）时出现，描述启动失败原因

**响应示例**（命令执行失败）：
```json
{
  "exit_code": 2,
  "stdout": "",
  "stderr": "ls: cannot access '/nonexistent': No such file or directory"
}
```

**响应示例**（命令无法启动）：
```json
{
  "exit_code": -1,
  "stdout": "",
  "stderr": "",
  "error": "exec: \"/bin/bash\": file does not exist"
}
```

# 命令执行 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/exec | POST | 执行命令 |

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

**响应格式**：
```json
"command output"
```

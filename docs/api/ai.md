# AI 命令助手 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/ai/generate-command | POST | 生成命令 |
| /api/ai/explain-command | POST | 解释命令 |
| /api/ai/fix-command | POST | 修复命令 |

---

## POST /api/ai/generate-command

**功能描述**：根据自然语言描述生成命令

**请求体**：
```json
{
  "prompt": "查看最近的nginx错误日志",
  "context": {
    "server_os": "ubuntu",
    "services": ["nginx", "mysql"]
  }
}
```

**参数说明**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | 是 | 自然语言描述 |
| context | object | 否 | 上下文信息 |

**响应**：
```json
{
  "command": "tail -n 100 /var/log/nginx/error.log",
  "explanation": "查看 nginx 错误日志的最后 100 行",
  "alternatives": [
    {
      "command": "journalctl -u nginx -p err -n 100",
      "explanation": "通过 systemd 查看nginx错误日志"
    }
  ],
  "is_dangerous": false
}
```

---

## POST /api/ai/explain-command

**功能描述**：解释命令的含义和作用

**请求体**：
```json
{
  "command": "find /var/log -name '*.log' -mtime +7 -delete"
}
```

**响应**：
```json
{
  "explanation": "查找 /var/log 目录下所有 7 天前修改过的 .log 文件并删除",
  "breakdown": [
    {
      "part": "find /var/log",
      "meaning": "在 /var/log 目录下查找文件"
    },
    {
      "part": "-name '*.log'",
      "meaning": "匹配所有 .log 结尾的文件"
    },
    {
      "part": "-mtime +7",
      "meaning": "修改时间超过 7 天"
    },
    {
      "part": "-delete",
      "meaning": "删除匹配的文件"
    }
  ],
  "warnings": ["此命令会删除文件，请谨慎使用"],
  "is_dangerous": true
}
```

---

## POST /api/ai/fix-command

**功能描述**：修复错误的命令

**请求体**：
```json
{
  "command": "docker ps -a | grep exited | xargs docker rm",
  "error": "unknown shorthand flag: 'a' in -a\nSee 'docker ps --help'"
}
```

**响应**：
```json
{
  "fixed_command": "docker container ls -a --filter 'status=exited' -q | xargs -r docker rm",
  "explanation": "原命令在部分 Docker 版本中语法不兼容，已修复为兼容性更好的写法"
}
```

---

## 实现建议

### AI 助手实现方案

**方案 A：后端集成 OpenAI/其他 LLM**
- 优点：统一管理 API Key，前端无需关心实现
- 缺点：需要后端开发

**方案 B：前端直接调用 OpenAI API**
- 优点：实现简单，后端无需改动
- 缺点：用户需要自己配置 API Key

**建议采用方案 A**，后端提供一个代理接口，可以：
1. 统一管理 API Key
2. 添加使用配额控制
3. 记录使用日志
4. 后续可切换不同 AI 提供商

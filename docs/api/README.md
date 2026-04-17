# Vigil API 文档

本文档描述了 Vigil 系统的 HTTP API 接口。

## API 概览

| 模块 | 文件 | 基础路径 |
|------|------|----------|
| 健康检查 | [health.md](health.md) | `/health` |
| 授权特征码 | [license.md](license.md) | `/api/license` |
| 进程管理 | [processes.md](processes.md) | `/api/processes/*`, `/api/namespaces/*` |
| 资源监控 | [resources.md](resources.md) | `/api/resources/*` |
| 配置管理 | [config.md](config.md) | `/api/config` |
| 命令执行 | [exec.md](exec.md) | `/api/exec` |
| 巡检检查 | [inspect.md](inspect.md) | `/api/inspect` |
| 文件管理 | [files.md](files.md) | `/api/files/*` |
| 用户管理 | [users.md](users.md) | `/api/users/*` |
| VM 管理 | [vms.md](vms.md) | `/api/vms/*` |
| 命令模板与历史 | [commands.md](commands.md) | `/api/commands/*` |
| 定时任务 | [schedules.md](schedules.md) | `/api/schedules/*` |
| AI 命令助手 | [ai.md](ai.md) | `/api/ai/*` |

## 快速导航

### 系统接口
- [健康检查](health.md) - 系统健康状态检查
- [授权特征码](license.md) - 获取服务器物理网卡的授权特征码
- [配置管理](config.md) - 系统配置读写
- [资源监控](resources.md) - 系统和进程资源监控

### 进程管理
- [进程管理](processes.md) - 进程的增删改查、启停控制

### 文件操作
- [文件管理](files.md) - 本地文件的上传、下载、列表、复制、移动

### VM 管理
- [VM 管理](vms.md) - VM 服务器管理、分组、SSH 连接、文件操作、权限管理、批量执行、资源监控

### 命令相关
- [命令执行](exec.md) - 远程命令执行
- [命令模板与历史](commands.md) - 命令模板管理、历史记录
- [AI 命令助手](ai.md) - AI 生成、解释、修复命令

### 定时任务
- [定时任务](schedules.md) - 定时任务管理、执行历史

### 系统运维
- [巡检检查](inspect.md) - 系统巡检检查

### 用户管理
- [用户管理](users.md) - 用户注册、登录、配置管理

## 通用说明

### 认证方式

API 支持两种认证方式：

1. **Basic Auth**（配置文件中预设的超级管理员）
   ```
   Authorization: Basic base64(username:password)
   ```

2. **数据库用户认证** - 通过 `/api/users/login` 接口获取用户信息

### 错误处理

API 使用标准 HTTP 状态码：

| 状态码 | 描述 |
|--------|------|
| 200 OK | 请求成功 |
| 201 Created | 创建成功 |
| 400 Bad Request | 请求参数错误 |
| 401 Unauthorized | 认证失败 |
| 404 Not Found | 资源未找到 |
| 409 Conflict | 资源冲突 |
| 500 Internal Server Error | 服务器内部错误 |

错误响应格式：
```json
{
  "error": "Error message"
}
```

### 版本控制

当前 API 版本为 v1，所有接口路径均以 `/api/` 开头。

## 示例客户端

### cURL 示例

**获取系统健康状态**：
```bash
curl -X GET http://localhost:8080/health
```

**列出进程**：
```bash
curl -X GET http://localhost:8080/api/namespaces/default/processes
```

**添加进程**：
```bash
curl -X POST -H "Content-Type: application/json" -d '{"metadata":{"name":"test-process","namespace":"default"},"command":"sleep","args":["10"]}' http://localhost:8080/api/namespaces/default/processes/test-process/add
```

### Python 示例

```python
import requests

# 获取系统健康状态
response = requests.get("http://localhost:8080/health")
print(response.json())

# 列出进程
response = requests.get("http://localhost:8080/api/namespaces/default/processes")
print(response.json())
```

### Go 示例

```go
package main

import (
  "encoding/json"
  "fmt"
  "net/http"
)

func main() {
  // 获取系统健康状态
  resp, err := http.Get("http://localhost:8080/health")
  if err != nil {
    panic(err)
  }
  defer resp.Body.Close()

  var result map[string]interface{}
  json.NewDecoder(resp.Body).Decode(&result)
  fmt.Println(result)
}
```

## 变更日志

### v1.2.0

- 新增定时任务 API `/api/schedules`
- 新增 AI 助手 API `/api/ai/*`

### v1.1.0

- 新增批量执行命令 API `/api/vms/batch/exec`
- 新增服务器资源监控 API `/api/vms/servers/{name}/resources`
- 新增批量获取服务器资源 API `/api/vms/resources/batch`
- 新增命令模板 API `/api/commands/templates`
- 新增命令历史 API `/api/commands/history`
- 新增跨服务器文件传输 API `/api/vms/files/transfer`
- 新增 VM 分组增强功能
- 新增用户登录接口 `/api/users/login`
- 用户数据迁移到 SQLite 数据库
- 支持启动时自动从 JSON 文件迁移用户数据

### v1.0.0

- 初始版本
- 支持进程管理、资源监控、配置管理等功能
- 支持 VM 管理、文件管理、权限管理等功能
- 支持 WebSocket SSH 连接

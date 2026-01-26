# HTTP API 接口说明文档

## 1. 概述

本文档描述了 Vigil 系统的 HTTP API 接口，包括接口路径、请求方法、请求参数、响应格式等。

## 2. 接口列表

| 模块 | 接口路径 | 请求方法 | 功能描述 |
|------|----------|----------|----------|
| 健康检查 | /health | GET | 检查系统健康状态 |
| 进程管理 | /api/processes/scan | GET | 扫描进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name}/add | POST | 添加进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name}/start | POST | 启动进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name}/stop | POST | 停止进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name}/restart | POST | 重启进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name} | GET | 获取进程详情 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name} | PUT | 编辑进程 |
| 进程管理 | /api/namespaces/{namespace}/processes/{name} | DELETE | 删除进程 |
| 进程管理 | /api/namespaces/{namespace}/processes | GET | 列出进程 |
| 资源监控 | /api/resources/system | GET | 获取系统资源信息 |
| 资源监控 | /api/resources/process/{pid} | GET | 获取进程资源信息 |
| 配置管理 | /api/config | GET | 获取配置 |
| 配置管理 | /api/config | PUT | 更新配置 |
| 命令执行 | /api/exec | POST | 执行命令 |
| 巡检检查 | /api/inspect | POST | 执行巡检检查 |
| VM 管理 | /api/vms | POST | 添加 VM |
| VM 管理 | /api/vms | GET | 列出 VM |
| VM 管理 | /api/vms/{name} | GET | 获取 VM 详情 |
| VM 管理 | /api/vms/{name} | PUT | 更新 VM |
| VM 管理 | /api/vms/{name} | DELETE | 删除 VM |
| VM 组管理 | /api/vms/groups | POST | 添加 VM 组 |
| VM 组管理 | /api/vms/groups | GET | 列出 VM 组 |
| VM 组管理 | /api/vms/groups/{name} | GET | 获取 VM 组详情 |
| VM 组管理 | /api/vms/groups/{name} | PUT | 更新 VM 组 |
| VM 组管理 | /api/vms/groups/{name} | DELETE | 删除 VM 组 |
| VM SSH | /api/vms/ssh/ws | WebSocket | WebSocket SSH 连接 |
| VM 文件管理 | /api/vms/files/upload | POST | 上传文件到 VM |
| VM 文件管理 | /api/vms/files/download | POST | 从 VM 下载文件 |
| VM 文件管理 | /api/vms/files/list | POST | 列出 VM 中的文件 |
| 权限管理 | /api/vms/permissions | POST | 添加权限 |
| 权限管理 | /api/vms/permissions | DELETE | 移除权限 |
| 权限管理 | /api/vms/permissions/check | POST | 检查权限 |
| 权限管理 | /api/vms/{name}/permissions | GET | 列出权限 |
| 文件管理 | /api/files/upload | POST | 上传文件 |
| 文件管理 | /api/files/download | POST | 下载文件 |
| 文件管理 | /api/files/list | POST | 列出文件 |
| 文件管理 | /api/files/delete | POST | 删除文件 |
| 文件管理 | /api/files/copy | POST | 复制文件 |
| 文件管理 | /api/files/move | POST | 移动文件 |

## 3. 详细接口说明

### 3.1 健康检查

#### GET /health

**功能描述**：检查系统健康状态

**请求参数**：无

**响应格式**：
```json
{
  "status": "ok"
}
```

### 3.2 进程管理

#### GET /api/processes/scan

**功能描述**：扫描进程

**请求参数**：
- `query`：扫描查询条件

**响应格式**：
```json
[
  {
    "metadata": {
      "name": "process-name",
      "namespace": "default",
      "description": "Process description"
    },
    "command": "command",
    "args": ["arg1", "arg2"],
    "env": {"key": "value"},
    "working_dir": "/path/to/dir",
    "restart": true,
    "user": "username"
  }
]
```

#### POST /api/namespaces/{namespace}/processes/{name}/add

**功能描述**：添加进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）
- 请求体：进程配置

**请求体示例**：
```json
{
  "metadata": {
    "name": "process-name",
    "namespace": "default",
    "description": "Process description"
  },
  "command": "command",
  "args": ["arg1", "arg2"],
  "env": {"key": "value"},
  "working_dir": "/path/to/dir",
  "restart": true,
  "user": "username"
}
```

**响应格式**：
- 成功：201 Created
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/namespaces/{namespace}/processes/{name}/start

**功能描述**：启动进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

#### POST /api/namespaces/{namespace}/processes/{name}/stop

**功能描述**：停止进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

#### POST /api/namespaces/{namespace}/processes/{name}/restart

**功能描述**：重启进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

#### GET /api/namespaces/{namespace}/processes/{name}

**功能描述**：获取进程详情

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
```json
{
  "metadata": {
    "name": "process-name",
    "namespace": "default",
    "description": "Process description"
  },
  "command": "command",
  "args": ["arg1", "arg2"],
  "env": {"key": "value"},
  "working_dir": "/path/to/dir",
  "restart": true,
  "user": "username"
}
```

#### PUT /api/namespaces/{namespace}/processes/{name}

**功能描述**：编辑进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）
- 请求体：进程配置

**请求体示例**：同添加进程

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 404 Not Found 或 500 Internal Server Error

#### DELETE /api/namespaces/{namespace}/processes/{name}

**功能描述**：删除进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

#### GET /api/namespaces/{namespace}/processes

**功能描述**：列出进程

**请求参数**：
- `namespace`：命名空间（路径参数）

**响应格式**：
```json
[
  {
    "metadata": {
      "name": "process-name",
      "namespace": "default",
      "description": "Process description"
    },
    "command": "command",
    "args": ["arg1", "arg2"],
    "env": {"key": "value"},
    "working_dir": "/path/to/dir",
    "restart": true,
    "user": "username"
  }
]
```

### 3.3 资源监控

#### GET /api/resources/system

**功能描述**：获取系统资源信息

**请求参数**：无

**响应格式**：
```json
{
  "cpu_usage": 10.5,
  "memory_usage": 50.2,
  "disk_usage": 75.8,
  "network_stats": {
    "rx_bytes": 1024,
    "tx_bytes": 2048
  }
}
```

#### GET /api/resources/process/{pid}

**功能描述**：获取进程资源信息

**请求参数**：
- `pid`：进程 ID（路径参数）

**响应格式**：
```json
{
  "pid": 1234,
  "cpu_usage": 5.2,
  "memory_usage": 20.5,
  "disk_usage": 10.1,
  "network_stats": {
    "rx_bytes": 512,
    "tx_bytes": 1024
  }
}
```

### 3.4 配置管理

#### GET /api/config

**功能描述**：获取配置

**请求参数**：无

**响应格式**：
```json
{
  "addr": ":8080",
  "auth": {
    "enabled": true,
    "username": "username",
    "password": "password"
  },
  "log": {
    "level": "info"
  },
  "monitor": {
    "rate": 60
  },
  "process": {
    "pid_file": "/path/to/pid/file"
  },
  "security": {
    "encryption_key": "encryption-key"
  },
  "https": {
    "enabled": false,
    "cert_path": "",
    "key_path": ""
  }
}
```

#### PUT /api/config

**功能描述**：更新配置

**请求参数**：
- 请求体：配置信息

**请求体示例**：同获取配置

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

### 3.5 命令执行

#### POST /api/exec

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

### 3.6 巡检检查

#### POST /api/inspect

**功能描述**：执行巡检检查

**请求参数**：
- 请求体：巡检请求

**请求体示例**：
```json
{
  "type": "type",
  "target": "target",
  "parameters": {
    "key": "value"
  }
}
```

**响应格式**：
```json
{
  "type": "type",
  "target": "target",
  "status": "success",
  "result": {
    "key": "value"
  },
  "errors": ["error message"]
}
```

### 3.7 VM 管理

#### POST /api/vms

**功能描述**：添加 VM

**请求参数**：
- 请求体：VM 信息

**请求体示例**：
```json
{
  "name": "vm-name",
  "ip": "192.168.1.1",
  "port": 22,
  "username": "username",
  "password": "password",
  "key_path": "/path/to/key"
}
```

**响应格式**：
```json
{
  "name": "vm-name",
  "ip": "192.168.1.1",
  "port": 22,
  "username": "username",
  "password": "password",
  "key_path": "/path/to/key"
}
```

#### GET /api/vms

**功能描述**：列出 VM

**请求参数**：无

**响应格式**：
```json
[
  {
    "name": "vm-name",
    "ip": "192.168.1.1",
    "port": 22,
    "username": "username",
    "password": "password",
    "key_path": "/path/to/key"
  }
]
```

#### GET /api/vms/{name}

**功能描述**：获取 VM 详情

**请求参数**：
- `name`：VM 名称（路径参数）

**响应格式**：
```json
{
  "name": "vm-name",
  "ip": "192.168.1.1",
  "port": 22,
  "username": "username",
  "password": "password",
  "key_path": "/path/to/key"
}
```

#### PUT /api/vms/{name}

**功能描述**：更新 VM

**请求参数**：
- `name`：VM 名称（路径参数）
- 请求体：VM 信息

**请求体示例**：
```json
{
  "password": "new-password",
  "key_path": "/path/to/new/key"
}
```

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

#### DELETE /api/vms/{name}

**功能描述**：删除 VM

**请求参数**：
- `name`：VM 名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

### 3.8 VM 组管理

#### POST /api/vms/groups

**功能描述**：添加 VM 组

**请求参数**：
- 请求体：组信息

**请求体示例**：
```json
{
  "name": "group-name",
  "description": "Group description",
  "vms": ["vm1", "vm2"]
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### GET /api/vms/groups

**功能描述**：列出 VM 组

**请求参数**：无

**响应格式**：
```json
[
  {
    "name": "group-name",
    "description": "Group description",
    "vms": ["vm1", "vm2"]
  }
]
```

#### GET /api/vms/groups/{name}

**功能描述**：获取 VM 组详情

**请求参数**：
- `name`：组名称（路径参数）

**响应格式**：
```json
{
  "name": "group-name",
  "description": "Group description",
  "vms": ["vm1", "vm2"]
}
```

#### PUT /api/vms/groups/{name}

**功能描述**：更新 VM 组

**请求参数**：
- `name`：组名称（路径参数）
- 请求体：组信息

**请求体示例**：
```json
{
  "description": "New group description",
  "vms": ["vm1", "vm2", "vm3"]
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 404 Not Found 或 500 Internal Server Error

#### DELETE /api/vms/groups/{name}

**功能描述**：删除 VM 组

**请求参数**：
- `name`：组名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

### 3.9 VM SSH

#### WebSocket /api/vms/ssh/ws

**功能描述**：WebSocket SSH 连接

**请求参数**：
- 查询参数：
  - `vm_name`：VM 名称

**响应格式**：WebSocket 连接，双向通信

### 3.10 VM 文件管理

#### POST /api/vms/files/upload

**功能描述**：上传文件到 VM

**请求参数**：
- 表单数据：
  - `file`：文件内容
  - `vm_name`：VM 名称
  - `target_path`：目标路径

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/vms/files/download

**功能描述**：从 VM 下载文件

**请求参数**：
- 请求体：
  - `vm_name`：VM 名称
  - `source_path`：源路径

**请求体示例**：
```json
{
  "vm_name": "vm-name",
  "source_path": "/path/to/file"
}
```

**响应格式**：
- 成功：文件内容
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/vms/files/list

**功能描述**：列出 VM 中的文件

**请求参数**：
- 请求体：
  - `vm_name`：VM 名称
  - `path`：路径
  - `max_depth`：最大深度

**请求体示例**：
```json
{
  "vm_name": "vm-name",
  "path": "/path/to/dir",
  "max_depth": 1
}
```

**响应格式**：
```json
[
  {
    "name": "file-name",
    "path": "/path/to/file",
    "size": 1024,
    "is_dir": false,
    "mode": "-rw-r--r--",
    "mod_time": "2023-01-01T00:00:00Z",
    "depth": 1
  }
]
```

### 3.11 权限管理

#### POST /api/vms/permissions

**功能描述**：添加权限

**请求参数**：
- 请求体：权限信息

**请求体示例**：
```json
{
  "vm_name": "vm-name",
  "username": "username",
  "permissions": ["read", "write"]
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### DELETE /api/vms/permissions

**功能描述**：移除权限

**请求参数**：
- 请求体：权限信息

**请求体示例**：
```json
{
  "vm_name": "vm-name",
  "username": "username",
  "permissions": ["read", "write"]
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/vms/permissions/check

**功能描述**：检查权限

**请求参数**：
- 请求体：
  - `vm_name`：VM 名称
  - `username`：用户名
  - `permission`：权限

**请求体示例**：
```json
{
  "vm_name": "vm-name",
  "username": "username",
  "permission": "read"
}
```

**响应格式**：
```json
{
  "has_permission": true
}
```

#### GET /api/vms/{name}/permissions

**功能描述**：列出权限

**请求参数**：
- `name`：VM 名称（路径参数）

**响应格式**：
```json
[
  {
    "vm_name": "vm-name",
    "username": "username",
    "permissions": ["read", "write"]
  }
]
```

### 3.12 文件管理

#### POST /api/files/upload

**功能描述**：上传文件

**请求参数**：
- 表单数据：
  - `file`：文件内容
  - `target_path`：目标路径

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/files/download

**功能描述**：下载文件

**请求参数**：
- 请求体：
  - `source_path`：源路径

**请求体示例**：
```json
{
  "source_path": "/path/to/file"
}
```

**响应格式**：
- 成功：文件内容
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/files/list

**功能描述**：列出文件

**请求参数**：
- 请求体：
  - `path`：路径
  - `max_depth`：最大深度

**请求体示例**：
```json
{
  "path": "/path/to/dir",
  "max_depth": 1
}
```

**响应格式**：
```json
[
  {
    "name": "file-name",
    "path": "/path/to/file",
    "size": 1024,
    "is_dir": false,
    "mode": "-rw-r--r--",
    "mod_time": "2023-01-01T00:00:00Z",
    "depth": 1
  }
]
```

#### POST /api/files/delete

**功能描述**：删除文件

**请求参数**：
- 请求体：
  - `path`：路径

**请求体示例**：
```json
{
  "path": "/path/to/file"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/files/copy

**功能描述**：复制文件

**请求参数**：
- 请求体：
  - `source_path`：源路径
  - `target_path`：目标路径

**请求体示例**：
```json
{
  "source_path": "/path/to/source",
  "target_path": "/path/to/target"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

#### POST /api/files/move

**功能描述**：移动文件

**请求参数**：
- 请求体：
  - `source_path`：源路径
  - `target_path`：目标路径

**请求体示例**：
```json
{
  "source_path": "/path/to/source",
  "target_path": "/path/to/target"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

## 4. 错误处理

API 使用标准 HTTP 状态码来表示错误：

| 状态码 | 描述 |
|--------|------|
| 400 Bad Request | 请求参数错误 |
| 404 Not Found | 资源未找到 |
| 500 Internal Server Error | 服务器内部错误 |

错误响应格式：
```json
{
  "error": "Error message"
}
```

## 5. 认证

API 支持 Basic Auth 认证，需要在请求头中添加 Authorization 字段：

```
Authorization: Basic base64(username:password)
```

## 6. 版本控制

当前 API 版本为 v1，所有接口路径均以 `/api/` 开头。

## 7. 速率限制

API 目前没有速率限制，但建议客户端合理控制请求频率。

## 8. 最佳实践

1. 使用 HTTPS 协议访问 API（如果启用）
2. 合理设置超时时间
3. 处理错误响应
4. 不要在请求中包含敏感信息
5. 定期更新 API 客户端

## 9. 示例客户端

### 9.1 cURL 示例

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

### 9.2 Python 示例

```python
import requests

# 获取系统健康状态
response = requests.get("http://localhost:8080/health")
print(response.json())

# 列出进程
response = requests.get("http://localhost:8080/api/namespaces/default/processes")
print(response.json())
```

### 9.3 Go 示例

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

## 10. 变更日志

### v1.0.0

- 初始版本
- 支持进程管理、资源监控、配置管理等功能
- 支持 VM 管理、文件管理、权限管理等功能
- 支持 WebSocket SSH 连接

## 11. 联系我们

如果您在使用 API 过程中遇到问题，请联系我们：

- 电子邮件：support@example.com
- 文档：https://example.com/docs
- GitHub：https://github.com/example/vigil

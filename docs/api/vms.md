# VM 管理 API

## 接口列表

| 模块 | 接口路径 | 请求方法 | 功能描述 |
|------|---------|----------|----------|
| VM 服务器管理 | /api/vms/servers/{name} | POST | 添加 VM |
| VM 服务器管理 | /api/vms/servers | GET | 列出 VM |
| VM 服务器管理 | /api/vms/servers/{name} | GET | 获取 VM 详情 |
| VM 服务器管理 | /api/vms/servers/{name} | PUT | 更新 VM |
| VM 服务器管理 | /api/vms/servers/{name} | DELETE | 删除 VM |
| VM 组管理 | /api/vms/groups/{name} | POST | 添加 VM 组 |
| VM 组管理 | /api/vms/groups | GET | 列出 VM 组 |
| VM 组管理 | /api/vms/groups/{name} | GET | 获取 VM 组详情 |
| VM 组管理 | /api/vms/groups/{name} | PUT | 更新 VM 组 |
| VM 组管理 | /api/vms/groups/{name} | DELETE | 删除 VM 组 |
| VM SSH | /api/vms/ssh/ws | WebSocket | WebSocket SSH 连接 |
| VM 文件管理 | /api/vms/files/{name}/upload | POST | 上传文件到 VM |
| VM 文件管理 | /api/vms/files/{name}/download | POST | 从 VM 下载文件 |
| VM 文件管理 | /api/vms/files/{name}/list | POST | 列出 VM 中的文件 |
| VM 文件管理 | /api/vms/files/{name}/delete | POST | 删除 VM 上的文件 |
| VM 文件管理 | /api/vms/files/{name}/mkdir | POST | 在 VM 上新建文件夹 |
| VM 文件管理 | /api/vms/files/{name}/touch | POST | 在 VM 上新建文件 |
| VM 文件管理 | /api/vms/files/{name}/rmdir | POST | 删除 VM 上的文件夹 |
| 权限管理 | /api/vms/permissions/{name} | POST | 添加权限 |
| 权限管理 | /api/vms/permissions/{name} | DELETE | 移除权限 |
| 权限管理 | /api/vms/permissions/{name}/check | POST | 检查权限 |
| 权限管理 | /api/vms/servers/{name}/permissions | GET | 列出权限 |
| VM 命令执行 | /api/vms/servers/{name}/exec | POST | 在 VM 上执行命令 |
| VM 连接测试 | /api/vms/servers/{name}/ping | GET | 测试 VM 连接 |
| VM SSH 连接管理 | /api/vms/ssh/connections | GET | 列出活动的 SSH 连接 |
| VM SSH 连接管理 | /api/vms/ssh/connections | DELETE | 关闭所有 SSH 连接 |
| VM SSH 连接管理 | /api/vms/ssh/connections/{id} | DELETE | 关闭特定 SSH 连接 |
| 批量执行 | /api/vms/batch/exec | POST | 批量执行命令 |
| 资源监控 | /api/vms/servers/{name}/resources | GET | 单服务器资源监控 |
| 资源监控 | /api/vms/resources/batch | POST | 批量获取服务器资源 |
| 文件传输 | /api/vms/files/transfer | POST | 跨服务器文件传输 |

---

## VM 服务器管理

### POST /api/vms/servers/{name}

**功能描述**：添加 VM

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：VM 信息（不包含 name 字段）

**请求体示例**：
```json
{
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
  "message": "VM added successfully"
}
```

---

### GET /api/vms/servers

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
    "key_path": "/path/to/key"
  }
]
```

---

### GET /api/vms/servers/{name}

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
  "key_path": "/path/to/key"
}
```

---

### PUT /api/vms/servers/{name}

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
```json
{
  "message": "VM updated successfully"
}
```

---

### DELETE /api/vms/servers/{name}

**功能描述**：删除 VM

**请求参数**：
- `name`：VM 名称（路径参数）

**响应格式**：
```json
{
  "message": "VM deleted successfully"
}
```

---

## VM 组管理

### POST /api/vms/groups/{name}

**功能描述**：添加 VM 组

**请求参数**：
- 路径参数：
  - `name`：组名称
- 请求体：组信息（不包含 name 字段）

**请求体示例**：
```json
{
  "description": "Group description",
  "vms": ["vm1", "vm2"],
  "is_shared": true,
  "shared_with": []
}
```

**新增字段说明**：
| 字段 | 类型 | 说明 |
|------|------|------|
| is_shared | boolean | 是否共享给团队，默认 false |
| shared_with | string[] | 共享给指定用户，空数组表示全员可见 |

**响应格式**：
```json
{
  "message": "Group added successfully"
}
```

---

### GET /api/vms/groups

**功能描述**：列出 VM 组

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| shared_only | boolean | 仅获取共享分组 |
| mine_only | boolean | 仅获取自己创建的分组 |

**响应格式**：
```json
[
  {
    "name": "group-name",
    "description": "Group description",
    "vms": ["vm1", "vm2"],
    "owner": "user1",
    "is_shared": true,
    "shared_with": [],
    "created_at": "2026-03-21T10:00:00Z"
  }
]
```

---

### GET /api/vms/groups/{name}

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

---

### PUT /api/vms/groups/{name}

**功能描述**：更新 VM 组

**请求参数**：
- `name`：组名称（路径参数）
- 请求体：组信息

**请求体示例**：
```json
{
  "description": "New group description",
  "vms": ["vm1", "vm2", "vm3"],
  "is_shared": true,
  "shared_with": ["user2"]
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 404 Not Found 或 500 Internal Server Error

---

### DELETE /api/vms/groups/{name}

**功能描述**：删除 VM 组

**请求参数**：
- `name`：组名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

---

## VM SSH

### WebSocket /api/vms/ssh/ws

**功能描述**：WebSocket SSH 连接

**请求参数**：
- 查询参数：
  - `vm_name`：VM 名称

**响应格式**：WebSocket 连接，双向通信

**使用说明**：
1. 客户端通过 WebSocket 协议连接到此端点
2. 连接时需要提供 `vm_name` 查询参数指定要连接的 VM
3. 连接建立后，客户端可以发送 SSH 命令，服务端返回命令执行结果
4. 支持窗口大小调整，客户端可以发送 `resize:{"cols":120,"rows":40}` 格式的消息调整终端大小

---

## VM SSH 连接管理

### GET /api/vms/ssh/connections

**功能描述**：列出活动的 SSH 连接

**请求参数**：
- 查询参数：
  - `vm_name`：（可选）过滤特定 VM 的连接
  - `user_name`：（可选）过滤特定用户的连接
  - `client_ip`：（可选）过滤来自特定 IP 的连接

**响应格式**：
```json
[
  {
    "id": "connection-id",
    "vm_name": "vm-name",
    "client_ip": "192.168.1.100",
    "username": "authenticated-user",
    "connected_at": "2023-01-01T12:00:00Z",
    "duration": "5m23s"
  }
]
```

---

### DELETE /api/vms/ssh/connections

**功能描述**：关闭所有 SSH 连接

**请求参数**：无

**响应格式**：
```json
{
  "message": "All SSH connections closed successfully",
  "count": 3
}
```

---

### DELETE /api/vms/ssh/connections/{id}

**功能描述**：关闭特定 SSH 连接

**请求参数**：
- `id`：连接 ID（路径参数）

**响应格式**：
```json
{
  "message": "SSH connection closed successfully"
}
```

---

## VM 文件管理

### POST /api/vms/files/{name}/upload

**功能描述**：上传文件到 VM

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 表单数据：
  - `file`：文件内容
  - `target_path`：目标路径

**响应格式**：
```json
{
  "message": "File uploaded successfully"
}
```

---

### POST /api/vms/files/{name}/download

**功能描述**：从 VM 下载文件

**请求参数**：
- 路径参数：
  - `name`：VM 名称
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

---

### POST /api/vms/files/{name}/list

**功能描述**：列出 VM 中的文件

**请求参数**：
- 路径参数：
  - `name`：VM 名称
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

---

### POST /api/vms/files/{name}/delete

**功能描述**：删除 VM 上的文件

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `path`：文件路径

**请求体示例**：
```json
{
  "path": "/tmp/old.log"
}
```

**响应格式**：
```json
{
  "message": "File deleted successfully"
}
```

---

### POST /api/vms/files/{name}/mkdir

**功能描述**：在 VM 上新建文件夹

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `path`：目录路径
  - `parents`：是否创建父目录（可选，默认 false）

**请求体示例**：
```json
{
  "path": "/home/app/logs",
  "parents": true
}
```

**响应格式**：
```json
{
  "message": "Directory created successfully"
}
```

---

### POST /api/vms/files/{name}/touch

**功能描述**：在 VM 上新建文件

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `path`：文件路径

**请求体示例**：
```json
{
  "path": "/tmp/newfile.txt"
}
```

**响应格式**：
```json
{
  "message": "File created successfully"
}
```

---

### POST /api/vms/files/{name}/rmdir

**功能描述**：删除 VM 上的文件夹

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `path`：目录路径
  - `recursive`：是否递归删除（可选，默认 false）

**请求体示例**：
```json
{
  "path": "/var/log/nginx",
  "recursive": true
}
```

**响应格式**：
```json
{
  "message": "Directory deleted successfully"
}
```

---

## 权限管理

### POST /api/vms/permissions/{name}

**功能描述**：添加权限

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：权限信息

**请求体示例**：
```json
{
  "username": "username",
  "permissions": ["read", "write"]
}
```

**响应格式**：
```json
{
  "message": "Permission added successfully"
}
```

---

### DELETE /api/vms/permissions/{name}

**功能描述**：移除权限

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：权限信息

**请求体示例**：
```json
{
  "username": "username",
  "permissions": ["read", "write"]
}
```

**响应格式**：
```json
{
  "message": "Permission removed successfully"
}
```

---

### POST /api/vms/permissions/{name}/check

**功能描述**：检查权限

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `username`：用户名
  - `permission`：权限

**请求体示例**：
```json
{
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

---

### GET /api/vms/servers/{name}/permissions

**功能描述**：列出权限

**请求参数**：
- 路径参数：
  - `name`：VM 名称

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

---

## VM 命令执行

### POST /api/vms/servers/{name}/exec

**功能描述**：在虚拟机上执行远程命令

**请求参数**：
- 路径参数：
  - `name`：VM 名称
- 请求体：
  - `command`：要执行的命令
  - `timeout`：超时时间（秒，可选，默认 30）

**请求体示例**：
```json
{
  "command": "uptime",
  "timeout": 30
}
```

**响应格式**：
```json
{
  "vm": "vm-name",
  "command": "uptime",
  "output": " 10:30:45 up 1 day,  2:15,  1 user,  load average: 0.05, 0.10, 0.05",
  "status": "success"
}
```

**错误响应**：
```json
{
  "error": "Failed to connect to VM"
}
```

---

## VM 连接测试

### GET /api/vms/servers/{name}/ping

**功能描述**：测试虚拟机 SSH 端口的连接性

**请求参数**：
- 路径参数：
  - `name`：VM 名称

**成功响应示例**：
```json
{
  "vm": "vm1",
  "ip": "192.168.1.100",
  "port": 22,
  "success": true,
  "status": "OK",
  "latency_ms": 2.45
}
```

**失败响应示例**：
```json
{
  "vm": "vm2",
  "ip": "192.168.1.101",
  "port": 22,
  "success": false,
  "status": "TIMEOUT",
  "latency_ms": null,
  "message": "connection timeout"
}
```

---

## 批量执行命令

### POST /api/vms/batch/exec

**功能描述**：在多台服务器上批量执行命令

**请求体**：
```json
{
  "vm_names": ["vm1", "vm2", "vm3"],
  "command": "uptime",
  "timeout": 30,
  "parallel": true
}
```

**参数说明**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| vm_names | string[] | 是 | 目标服务器名称列表 |
| command | string | 是 | 要执行的命令 |
| timeout | number | 否 | 超时时间（秒），默认 30 |
| parallel | boolean | 否 | 是否并行执行，默认 true |

**响应**：
```json
{
  "task_id": "task_123",
  "total": 3,
  "success": 2,
  "failed": 1,
  "results": [
    {
      "vm_name": "vm1",
      "status": "success",
      "output": " 10:30:45 up 1 day,  2:15,  1 user,  load average: 0.05, 0.10, 0.05",
      "duration_ms": 150
    },
    {
      "vm_name": "vm2",
      "status": "success",
      "output": " 10:30:46 up 5 days,  1:30,  2 users,  load average: 0.15, 0.20, 0.18",
      "duration_ms": 200
    },
    {
      "vm_name": "vm3",
      "status": "failed",
      "output": "",
      "error": "connection timeout",
      "duration_ms": 30000
    }
  ]
}
```

---

## 资源监控

### GET /api/vms/servers/{name}/resources

**功能描述**：获取指定服务器的实时资源使用情况

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | VM 名称 |

**响应**：
```json
{
  "vm_name": "vm1",
  "cpu_usage": 45.2,
  "memory_usage": 62.8,
  "memory_total_gb": 16,
  "memory_used_gb": 10.05,
  "disk_usage": 55.0,
  "disk_total_gb": 500,
  "disk_used_gb": 275,
  "load_average": [0.5, 0.8, 1.2],
  "uptime": "10 days, 5:30:00",
  "network": {
    "rx_bytes_per_sec": 1024000,
    "tx_bytes_per_sec": 512000
  },
  "collected_at": "2026-03-21T10:00:00Z"
}
```

---

### POST /api/vms/resources/batch

**功能描述**：批量获取多台服务器的资源使用情况

**请求体**：
```json
{
  "vm_names": ["vm1", "vm2", "vm3"]
}
```

**响应**：
```json
[
  {
    "vm_name": "vm1",
    "cpu_usage": 45.2,
    "memory_usage": 62.8,
    "disk_usage": 55.0,
    "status": "ok"
  },
  {
    "vm_name": "vm2",
    "cpu_usage": 85.5,
    "memory_usage": 78.2,
    "disk_usage": 90.0,
    "status": "warning"
  },
  {
    "vm_name": "vm3",
    "status": "error",
    "error": "connection failed"
  }
]
```

**说明**：
- 返回简化版的资源信息，适合列表展示
- `status` 字段：ok（正常）/ warning（警告）/ error（异常）

---

## 跨服务器文件传输

### POST /api/vms/files/transfer

**功能描述**：在两台服务器之间直接传输文件，无需下载到本地

**请求体**：
```json
{
  "source_vm": "vm1",
  "source_path": "/var/log/app.log",
  "target_vm": "vm2",
  "target_path": "/backup/app.log"
}
```

**参数说明**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source_vm | string | 是 | 源服务器名称 |
| source_path | string | 是 | 源文件路径 |
| target_vm | string | 是 | 目标服务器名称 |
| target_path | string | 是 | 目标文件路径 |

**响应**：
```json
{
  "message": "File transferred successfully",
  "bytes_transferred": 1024000,
  "duration_ms": 2500
}
```

**错误响应**：
```json
{
  "error": "Source file not found",
  "source_vm": "vm1",
  "source_path": "/var/log/app.log"
}
```

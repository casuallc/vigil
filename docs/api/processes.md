# 进程管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/processes/scan | GET | 扫描进程 |
| /api/namespaces/{namespace}/processes/{name}/add | POST | 添加进程 |
| /api/namespaces/{namespace}/processes/{name}/start | POST | 启动进程 |
| /api/namespaces/{namespace}/processes/{name}/stop | POST | 停止进程 |
| /api/namespaces/{namespace}/processes/{name}/restart | POST | 重启进程 |
| /api/namespaces/{namespace}/processes/{name} | GET | 获取进程详情 |
| /api/namespaces/{namespace}/processes/{name} | PUT | 编辑进程 |
| /api/namespaces/{namespace}/processes/{name} | DELETE | 删除进程 |
| /api/namespaces/{namespace}/processes | GET | 列出进程 |

---

## GET /api/processes/scan

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

---

## POST /api/namespaces/{namespace}/processes/{name}/add

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

---

## POST /api/namespaces/{namespace}/processes/{name}/start

**功能描述**：启动进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

---

## POST /api/namespaces/{namespace}/processes/{name}/stop

**功能描述**：停止进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

---

## POST /api/namespaces/{namespace}/processes/{name}/restart

**功能描述**：重启进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

---

## GET /api/namespaces/{namespace}/processes/{name}

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

---

## PUT /api/namespaces/{namespace}/processes/{name}

**功能描述**：编辑进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）
- 请求体：进程配置

**请求体示例**：同添加进程

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 404 Not Found 或 500 Internal Server Error

---

## DELETE /api/namespaces/{namespace}/processes/{name}

**功能描述**：删除进程

**请求参数**：
- `namespace`：命名空间（路径参数）
- `name`：进程名称（路径参数）

**响应格式**：
- 成功：200 OK
- 失败：404 Not Found 或 500 Internal Server Error

---

## GET /api/namespaces/{namespace}/processes

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

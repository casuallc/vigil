# 命令模板与历史 API

## 接口列表

| 模块 | 接口路径 | 请求方法 | 功能描述 |
|------|---------|----------|----------|
| 命令模板 | /api/commands/templates | POST | 创建命令模板 |
| 命令模板 | /api/commands/templates | GET | 列出命令模板 |
| 命令模板 | /api/commands/templates/{id} | GET | 获取模板详情 |
| 命令模板 | /api/commands/templates/{id} | PUT | 更新模板 |
| 命令模板 | /api/commands/templates/{id} | DELETE | 删除模板 |
| 命令历史 | /api/commands/history | POST | 记录命令执行 |
| 命令历史 | /api/commands/history | GET | 获取命令历史 |
| 命令历史 | /api/commands/history/{id} | DELETE | 删除历史记录 |

---

## 命令模板

### POST /api/commands/templates

**功能描述**：创建命令模板

**请求体**：
```json
{
  "name": "查看最近错误日志",
  "description": "查看指定服务的最近 N 条错误日志",
  "command": "tail -n ${lines} /var/log/${service}/error.log",
  "variables": [
    {
      "name": "lines",
      "label": "行数",
      "type": "number",
      "default": "50"
    },
    {
      "name": "service",
      "label": "服务",
      "type": "select",
      "default": "nginx",
      "options": ["nginx", "apache", "app"]
    }
  ],
  "category": "diagnostic",
  "is_shared": true
}
```

**响应**：
```json
{
  "id": "tpl_123",
  "name": "查看最近错误日志",
  "description": "查看指定服务的最近 N 条错误日志",
  "command": "tail -n ${lines} /var/log/${service}/error.log",
  "variables": [...],
  "category": "diagnostic",
  "is_shared": true,
  "created_by": "user1",
  "created_at": "2026-03-21T10:00:00Z"
}
```

---

### GET /api/commands/templates

**功能描述**：列出命令模板

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| category | string | 按分类筛选 |
| shared | boolean | 仅共享模板 |
| search | string | 搜索关键词 |

**响应**：
```json
[
  {
    "id": "tpl_123",
    "name": "查看最近错误日志",
    "description": "...",
    "category": "diagnostic",
    "is_shared": true,
    "created_by": "user1"
  }
]
```

---

### GET /api/commands/templates/{id}

**功能描述**：获取模板详情

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 模板 ID |

---

### PUT /api/commands/templates/{id}

**功能描述**：更新模板

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 模板 ID |

---

### DELETE /api/commands/templates/{id}

**功能描述**：删除模板

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 模板 ID |

---

## 命令历史

### POST /api/commands/history

**功能描述**：记录命令执行

**请求体**：
```json
{
  "vm_name": "vm1",
  "command": "tail -n 50 /var/log/nginx/error.log",
  "status": "success",
  "duration_ms": 150
}
```

**响应**：
```json
{
  "id": "hist_123",
  "vm_name": "vm1",
  "command": "tail -n 50 /var/log/nginx/error.log",
  "executed_by": "user1",
  "executed_at": "2026-03-21T10:00:00Z",
  "status": "success",
  "duration_ms": 150
}
```

---

### GET /api/commands/history

**功能描述**：获取命令历史

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| vm_name | string | 按服务器筛选 |
| search | string | 搜索命令内容 |
| page | number | 页码，默认 1 |
| page_size | number | 每页数量，默认 20 |

**响应**：
```json
{
  "total": 150,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "id": "hist_123",
      "vm_name": "vm1",
      "command": "tail -n 50 /var/log/nginx/error.log",
      "executed_by": "user1",
      "executed_at": "2026-03-21T10:00:00Z",
      "status": "success",
      "duration_ms": 150
    }
  ]
}
```

---

### DELETE /api/commands/history/{id}

**功能描述**：删除历史记录

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 历史记录 ID |

---

## 数据库表结构

### 命令模板表 (command_templates)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 模板 ID |
| name | TEXT | 模板名称 |
| description | TEXT | 描述 |
| command | TEXT | 命令内容 |
| variables | TEXT | 变量定义（JSON） |
| category | TEXT | 分类 |
| is_shared | INTEGER | 是否共享（0/1） |
| created_by | TEXT | 创建者用户名 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 命令历史表 (command_history)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 记录 ID |
| vm_name | TEXT | 服务器名称 |
| command | TEXT | 命令内容 |
| executed_by | TEXT | 执行者用户名 |
| executed_at | DATETIME | 执行时间 |
| status | TEXT | 执行状态 |
| duration_ms | INTEGER | 耗时（毫秒） |

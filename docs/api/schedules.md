# 定时任务 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/schedules | POST | 创建定时任务 |
| /api/schedules | GET | 获取定时任务列表 |
| /api/schedules/{id} | GET | 获取定时任务详情 |
| /api/schedules/{id} | PUT | 更新定时任务 |
| /api/schedules/{id} | DELETE | 删除定时任务 |
| /api/schedules/{id}/toggle | POST | 启用/禁用定时任务 |
| /api/schedules/{id}/run | POST | 立即执行一次 |
| /api/schedules/{id}/history | GET | 获取任务执行历史 |

---

## POST /api/schedules

**功能描述**：创建定时任务

**请求体**：
```json
{
  "name": "清理日志文件",
  "description": "每天凌晨清理7天前的日志",
  "command": "find /var/log -name '*.log' -mtime +7 -delete",
  "vm_names": ["vm1", "vm2"],
  "cron": "0 0 * * *",
  "enabled": true,
  "timeout": 300
}
```

**参数说明**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 任务名称 |
| description | string | 否 | 任务描述 |
| command | string | 是 | 要执行的命令 |
| vm_names | string[] | 是 | 目标服务器列表 |
| cron | string | 是 | Cron 表达式（5位） |
| enabled | boolean | 否 | 是否启用，默认 true |
| timeout | number | 否 | 超时时间（秒），默认 300 |

**响应**：
```json
{
  "id": "schedule_123",
  "name": "清理日志文件",
  "description": "每天凌晨清理7天前的日志",
  "command": "find /var/log -name '*.log' -mtime +7 -delete",
  "vm_names": ["vm1", "vm2"],
  "cron": "0 0 * * *",
  "enabled": true,
  "timeout": 300,
  "created_by": "user1",
  "created_at": "2026-03-22T10:00:00Z",
  "next_run_at": "2026-03-23T00:00:00Z"
}
```

---

## GET /api/schedules

**功能描述**：获取定时任务列表

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| enabled | boolean | 按启用状态筛选 |
| vm_name | string | 按服务器筛选 |

**响应**：
```json
[
  {
    "id": "schedule_123",
    "name": "清理日志文件",
    "command": "find /var/log -name '*.log' -mtime +7 -delete",
    "vm_names": ["vm1", "vm2"],
    "cron": "0 0 * * *",
    "enabled": true,
    "next_run_at": "2026-03-23T00:00:00Z",
    "last_run_at": "2026-03-22T00:00:00Z",
    "last_run_status": "success"
  }
]
```

---

## GET /api/schedules/{id}

**功能描述**：获取定时任务详情

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

---

## PUT /api/schedules/{id}

**功能描述**：更新定时任务

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

---

## DELETE /api/schedules/{id}

**功能描述**：删除定时任务

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

---

## POST /api/schedules/{id}/toggle

**功能描述**：启用/禁用定时任务

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

**响应**：
```json
{
  "id": "schedule_123",
  "enabled": false
}
```

---

## POST /api/schedules/{id}/run

**功能描述**：立即执行一次

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

**响应**：
```json
{
  "message": "Task triggered successfully",
  "execution_id": "exec_456"
}
```

---

## GET /api/schedules/{id}/history

**功能描述**：获取任务执行历史

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | number | 页码 |
| page_size | number | 每页数量 |

**响应**：
```json
{
  "total": 30,
  "items": [
    {
      "id": "exec_456",
      "schedule_id": "schedule_123",
      "triggered_at": "2026-03-22T00:00:00Z",
      "completed_at": "2026-03-22T00:00:05Z",
      "status": "success",
      "results": [
        {
          "vm_name": "vm1",
          "status": "success",
          "output": "Deleted 15 files"
        }
      ]
    }
  ]
}
```

---

## 数据库表结构

### 定时任务表 (schedules)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 任务 ID |
| name | TEXT | 任务名称 |
| description | TEXT | 任务描述 |
| command | TEXT | 执行命令 |
| vm_names | TEXT | 目标服务器（JSON 数组） |
| cron | TEXT | Cron 表达式 |
| enabled | INTEGER | 是否启用（0/1） |
| timeout | INTEGER | 超时时间（秒） |
| created_by | TEXT | 创建者 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |
| last_run_at | DATETIME | 上次执行时间 |
| last_run_status | TEXT | 上次执行状态 |

### 定时任务执行历史表 (schedule_executions)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 执行 ID |
| schedule_id | TEXT | 关联的任务 ID |
| triggered_at | DATETIME | 触发时间 |
| completed_at | DATETIME | 完成时间 |
| status | TEXT | 执行状态 |
| results | TEXT | 执行结果（JSON） |

---

## Cron 表达式说明

使用标准 5 位格式：`分 时 日 月 周`

示例：
- `0 0 * * *` - 每天凌晨 0 点
- `*/30 * * * *` - 每 30 分钟
- `0 9 * * 1-5` - 工作日早上 9 点
- `0 0 1 * *` - 每月 1 号凌晨

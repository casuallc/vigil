# 系统管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/system/upgrade | POST | 触发平滑升级（仅 super admin） |
| /api/system/status | GET | 获取服务器运行状态 |
| /api/system/upgrade/status | GET | 获取升级进度（轮询用） |

---

## POST /api/system/upgrade

**功能描述**：触发 bbx-server 平滑升级。新进程启动并完成自检后，旧进程自动优雅退出，新进程接管端口。

**权限要求**：仅 super admin 可调用（启用 Basic Auth 时）。

**Content-Type**：`multipart/form-data`（上传方式）或 `application/json`（下载方式）

### 方式一：API 上传

**请求参数**（multipart/form-data）：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| method | string | 是 | 固定值 `"upload"` |
| file | file | 是 | 新二进制文件 |
| checksum | string | 否 | SHA256 校验值，格式 `"sha256:abc123..."` |

**请求示例**：
```bash
curl -u admin:password -X POST http://localhost:57575/api/system/upgrade \
  -F "method=upload" \
  -F "file=@./bbx-server.new" \
  -F "checksum=sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
```

### 方式二：远程下载

**请求参数**（application/json）：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| method | string | 是 | 固定值 `"download"` |
| url | string | 是 | 新二进制下载地址 |
| checksum | string | 否 | SHA256 校验值，格式 `"sha256:abc123..."` |

**请求示例**：
```bash
curl -u admin:password -X POST http://localhost:57575/api/system/upgrade \
  -H "Content-Type: application/json" \
  -d '{
    "method": "download",
    "url": "https://artifacts.example.com/bbx-server/v1.3.0/linux-amd64/bbx-server",
    "checksum": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  }'
```

**响应格式**：

成功（202 Accepted）：
```json
{
  "message": "upgrade started",
  "state": "downloading"
}
```

已有升级进行中（409 Conflict）：
```json
{
  "error": "upgrade already in progress: downloading"
}
```

权限不足（403 Forbidden）：
```json
{
  "error": "only super admin can perform upgrade"
}
```

---

## GET /api/system/status

**功能描述**：获取服务器当前运行状态，包括 PID、版本、运行时长、升级状态、监听地址、可执行文件路径等。

**请求参数**：无

**响应格式**：
```json
{
  "pid": 1234,
  "version": "1.2.3",
  "uptime_seconds": 86400,
  "started_at": "2026-05-26T10:00:00Z",
  "upgrade_state": "idle",
  "listeners": [":57575"],
  "executable_path": "/usr/local/bin/bbx-server",
  "go_version": "go1.24.0",
  "os": "linux",
  "arch": "amd64"
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| pid | int | 当前进程 PID |
| version | string | 程序版本号 |
| uptime_seconds | int | 运行时长（秒） |
| started_at | string | 启动时间（RFC3339） |
| upgrade_state | string | 升级状态，见下方状态机说明 |
| listeners | array | 监听地址列表 |
| executable_path | string | 可执行文件绝对路径 |
| go_version | string | Go 版本 |
| os | string | 操作系统 |
| arch | string | CPU 架构 |

---

## GET /api/system/upgrade/status

**功能描述**：获取当前升级任务的详细进度，用于客户端轮询。

**请求参数**：无

**响应格式**：

无升级进行时：
```json
{
  "state": "idle"
}
```

升级进行中：
```json
{
  "state": "health_checking",
  "new_pid": 5678,
  "started_at": "2026-05-26T10:01:00Z",
  "completed_at": null,
  "error": "",
  "binary_path": "data/bbx-server.new"
}
```

升级失败：
```json
{
  "state": "failed",
  "new_pid": 5678,
  "started_at": "2026-05-26T10:01:00Z",
  "completed_at": null,
  "error": "health check failed: connection refused",
  "binary_path": "data/bbx-server.new"
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| state | string | 升级状态 |
| new_pid | int | 新进程 PID |
| started_at | string | 升级开始时间（RFC3339） |
| completed_at | string / null | 升级完成时间（RFC3339） |
| error | string | 失败原因（失败时非空） |
| binary_path | string | 新二进制临时路径 |

---

## 升级状态机

| 状态 | 含义 |
|------|------|
| `idle` | 无升级进行中 |
| `downloading` | 正在下载/接收新二进制 |
| `verifying` | 正在校验新二进制 |
| `starting` | 新进程已启动，正在初始化 |
| `health_checking` | 新进程自检中 |
| `ready` | 新进程自检通过，等待旧进程释放端口 |
| `completed` | 端口交接完成，新进程已接管 |
| `failed` | 升级失败，旧进程继续运行 |

---

## 错误码说明

| HTTP 状态码 | 场景 |
|-------------|------|
| 202 Accepted | 升级请求已接受，正在后台执行 |
| 400 Bad Request | 请求参数错误（如缺少 url、非法 method） |
| 403 Forbidden | 权限不足（非 super admin） |
| 409 Conflict | 已有升级正在进行中 |
| 500 Internal Server Error | 读取状态文件失败等内部错误 |

---

## 完整升级流程示例

```bash
# 1. 触发升级
curl -u admin:password -X POST http://localhost:57575/api/system/upgrade \
  -F "method=upload" -F "file=@./bbx-server.new"
# => {"message":"upgrade started","state":"downloading"}

# 2. 轮询升级进度（每 1~2 秒查询一次）
curl -u admin:password http://localhost:57575/api/system/upgrade/status
# => {"state":"health_checking","new_pid":5678,...}

# 3. 升级完成后，新进程已接管端口，旧进程自动退出
# 无需手动操作，继续正常调用 API 即可
```

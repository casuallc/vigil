# Poll Mode 上游协议

Poll 模式用于**只允许出方向请求**的网络环境：bbx 主动向各上游服务长轮询拉取任务，执行后回执结果。本文档描述**上游服务需要实现**的协议接口（bbx 是客户端）。

设计与行为细节（busy/idle 状态机、topic 并发模型、可靠性语义）见 [../poll-mode.md](../poll-mode.md)；bbx 侧配置见 conf/config.yaml 的 `poll` 段。

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| {endpoint}/poll | GET | bbx 长轮询拉取任务 |
| {endpoint}/ack | POST | bbx 回执任务结果 |

## 通用说明

- **认证**：bbx 对每个请求携带 `Authorization: Basic base64(username:password)`，凭据在 bbx 配置中按上游独立设置；上游应校验。
- **数据面分离**：poll/ack 控制面只承载小 JSON（ack 内嵌结果 ≤ 64KB）；文件、日志流、交互式会话等大数据由任务携带目标地址、bbx 主动外拨传输（见任务类型）。
- **幂等**：上游发出任务后等待 ack，超时未收到应重派同一任务（同 `id`）；bbx 按 `id` 去重，重派命中直接返回缓存结果，不重复执行。

---

## GET {endpoint}/poll

**功能描述**：拉取任务。上游应 hold 住请求直至有任务可下发或 `wait` 超时；hold 期间 bbx 不再发其他请求，hold 本身即连接保活。

**请求参数**（Query）：
- `agent`：agent 标识（默认 bbx 主机名，可用 `poll.agent_id` 覆盖）
- `wait`：长轮询 hold 时长，Go duration 格式（如 `25s`）；`0s` 表示立即返回（bbx 处于 BUSY 态时使用）

**请求示例**：
```
GET /poll?agent=host-01&wait=25s
Authorization: Basic Ynh4Onh4eA==
```

**响应字段**：
- `tasks`：任务数组，可为空
- `has_more`：为 `true` 时 bbx 立即发起下一次拉取（用于任务洪峰时快速排空）

**响应示例**：
```json
{
  "tasks": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "topic": "docker",
      "timeout_sec": 120,
      "action": {"type": "api", "method": "POST", "path": "/api/process/scan", "body": {"query": "java"}},
      "ack_url": "",
      "created_at": "2026-09-03T10:00:00Z"
    }
  ],
  "has_more": false
}
```

### Task 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 任务唯一标识，重派必须保持同 id |
| topic | string | 是 | 串行调度键：**同 topic 的任务在 bbx 侧严格串行执行，不同 topic 并行**。bbx 不配置、不过滤 topic |
| timeout_sec | int | 否 | 任务级超时秒数，默认 120；长任务（push_file/tail_file/ws_bridge）应显式调大 |
| action | object | 是 | 任务内容，结构按 `action.type` 而定（见下） |
| ack_url | string | 否 | 覆盖默认回执地址（默认 `{来源 endpoint}/ack`） |
| created_at | string | 否 | 创建时间，仅作记录 |

### action.type = `api`（默认）

bbx 向自身 API 发起 loopback 调用（复用 bbx 全部 REST handler、鉴权与审计），响应内嵌进 ack。

| 字段 | 类型 | 说明 |
|------|------|------|
| method | string | HTTP 方法，默认 `GET` |
| path | string | bbx API 路径，须以 `/` 开头，如 `/api/process/scan` |
| body | object | 可选，JSON 请求体 |

```json
{"type": "api", "method": "POST", "path": "/api/v2/exec", "body": {"command": "df -h"}}
```

### action.type = `push_file`

调用方想获取 bbx 本地文件但无法入方向访问 → bbx 把文件流式推送到任务给定的地址。

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | bbx 本地文件路径 |
| push_url | string | 接收地址，bbx 以 raw body 流式推送（非 multipart），`Content-Type: application/octet-stream`，带 Content-Length |
| method | string | 推送方法，默认 `POST` |
| headers | object | 可选，附加请求头（凭据等，bbx 透传） |

```json
{"type": "push_file", "path": "/data/pkg.tar.gz", "push_url": "https://svc-a/upload/abc", "timeout_sec": 600}
```

### action.type = `pull_file`

管控台 → bbx 方向的文件下发：部署包分发、前端文件下发、代理升级包传输等。bbx 主动外拨下载任务给定的地址并落盘；先写同目录临时文件再 rename 就位，失败/中断不会在目标路径留下半截文件。

| 字段 | 类型 | 说明 |
|------|------|------|
| url | string | 下载地址（`http://` / `https://`），bbx 以 `GET` 拉取 |
| path | string | bbx 本地目标路径，父目录不存在时自动创建；已存在同名文件时覆盖 |
| headers | object | 可选，附加请求头（凭据等，bbx 透传） |
| sha256 | string | 可选，期望的文件摘要（hex）；不一致则任务失败且不落盘 |

```json
{"type": "pull_file", "url": "https://svc-a/packages/app-2.1.0.tar.gz", "path": "/data/packages/app-2.1.0.tar.gz", "sha256": "9f2c...", "timeout_sec": 600}
```

典型的升级流程：`pull_file` 拉升级包 → `api` 任务调 `/api/system/upgrade`（topic 相同即串行，保证先下载后升级）。

### action.type = `tail_file`

bbx tail 本地日志，持续向任务给定地址推送增量（chunked POST 长连接）。

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | bbx 本地日志路径 |
| push_url | string | 接收地址，chunked `POST`，`Content-Type: text/plain; charset=utf-8` |
| follow | bool | `true` 持续跟随增量；`false` 推完指定行数即结束 |
| lines | int | 初始推送文件末尾行数，默认 0（从当前末尾开始跟随） |
| headers | object | 可选，附加请求头 |

```json
{"type": "tail_file", "path": "/var/log/app.log", "push_url": "https://svc-a/logs/xyz", "follow": true, "lines": 200, "timeout_sec": 1800}
```

### action.type = `ws_bridge`

bbx 作为 WS client 主动外拨 `connect_url`，同时连接本地 WS handler，双向拷贝帧——用于交互式场景（如 docker exec）。

| 字段 | 类型 | 说明 |
|------|------|------|
| connect_url | string | 上游 WS 地址（`ws://` / `wss://`），可内嵌凭据参数 |
| headers | object | 可选，拨号请求头（如 `Authorization`） |
| local.path | string | 本地 WS handler 路径，如 `/api/docker/containers/{id}/exec/ws` |

```json
{"type": "ws_bridge", "connect_url": "wss://svc-a/ws/xyz", "headers": {"Authorization": "..."}, "local": {"path": "/api/docker/containers/c1/exec/ws"}, "timeout_sec": 3600}
```

注意：本地 handler 的协议约定不变——例如 docker exec WS 要求首条消息为 `{"command":"...","tty":true,...}`，上游连接后需按目标 handler 的协议先发言。

---

## POST {endpoint}/ack

**功能描述**：回执任务结果。任务从哪个上游拉到就回哪个上游，除非任务携带 `ack_url`。

**请求体**：
- `id`：任务 id
- `status`：`success` | `failed` | `timeout`
- `result`：结果对象（仅 success 时携带，按任务类型不同，见下）；超过 64KB 截断并标记 `truncated: true`
- `exit_code`：`0` 成功；`1` 失败；`124` 超时
- `duration_ms`：执行耗时
- `error`：失败/超时原因（成功时为空）

**请求体示例**：
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "status": "success",
  "result": {"status_code": 200, "body": "{\"exit_code\":0,\"stdout\":\"...\"}", "truncated": false},
  "exit_code": 0,
  "duration_ms": 1234
}
```

**响应**：任意 2xx 即视为回执成功。

### 各任务类型的 result

| type | result 字段 |
|------|------------|
| `api` | `status_code`：bbx API 响应码；`body`：响应体字符串；`truncated`：是否被 64KB 截断 |
| `push_file` | `size`：推送字节数；`sha256`：文件摘要（hex） |
| `pull_file` | `path`：落盘路径；`size`：下载字节数；`sha256`：实际摘要（hex，供上游核对） |
| `tail_file` | `lines`：推送行数；`bytes`：推送字节数；`end_reason`：`completed` / `timeout` / `peer_closed` / `read_error` |
| `ws_bridge` | `duration_ms`：桥接时长；`end_reason`：`closed` / `timeout` |

---

## 上游实现要点

1. **hold 时长**：实现 `wait` 参数的服务端 hold；hold 期间有任务立即返回，超时返回 `{"tasks": [], "has_more": false}`。bbx 默认 `wait=25s`，小于常见网关/LB 空闲超时。
2. **has_more**：一次返回多条任务后若队列仍非空，置 `has_more: true`，bbx 会立即再拉（不等 busy_interval）。
3. **超时重派**：发出任务后启动计时，超过 `timeout_sec` 未见 ack（或 agent 长时间不来 poll）即重派同 id 任务；bbx 幂等去重保证不会重复执行。
4. **topic 打标**：把有先后顺序依赖的任务打上同一 topic；相互独立的任务用不同 topic 以获得并行度。bbx 侧 topic 队列上限 32，超出后新 topic 落共享 `default` 队列串行执行。
5. **多上游独立**：bbx 支持配置多个上游，互不故障转移；同一任务不应投递给多个上游。

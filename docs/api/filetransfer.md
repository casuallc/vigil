# 文件传输 Agent API

文件传输 Agent 是 `bbx-server` 的子功能，提供文件系统浏览与分块文件传输任务管理。

> 启用方式：在 `conf/config.yaml` 的 `filetransfer.enabled: true`（详见末尾「配置」）。未启用时不注册以下接口。

## 认证

这些接口复用 **vigil 全局 Basic Auth**（`conf/config.yaml` 的 `auth`，即超管凭据或用户库中的注册用户），与其它 API 一致。它们**不再**使用独立的 Agent 凭据。

```
Authorization: Basic base64(username:password)
```

未携带或凭据错误返回 `401`，响应头带 `WWW-Authenticate: Basic realm="vigil"`。

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/fs/list?path= | GET | 列出目录内容 |
| /api/fs/stat?path= | GET | 获取文件/目录统计信息 |
| /api/transfer/tasks | POST | 创建任务 |
| /api/transfer/tasks | GET | 列出所有任务配置 |
| /api/transfer/tasks/{id} | GET | 获取任务配置 |
| /api/transfer/tasks/{id} | DELETE | 删除任务及本地持久化 |
| /api/transfer/tasks/{id}/start | POST | 启动任务 |
| /api/transfer/tasks/{id}/pause | POST | 暂停任务 |
| /api/transfer/tasks/{id}/resume | POST | 恢复任务 |
| /api/transfer/tasks/{id}/cancel | POST | 取消任务 |
| /api/transfer/tasks/{id}/status | GET | 聚合状态 + 每文件进度 |
| /api/transfer/tasks/{id}/progress | GET | 仅每文件进度（供续传查询） |
| /api/transfer/tasks/{id}/chunks?... | POST | 接收二进制分块（RECV 端） |

> 响应体采用 vigil 原生裸 JSON（非 `{code,msg,data}` 包裹），错误统一为 `{"error": "..."}`。

---

## GET /api/fs/list

**功能描述**：列出指定目录的内容。`path` 必须是目录且落在允许的根目录白名单内（见「路径监狱」）。

**请求参数**（Query）：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| path | string | 是 | 目录绝对路径 |

**响应格式**（`[]FsItem`）：

```json
[
  { "name": "logs", "dir": true,  "size": 0,     "mtime": 1751270400000 },
  { "name": "a.txt", "dir": false, "size": 1024, "mtime": 1751270500000 }
]
```

**响应字段说明**：

| 字段 | 类型 | 描述 |
|------|------|------|
| name | string | 条目名称（不含路径） |
| dir | bool | 是否为目录 |
| size | int64 | 文件大小（字节）；目录为 0 |
| mtime | int64 | 最后修改时间（毫秒时间戳） |

**cURL 示例**：

```bash
curl -u <username>:<password> \
  "http://localhost:57575/api/fs/list?path=/data/logs"
```

---

## GET /api/fs/stat

**功能描述**：获取文件或目录的统计信息。目录会递归汇总文件数与总字节数。

**请求参数**（Query）：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| path | string | 是 | 文件或目录绝对路径 |

**响应格式**：

```json
{
  "isDir": true,
  "size": 0,
  "fileCount": 12,
  "totalSize": 1048576
}
```

**响应字段说明**：

| 字段 | 类型 | 描述 |
|------|------|------|
| isDir | bool | 是否为目录 |
| size | int64 | 文件大小（字节）；目录为 0 |
| fileCount | int64 | 目录下文件总数（递归）；文件时为 0 |
| totalSize | int64 | 目录下文件总字节数（递归）；文件时为 0 |

---

## POST /api/transfer/tasks

**功能描述**：创建一个传输任务并以 `IDLE` 状态持久化。请求体为完整的 `TaskConfig`。

**TaskConfig 字段说明**：

| 字段 | 类型 | 描述 |
|------|------|------|
| taskId | int64 | 任务 ID（调用方分配，唯一） |
| role | string | `SEND` 或 `RECV` |
| relayType | string | `DIRECT` 或 `KAFKA` |
| manifest | FileEntry[] | 文件清单；SEND 端为空时按 sourcePaths 自动构建 |
| sourcePaths | string[] | 源路径（SEND 端，文件或目录） |
| chunkSize | int | 分块大小（字节）；0 表示用默认值 |
| parallelism | int | 并行度；0/1 为串行（默认），>1 时启用文件级 worker 池，KAFKA 模式下同时在单文件内并行发送分块（上限 16） |
| overwritePolicy | string | `OVERWRITE` / `SKIP` / `RENAME` |
| targets | TargetConfig[] | 目标列表（SEND 端 DIRECT） |
| targetDir | string | 落地目录（RECV 端） |
| recvToken | string | 接收令牌（RECV 端，可选） |
| kafka | KafkaConfig | Kafka 配置（relayType=KAFKA 时） |

`TargetConfig`：`host`、`port`、`authUser`、`authPass`、`recvToken`、`agentTaskId`（对端任务 ID）。其中 `authUser`/`authPass` 为**对端 vigil 的全局 Basic Auth 凭据**（SEND 端向对端 RECV 推送分块时使用）。
`KafkaConfig`：`bootstrapServers`、`topic`、`groupId`、`authEnabled`、`saslMechanism`、`securityProtocol`、`username`、`password`、`maxMessageBytes`、`compression`（`none`/`snappy`/`zstd`/`lz4`/`gzip`，默认 `snappy`；broker 透明解压，消费端无需配置）。

> 落盘时 `targets[].authPass` 与 `kafka.password` 以 AES-128-GCM 加密。

**请求示例**（RECV DIRECT）：

```json
{
  "taskId": 1001,
  "role": "RECV",
  "relayType": "DIRECT",
  "targetDir": "/data/incoming",
  "overwritePolicy": "OVERWRITE"
}
```

**响应格式**：

```json
{ "taskId": 1001 }
```

**错误响应**（任务已存在）：HTTP `409`

```json
{ "error": "task already exists: 1001" }
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d @task.json \
  http://localhost:57575/api/transfer/tasks
```

---

## GET /api/transfer/tasks

**功能描述**：列出所有任务配置，返回 `TaskConfig[]`。

---

## GET /api/transfer/tasks/{id}

**功能描述**：获取指定任务的配置，返回 `TaskConfig`。任务不存在返回 `404`。

---

## DELETE /api/transfer/tasks/{id}

**功能描述**：取消任务（若在运行）并删除其本地持久化目录。

**响应格式**：

```json
{ "status": "deleted" }
```

---

## POST /api/transfer/tasks/{id}/start | pause | resume | cancel

**功能描述**：驱动任务生命周期。

| 操作 | 允许的前置状态 | 说明 |
|------|----------------|------|
| start | IDLE、PAUSED | 启动执行 |
| pause | RUNNING | 暂停并落盘进度 |
| resume | PAUSED | 加载进度后继续 |
| cancel | 任意 | 取消执行 |

状态非法（如对 RUNNING 任务执行 start）返回 `409`。

**成功响应**：

```json
{ "status": "ok" }
```

---

## GET /api/transfer/tasks/{id}/status

**功能描述**：返回聚合状态与每文件进度。

**响应格式**（`TaskStatus`）：

```json
{
  "taskId": 1001,
  "state": "RUNNING",
  "progress": 42,
  "totalBytes": 10485760,
  "transferredBytes": 4404019,
  "totalFiles": 3,
  "completedFiles": 1,
  "files": [
    { "relPath": "a.bin", "receivedBytes": 1048576, "totalBytes": 1048576, "completed": true }
  ],
  "errorMsg": "",
  "startedAt": 1753100000000,
  "finishedAt": 0,
  "elapsedMs": 8300,
  "bytesPerSecond": 530604,
  "currentBytesPerSecond": 1048576
}
```

| 字段 | 类型 | 描述 |
|------|------|------|
| state | string | `IDLE`/`RUNNING`/`PAUSED`/`SUCCESS`/`FAILED`/`PARTIAL_FAILED`/`CANCELLED` |
| progress | int | 总体进度百分比（0-100） |
| files | FileProgress[] | 每文件进度 |
| errorMsg | string | 失败原因，正常时为空 |
| startedAt | int64 | 首次进入 RUNNING 的 epoch 毫秒；未启动为 0 |
| finishedAt | int64 | 到达终态（SUCCESS/FAILED/PARTIAL_FAILED/CANCELLED）的 epoch 毫秒；未结束为 0 |
| elapsedMs | int64 | 累计活跃耗时（毫秒），暂停期间冻结、不计入 |
| bytesPerSecond | int64 | 全程平均速率（transferredBytes / elapsedMs） |
| currentBytesPerSecond | int64 | 最近 5 秒滑动窗口速率；SEND 任务为发送速率，RECV 任务为接收速率 |

> 计时持久化在 `{dataDir}/tasks/{taskId}/timing.json`，服务重启后保留。

---

## GET /api/transfer/tasks/{id}/progress

**功能描述**：仅返回每文件进度 `FileProgress[]`，供 DIRECT 发送端查询续传位点。

```json
[
  { "relPath": "a.bin", "receivedBytes": 1048576, "totalBytes": 2097152, "completed": false }
]
```

---

## POST /api/transfer/tasks/{id}/chunks

**功能描述**：RECV 端接收一个二进制分块。请求体为**原始字节流**（`Content-Type: application/octet-stream`），分块元数据通过 Query 传递。EOF 块到达后校验整文件 SHA-256，并按覆盖策略将 `.part` 重命名为正式文件。

**请求参数**（Query）：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| relPath | string | 是 | 相对落地目录的路径（禁止含 `..`） |
| chunkIndex | int | 是 | 分块序号 |
| offset | int64 | 是 | 该块在文件中的起始偏移 |
| length | int | 是 | 该块字节数 |
| crc32 | uint32 | 是 | 该块 CRC32（IEEE） |
| eof | bool | 是 | 是否为最后一块 |
| sha256 | string | 否 | 整文件 SHA-256（EOF 块携带） |
| recvToken | string | 否 | 接收令牌；仅当任务配置了 `recvToken` 时校验 |

**成功响应**：

```json
{ "status": "ok" }
```

**错误响应示例**：

```json
{ "error": "sha256 mismatch for a.bin" }
```

```json
{ "error": "path traversal not allowed: ../escape" }
```

`recvToken` 不匹配返回 `403`。

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/octet-stream" \
  --data-binary @chunk0.bin \
  "http://localhost:57575/api/transfer/tasks/1001/chunks?relPath=a.bin&chunkIndex=0&offset=0&length=1048576&crc32=123456789&eof=false"
```

---

## 路径监狱（Path Jail）

`/api/fs/*` 与 `/chunks` 落地均受路径监狱约束：

- 路径为空或含 `..` 直接拒绝；
- 解析为绝对路径并清理后，必须落在允许的根目录内（`filetransfer.roots`，为空时默认仅用户主目录）；
- `/chunks` 的 `relPath` 必须落在任务的 `targetDir` 内。

## Kafka 中继消息格式

KAFKA 模式下，每个分块作为一条 Kafka 消息：

- key：`parallelism <= 1` 时为 `relPath`（同文件分块有序落在同一 partition）；`parallelism > 1` 时为 nil，producer 用 round-robin 分区器把分块打散到**所有 partition**（Kafka 的并行单元），由 worker 池并发发送。乱序到达由接收端按 offset 落盘、用区间集合跟踪已收字节，收齐整个文件后才校验 SHA-256 并改名落地（EOF 分块先到也能正确收尾）；消费端按 partition 并发 claim，多 partition 可同时消费；
- value 为二进制帧：`[4 字节大端 uint32 头长度][JSON(ChunkMeta)][原始 chunk 字节]`，chunk 不做 base64 编码，`chunkSize` 即为线上消息体大小的主要部分；
- producer：`acks=all`、`retries=3`、压缩按 `kafka.compression`（默认 snappy）；
- 单条消息上限取任务 `kafka.maxMessageBytes`（可选，默认 1,000,000 字节，低于 broker 默认 `message.max.bytes=1,000,012`）；`chunkSize` 超过「上限 − 帧头开销」时自动钳制并打日志。broker/topic 调大过 `max.message.bytes` 时，可相应调大 `kafka.maxMessageBytes`。

## 配置

`conf/config.yaml`：

```yaml
filetransfer:
  enabled: true
  data_dir: ""                          # 空则默认 ~/.vigil-file-transfer（旧版 ~/.admq-file-transfer-agent 自动迁移）
  default_chunk_size: 1048576           # 1MB
  encryption_key: "vigil-file-transfer-change-me"   # 经 SHA-256 派生 AES-256-GCM 密钥
  roots: []                             # 允许浏览/落地的根目录白名单；为空则仅用户主目录
```

> 认证复用全局 `auth`（见「认证」），本节不再有独立的 `auth_user` / `auth_pass`。

本地持久化目录结构：

```
{dataDir}/tasks/{taskId}/
├── config.json     # TaskConfig（敏感字段 AES-GCM 加密）
├── state.json      # 任务状态字符串
├── progress.json   # []FileProgress
└── timing.json     # 计时（startedAt/finishedAt/activeMs）
```

`bbx-server` 重启时扫描该目录恢复任务，`state=RUNNING` 的任务自动续传。

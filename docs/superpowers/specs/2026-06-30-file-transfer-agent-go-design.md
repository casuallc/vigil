# File Transfer Agent — Go 重构设计

> 原始版本：ADMQ Manager v2.0.6 中的 `file-transfer-agent`（Java/Spring Boot）
> 目标：在 vigil 仓库内，作为 `bbx-server` 的一个子功能用 Go 复刻 Agent 侧功能
> 日期：2026-06-30

---

## 0. 决策摘要

| 决策点 | 选择 |
|--------|------|
| 代码归属 | vigil 仓库内的子功能，复用现有 `bbx-server` HTTP 服务 |
| 核心代码位置 | 新增顶层 `filetransfer/` 目录 |
| 传输模式 | 本次实现 **DIRECT + KAFKA**（对齐 Java）；Pulsar/ActiveMQ 后续按同一 SPI 扩展 |
| 互通要求 | Agent REST API 路径/参数、Kafka 消息格式与 Java 对齐；本地持久化走自有方式 |
| 持久化 | 本地 JSON 文件，目录结构对齐 Java（`{dataDir}/tasks/{taskId}/`） |
| API 响应格式 | 沿用 vigil 现有风格（`writeJSON` + HTTP 状态码，非 Java 的 `{code,msg,data}` 包装） |
| CLI | REST API + `bbx-cli transfer` 子命令 |

---

## 1. 架构与模块划分

### 1.1 位置与复用
作为 vigil 的子功能，直接扩展现有 `bbx-server`：

- 核心代码放在新增顶层目录 `filetransfer/`。
- `api.Server` 持有并初始化 `filetransfer.Manager`。
- 在 `api/routes.go` 注册与 Java agent 兼容的 REST 路由，复用 `api.Server` 的 Basic Auth 中间件（`s.authMiddleware`）。
- 复用 vigil 已有依赖：`github.com/IBM/sarama`（Kafka）。Pulsar（`apache/pulsar-client-go`）、ActiveMQ（`go-stomp/stomp/v3`）依赖已存在，本次不使用。
- `bbx-cli` 增加 `transfer` 子命令，通过 `api.Client` 调用 REST。

### 1.2 包结构
```
filetransfer/
  manager.go          # 任务管理器：CRUD、生命周期、执行、状态聚合、启动恢复
  handlers.go         # HTTP handlers（注册到 gorilla/mux）
  fs.go               # 文件系统浏览 + Path Jail
  store.go            # 本地 JSON 持久化 + AES-GCM 加解密
  transport.go        # RelayTransport 接口 + 注册表
  transport_direct.go # DIRECT 发送（HTTP 直推）
  transport_kafka.go  # Kafka producer 发送 / consumer 接收
  models.go           # 数据模型与枚举
api/
  routes.go           # 新增 filetransfer 路由注册
  server.go           # Server 结构增加 *filetransfer.Manager 字段与初始化
cli/
  handlers_filetransfer.go  # bbx-cli transfer ... 子命令
config/
  config.go           # 增加 FiletransferConfig
```

`filetransfer.Manager` 由 `api.Server` 在启动时构造（传入配置），并在 `routes.go` 中把 handlers 挂到 mux。

### 1.3 设计理由
- 复用 `api.Server` 的 HTTP、Basic Auth、配置、日志、优雅关闭，无需单独的 agent 二进制。
- API 路径与 Java Manager 对齐，`/api/transfer/tasks`、`/api/fs/list` 等可被现有 Java Manager 直接编排，也可与对端 Java agent 互通。
- 传输 SPI 化（`RelayTransport` 接口 + 注册表），后续加 Pulsar/ActiveMQ 只需新增实现并注册。

---

## 2. 数据模型与状态机

### 2.1 枚举（字符串值与 Java 对齐）
```go
type Role string
const (
    RoleSend Role = "SEND"
    RoleRecv Role = "RECV"
)

type RelayType string
const (
    RelayDirect RelayType = "DIRECT"
    RelayKafka  RelayType = "KAFKA"
)

type OverwritePolicy string
const (
    Overwrite OverwritePolicy = "OVERWRITE"
    Skip      OverwritePolicy = "SKIP"
    Rename    OverwritePolicy = "RENAME"
)

type TaskState string
const (
    StateIdle          TaskState = "IDLE"
    StateRunning       TaskState = "RUNNING"
    StatePaused        TaskState = "PAUSED"
    StateCancelled     TaskState = "CANCELLED"
    StateSuccess       TaskState = "SUCCESS"
    StateFailed        TaskState = "FAILED"
    StatePartialFailed TaskState = "PARTIAL_FAILED"
)
```

### 2.2 模型（JSON 字段名与 Java 对齐）
```go
type FileEntry struct {
    RelPath string `json:"relPath"`
    Size    int64  `json:"size"`
    Sha256  string `json:"sha256"`
}

type TargetConfig struct {
    Host        string `json:"host"`
    Port        int    `json:"port"`
    AuthUser    string `json:"authUser"`
    AuthPass    string `json:"authPass"`    // 落盘时 AES-GCM 加密
    RecvToken   string `json:"recvToken"`
    AgentTaskID int64  `json:"agentTaskId"`
}

type KafkaConfig struct {
    BootstrapServers string `json:"bootstrapServers"`
    Topic            string `json:"topic"`
    GroupID          string `json:"groupId"`
    AuthEnabled      bool   `json:"authEnabled"`
    SaslMechanism    string `json:"saslMechanism"`
    SecurityProtocol string `json:"securityProtocol"`
    Username         string `json:"username"`
    Password         string `json:"password"`  // 落盘时 AES-GCM 加密
}

type ChunkMeta struct {
    RelPath    string `json:"relPath"`
    ChunkIndex int    `json:"chunkIndex"`
    Offset     int64  `json:"offset"`
    Length     int    `json:"length"`
    Crc32      uint32 `json:"crc32"`
    Eof        bool   `json:"eof"`
    Sha256     string `json:"sha256,omitempty"`
}

type FileProgress struct {
    RelPath       string `json:"relPath"`
    ReceivedBytes int64  `json:"receivedBytes"`
    TotalBytes    int64  `json:"totalBytes"`
    Completed     bool   `json:"completed"`
}

type TaskConfig struct {
    TaskID          int64           `json:"taskId"`
    Role            Role            `json:"role"`
    RelayType       RelayType       `json:"relayType"`
    Manifest        []FileEntry     `json:"manifest"`        // 运行前由 SEND 端生成
    SourcePaths     []string        `json:"sourcePaths"`     // SEND 端
    ChunkSize       int             `json:"chunkSize"`
    OverwritePolicy OverwritePolicy `json:"overwritePolicy"`
    Targets         []TargetConfig  `json:"targets"`         // SEND 端
    TargetDir       string          `json:"targetDir"`       // RECV 端
    RecvToken       string          `json:"recvToken"`       // RECV 端
    Kafka           *KafkaConfig    `json:"kafka"`
}

type TaskStatus struct {
    TaskID           int64          `json:"taskId"`
    State            TaskState      `json:"state"`
    Progress         int            `json:"progress"`        // 0-100
    TotalBytes       int64          `json:"totalBytes"`
    TransferredBytes int64          `json:"transferredBytes"`
    TotalFiles       int            `json:"totalFiles"`
    CompletedFiles   int            `json:"completedFiles"`
    Files            []FileProgress `json:"files"`
    ErrorMsg         string         `json:"errorMsg"`
}
```

> 注意 Crc32：Java 用 `long`（CRC32 是 32 位无符号，Java 放 long 表示）。Go 端用 `uint32` 计算，JSON 序列化为正整数，与 Java 的 long 值一致。

### 2.3 状态机
```
IDLE → RUNNING → SUCCESS
       ↓
   PAUSED → RUNNING
       ↓
   CANCELLED / FAILED / PARTIAL_FAILED
```
- `IDLE`、`PAUSED` 允许 `UpdateTask`。
- `start` / `resume` 启动执行 goroutine。
- `pause` 设置暂停标志并取消 goroutine。
- `cancel` 设置取消标志并取消 goroutine。

### 2.4 运行时
```go
type taskRuntime struct {
    taskID   int64
    config   TaskConfig
    state    TaskState
    errMsg   string
    cancel   context.CancelFunc   // 当前执行 goroutine 的取消句柄
    paused   atomic.Bool
    canceled atomic.Bool
    progress map[string]*FileProgress
    mu       sync.RWMutex
}
```
- `Manager` 持有 `runtimes map[int64]*taskRuntime` + `sync.RWMutex`。
- 每个执行任务一个 goroutine；`context.Context` 控制生命周期。

---

## 3. REST API 与认证

### 3.1 路由（前缀 `/api`，与 Java agent 对齐）

| 方法 | URL | 说明 |
|------|-----|------|
| `GET` | `/api/fs/list?path=` | 列目录，返回 `[]FsItem` |
| `GET` | `/api/fs/stat?path=` | 文件/目录信息 |
| `POST` | `/api/transfer/tasks` | 创建任务，body=TaskConfig |
| `GET` | `/api/transfer/tasks` | 列所有任务配置 |
| `GET` | `/api/transfer/tasks/{id}` | 获取任务配置 |
| `DELETE`| `/api/transfer/tasks/{id}` | 删除任务及本地持久化 |
| `POST` | `/api/transfer/tasks/{id}/start` | 启动 |
| `POST` | `/api/transfer/tasks/{id}/pause` | 暂停 |
| `POST` | `/api/transfer/tasks/{id}/resume` | 恢复 |
| `POST` | `/api/transfer/tasks/{id}/cancel` | 取消 |
| `GET` | `/api/transfer/tasks/{id}/status` | 聚合状态 + 每文件进度 |
| `GET` | `/api/transfer/tasks/{id}/progress` | 仅每文件进度（供源端续传查询） |
| `POST` | `/api/transfer/tasks/{id}/chunks?relPath=&chunkIndex=&offset=&length=&crc32=&eof=&sha256=` | 接收二进制分块 |

### 3.2 响应格式
- 沿用 vigil 现有风格：`writeJSON(w, statusCode, data)` 直接返回数据对象，错误用 `writeError(w, statusCode, msg)`（`{"error": ...}`）+ 对应 HTTP 状态码。
- **不**使用 Java 的 `{code,msg,data}` 包装。
- 这意味着 DIRECT 发送端（`transport_direct.go`）查询对端 progress 时，按 vigil 的响应结构解析（直接 `[]FileProgress`，而非 `Result<List>`）。Go agent ↔ Go agent 自洽；与 Java agent 互通时，progress 响应结构差异需由 Manager/调用方知悉（本设计两端均为 Go agent）。

### 3.3 认证
- 复用 vigil 现有 Basic Auth 中间件。
- 配置见 §5.1；密码内存明文校验（与 Java 一致，不走 bcrypt）。
- DIRECT 接收 `/chunks` 时，除 Basic Auth 外，校验 `recvToken`（与任务配置一致）。

### 3.4 FS 浏览与 Path Jail
- `FsItem`：`name`、`path`、`isDir`、`size`、`modTime`。
- 白名单 `roots`：为空时默认只允许用户主目录（`os.UserHomeDir()`）。
- Path Jail：`filepath.Abs` + `filepath.Clean`，拒绝含 `..` 的路径，解析后必须落在白名单根目录内（`strings.HasPrefix` 比较，注意补足分隔符避免前缀误判）。

---

## 4. 传输流程

### 4.1 SEND 端共同逻辑（`manager.go`）
1. 若 `Manifest` 为空，按 `SourcePaths` 构建清单：
   - 文件 → 直接加入；目录 → 递归遍历，相对路径统一用 `/`。
   - 每个文件计算整文件 SHA-256 和 size。
2. 初始化每文件 `FileProgress`。
3. 选择 transport（`RelayDirect` / `RelayKafka`）。
4. 对每个文件、每个目标：
   - DIRECT：先 `GET /progress` 取 `resumeOffset`。
   - 从 `resumeOffset` 起按 `chunkSize` 分块，每块算 CRC32。
   - 最后一块 `eof=true` 并带整文件 `sha256`。
5. 检查 `paused`/`canceled` 标志，及时落 `state.json` 并退出。
6. 全部成功 → `SUCCESS`；部分目标失败 → `PARTIAL_FAILED`。

### 4.2 DIRECT
**SEND（`transport_direct.go`）**
- `http.Client`，连接超时 30s。
- `GET {base}/api/transfer/tasks/{agentTaskId}/progress` → resumeOffset。
- `POST {base}/api/transfer/tasks/{agentTaskId}/chunks?relPath=...&offset=...&crc32=...&eof=...&sha256=...`，body 为原始二进制 chunk，`Content-Type: application/octet-stream`。
- Basic Auth 头 = `Basic base64(authUser:authPass)`。

**RECV（`handlers.go` 的 `/chunks`）**
- 复用 `Manager.receiveChunk(taskID, meta, body)`：
  - Path Jail 后写 `{targetDir}/{relPath}.part`。
  - `os.OpenFile` + `Seek(offset)` 写入，乱序/重发可覆盖。
  - 更新 `FileProgress.ReceivedBytes`。
  - EOF：校验整文件 SHA-256，通过后按覆盖策略重命名为正式文件。
  - 持久化 `progress.json`。

### 4.3 KAFKA
**SEND（`transport_kafka.go`）**
- `sarama` SyncProducer：`acks=all`、`retries=3`、`MaxMessageBytes=10MB`、`Producer.Return.Successes=true`。
- key = `relPath`（保证同文件有序），value = `JSON(ChunkMeta) + "\n" + base64(chunkData)`（与 Java 字节级一致）。
- SASL/PLAIN：从 `KafkaConfig` 设置 `Config.Net.SASL`（Mechanism、User、Password、Handshake），`securityProtocol` 决定是否 TLS。

**RECV（`manager.go` → `transport_kafka.go` consumer）**
- `executeRecvKafka`：启动 `sarama` consumer group（`groupId` 每个目标唯一），`auto.offset.reset=earliest`。
- poll/Consume 循环，遇 `paused`/`canceled` 退出；处理每条消息：
  - 拆分 `header\n base64body` → `ChunkMeta` + chunkData。
  - 调 `receiveChunk` 复用写入/校验逻辑。

### 4.4 覆盖策略（`applyOverwritePolicy`）
| 策略 | 行为 |
|------|------|
| `OVERWRITE` | `.part` 重命名覆盖目标 |
| `SKIP` | 目标存在则删 `.part`，否则重命名 |
| `RENAME` | 目标存在则自动 `name_1.ext`、`name_2.ext` |

---

## 5. 持久化与配置

### 5.1 配置（`conf/config.yaml`）
```yaml
filetransfer:
  enabled: true
  auth_user: admq
  auth_pass: admq-file-transfer
  data_dir: ""                          # 空则默认 ~/.admq-file-transfer-agent
  default_chunk_size: 1048576           # 1MB
  encryption_key: "admq-file-transfer-agent-key-16"
  roots: []                             # 允许浏览/落地的根目录白名单
```
对应 `config.FiletransferConfig` 结构，挂到 `config.Config`。

### 5.2 本地目录结构（对齐 Java）
```
{dataDir}/tasks/{taskId}/
├── config.json     # TaskConfig（敏感字段 AES-GCM 加密）
├── state.json      # TaskState 字符串（JSON 字符串）
└── progress.json   # []FileProgress
```

### 5.3 持久化行为（`store.go`）
- 创建任务：写 `config.json` + `state.json="IDLE"`。
- 状态变更：即时写 `state.json`。
- 进度：文件完成或状态变更时写 `progress.json`。
- 启动恢复：扫描 `{dataDir}/tasks/` 下所有数字目录，加载 `config.json`/`state.json`；`state=RUNNING` 自动 `resume`（先加载 `progress.json` 到内存）。
- `deleteTask`：删除整个 `{taskId}/` 目录。

### 5.4 加密
- 字段：`TargetConfig.AuthPass`、`KafkaConfig.Password`。
- 算法：AES-128-GCM，IV 12 字节，Tag 128 位。
- key 派生：配置 `encryption_key` 前 16 字节 UTF-8（与 Java 一致）。
- 密文格式：`Base64(iv || ciphertext)`。
- 加/解密失败时降级为明文（与 Java 行为一致）。

---

## 6. CLI 命令

`bbx-cli transfer` 子命令，通过 `api.Client` 调用 REST，走 Basic Auth：

```bash
# 文件系统
bbx-cli transfer fs list --path /data
bbx-cli transfer fs stat --path /data/file.txt

# 任务 CRUD
bbx-cli transfer task create --file task.json   # task.json 为完整 TaskConfig
bbx-cli transfer task list
bbx-cli transfer task get --id 1
bbx-cli transfer task delete --id 1

# 生命周期
bbx-cli transfer task start  --id 1
bbx-cli transfer task pause  --id 1
bbx-cli transfer task resume --id 1
bbx-cli transfer task cancel --id 1

# 状态
bbx-cli transfer task status   --id 1
bbx-cli transfer task progress --id 1
```
- `--id` 对应 `taskId`。
- 输出沿用 vigil CLI 现有风格（表格/JSON）。

---

## 7. 实现注意点

1. **Path Jail**：`filepath.Abs` + `filepath.Clean` + 边界比较；拒绝 `..`；注意 Windows 盘符与分隔符。
2. **并发**：每任务一 goroutine + `context.Context`；`runtimes` 用 `sync.RWMutex`；`paused`/`canceled` 用 `atomic.Bool`。
3. **续传**：DIRECT 以对端返回 `receivedBytes` 为 offset；写 `.part` 用 `os.OpenFile(O_RDWR|O_CREATE)` + `Seek`。
4. **Kafka 消息格式**：与 Java 保持字节级一致（`header JSON + "\n" + base64 chunk`，key=relPath）。
5. **CRC32**：用 `hash/crc32`（IEEE 多项式，与 Java `java.util.zip.CRC32` 一致）。
6. **SHA-256**：`crypto/sha256`，整文件流式计算，输出小写 hex。
7. **AES-GCM**：key 取前 16 字节；解密失败回退明文。
8. **优雅关闭**：`bbx-server` 退出时不强行改任务状态（保持 RUNNING，下次启动自动恢复）；正在执行的 goroutine 通过 context 取消。
9. **transport 注册表**：`map[RelayType]RelayTransport`，Manager 按 `relayType` 取实现，避免模式判断散落。
10. **消息大小**：创建任务时可选校验 `chunkSize` ≤ Kafka `max.message.bytes`。

---

## 8. 文件清单（预计新增/改动）

新增：
- `filetransfer/models.go`
- `filetransfer/store.go`
- `filetransfer/fs.go`
- `filetransfer/transport.go`
- `filetransfer/transport_direct.go`
- `filetransfer/transport_kafka.go`
- `filetransfer/manager.go`
- `filetransfer/handlers.go`
- `cli/handlers_filetransfer.go`

改动：
- `config/config.go`：增加 `FiletransferConfig`
- `api/server.go`：`Server` 增加 `*filetransfer.Manager` 字段与初始化、启动恢复
- `api/routes.go`：注册 filetransfer 路由
- `cli/`：注册 `transfer` 命令（`NewCLI` 处）
- `conf/config.yaml`：增加 `filetransfer` 段

---

## 9. 后续扩展（本次不实现）

- PULSAR transport（`apache/pulsar-client-go`）：message key=relPath，独立 subscription 扇出，消息体与 Kafka 兼容。
- ACTIVEMQ transport（`go-stomp/stomp/v3`）：每目标独立 Queue 或 VirtualTopic，BytesMessage + 属性承载 ChunkMeta。
- 二者按 `RelayTransport` 接口实现并注册到 transport 注册表即可接入。

---

## 10. 参考

- 设计总结：`D:\apusic\admq\admq-manager\tmp\file-transfer-agent-go-design-summary.md`
- Java 源码：`D:\apusic\admq\admq-manager\file-transfer-agent\src\main\java\com\admq\manager\filetransfer\agent\`

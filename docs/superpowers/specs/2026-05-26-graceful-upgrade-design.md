# Vigil 平滑升级设计

## 背景

Vigil (bbx-server) 作为代理部署在各个服务器上，控制端通过 API 调用进行工作。当前每次 bbx 有 bug 或需要升级时，需要 SSH 到每台服务器手动操作，非常繁琐。

本设计目标：bbx-server 具备类似 nginx 的自升级能力——通过自身 API 接收新二进制并平滑替换运行中的进程，无需 SSH 登录。

## 设计决策

- **方案选择**：端口竞争 + 状态文件（跨平台兼容，改动最小）
- **升级分发**：同时支持 API 上传和远程拉取
- **连接处理**：强制接管模式（旧进程关闭监听，新进程绑定端口，旧进程保持现有连接直到完成）
- **失败处理**：新进程健康自检通过后旧进程才退出（双重保险）

## 核心架构改进（相比原方案）

### 1. Windows 文件锁定解决

Windows 会锁定正在运行的可执行文件，无法直接覆盖或执行同名文件。解决方案：

```
[旧进程 bbx-server.exe] --写入--> data/bbx-server.new
                          └──exec──> data/bbx-server.new (新进程)
                                           │
                                           ▼
                          接管完成后，旧进程退出，文件锁释放
                                           │
                                           ▼
                          [新进程] os.Rename(data/bbx-server.new, bbx-server.exe)
```

- 旧进程运行的是 `bbx-server.exe`（被系统锁定）
- 新进程从 `data/bbx-server.new` 启动（不受锁）
- 旧进程退出后，新进程将自己重命名回标准路径
- Linux 上可直接覆盖，此步骤为 no-op

### 2. 状态文件原子写入 + 文件锁

状态文件是新旧进程协调的唯一媒介，并发读写可能损坏文件：

```go
// 写状态：先写临时文件，再原子 rename
tmpFile := statePath + ".tmp"
writeJSON(tmpFile, state)
os.Rename(tmpFile, statePath)  // 原子操作，不会半写

// 读状态：配合跨平台文件锁
lock(stateFile, shared)    // 共享锁读
lock(stateFile, exclusive) // 独占锁写
```

### 3. SQLite WAL 模式

新进程启动时检测并强制启用 WAL，避免新旧进程访问同一数据库时锁定冲突：

```go
db.Exec("PRAGMA journal_mode=WAL;")
```

WAL 模式允许多个进程并发读取，写操作不会阻塞读操作。

### 4. Draining 硬上限

旧进程收到 `completed` 后：
- 立即关闭所有 WebSocket 连接（发送 close frame）
- 等待普通 HTTP 连接完成
- **30 秒硬上限**，超时后强制 `server.Close()` 关闭剩余连接

### 5. 去掉 `rolling_back` 状态

升级失败时旧进程自动继续运行，这就是回滚。不需要额外状态机状态。

## 平滑升级流程

```
[控制端] --POST /api/system/upgrade--> [旧进程 :57575]
                                            |
                                            v
                               1. 权限检查（仅 super admin）
                               2. 检查 upgrade.state != "idle"
                                  是 → 返回 409 Conflict
                               3. 标记 state = "downloading"
                               4. 接收新二进制（API上传或远程下载）
                               5. 写入临时路径：data/bbx-server.new
                               6. 标记 state = "verifying"
                               7. 校验（可执行权限、sha256）
                               8. 标记 state = "starting"
                               8. fork/exec 新进程
                                            |
                               环境变量：BBX_UPGRADE=1
                                         BBX_OLD_PID=<old_pid>
                                         BBX_CONFIG_PATH=<path>
                                         BBX_INTERNAL_ADDR=:57576
                                         BBX_NEW_BINARY_PATH=data/bbx-server.new
                                            |
                                            v
                               [新进程 :57576] 启动
                               +- 检测到 BBX_UPGRADE=1，进入升级模式
                               +- 加载配置
                               +- 强制 SQLite PRAGMA journal_mode=WAL
                               +- 初始化所有组件（DB、VM、调度器等）
                               +- 启动内部 HTTP 服务（:57576）
                               +- 标记 state = "health_checking"
                               +- 自检：GET /health -> 200 OK（5s 超时）
                                            |
                               自检通过后（原子写入 + 独占锁）：
                               upgrade_state = "ready"
                               new_pid = <new_pid>
                                            |
                                            v
                               [旧进程] 检测到状态文件变更
                               +- 停止接受新连接（关闭 Listener）
                               +- 关闭所有 WebSocket 连接（发送 close frame）
                               +- 标记为 draining，等待现有连接完成
                               +- 30s 硬上限
                                            |
                                            v
                               [新进程] 绑定 :57575（重试10次，指数退避）
                               成功 → 写入状态文件：upgrade_state = "completed"
                                            |
                                            v
                               [旧进程] 检测到 completed
                               +- 等待最多 30s（已计时）
                               +- 关闭所有资源（Stop()）
                               +- 退出（exit 0）
                                            |
                                            v
                               [新进程] 检测到旧进程 PID 已退出
                               +- 将 data/bbx-server.new 移动到标准路径：
                                  Linux: os.Rename(data/bbx-server.new, 标准路径)
                                  Windows:
                                    1. os.Remove(标准路径)  // 旧进程已退出，文件锁释放
                                    2. os.Rename(data/bbx-server.new, 标准路径)
                                    3. 若仍失败，调用 MoveFileEx + MOVEFILE_DELAY_UNTIL_REBOOT
                                       在下次重启时完成替换
                               +- 清理状态文件中的 upgrade 字段（恢复 idle）
                               +- 正常运行
```

## 状态文件

**`data/bbx.state`**（JSON，原子写入，文件锁保护）：

```json
{
  "pid": 1234,
  "version": "1.2.3",
  "started_at": "2026-05-26T10:00:00Z",
  "upgrade": {
    "state": "idle",
    "new_pid": 0,
    "started_at": "",
    "completed_at": "",
    "error": "",
    "binary_path": ""
  }
}
```

**`upgrade.state` 状态机：**

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

## API 设计

### 系统管理

```
POST /api/system/upgrade          // 触发升级（仅 super admin）
GET  /api/system/status           // 获取运行状态
GET  /api/system/upgrade/status   // 获取升级进度（轮询用）
```

**`POST /api/system/upgrade` 权限控制：**

```go
func (s *Server) canUpgrade(r *http.Request) bool {
    if !s.config.BasicAuth.Enabled {
        return true  // 未启用认证时允许（内部网络场景）
    }
    username, _, _ := r.BasicAuth()
    return username == s.config.BasicAuth.Username  // 仅 super admin
}
```

**请求体（方式1：API 上传）：**

```json
{
  "method": "upload",
  "file": <binary_file>
}
```

**请求体（方式2：远程下载）：**

```json
{
  "method": "download",
  "url": "https://artifacts.example.com/bbx-server/v1.3.0/linux-amd64/bbx-server",
  "checksum": "sha256:abc123..."
}
```

**`GET /api/system/status` 响应：**

```json
{
  "pid": 1234,
  "version": "1.2.3",
  "uptime_seconds": 86400,
  "started_at": "2026-05-26T10:00:00Z",
  "upgrade_state": "idle",
  "listeners": [":57575"],
  "executable_path": "/usr/local/bin/bbx-server"
}
```

## 错误处理

| 场景 | 处理 |
|------|------|
| 升级权限不足 | 返回 403 Forbidden |
| 已有升级进行中 | 返回 409 Conflict，`error: "upgrade already in progress: <state>"` |
| 新二进制校验失败 | 拒绝升级，清理 `data/bbx-server.new`，返回 400 |
| 新进程启动失败（exit 非0） | 状态文件标记 `failed`，旧进程继续，清理临时文件 |
| 新进程健康自检失败（5s 超时） | 同上，新进程自动退出 |
| 端口绑定冲突（旧未释放） | 新进程重试 10 次，指数退避（1s, 2s, 4s...），仍失败则标记 `failed` |
| 升级超时（60s） | 旧进程强制终止新进程，状态恢复 `idle`，清理临时文件 |
| 旧进程 draining 时崩溃 | 新进程检测到旧进程 PID 不存在，直接绑定端口继续服务 |

## 新增文件布局

```
├── cmd/bbx-server/main.go          // 增加 upgrade.IsUpgradeMode() 检测入口
├── api/
│   ├── server.go                   // 增加 system API handler 注册
│   ├── routes.go                   // 增加 system 路由
│   ├── handlers_system.go          // 新增：system 相关 API handler
│   └── middleware.go               // 新增/复用：upgrade 权限检查中间件
├── upgrade/
│   ├── upgrade.go                  // 核心：升级流程控制、fork/exec、超时处理
│   ├── health.go                   // 自检逻辑（新进程自检）
│   ├── state.go                    // 状态文件读写（原子写入 + 文件锁）
│   ├── binary.go                   // 二进制校验/下载/替换（含 Windows 特殊处理）
│   └── filelock.go                 // 跨平台文件锁（flock / LockFile）
```

## 最小改动原则

- `cmd/bbx-server/main.go` 只在开头加 `upgrade.IsUpgradeMode()` 检测
- `api/server.go` 的 `Stop()` 方法已有关闭资源逻辑，升级时复用
- `api/server.go` 增加 `canUpgrade()` 权限检查和 system handler 注册
- 现有所有业务 handler 不改动
- 配置结构不改动，升级参数通过环境变量传递

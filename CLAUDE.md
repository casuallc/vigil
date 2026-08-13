# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
# Build all components (cross-platform: linux/amd64, linux/arm64)
./build_all.sh

# Build specific commands
./build_all.sh server cli

# Build CLI only
go build -o bbx-cli ./cmd/bbx-cli

# Build server only
go build -o bbx-server ./cmd/bbx-server

# Build tar.gz + RPM in WSL (recommended on Windows)
# This auto-installs Go and nfpm inside WSL if missing
.\build-wsl.ps1
.\build-wsl.ps1 -Version 2.1.0

# loongarch64 旧世界构建（麒麟 V10 / Loongnix 20 / UOS V20，ABI1.0）
# 上游 Go 编出的 loong64 二进制是「新世界」，在旧世界内核（4.19/5.4/5.10）上启动即段错误。
# build_all.sh 检测到龙芯 abi1.0 工具链（默认 ~/toolchains/go-abi1.0/bin/go，
# 可用 LOONGSON_GO 环境变量覆盖）时会额外产出 linux-loong64-abi1 构建，
# package.sh / push.sh 会生成对应的 *-linux-loong64-abi1.tar.gz / *.loongarch64-abi1.rpm / *.loong64-abi1.deb。
# 工具链下载: http://ftp.loongnix.cn/toolchain/golang/go-1.25/abi1.0/

# Run tests
go test ./...

# Run specific package tests
go test ./inspection/...
go test ./proc/...

# Run specific test
go test ./inspection/... -run TestInspectionRules

# Run the server
./bbx-server
./bbx-server -config conf/config.yaml
./bbx-server -addr :8080

# Run the CLI (example commands)
./bbx-cli proc scan -q "java" -H http://127.0.0.1:57575
./bbx-cli proc list -H http://127.0.0.1:57575
```

## Git 提交规范

- **每次修复问题或新增功能后都必须提交（commit）**。完成一处 bug 修复或一个功能点后，应立即创建一个聚焦、独立的提交，不要把多个不相关的改动混在一起。
- 提交信息使用约定式提交格式（Conventional Commits），例如 `fix(filetransfer): ...`、`feat(network): ...`、`docs(...): ...`，与现有历史保持一致。

## Project Structure

Vigil is a Go-based process management and message queue client tool (module: `github.com/casuallc/vigil`).

```
├── api/              # HTTP API server and client implementations
├── cli/              # Cobra-based CLI commands (entry: NewCLI)
├── client/           # Message queue clients (kafka, mqtt, pulsar, rabbitmq, redis, rocketmq, zookeeper)
├── cmd/              # Application entry points (bbx-cli, bbx-server)
├── common/           # Shared utilities (exec.go, utils.go)
├── config/           # Configuration loading (config.yaml, scan.yaml)
├── crypto/           # Encryption utilities
├── docker/           # Docker container, compose, and registry management
├── docs/             # Documentation (CLI references, testing guides, API docs)
├── inspection/       # Cosmic inspection rules and evaluation engine
├── proc/             # Process management (scan, create, lifecycle, mounts)
├── vm/               # VM management (SSH, file transfer, groups, permissions)
├── audit/            # Audit logging for API requests
├── models/           # Data models (users, processes, login logs)
├── sql/              # SQLite database helpers
└── version/          # Version info injected at build time
```

## Architecture

- **Server (`bbx-server`)**: REST API server using `gorilla/mux`, supports HTTP/HTTPS. The `api.Server` struct centralizes all domain managers (proc, vm, scheduler, audit, user DB).
- **CLI (`bbx-cli`)**: Uses `spf13/cobra`, communicates with the server via `api.Client` or makes direct client connections. Entry point is `cli.NewCLI(apiHost)`.
- **Message Queue Clients**: Unified interface pattern across 7 MQ systems (Redis, RabbitMQ, RocketMQ, Kafka, MQTT, Pulsar, Zookeeper). Each client implements `Connect()`, `Disconnect()`, and message counting (produced/consumed totals printed on exit).
- **Process Manager**: `proc.Manager` implements `ProcessScanner`, `ProcessLifecycle`, `ProcessInfo`, `ProcessConfig`, and `ProcessMonitor`. Processes are identified by `(namespace, name)` and persisted via `proc.ProcessStore` (SQLite).
- **VM Manager**: `vm.Manager` uses SQLite for persistence and supports simulated SSH (WebSocket), SFTP file operations, command execution with allowlists, batch ops, and group/permission management.
- **Scheduler**: `api.Scheduler` runs as a background goroutine in the server, checking `scheduleDB` every minute to execute scheduled VM commands.
- **AI Assistant**: Server exposes `/api/ai/*` endpoints for generating, explaining, and fixing commands.
- **User/Auth System**: SQLite-backed user database (`models.SQLiteUserDatabase`) with basic auth, login logs, and per-user configs.
- **Audit System**: `audit.Logger` records all API requests including operation type, timestamp, user, IP, and status.
- **Persistence**: Server uses `modernc.org/sqlite` for multiple features (users, login logs, command templates/history, schedules, VM data).
- **Docker Manager**: `docker.Manager` wraps the official Docker SDK to manage local containers (list, inspect, create, start/stop/restart, pause/unpause, exec, logs, stats) and compose projects. Exposes `/api/docker/*` REST and WebSocket endpoints.
- **Docker Registry**: Embedded Docker Registry HTTP API V2 implementation (`api/handlers_docker_registry.go`) backed by local filesystem storage, enabling `docker login / tag / push / pull` against the server.

## Key Configuration Files

- `conf/config.yaml`: Server configuration (addr, auth, log level, HTTPS, encryption key, database)
- `conf/scan.yaml`: Batch process scan configuration
- `conf/cosmic/`: Cosmic inspection rules (YAML-based)
- `conf/config.yaml` Docker sections: `docker` (container/compose enablement) and `docker_registry` (embedded registry enablement and storage path)

## Testing Patterns

- Tests use the standard `testing` package
- Integration tests in `tests/` directory (e.g., `tests/cosmic_test.go`)
- Inspection rules tests validate YAML configs and expression evaluation using `expr-lang/expr`
- Resource monitor tests are in `proc/resource_monitor_test.go`

## Common Patterns

- All MQ clients implement message counting (produced/consumed totals printed on exit)
- CLI handlers are in `cli/handlers_*.go` — main command logic
- CLI protocol-specific clients are in `cli/client_*.go`
- Process registration uses `models.ManagedProcess` with Metadata/Spec/Status
- Cosmic inspection uses script-based checks with expression evaluation for thresholds (`inspection.ExecuteCheck`)
- Version info is injected at build time via ldflags into the `version` package (see `build_all.sh`)
- API routes are registered centrally in `api/routes.go`
- **File upload strategy**: Files ≤100MB use multipart `/api/files/upload` (or `/api/vms/files/{name}/upload` for VMs). Files >100MB use raw-body stream endpoints `/api/files/stream` (or `/api/vms/files/{name}/stream`). Client auto-switches based on file size. Server stream handlers use `io.Copy(r.Body, dstFile)` directly — do NOT call `ParseMultipartForm` for stream endpoints.

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

## Key Configuration Files

- `conf/config.yaml`: Server configuration (addr, auth, log level, HTTPS, encryption key, database)
- `conf/scan.yaml`: Batch process scan configuration
- `conf/cosmic/`: Cosmic inspection rules (YAML-based)

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

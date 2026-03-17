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

# Run tests
go test ./...

# Run specific test
go test ./inspection/... -run TestInspectionRules
```

## Project Structure

Vigil is a Go-based process management and message queue client tool.

```
├── api/              # HTTP API server and client implementations
├── cli/              # Cobra-based CLI commands (entry: NewCLI)
├── client/           # Message queue clients (kafka, mqtt, pulsar, rabbitmq, redis, rocketmq, zookeeper)
├── cmd/              # Application entry points (bbx-cli, bbx-server)
├── common/           # Shared utilities (exec.go, utils.go)
├── config/           # Configuration loading (config.yaml, scan.yaml)
├── crypto/           # Encryption utilities
├── docs/             # Documentation (CLI references, testing guides)
├── inspection/       # Cosmic inspection rules and evaluation engine
├── proc/             # Process management (scan, create, lifecycle, mounts)
├── vm/               # VM management (SSH, file transfer, groups, permissions)
├── audit/            # Audit logging for API requests
└── version/          # Version info injected at build time
```

## Architecture

- **Server (`bbx-server`)**: REST API server using `gorilla/mux`, supports HTTP/HTTPS
- **CLI (`bbx-cli`)**: Uses `spf13/cobra`, communicates with server API or direct client connections
- **Message Queue Clients**: Unified interface pattern across 7 MQ systems (Redis, RabbitMQ, RocketMQ, Kafka, MQTT, Pulsar, Zookeeper)
- **Process Manager**: Manages system processes with support for scanning, lifecycle control, and mount management
- **VM Manager**: Simulated SSH service with file transfer (SFTP), command execution, and batch operations

## Key Configuration Files

- `conf/config.yaml`: Server configuration (port, auth, HTTPS, encryption key)
- `conf/scan.yaml`: Batch process scan configuration
- `conf/cosmic/`: Cosmic inspection rules (YAML-based)

## Testing Patterns

- Tests use standard `testing` package
- Integration tests in `tests/` directory
- Inspection rules tests validate YAML configs and expression evaluation (using `expr-lang/expr`)

## Common Patterns

- All MQ clients implement message counting (produced/consumed totals printed on exit)
- CLI handlers in `cli/handlers.go` - main command logic
- CLI clients in `cli/client_*.go` - protocol-specific implementations
- Process registration uses `proc.ManagedProcess` struct with Metadata/Spec/Status
- Cosmic inspection uses script-based checks with expression evaluation for thresholds

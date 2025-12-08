# Vigil CLI 命令汇总

本文件汇总了 Vigil CLI 的所有命令，包括命令格式、参数说明和示例。

## 命令结构

Vigil CLI 命令采用层级结构，主要分为以下几类：

1. **进程管理命令** - 用于管理和监控系统进程
2. **资源管理命令** - 用于查看系统和进程资源
3. **配置管理命令** - 用于查看和管理系统配置
4. **命令执行命令** - 用于在服务器上执行命令或脚本
5. **消息队列命令** - 用于与各种消息队列系统交互
6. **集成测试命令** - 用于测试各种服务的功能
7. **其他命令** - 其他辅助功能

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--host` | `-H` | API 服务器地址 | http://localhost:8181 |
| `--help` | `-h` | 显示命令帮助信息 | - |
| `--version` | - | 显示版本信息 | - |

## 进程管理命令

### 命令格式

```
bbx-cli proc [subcommand] [flags]
```

### 子命令列表

| 命令 | 说明 | 详细文档 |
|------|------|----------|
| `scan` | 扫描系统进程 | [cli_proc.md](./cli_proc.md#scan---扫描进程) |
| `create` | 创建新的托管进程 | [cli_proc.md](./cli_proc.md#create---创建进程) |
| `start` | 启动托管进程 | [cli_proc.md](./cli_proc.md#start---启动进程) |
| `stop` | 停止托管进程 | [cli_proc.md](./cli_proc.md#stop---停止进程) |
| `restart` | 重启托管进程 | [cli_proc.md](./cli_proc.md#restart---重启进程) |
| `delete` | 删除托管进程 | [cli_proc.md](./cli_proc.md#delete---删除进程) |
| `list` | 列出所有托管进程 | [cli_proc.md](./cli_proc.md#list---列出进程) |
| `status` | 检查进程状态 | [cli_proc.md](./cli_proc.md#status---检查进程状态) |
| `edit` | 编辑进程定义 | [cli_proc.md](./cli_proc.md#edit---编辑进程定义) |
| `get` | 获取进程详情 | [cli_proc.md](./cli_proc.md#get---获取进程详情) |
| `mount` | 管理进程挂载点 | [cli_proc.md](./cli_proc.md#挂载管理命令) |

## 资源管理命令

### 命令格式

```
bbx-cli resources [subcommand] [flags]
```

### 子命令列表

| 命令 | 说明 | 详细文档                                                   |
|------|------|--------------------------------------------------------|
| `system` | 获取系统资源使用信息 | [cli_resources.md](./cli_resources.md#system---获取系统资源) |
| `process` | 获取特定进程的资源使用信息 | [cli_resources.md](./cli_resources.md#process---获取进程资源)  |

## 配置管理命令

### 命令格式

```
bbx-cli config [subcommand] [flags]
```

### 子命令列表

| 命令 | 说明 | 详细文档 |
|------|------|----------|
| `get` | 获取当前配置 | [cli_config.md](./cli_config.md#get---获取配置) |

## 命令执行命令

### 命令格式

```
bbx-cli exec [command/script] [flags]
```

### 详细文档

[cli_exec.md](./cli_exec.md)

## 消息队列命令

Vigil 支持多种消息队列系统，提供统一的 API 接口。

### 支持的消息队列

| 消息队列 | 命令前缀 | 功能 | 详细文档 |
|---------|---------|------|----------|
| Redis | `redis` | 支持发布/订阅消息 | [cli_redis.md](./cli_redis.md) |
| RabbitMQ | `rabbit` | 支持生产/消费消息 | [cli_rabbit.md](./cli_rabbit.md) |
| RocketMQ | `rocket` | 支持生产/消费消息 | [cli_rocket.md](./cli_rocket.md) |
| Kafka | `kafka` | 支持生产/消费消息 | [cli_kafka.md](./cli_kafka.md) |
| MQTT | `mqtt` | 支持发布/订阅消息 | [cli_mqtt.md](./cli_mqtt.md) |
| Pulsar | `pulsar` | 支持生产/消费消息 | [cli_pulsar.md](./cli_pulsar.md) |
| Zookeeper | `zk` | 支持基本操作 | [cli_zk.md](./cli_zk.md) |

### 通用功能

所有消息队列客户端都实现了消息计数功能，在退出时会打印生产和消费的消息总数。

## 集成测试命令

### 命令格式

```
bbx-cli test [service] [subcommand] [flags]
```

### 支持的测试服务

| 服务 | 命令 | 详细文档 |
|------|------|----------|
| MQTT | `test mqtt` | [cli_test_mqtt.md](./cli_test_mqtt.md) |

## 其他命令

### cosmic 命令

用于管理 Cosmic 相关功能。

**用法：**
```
bbx-cli cosmic [subcommand]
```

### 详细文档

[cli_cosmic.md](./cli_cosmic.md)

## 快速开始示例

### 启动服务器

```bash
# 使用默认配置启动服务器
./bbx-server

# 使用指定配置文件启动服务器
./bbx-server -config path/to/config.yaml
```

### 基本操作

```bash
# 查看版本信息
./bbx-cli version

# 扫描包含 "java" 的进程
./bbx-cli proc scan -q "java" -H http://localhost:8181

# 扫描并注册进程
./bbx-cli proc scan -q "java" -r -H http://localhost:8181

# 列出所有托管进程
./bbx-cli proc list

# 查看系统资源
./bbx-cli resources system

# 使用 Redis 发布消息
./bbx-cli redis publish -c channel1 -m "hello world"
```

## 获取帮助

要获取特定命令的帮助信息，可以使用 `--help` 或 `-h` 参数：

```bash
# 获取进程管理命令的帮助
./bbx-cli proc --help

# 获取扫描命令的帮助
./bbx-cli proc scan --help
```

## 注意事项

1. 确保服务器正在运行，并且 CLI 可以访问服务器 API
2. 对于需要服务器配置的功能（如 ACL、TLS 等），请确保服务器已正确配置
3. 建议在测试环境中运行命令，避免影响生产系统
4. 详细的命令参数和示例请参考各个命令的具体文档

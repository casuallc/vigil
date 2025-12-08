# Vigil

Vigil 是一个功能强大的进程管理和消息队列客户端工具，用于监控、管理系统进程和与多种消息队列系统进行交互。

## 功能特性

### 进程管理
- **进程扫描**：基于查询字符串或正则表达式扫描系统进程
- **进程创建**：创建新的托管进程
- **进程生命周期管理**：启动、停止、重启、删除进程
- **进程状态监控**：查看进程状态、资源使用情况
- **进程配置管理**：列出、编辑、获取进程配置
- **挂载管理**：为进程添加、移除和列出挂载点（支持 bind、tmpfs、volume 类型）

### 资源管理
- **系统资源监控**：查看系统资源使用情况
- **进程资源监控**：查看特定进程的资源使用情况

### 消息队列客户端
Vigil 支持多种消息队列系统，提供统一的 API 接口：

| 消息队列 | 功能 | 消息计数 |
|---------|------|---------|
| Redis   | 支持发布/订阅消息 | ✅ |
| RabbitMQ | 支持生产/消费消息 | ✅ |
| RocketMQ | 支持生产/消费消息 | ✅ |
| Kafka | 支持生产/消费消息 | ✅ |
| MQTT | 支持发布/订阅消息 | ✅ |
| Pulsar | 支持生产/消费消息 | ✅ |
| Zookeeper | 支持基本操作 | ❌ |

所有消息队列客户端都实现了消息计数功能，在退出时会打印生产和消费的消息总数。

### 其他功能
- **配置管理**：查看和管理系统配置
- **命令执行**：执行命令或脚本
- **批量扫描**：从配置文件加载并扫描多个进程

## 项目结构

```
├── api/              # API 服务器和客户端实现
├── cli/              # 命令行界面实现
├── client/           # 各种消息队列客户端
│   ├── kafka/        # Kafka 客户端
│   ├── mqtt/         # MQTT 客户端
│   ├── pulsar/       # Pulsar 客户端
│   ├── rabbitmq/     # RabbitMQ 客户端
│   ├── redis/        # Redis 客户端
│   ├── rocketmq/     # RocketMQ 客户端
│   └── zookeeper/    # Zookeeper 客户端
├── cmd/              # 命令行入口
│   ├── bbx-cli/      # CLI 入口
│   └── bbx-server/   # 服务器入口
├── common/           # 通用工具函数
├── conf/             # 配置文件
├── config/           # 配置加载和管理
├── docs/             # 文档
├── inspection/       # 检查规则和实现
├── proc/             # 进程管理核心逻辑
├── scripts/          # 脚本文件
├── tests/            # 测试文件
└── version/          # 版本信息
```

## 快速开始

### 启动服务器

```bash
# 使用默认配置启动服务器
./bbx-server

# 使用指定配置文件启动服务器
./bbx-server -config path/to/config.yaml

# 指定服务器监听地址
./bbx-server -addr :8080
```

### 使用 CLI

```bash
# 查看版本信息
./bbx-cli version

# 查看帮助信息
./bbx-cli help

# 扫描进程
./bbx-cli proc scan -q "MQ" -H http://localhost:8181

# 列出所有托管进程
./bbx-cli proc list

# 查看系统资源
./bbx-cli resources system

# 使用 Redis 发布消息
./bbx-cli redis publish -c channel1 -m "hello world"

# 使用 RocketMQ 发送消息
./bbx-cli rocket send -t topic1 -m "hello rocketmq"
```

## 命令行参考

### 进程管理命令

```
bbx-cli proc [command]

命令列表：
  scan        扫描进程
  create      创建进程
  start       启动进程
  stop        停止进程
  restart     重启进程
  delete      删除进程
  list        列出进程
  status      检查进程状态
  edit        编辑进程定义
  get         获取进程详情
  mount       管理进程挂载点
```

### 资源管理命令

```
bbx-cli resources [command]

命令列表：
  system      获取系统资源
  process     获取进程资源
```

### 消息队列命令

每种消息队列系统都有对应的命令组，例如：

```
bbx-cli redis [command]
bbx-cli rabbit [command]
bbx-cli rocket [command]
bbx-cli kafka [command]
bbx-cli mqtt [command]
bbx-cli pulsar [command]
bbx-cli zk [command]
```

## 配置文件

### 服务器配置

服务器配置文件默认路径为 `config.yaml`，可以通过 `-config` 参数指定。配置示例：

```yaml
# 服务器配置示例
server:
  host: "0.0.0.0"
  port: 8181
  timeout: 30s

# 进程管理配置
process:
  default_namespace: "default"
  managed_processes_file: "proc/managed_processes.yaml"
```

### 扫描配置

批量扫描配置文件默认路径为 `conf/scan.yaml`，可以通过 `-c` 参数指定。配置示例：

```yaml
# 扫描配置示例
scans:
  - query: "redis-server"
    namespace: "default"
    register: true
  - query: "mongodb"
    namespace: "db"
    register: false
```

## 构建

### 构建所有组件

```bash
./build_all.sh
```

### 单独构建 CLI

```bash
go build -o bbx-cli ./cmd/bbx-cli
```

### 单独构建服务器

```bash
go build -o bbx-server ./cmd/bbx-server
```

## 运行环境

- Go 1.24.6 或更高版本
- 支持 Linux、Windows、macOS 等主流操作系统

## 依赖管理

项目使用 Go Modules 进行依赖管理，主要依赖包括：

- github.com/spf13/cobra - 命令行框架
- github.com/IBM/sarama - Kafka 客户端
- github.com/apache/pulsar-client-go - Pulsar 客户端
- github.com/apache/rocketmq-client-go/v2 - RocketMQ 客户端
- github.com/eclipse/paho.mqtt.golang - MQTT 客户端
- github.com/rabbitmq/amqp091-go - RabbitMQ 客户端
- github.com/redis/go-redis/v9 - Redis 客户端
- github.com/samuel/go-zookeeper - Zookeeper 客户端

## 使用示例

### 扫描并注册进程

```bash
# 扫描包含 "java" 的进程并注册
./bbx-cli proc scan -q "java" -r -H http://localhost:8181

# 批量扫描并注册进程
./bbx-cli proc scan -b -c conf/scan.yaml -r -H http://localhost:8181
```

### 管理进程

```bash
# 启动进程
./bbx-cli proc start my-process

# 停止进程
./bbx-cli proc stop my-process

# 查看进程状态
./bbx-cli proc status my-process

# 列出所有进程
./bbx-cli proc list
```

### 使用 Redis 客户端

```bash
# 发布消息
./bbx-cli redis publish -c my-channel -m "hello redis"

# 订阅消息
./bbx-cli redis subscribe -c my-channel
```

### 使用 RocketMQ 客户端

```bash
# 发送消息
./bbx-cli rocket send -t my-topic -m "hello rocketmq"

# 消费消息
./bbx-cli rocket consume -t my-topic -g my-group
```

## 消息计数

所有消息队列客户端都会记录生产和消费的消息总数，并在退出时打印：

```
Total messages produced: 10
Total messages consumed: 5
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[MIT License](LICENSE)

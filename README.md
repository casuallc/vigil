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
- **集成测试**：支持多种服务的集成测试，如MQTT测试
- **审计模块**：记录所有API请求的审计日志，包括操作类型、时间戳、用户、IP地址、状态等信息
- **HTTPS支持**：服务端支持HTTPS加密通信，提高安全性
- **TLS证书管理**：提供命令行工具生成自签名TLS证书

### VM管理
- **模拟SSH服务**：提供模拟的SSH服务，无需真实网络连接
- **命令执行与限制**：支持命令执行，记录命令执行日志，并限制危险命令
- **本地文件操作**：直接通过API进行文件上传、下载和列表操作
- **权限控制**：管理VM的访问权限
- **SSH转发服务**：支持为VM启动SSH转发服务，实现本地端口到VM的转发
- **加密存储**：使用加密方式存储SSH密钥路径和密码，确保敏感数据安全
- **非交互式VM选择**：支持通过命令行参数指定VM，无需交互式提示
- **VM组管理**：创建、列出、获取、更新和删除VM组，每个组可包含多个VM
- **批量文件传输**：支持向多个VM或VM组批量上传、下载文件和列出文件
- **SFTP文件传输**：基于SFTP协议的安全文件传输功能

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
├── crypto/           # 加密相关功能
├── docs/             # 文档
├── inspection/       # 检查规则和实现
├── proc/             # 进程管理核心逻辑
├── audit/            # 审计模块
├── vm/               # VM管理核心逻辑
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

# 使用HTTPS（需要在配置文件中设置证书和密钥路径）
./bbx-server -config path/to/config.yaml
```

### 使用 CLI

```bash
# 查看版本信息
./bbx-cli version

# 查看帮助信息
./bbx-cli help

# 扫描进程（HTTP）
./bbx-cli proc scan -q "MQ" -H http://127.0.0.1:8181

# 扫描进程（HTTPS）
./bbx-cli proc scan -q "MQ" -H https://127.0.0.1:8181

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

### VM管理命令

```
bbx-cli vm [command]

命令列表：
  list        列出所有VM
  add         添加VM
  get         获取VM详情
  delete      删除VM
  ssh         SSH连接到VM
  file        文件管理（上传、下载、列表）
  group       VM组管理（添加、列出、获取、更新、删除组）
  permission  权限管理
```

### SSH转发命令

```bash
# 为VM启动SSH转发服务
./bbx-cli vm ssh --vm my-vm --port 2222

# 使用自定义参数
./bbx-cli vm ssh --vm my-vm --host localhost --port 2222 --target-host 192.168.1.100 --target-port 22 --audit-log /tmp/ssh-audit.log
```

参数说明：
- `--vm, -n`: VM名称（必填）
- `--host, -l`: 本地绑定地址（默认：localhost）
- `--port, -p`: 本地绑定端口（默认：2222）
- `--target-host, -t`: 目标主机（默认：VM的IP）
- `--target-port, -T`: 目标端口（默认：VM的端口）
- `--target-username, -U`: 目标用户名（默认：VM的用户名）
- `--target-password, -P`: 目标密码
- `--target-key, -k`: 目标私钥路径
- `--audit-log, -a`: 审计日志路径

### VM组管理命令

```bash
# 添加VM组
./bbx-cli vm group add -n test-group -d "Test VM group" -v vm1 -v vm2

# 列出所有VM组
./bbx-cli vm group list

# 获取VM组详情
./bbx-cli vm group get -n test-group

# 更新VM组
./bbx-cli vm group update -n test-group -d "Updated test group" -v vm1 -v vm2 -v vm3

# 删除VM组
./bbx-cli vm group delete -n test-group
```

参数说明：
- `--name, -n`: 组名称（必填）
- `--description, -d`: 组描述
- `--vms, -v`: VM名称（可多次使用，添加多个VM）

### 批量文件传输命令

```bash
# 向多个VM上传文件
./bbx-cli vm file upload --vm vm1 --vm vm2 --source local.txt --target /remote/path

# 向VM组上传文件
./bbx-cli vm file upload --group test-group --source local.txt --target /remote/path

# 从多个VM下载文件
./bbx-cli vm file download --vm vm1 --vm vm2 --source /remote/path --target local/dir

# 列出多个VM上的文件
./bbx-cli vm file list --vm vm1 --vm vm2 --path /remote/path
```

参数说明：
- `--vm, -v`: VM名称（可多次使用，指定多个VM）
- `--group, -g`: 组名称（可多次使用，指定多个组）
- `--source, -s`: 源文件路径
- `--target, -t`: 目标文件路径
- `--path, -p`: 目录路径（用于list命令）

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

### 集成测试命令

使用集成测试命令可以测试各种服务的功能：

```
bbx-cli test [service]
```

目前支持的测试服务：

- **mqtt**：MQTT消息队列集成测试
  - `bbx-cli test mqtt all`：运行所有 MQTT 测试
  - `bbx-cli test mqtt v3`：运行MQTT 3.1.1 测试
  - `bbx-cli test mqtt v5`：运行MQTT 5.0 测试
  - `bbx-cli test mqtt emqx`：运行 EMQX 特定测试

### TLS证书管理命令

使用 TLS 命令可以生成自签名的 HTTPS 证书：

```
bbx-cli tls [command]

命令列表：
  generate    生成TLS证书
```

#### 生成TLS证书

```bash
# 使用默认参数生成证书
./bbx-cli tls generate

# 指定证书和密钥路径
./bbx-cli tls generate --cert cert.pem --key key.pem

# 指定主机名
./bbx-cli tls generate --host example.com
```

参数说明：
- `--cert, -c`：证书文件路径（默认：cert.pem）
- `--key, -k`：私钥文件路径（默认：key.pem）
- `--host, -H`：证书的主机名或IP地址（默认：localhost）

## 配置文件

### 服务器配置

服务器配置文件默认路径为 `conf/config.yaml`，可以通过 `-config` 参数指定。配置示例：

```yaml
log:
  level: info

monitor:
  rate: 5          # seconds

process:
  pid_file: ./../app.pid

security:
  encryption_key: 8FVKXDQxzgdEH8DR8wQPnCo6Ke5IwQ+CYdqdmjmi/Lk=

https:
  enabled: true
  cert_path: conf/cert.pem
  key_path: conf/key.pem

managed_apps: []
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
- github.com/google/uuid - 生成唯一ID

## 使用示例

### 扫描并注册进程

```bash
# 扫描包含 "java" 的进程并注册
./bbx-cli proc scan -q "java" -r -H http://127.0.0.1:8181

# 批量扫描并注册进程
./bbx-cli proc scan -b -c conf/scan.yaml -r -H http://127.0.0.1:8181
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

[Apache License](LICENSE)

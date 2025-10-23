# Vigil CLI 命令说明

Vigil 是一个进程管理与监控工具。本文档覆盖 CLI 中的所有命令、用途与参数，帮助你快速定位命令用法。

- 根命令：`vigil`
- 全局选项：`--host, -H` 指定 API 服务地址，默认 `http://localhost:8080`
- 版本：设置了 `Version: 1.0.0`，可使用 `vigil --version`

## 进程管理：`vigil proc`

用于管理和监控进程的所有操作。

- `scan`
  - 简介：Scan processes
  - 说明：Scan system processes based on query string or regex
  - 用法：`vigil proc scan --query <expr> [--register] [--namespace <ns>]`
  - 选项：
    - `--query, -q` 搜索关键字或正则（必填）
    - `--register, -r` 扫描后注册为受管进程
    - `--namespace, -n` 命名空间，默认 `default`

- `create [name]`
  - 简介：Create process
  - 说明：Create a new managed process
  - 用法：`vigil proc create [name] --command <path> [--namespace <ns>]`
  - 选项：
    - `--name, -N` 进程名（可替代位置参数）
    - `--command, -c` 启动命令路径
    - `--namespace, -n` 命名空间，默认 `default`

- `start [name]`
  - 简介：Start process
  - 说明：未传入 name 时会进入交互式选择
  - 用法：`vigil proc start [name] [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `stop [name]`
  - 简介：Stop process
  - 说明：未传入 name 时会进入交互式选择
  - 用法：`vigil proc stop [name] [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `restart [name]`
  - 简介：Restart process
  - 说明：未传入 name 时会进入交互式选择
  - 用法：`vigil proc restart [name] [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `delete [name]`
  - 简介：Delete a managed process
  - 说明：删除前会停止运行中的进程；未传入 name 时进入交互式选择
  - 用法：`vigil proc delete [name] [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `list`
  - 简介：List processes
  - 说明：列出所有受管进程
  - 用法：`vigil proc list [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `status [name]`
  - 简介：Check process status
  - 说明：查看指定受管进程状态
  - 用法：`vigil proc status <name> [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `edit [name]`
  - 简介：Edit process definition
  - 说明：通过 vim 编辑进程定义；未传入 name 时进入交互式选择
  - 用法：`vigil proc edit <name> [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

- `get [name]`
  - 简介：Get process details
  - 说明：获取进程详情；未传入 name 时进入交互式选择
  - 用法：`vigil proc get <name> [--format <yaml|text>] [--namespace <ns>]`
  - 选项：
    - `--format, -f` 输出格式，默认 `yaml`（可选 `text`）
    - `--namespace, -n` 命名空间，默认 `default`

### 挂载管理：`vigil proc mount`

用于管理受管进程的挂载（类似 Docker volume），支持 `bind/tmpfs/volume`。

- `add [name]`
  - 简介：Add a mount to a process
  - 说明：支持 `type=bind/tmpfs/volume`
  - 用法：`vigil proc mount add <name> --type <bind|tmpfs|volume> --target <path> [options]`
  - 选项：
    - `--type, -t` 挂载类型（`bind|tmpfs|volume`），默认 `bind`
    - `--target, -T` 进程内目标路径（必填）
    - `--source, -s` bind 挂载源路径
    - `--volume, -v` volume 名称
    - `--read-only, -r` 只读挂载
    - `--option, -o` 挂载选项（可重复）
    - `--namespace, -n` 命名空间，默认 `default`

- `remove [name]`
  - 简介：Remove mount(s) from a process
  - 说明：按目标路径或索引移除
  - 用法：`vigil proc mount remove <name> [--target <path>] [--index <i>] [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`
    - `--target, -T` 要移除的目标路径
    - `--index, -i` 要移除的挂载索引（数字）

- `list [name]`
  - 简介：List mounts of a process
  - 说明：列出进程所有挂载配置
  - 用法：`vigil proc mount list <name> [--namespace <ns>]`
  - 选项：
    - `--namespace, -n` 命名空间，默认 `default`

## 资源查看：`vigil resources`

用于查看系统与进程资源信息。

- `system`
  - 简介：Get system resources
  - 说明：获取系统资源使用信息
  - 用法：`vigil resources system`

- `process [pid]`
  - 简介：Get process resources
  - 说明：获取指定进程的资源使用信息
  - 用法：`vigil resources process <pid>`

## 配置管理：`vigil config`

用于查看和管理系统配置。

- `get`
  - 简介：Get configuration
  - 说明：获取当前配置
  - 用法：`vigil config get`

## 远程执行：`vigil exec [command/script]`

在服务器上执行命令或脚本，可设置环境变量或将输出写入文件。

- 简介：Execute a command or script
- 说明：Execute a command or script file on the server, with optional environment variables and output to file.
- 用法：`vigil exec <cmdOrScript> [--file] [--env KEY=VALUE]... [--result <file>]`
- 选项：
  - `--file, -f` 将参数视为脚本文件路径
  - `--env, -e` 设置环境变量（可重复，格式 `KEY=VALUE`）
  - `--result, -r` 将执行结果写入文件，而非输出到控制台

## Redis：`vigil redis`

执行 Redis 操作：`get/set/delete/info`。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--server, -s` 服务器地址，默认 `localhost`
  - `--port, -p` 端口，默认 `6379`
  - `--password, -P` 密码
  - `--db, -d` 数据库编号，默认 `0`

- `get`
  - 简介：Get value from Redis
  - 用法：`vigil redis get --key <k>`
  - 选项：
    - `--key, -k` 键

- `set`
  - 简介：Set value in Redis
  - 用法：`vigil redis set --key <k> --value <v>`
  - 选项：
    - `--key, -k` 键
    - `--value, -v` 值

- `delete`
  - 简介：Delete key from Redis
  - 用法：`vigil redis delete --key <k>`
  - 选项：
    - `--key, -k` 键

- `info`
  - 简介：Get Redis server information
  - 用法：`vigil redis info`

## RabbitMQ：`vigil rabbitmq`

执行 RabbitMQ 操作：声明/删除交换与队列、绑定/解绑、发布/消费等。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--server, -s` 服务器地址，默认 `localhost`
  - `--port, -p` 端口，默认 `5672`
  - `--vhost, -v` 虚拟主机，默认 `/`
  - `--user, -u` 用户名，默认 `guest`
  - `--password, -P` 密码，默认 `guest`

- `declare-exchange`
  - 简介：Declare a RabbitMQ exchange
  - 用法：`vigil rabbitmq declare-exchange --name <n> [--type <t>] [--durable] [--auto-delete]`
  - 选项：
    - `--name, -n` 交换机名
    - `--type, -t` 类型，默认 `direct`
    - `--durable, -d` 持久化（重启后保留）
    - `--auto-delete, -a` 自动删除

- `delete-exchange`
  - 简介：Delete a RabbitMQ exchange
  - 用法：`vigil rabbitmq delete-exchange --name <n>`
  - 选项：
    - `--name, -n` 交换机名

- `declare-queue`
  - 简介：Declare a RabbitMQ queue
  - 用法：`vigil rabbitmq declare-queue --name <n> [--durable] [--auto-delete] [--exclusive]`
  - 选项：
    - `--name, -n` 队列名
    - `--durable, -d` 持久化
    - `--auto-delete, -a` 无消费者时删除
    - `--exclusive, -e` 仅当前连接访问

- `delete-queue`
  - 简介：Delete a RabbitMQ queue
  - 用法：`vigil rabbitmq delete-queue --name <n>`
  - 选项：
    - `--name, -n` 队列名

- `queue-bind`
  - 简介：Bind a queue to an exchange
  - 用法：`vigil rabbitmq queue-bind --queue <q> --exchange <e> [--routing-key <r>] [--args key=val,...]`
  - 选项：
    - `--queue, -q` 队列名
    - `--exchange, -e` 交换机名
    - `--routing-key, -r` 路由键
    - `--args, -a` 额外参数，格式 `k1=v1,k2=v2`

- `queue-unbind`
  - 简介：Unbind a queue from an exchange
  - 用法：`vigil rabbitmq queue-unbind --queue <q> --exchange <e> [--routing-key <r>] [--args key=val,...]`
  - 选项：
    - `--queue, -q` 队列名
    - `--exchange, -e` 交换机名
    - `--routing-key, -r` 路由键
    - `--args, -a` 绑定参数

- `publish`
  - 简介：Publish a message to an exchange
  - 用法：`vigil rabbitmq publish --exchange <e> --routing-key <r> --message <m> [options]`
  - 选项：
    - `--print-log` 打印日志
    - `--exchange, -e` 交换机名
    - `--routing-key, -r` 路由键
    - `--message, -m` 消息内容
    - `--interval, -i` 发送间隔（毫秒），默认 `1000`
    - `--repeat, -t` 重复次数，默认 `10`
    - `--rate-limit, -l` 速率限制

- `consume`
  - 简介：Consume messages from a queue
  - 用法：`vigil rabbitmq consume --queue <q> [--consumer <c>] [--auto-ack] [--timeout <s>]`
  - 选项：
    - `--queue, -q` 队列名
    - `--consumer, -c` 消费者名称
    - `--auto-ack, -a` 自动确认，默认启用
    - `--timeout, -t` 等待超时秒数，默认 `10`

## Kafka：`vigil kafka`

执行 Kafka 操作：发送与接收消息。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--servers, -s` 服务器地址（逗号分隔），默认 `localhost`
  - `--port, -p` 端口，默认 `9092`
  - `--user, -u` 用户名
  - `--password` 密码
  - `--sasl-mechanism` SASL 机制（`PLAIN|SCRAM-SHA-256|SCRAM-SHA-512`）
  - `--sasl-protocol` SASL 协议，默认 `SASL_PLAINTEXT`
  - `--timeout` 连接超时（秒），默认 `30`

- `send`
  - 简介：Send message to Kafka
  - 用法：`vigil kafka send --topic <t> --message <m> [options]`
  - 选项：
    - `--topic, -t` 主题
    - `--message, -m` 内容
    - `--key, -k` 键
    - `--repeat, -r` 重复次数，默认 `10`
    - `--interval, -i` 发送间隔（毫秒），默认 `1000`
    - `--print-log` 打印日志
    - `--acks, -a` 确认级别（`0|1|-1/all`），默认 `1`
    - `--message-length` 填充消息长度
    - `--compression, -c` 压缩（`gzip|snappy|lz4|zstd`）
    - `--headers` 头部，格式 `name=value,name2=value2`

- `receive`
  - 简介：Receive messages from Kafka
  - 用法：`vigil kafka receive --topic <t> [--group-id <g>] [options]`
  - 选项：
    - `--topic, -t` 主题
    - `--group-id, -g` 消费者组 ID，默认 `default_consumer_group`
    - `--offset, -o` 指定偏移（`offset-type` 为 `specific` 时生效）
    - `--offset-type` 偏移类型（`earliest|latest|specific`），默认 `latest`
    - `--timeout` 超时（秒）
    - `--print-log` 打印日志
    - `--max-messages` 最大消息数（0 为无限）

## Pulsar：`vigil pulsar`

执行 Apache Pulsar 操作：发送与接收消息。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--url` 服务地址，默认 `pulsar://localhost:6650`
  - `--token` 认证令牌
  - `--timeout, -o` 超时（秒），默认 `30`

- `send`
  - 简介：Send message to Pulsar
  - 用法：`vigil pulsar send --topic <t> --message <m> [options]`
  - 选项：
    - `--topic, -t` 主题
    - `--message, -m` 内容
    - `--key, -k` 键
    - `--send-timeout` 发送超时（毫秒）
    - `--enable-batching` 启用批量
    - `--batching-max-delay` 批量最大延迟（毫秒）
    - `--batching-max-messages` 批量消息数
    - `--message-length` 填充消息长度
    - `--repeat, -r` 重复次数
    - `--interval, -i` 发送间隔（毫秒）
    - `--print-log` 打印日志
    - `--delay-time` 延时消息（毫秒）
    - `--deliver-time` 定时投递时间（`YYYY-MM-DD HH:mm:ss`）
    - `--enable-compression` 启用压缩
    - `--properties` 属性，格式 `key=val,key=val`

- `receive`
  - 简介：Receive messages from Pulsar
  - 用法：`vigil pulsar receive --topic <t> [--subscription <s>] [options]`
  - 选项：
    - `--topic, -t` 主题
    - `--subscription, -s` 订阅名，默认 `default-subscription`
    - `--subscription-type` 订阅类型（`Exclusive|Shared|Failover|Key_Shared`），默认 `Exclusive`
    - `--receive-timeout` 接收超时（毫秒），默认 `10000`
    - `--message-timeout` 处理超时（秒）
    - `--initial-position` 初始位置（`Earliest|Latest`），默认 `Latest`
    - `--auto-ack` 自动确认，默认启用
    - `--count, -c` 接收消息数量（0 为无限）

## RocketMQ：`vigil rocketmq`

执行 RocketMQ 操作：发送与接收消息（含批量与事务）。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--server, -s` 服务器地址，默认 `localhost`
  - `--port, -p` 端口，默认 `9876`
  - `--user, -u` 用户名
  - `--namespace, -n` 命名空间
  - `--access-key` 访问键
  - `--secret-key` 密钥

- `send`
  - 简介：Send message to RocketMQ
  - 用法：`vigil rocketmq send --topic <t> --message <m> [options]`
  - 选项：
    - `--group, -g` 生产者组，默认 `default_group`
    - `--topic, -t` 主题
    - `--tags` 标签
    - `--keys, -k` 键
    - `--message, -m` 内容
    - `--repeat, -r` 重复次数
    - `--interval, -i` 发送间隔（毫秒）
    - `--send-type` 发送类型（`sync|async`），默认 `sync`
    - `--delay-level` 延时等级
    - `--print-log` 打印日志
    - `--trace` 启用消息追踪
    - `--message-length` 填充消息长度

- `receive`
  - 简介：Receive messages from RocketMQ
  - 用法：`vigil rocketmq receive --topic <t> [--group <g>] [options]`
  - 选项：
    - `--group, -g` 消费者组，默认 `default_consumer_group`
    - `--topic, -t` 主题
    - `--tags` 标签过滤（`*` 表示全部）
    - `--timeout` 超时（秒）
    - `--start-pos` 起始位置（`FIRST|LAST|TIMESTAMP`）
    - `--timestamp` 时间戳（`20060102150405`）
    - `--consume-type` 消费类型（`SYNC|ASYNC`）
    - `--print-log` 打印日志
    - `--retry-count` 重试次数
    - `--trace` 启用消息追踪

- `batch-send`
  - 简介：Batch send messages to RocketMQ
  - 用法：`vigil rocketmq batch-send --topic <t> --message <m> [options]`
  - 选项：
    - `--group, -g` 生产者组，默认 `default_group`
    - `--topic, -t` 主题
    - `--tags` 标签
    - `--keys, -k` 键
    - `--message, -m` 内容
    - `--repeat, -r` 批次数
    - `--interval, -i` 批次间隔（毫秒）
    - `--batch-size` 每批消息数，默认 `10`
    - `--print-log` 打印日志
    - `--trace` 启用消息追踪

- `transaction-send`
  - 简介：Send transaction messages to RocketMQ
  - 用法：`vigil rocketmq transaction-send --topic <t> --message <m> [options]`
  - 选项：
    - `--group, -g` 事务生产者组，默认 `default_transaction_group`
    - `--topic, -t` 主题
    - `--tags` 标签
    - `--keys, -k` 键
    - `--message, -m` 内容
    - `--repeat, -r` 重复次数
    - `--interval, -i` 间隔（毫秒）
    - `--print-log` 打印日志
    - `--check-times` 事务检查次数，默认 `3`

## MQTT：`vigil mqtt`

执行 MQTT 操作：发布与订阅。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--server` 服务器地址，默认 `localhost`
  - `--port` 端口，默认 `1883`
  - `--user` 用户名
  - `--password` 密码
  - `--client-id` 客户端 ID
  - `--clean-start` 清理会话，默认启用
  - `--keep-alive` 保活间隔（秒），默认 `60`
  - `--timeout` 连接超时（秒），默认 `30`

- `publish`
  - 简介：Publish a message to an MQTT topic
  - 用法：`vigil mqtt publish --topic <t> [--qos <0|1|2>] [--message <m>] [options]`
  - 选项：
    - `--topic, -t` 主题
    - `--qos, -q` 服务质量等级（`0|1|2`），默认 `0`
    - `--message, -m` 内容，默认 `Hello, MQTT!`
    - `--repeat, -r` 重复次数，默认 `1`
    - `--interval, -i` 间隔（毫秒），默认 `1000`
    - `--retained, -R` 保留消息
    - `--print-log` 打印日志，默认启用

- `subscribe`
  - 简介：Subscribe to an MQTT topic
  - 用法：`vigil mqtt subscribe --topic <t> [--qos <0|1|2>] [--timeout <s>]`
  - 选项：
    - `--topic, -t` 主题
    - `--qos, -q` 服务质量等级，默认 `0`
    - `--timeout, -o` 超时（秒，`0` 为无限）
    - `--print-log` 打印日志，默认启用

## Zookeeper：`vigil zookeeper`

执行 Zookeeper 节点操作。父命令提供持久化连接参数，子命令继承。

- 父级选项：
  - `--server, -s` 服务器地址，默认 `localhost`
  - `--port, -p` 端口，默认 `2181`
  - `--timeout, -t` 连接超时（秒），默认 `30`

- `create`
  - 简介：Create a Zookeeper node
  - 用法：`vigil zookeeper create --path <p> [--data <d>]`
  - 选项：
    - `--path` 节点路径
    - `--data, -d` 节点数据

- `delete`
  - 简介：Delete a Zookeeper node
  - 用法：`vigil zookeeper delete --path <p>`
  - 选项：
    - `--path` 节点路径

- `exists`
  - 简介：Check if a Zookeeper node exists
  - 用法：`vigil zookeeper exists --path <p>`
  - 选项：
    - `--path` 节点路径

- `get`
  - 简介：Get data from a Zookeeper node
  - 用法：`vigil zookeeper get --path <p>`
  - 选项：
    - `--path` 节点路径

- `set`
  - 简介：Set data for a Zookeeper node
  - 用法：`vigil zookeeper set --path <p> --data <d>`
  - 选项：
    - `--path` 节点路径
    - `--data, -d` 节点数据
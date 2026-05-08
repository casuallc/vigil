# RabbitMQ 命令

RabbitMQ命令用于与RabbitMQ服务器进行交互，支持交换机/队列管理和消息的生产消费。

## 命令格式

```
bbx-cli rabbitmq [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | RabbitMQ服务器地址 | 127.0.0.1 |
| `--port` | `-p` | RabbitMQ服务器端口 | 5672 |
| `--user` | `-u` | RabbitMQ用户名 | guest |
| `--password` | `-P` | RabbitMQ密码 | guest |
| `--vhost` | `-v` | RabbitMQ虚拟主机 | / |

## 命令列表

### declare-exchange - 声明交换机

声明一个RabbitMQ交换机。

**用法：**
```
bbx-cli rabbitmq declare-exchange [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--name` | `-n` | 交换机名称（必填） | - |
| `--type` | `-t` | 交换机类型 | direct |
| `--durable` | `-d` | 交换机是否在broker重启后保留 | true |
| `--auto-delete` | `-a` | 是否自动删除 | false |

### delete-exchange - 删除交换机

删除一个RabbitMQ交换机。

**用法：**
```
bbx-cli rabbitmq delete-exchange [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--name` | `-n` | 交换机名称（必填） | - |

### declare-queue - 声明队列

声明一个RabbitMQ队列。

**用法：**
```
bbx-cli rabbitmq declare-queue [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--name` | `-n` | 队列名称（必填） | - |
| `--durable` | `-d` | 队列是否在broker重启后保留 | true |
| `--auto-delete` | `-a` | 最后一个消费者取消订阅后是否自动删除 | false |
| `--exclusive` | `-e` | 是否仅限当前连接访问 | false |

### delete-queue - 删除队列

删除一个RabbitMQ队列。

**用法：**
```
bbx-cli rabbitmq delete-queue [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--name` | `-n` | 队列名称（必填） | - |

### queue-bind - 绑定队列

将队列绑定到交换机。

**用法：**
```
bbx-cli rabbitmq queue-bind [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--queue` | `-q` | 队列名称（必填） | - |
| `--exchange` | `-e` | 交换机名称（必填） | - |
| `--routing-key` | `-r` | 路由键 | - |
| `--args` | `-a` | 额外绑定参数，格式：key1=value1,key2=value2 | - |

### queue-unbind - 解绑队列

将队列从交换机解绑。

**用法：**
```
bbx-cli rabbitmq queue-unbind [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--queue` | `-q` | 队列名称（必填） | - |
| `--exchange` | `-e` | 交换机名称（必填） | - |
| `--routing-key` | `-r` | 路由键 | - |
| `--args` | `-a` | 绑定参数，格式：key1=value1,key2=value2 | - |

### publish - 发布消息

向RabbitMQ交换机发布消息。支持直接发送文本、从文件读取内容发送（支持超大文件），或遍历文件夹将每个文件作为一条消息发送。

**用法：**
```
bbx-cli rabbitmq publish [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件或文件夹路径，读取内容作为消息发送 | - |
| `--recursive` | `-R` | 递归发送子目录中的文件（配合 `--file` 目录时使用） | false |
| `--exchange` | `-e` | 交换机名称 | - |
| `--routing-key` | `-r` | 路由键 | - |
| `--interval` | `-i` | 发送消息间隔（毫秒） | 1000 |
| `--repeat` | `-t` | 重复发送次数 | 10 |
| `--rate-limit` | `-l` | 发送速率限制 | 0 |
| `--print-log` | - | 是否打印日志 | true |

### consume - 消费消息

从RabbitMQ队列消费消息。

**用法：**
```
bbx-cli rabbitmq consume [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--queue` | `-q` | 队列名称（必填） | - |
| `--consumer` | `-c` | 消费者名称 | - |
| `--auto-ack` | `-a` | 是否自动确认消息 | true |
| `--timeout` | `-t` | 等待消息超时时间（秒） | 10 |

## 示例

```bash
# 声明交换机
bbx-cli rabbitmq declare-exchange -n my-exchange -t direct -s 127.0.0.1 -p 5672

# 声明队列
bbx-cli rabbitmq declare-queue -n my-queue -s 127.0.0.1 -p 5672

# 绑定队列到交换机
bbx-cli rabbitmq queue-bind -q my-queue -e my-exchange -r my-key

# 发布文本消息
bbx-cli rabbitmq publish -m "hello world" -e my-exchange -r my-key -s 127.0.0.1 -p 5672

# 发布文件内容（支持超大文件）
bbx-cli rabbitmq publish -f /path/to/large_file.log -e my-exchange -r my-key

# 发布文件夹下所有文件，每个文件作为一条消息
bbx-cli rabbitmq publish -f /path/to/dir -e my-exchange -r my-key

# 递归发送文件夹及子目录中的所有文件
bbx-cli rabbitmq publish -f /path/to/dir -R -e my-exchange -r my-key

# 消费消息
bbx-cli rabbitmq consume -q my-queue -a true -t 10 -s 127.0.0.1 -p 5672

# 删除队列
bbx-cli rabbitmq delete-queue -n my-queue

# 删除交换机
bbx-cli rabbitmq delete-exchange -n my-exchange
```

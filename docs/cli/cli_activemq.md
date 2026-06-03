# ActiveMQ 命令

ActiveMQ 命令用于与 ActiveMQ 消息队列进行交互，通过 STOMP 协议支持发送和接收消息。

## 命令格式

```
bbx-cli activemq [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | ActiveMQ 服务器地址 | 127.0.0.1 |
| `--port` | `-p` | ActiveMQ STOMP 端口 | 61613 |
| `--user` | `-u` | 认证用户名 | - |
| `--password` | - | 认证密码 | - |
| `--timeout` | - | 连接超时时间（秒） | 30 |
| `--vhost` | - | 虚拟主机 | / |
| `--heartbeat` | - | 心跳间隔（秒，0 表示禁用） | 0 |

## 命令列表

### send - 发送消息

向 ActiveMQ 目标地址（Queue 或 Topic）发送消息。支持直接发送文本、从文件读取内容发送，或遍历文件夹将每个文件作为一条消息发送。

**用法：**
```
bbx-cli activemq send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--destination` | `-d` | 目标地址（必填，如 `/queue/test` 或 `/topic/test`） | - |
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件或文件夹路径，读取内容作为消息发送 | - |
| `--recursive` | `-R` | 递归发送子目录中的文件（配合 `--file` 目录时使用） | false |
| `--repeat` | `-r` | 重复发送次数 | 10 |
| `--interval` | `-i` | 发送消息间隔（毫秒） | 1000 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--headers` | - | 消息头，格式：`name=value,name2=value2` | - |
| `--persistent` | - | 发送持久化消息 | false |

### receive - 接收消息

从 ActiveMQ 目标地址接收消息。

**用法：**
```
bbx-cli activemq receive [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--destination` | `-d` | 目标地址（必填，如 `/queue/test` 或 `/topic/test`） | - |
| `--timeout` | - | 消费者超时时间（秒，0 表示无超时） | 0 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--max-messages` | - | 最大接收消息数（0 表示无限制） | 0 |
| `--durable` | - | 创建持久订阅（仅 Topic 有效） | false |
| `--subscription-name` | - | 持久订阅名称 | - |
| `--ack-mode` | - | 确认模式（auto, client, client-individual） | auto |

## 示例

```bash
# 发送消息到 Queue
bbx-cli activemq send -d /queue/test -m "hello activemq" -s 127.0.0.1 -p 61613

# 发送消息到 Topic
bbx-cli activemq send -d /topic/test -m "hello topic" -r 5 -i 500 -s 127.0.0.1

# 发送持久化消息
bbx-cli activemq send -d /queue/test -m "persistent msg" --persistent -s 127.0.0.1

# 发送文件内容
bbx-cli activemq send -d /queue/test -f /path/to/file.log -s 127.0.0.1

# 发送文件夹下所有文件，每个文件作为一条消息
bbx-cli activemq send -d /queue/test -f /path/to/dir -s 127.0.0.1

# 递归发送文件夹及子目录中的所有文件
bbx-cli activemq send -d /queue/test -f /path/to/dir -R -s 127.0.0.1

# 接收 Queue 消息
bbx-cli activemq receive -d /queue/test -s 127.0.0.1 -p 61613

# 接收 Topic 消息（带超时和最大消息数限制）
bbx-cli activemq receive -d /topic/test --timeout 30 --max-messages 100

# 使用 client 确认模式接收消息
bbx-cli activemq receive -d /queue/test --ack-mode client

# 创建持久订阅接收 Topic 消息
bbx-cli activemq receive -d /topic/test --durable --subscription-name my-sub

# 带认证发送消息
bbx-cli activemq send -d /queue/test -m "hello" -u admin --password admin -s 127.0.0.1
```

## 注意事项

1. ActiveMQ 默认 STOMP 端口为 **61613**，请确保防火墙已放行
2. Destination 格式为 `/queue/<name>` 或 `/topic/<name>`
3. 持久订阅仅对 Topic 有效，需要指定 `--subscription-name`
4. `client` 和 `client-individual` 确认模式下，消息会被手动确认；如果处理失败，消息可能重新投递
5. 所有 MQ 客户端在退出时会打印生产和消费的消息总数

# Kafka 命令

Kafka命令用于与Kafka集群进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli kafka [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--servers` | `-s` | Kafka服务器地址（逗号分隔） | 127.0.0.1 |
| `--port` | `-p` | Kafka服务器端口 | 9092 |
| `--user` | `-u` | 认证用户名 | - |
| `--password` | - | 认证密码 | - |
| `--sasl-mechanism` | - | SASL机制（PLAIN, SCRAM-SHA-256, SCRAM-SHA-512） | - |
| `--sasl-protocol` | - | SASL协议 | SASL_PLAINTEXT |
| `--timeout` | - | 连接超时时间（秒） | 30 |

## 命令列表

### send - 发送消息

向Kafka主题发送消息。

**用法：**
```
bbx-cli kafka send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--message` | `-m` | 消息内容（必填） | - |
| `--key` | `-k` | 消息键 | - |
| `--repeat` | `-r` | 重复发送次数 | 10 |
| `--interval` | `-i` | 发送消息间隔（毫秒） | 1000 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--acks` | `-a` | 确认级别（0, 1, -1/all） | 1 |
| `--message-length` | - | 消息长度，不足时以空格填充 | 0 |
| `--compression` | `-c` | 压缩类型（gzip, snappy, lz4, zstd） | - |
| `--headers` | - | 消息头，格式：name=value,name2=value2 | - |

### receive - 接收消息

从Kafka主题接收消息。

**用法：**
```
bbx-cli kafka receive [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--group-id` | `-g` | 消费者组ID | default_consumer_group |
| `--offset` | `-o` | 偏移量（仅在 `--offset-type=specific` 时有效） | 0 |
| `--offset-type` | - | 偏移量类型（earliest, latest, specific） | latest |
| `--timeout` | - | 消费者超时时间（秒，0表示无超时） | 0 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--max-messages` | - | 最大接收消息数（0表示无限制） | 0 |

## 示例

```bash
# 发送消息到主题
bbx-cli kafka send -t topic1 -m "hello kafka" -s 127.0.0.1 -p 9092

# 发送消息并指定消息键
bbx-cli kafka send -t topic1 -m "hello kafka" -k mykey -r 5 -i 500 -s 127.0.0.1

# 接收消息
bbx-cli kafka receive -t topic1 -g my_group -s 127.0.0.1 -p 9092

# 从最早偏移量接收消息
bbx-cli kafka receive -t topic1 --offset-type earliest --max-messages 100
```

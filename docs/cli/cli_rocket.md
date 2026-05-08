# RocketMQ 命令

RocketMQ命令用于与RocketMQ服务器进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli rocketmq [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | RocketMQ服务器地址 | 127.0.0.1 |
| `--port` | `-p` | RocketMQ服务器端口 | 9876 |
| `--user` | `-u` | 用户名 | - |
| `--namespace` | `-n` | 命名空间 | - |
| `--access-key` | - | Access Key | - |
| `--secret-key` | - | Secret Key | - |

## 命令列表

### send - 发送消息

向RocketMQ主题发送消息。支持直接发送文本、从文件读取内容发送（支持超大文件）。

**用法：**
```
bbx-cli rocketmq send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件路径，读取内容作为消息发送 | - |
| `--group` | `-g` | 生产者组名 | default_group |
| `--tags` | - | 消息标签 | - |
| `--keys` | `-k` | 消息键 | - |
| `--repeat` | `-r` | 重复发送次数 | 1 |
| `--interval` | `-i` | 发送间隔（毫秒） | 1000 |
| `--send-type` | - | 发送类型（sync/async） | sync |
| `--delay-level` | - | 延迟消息级别 | 0 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--trace` | - | 是否使用消息轨迹 | false |
| `--message-length` | - | 消息长度，不足时以空格填充 | 0 |

### batch-send - 批量发送消息

批量向RocketMQ主题发送消息。支持从文件读取内容。

**用法：**
```
bbx-cli rocketmq batch-send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件路径，读取内容作为消息发送 | - |
| `--group` | `-g` | 生产者组名 | default_group |
| `--tags` | - | 消息标签 | - |
| `--keys` | `-k` | 消息键 | - |
| `--repeat` | `-r` | 重复发送次数 | 1 |
| `--interval` | `-i` | 批次间隔（毫秒） | 1000 |
| `--batch-size` | - | 每批消息数量 | 10 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--trace` | - | 是否使用消息轨迹 | false |

### transaction-send - 发送事务消息

向RocketMQ主题发送事务消息。支持从文件读取内容。

**用法：**
```
bbx-cli rocketmq transaction-send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件路径，读取内容作为消息发送 | - |
| `--group` | `-g` | 事务生产者组名 | default_transaction_group |
| `--tags` | - | 消息标签 | - |
| `--keys` | `-k` | 消息键 | - |
| `--repeat` | `-r` | 重复发送次数 | 1 |
| `--interval` | `-i` | 发送间隔（毫秒） | 1000 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--check-times` | - | 事务回查次数 | 3 |

### receive - 消费消息

从RocketMQ主题消费消息。

**用法：**
```
bbx-cli rocketmq receive [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--group` | `-g` | 消费者组名 | default_consumer_group |
| `--tags` | - | 消息标签过滤（`*` 表示全部） | * |
| `--timeout` | - | 消费者超时时间（秒，0表示无超时） | 0 |
| `--start-pos` | - | 开始消费位置（FIRST/LAST/TIMESTAMP） | LAST |
| `--consume-type` | - | 消费类型（SYNC/ASYNC） | SYNC |
| `--print-log` | - | 是否打印详细日志 | true |
| `--retry-count` | - | 消息重试次数 | 0 |
| `--trace` | - | 是否使用消息轨迹 | false |

## 示例

```bash
# 发送消息到主题
bbx-cli rocketmq send -t topic1 -m "hello rocketmq" -s 127.0.0.1 -p 9876

# 发送文件内容
bbx-cli rocketmq send -t topic1 -f /path/to/file.txt -s 127.0.0.1

# 批量发送
bbx-cli rocketmq batch-send -t topic1 -m "hello" -s 127.0.0.1

# 发送事务消息
bbx-cli rocketmq transaction-send -t topic1 -m "hello" -s 127.0.0.1

# 消费消息
bbx-cli rocketmq receive -t topic1 -g my_group -s 127.0.0.1 -p 9876
```

# Pulsar 命令

Pulsar命令用于与Pulsar集群进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli pulsar [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--url` | - | Pulsar服务URL | pulsar://127.0.0.1:6650 |
| `--token` | - | 认证Token | eyJhbGciOiJIUzI1NiJ9... |
| `--timeout` | `-o` | 连接超时时间（秒） | 30 |

## 命令列表

### send - 发送消息

向Pulsar主题发送消息。支持直接发送文本、从文件读取内容发送（支持超大文件），或遍历文件夹将每个文件作为一条消息发送。

**用法：**
```
bbx-cli pulsar send [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--message` | `-m` | 消息内容（与 `--file` 至少填一个） | - |
| `--file` | `-f` | 文件或文件夹路径，读取内容作为消息发送 | - |
| `--recursive` | `-R` | 递归发送子目录中的文件（配合 `--file` 目录时使用） | false |
| `--key` | `-k` | 消息键 | - |
| `--send-timeout` | - | 发送超时时间（毫秒） | 30000 |
| `--enable-batching` | - | 是否启用消息批处理 | false |
| `--batching-max-delay` | - | 批处理最大延迟（毫秒） | 10 |
| `--batching-max-messages` | - | 每批最大消息数量 | 1000 |
| `--message-length` | - | 消息长度，不足时以空格填充 | 0 |
| `--repeat` | `-r` | 重复发送次数 | 1 |
| `--interval` | `-i` | 发送消息间隔（毫秒） | 1000 |
| `--print-log` | - | 是否打印详细日志 | true |
| `--delay-time` | - | 延迟消息延迟时间（毫秒） | 0 |
| `--deliver-time` | - | 定时消息投递时间（格式：2006-01-02 15:04:05） | - |
| `--enable-compression` | - | 是否启用消息压缩 | false |
| `--properties` | - | 消息属性，格式：key=val,key=val | - |

### receive - 接收消息

从Pulsar主题接收消息。

**用法：**
```
bbx-cli pulsar receive [flags]
```

**参数：**

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--topic` | `-t` | 主题名称（必填） | - |
| `--subscription` | `-s` | 订阅名称 | default-subscription |
| `--subscription-type` | - | 订阅类型（Exclusive/Shared/Failover/Key_Shared） | Exclusive |
| `--receive-timeout` | - | 接收超时时间（毫秒） | 10000 |
| `--message-timeout` | - | 消息处理超时时间（秒，0表示无超时） | 0 |
| `--initial-position` | - | 初始位置（Earliest/Latest） | Latest |
| `--auto-ack` | - | 是否自动确认消息 | true |
| `--count` | `-c` | 接收消息数量（0表示无限制） | 0 |

## 示例

```bash
# 发送消息到主题
bbx-cli pulsar send -t topic1 -m "hello pulsar"

# 发送文件内容（支持超大文件）
bbx-cli pulsar send -t topic1 -f /path/to/large_file.log

# 发送文件夹下所有文件，每个文件作为一条消息
bbx-cli pulsar send -t topic1 -f /path/to/dir

# 递归发送文件夹及子目录中的所有文件
bbx-cli pulsar send -t topic1 -f /path/to/dir -R

# 接收消息
bbx-cli pulsar receive -t topic1 -c 10 -s my_subscription
```

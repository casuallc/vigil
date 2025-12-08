# Kafka 命令

Kafka命令用于与Kafka集群进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli kafka [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--brokers` | `-b` | Kafka brokers地址 | localhost:9092 |
| `--group` | `-g` | 消费者组名 | default_group |

## 命令列表

### send - 发送消息

向Kafka主题发送消息。

**用法：**
```
bbx-cli kafka send [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-m, --message string`：要发送的消息
- `-k, --key string`：消息键

### consume - 消费消息

从Kafka主题消费消息。

**用法：**
```
bbx-cli kafka consume [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-c, --count int`：要消费的消息数量（默认：1）
- `-o, --offset string`：消费起始偏移量（earliest/latest）

## 示例

```bash
# 发送消息到主题
./bbx-cli kafka send -t topic1 -m "hello kafka" -b localhost:9092

# 消费消息
./bbx-cli kafka consume -t topic1 -c 10 -g my_group -b localhost:9092
```

# RocketMQ 命令

RocketMQ命令用于与RocketMQ服务器进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli rocket [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--namesrv` | `-n` | NameServer地址 | localhost:9876 |
| `--group` | `-g` | 消费者组名 | default_group |

## 命令列表

### send - 发送消息

向RocketMQ主题发送消息。

**用法：**
```
bbx-cli rocket send [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-m, --message string`：要发送的消息
- `-k, --key string`：消息键
- `-t, --tags string`：消息标签

### consume - 消费消息

从RocketMQ主题消费消息。

**用法：**
```
bbx-cli rocket consume [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-c, --count int`：要消费的消息数量（默认：1）
- `-s, --selector string`：消息选择器

## 示例

```bash
# 发送消息到主题
./bbx-cli rocket send -t topic1 -m "hello rocketmq" -n localhost:9876

# 消费消息
./bbx-cli rocket consume -t topic1 -c 5 -g my_group -n localhost:9876
```

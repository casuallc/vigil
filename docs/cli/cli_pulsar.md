# Pulsar 命令

Pulsar命令用于与Pulsar集群进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli pulsar [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--service-url` | `-s` | Pulsar服务URL | pulsar://localhost:6650 |
| `--admin-url` | `-a` | Pulsar管理API URL | http://localhost:8080 |
| `--topic` | `-t` | 主题名称 |  |

## 命令列表

### send - 发送消息

向Pulsar主题发送消息。

**用法：**
```
bbx-cli pulsar send [flags]
```

**参数：**
- `-m, --message string`：要发送的消息
- `-k, --key string`：消息键

### receive - 接收消息

从Pulsar主题接收消息。

**用法：**
```
bbx-cli pulsar receive [flags]
```

**参数：**
- `-c, --count int`：要接收的消息数量（默认：1）
- `-s, --subscription string`：订阅名称

## 示例

```bash
# 发送消息到主题
./bbx-cli pulsar send -t topic1 -m "hello pulsar" -s pulsar://localhost:6650

# 接收消息
./bbx-cli pulsar receive -t topic1 -c 10 -s my_subscription -s pulsar://localhost:6650
```

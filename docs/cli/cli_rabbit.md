# RabbitMQ 命令

RabbitMQ命令用于与RabbitMQ服务器进行交互，支持生产和消费消息。

## 命令格式

```
bbx-cli rabbit [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | RabbitMQ服务器地址 | localhost |
| `--port` | `-p` | RabbitMQ服务器端口 | 5672 |
| `--username` | `-u` | RabbitMQ用户名 | guest |
| `--password` | `-P` | RabbitMQ密码 | guest |
| `--vhost` | `-v` | RabbitMQ虚拟主机 | / |

## 命令列表

### send - 发送消息

向RabbitMQ队列发送消息。

**用法：**
```
bbx-cli rabbit send [flags]
```

**参数：**
- `-t, --topic string`：主题或队列名称
- `-m, --message string`：要发送的消息
- `-e, --exchange string`：交换机名称
- `-r, --routing-key string`：路由键

### receive - 接收消息

从RabbitMQ队列接收消息。

**用法：**
```
bbx-cli rabbit receive [flags]
```

**参数：**
- `-t, --topic string`：主题或队列名称
- `-c, --count int`：要接收的消息数量（默认：1）

## 示例

```bash
# 发送消息到队列
./bbx-cli rabbit send -t queue1 -m "hello world" -s localhost -p 5672

# 从队列接收消息
./bbx-cli rabbit receive -t queue1 -c 3 -s localhost -p 5672
```

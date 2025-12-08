# MQTT 命令

MQTT命令用于与MQTT服务器进行交互，支持发布和订阅消息。

## 命令格式

```
bbx-cli mqtt [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | MQTT服务器地址 | localhost |
| `--port` | `-p` | MQTT服务器端口 | 1883 |
| `--client-id` | `-c` | 客户端ID | 自动生成 |
| `--username` | `-u` | MQTT用户名 |  |
| `--password` | `-P` | MQTT密码 |  |
| `--keepalive` | `-k` | Keepalive时间（秒） | 60 |
| `--clean` | `-C` | Clean session标志 | true |

## 命令列表

### publish - 发布消息

向MQTT主题发布消息。

**用法：**
```
bbx-cli mqtt publish [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-m, --message string`：要发布的消息
- `-q, --qos int`：消息QoS级别（0/1/2）
- `-r, --retained`：保留消息标志

### subscribe - 订阅消息

订阅MQTT主题并接收消息。

**用法：**
```
bbx-cli mqtt subscribe [flags]
```

**参数：**
- `-t, --topic string`：主题名称
- `-q, --qos int`：订阅QoS级别（0/1/2）
- `-c, --count int`：要接收的消息数量（默认：1）

## 示例

```bash
# 发布消息到主题
./bbx-cli mqtt publish -t topic1 -m "hello mqtt" -q 1 -s localhost -p 1883

# 订阅主题接收消息
./bbx-cli mqtt subscribe -t topic1 -q 0 -c 5 -s localhost -p 1883
```

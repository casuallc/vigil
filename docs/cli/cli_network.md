# Network 命令文档

## 概述

`network` 命令组用于网络诊断和探测，支持对指定目标 IP 和端口进行连通性测试并测量延迟。

## 命令结构

```
bbx-cli network [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `probe` | 探测目标 IP 和端口的连通性 |

## 命令详情

### network probe

探测指定目标 IP 和端口的网络连通性，返回是否可达以及连接延迟。

**语法：**

```
bbx-cli network probe --target <ip> --port <port> [--protocol <protocol>] [--timeout <ms>]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--target` | `-t` | 目标 IP 地址或主机名 | 是 | - |
| `--port` | `-p` | 目标端口号 | 是 | - |
| `--protocol` | - | 协议类型（tcp、udp） | 否 | `tcp` |
| `--timeout` | - | 超时时间（毫秒） | 否 | `5000` |

**示例：**

```bash
# 探测 SSH 端口连通性
./bbx-cli network probe -t 192.168.1.100 -p 22

# 探测 HTTP 端口
./bbx-cli network probe --target example.com --port 80

# 使用 UDP 协议探测 DNS 端口
./bbx-cli network probe -t 8.8.8.8 -p 53 --protocol udp

# 设置超时时间为 3 秒
./bbx-cli network probe -t 192.168.1.100 -p 3306 --timeout 3000

# 指定服务器地址
./bbx-cli network probe -t 192.168.1.100 -p 22 -H http://10.0.0.1:57575
```

**输出示例**（可达）：

```
Probe successful: 192.168.1.100:22 is reachable (latency: 12ms)
```

**输出示例**（不可达）：

```
Probe failed: 192.168.1.100:22 is unreachable (error: dial tcp 192.168.1.100:22: connect: connection refused)
```

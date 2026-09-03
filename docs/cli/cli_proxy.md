# Proxy 命令文档

## 概述

`proxy` 命令组用于管理 bbx-server 上的 HTTP 反向代理实例：创建、查看、启停和删除。通过 API 创建的实例会持久化到服务器数据库，重启后自动恢复运行状态。

代理目标受白名单管控（CIDR / 域名后缀 / 精确主机），默认拒绝一切未授权目标；回环、私网、链路本地地址需显式 `--allow-private`；云元数据地址（`169.254.169.254`、`metadata.google.internal`）永远拒绝。详见 [proxy 功能文档](../proxy.md)。

## 命令结构

```
bbx-cli proxy [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `add` | 创建代理实例 |
| `list` | 列出所有代理实例 |
| `get` | 查看实例配置与状态 |
| `status` | 查看实例运行状态与统计 |
| `start` | 启动实例 |
| `stop` | 停止实例 |
| `delete` | 删除实例 |

## 命令详情

### proxy add

创建一个反向代理实例，将 `listen` 地址上的 HTTP 请求转发到 `target`。

**语法：**

```
bbx-cli proxy add --name <name> --listen <addr> --target <url> [flags]
```

**参数：**

| 参数 | 描述 | 必填 | 默认值 |
|------|------|------|--------|
| `--name` | 实例名称（唯一） | 是 | - |
| `--listen` | 监听地址，如 `127.0.0.1:8080` | 是 | - |
| `--target` | 上游目标，如 `http://10.0.0.5:9000` | 是 | - |
| `--whitelist` | 允许的目标条目（可重复）：CIDR `10.0.0.0/8`、后缀 `*.corp.local`、主机 `db01[:5432]` | 否 | 仅 target 主机 |
| `--allow-private` | 允许回环/私网/链路本地目标 | 否 | `false` |
| `--max-body-mb` | 请求体上限（MB），0 为不限 | 否 | `0` |
| `--header` | 注入上游请求的额外头，`Key:Value`（可重复） | 否 | - |
| `--start` | 创建后立即启动 | 否 | `false` |
| `--tls` | 监听侧启用 TLS 终止 | 否 | `false` |
| `--cert` | TLS 证书路径（配合 `--tls`） | 否 | - |
| `--key` | TLS 私钥路径（配合 `--tls`） | 否 | - |

**示例：**

```bash
# 最简：转发到内网 Web 服务
./bbx-cli proxy add --name web --listen 127.0.0.1:8080 --target http://10.0.0.5:9000 --allow-private

# 带白名单与自定义头，并立即启动
./bbx-cli proxy add --name api --listen :8081 \
  --target http://10.0.0.6:8080 --whitelist 10.0.0.0/8 \
  --header 'X-Forwarded-By:vigil' --allow-private --start

# 监听侧 TLS 终止
./bbx-cli proxy add --name secure --listen :8443 --target http://10.0.0.7:9000 \
  --tls --cert /etc/vigil/cert.pem --key /etc/vigil/key.pem --allow-private

# 指定服务器地址
./bbx-cli proxy add --name web --listen :8080 --target http://10.0.0.5:9000 \
  --allow-private -H http://10.0.0.1:57575
```

**输出示例：**

```
Proxy instance web created (running): 127.0.0.1:8080 -> http://10.0.0.5:9000
```

### proxy list

以表格列出所有代理实例及其状态、来源（config/api）、流量统计。

**语法：**

```
bbx-cli proxy list
```

**输出示例：**

```
┌─────────────┬─────────┬────────┬──────────────────┬────────────────────────┬──────────┬───────────┐
│ NAME        │ STATE   │ ORIGIN │ LISTEN           │ TARGET                 │ REQUESTS │ BYTES OUT │
├─────────────┼─────────┼────────┼──────────────────┼────────────────────────┼──────────┼───────────┤
│ web         │ running │ api    │ 127.0.0.1:8080   │ http://10.0.0.5:9000   │ 128      │ 34012     │
│ static-web  │ running │ config │ 127.0.0.1:18080  │ http://127.0.0.1:19001 │ 3        │ 57        │
└─────────────┴─────────┴────────┴──────────────────┴────────────────────────┴──────────┴───────────┘
```

### proxy get / status

查看单个实例的完整配置、运行状态、请求统计和最近一次错误。

**语法：**

```
bbx-cli proxy get <name>
bbx-cli proxy status <name>
```

**输出示例：**

```
Name:      web
State:     running
Origin:    api
Listen:    127.0.0.1:8080
Target:    http://10.0.0.5:9000
Whitelist: 10.0.0.0/8
Requests:  128 (upstream errors: 0)
Traffic:   in=4096B out=34012B active=1
StartedAt: 2026-09-03 13:02:37
```

### proxy start / stop

启动或停止一个实例。停止只是关闭监听器，配置仍保留；`start`/`stop` 会更新持久化的期望状态，服务器重启后按该状态恢复。

**语法：**

```
bbx-cli proxy start <name>
bbx-cli proxy stop <name>
```

**输出示例：**

```
Proxy instance web: running
```

### proxy delete

删除一个 API 创建的实例（先停止再删除）。**配置文件（`proxy.instances`）中定义的实例不能通过此命令删除**，会返回 409 冲突，请改为编辑配置文件。

**语法：**

```
bbx-cli proxy delete <name>
```

**输出示例：**

```
Proxy instance web deleted
```

## 注意事项

- 所有命令经由 `/api/proxy/instances` REST API 操作，服务器需开启 `proxy.enabled: true`，否则返回 503；
- 管理接口走全局 Basic Auth 与审计，被转发流量的明细记录在服务器 `logs/proxy/` 目录；
- 反向代理实例原生支持 WebSocket 透传，无需额外配置；
- 无公网入口的内网服务器可结合 poll 模式的 `proxy_session` 隧道对外提供反向访问，见 [poll-mode 文档](../poll-mode.md)。

# HTTP 代理（proxy）

bbx-server 内置的 HTTP 代理功能，每个实例有两种模式：

- **反向代理**（`mode: reverse`，默认）：监听地址固定转发到一个上游 target，等价于 nginx 的 `proxy_pass`；
- **正向代理**（`mode: forward`）：经典正向代理，客户端在请求里指定目的地（绝对 URI 或 `CONNECT`），白名单逐目标管控。

支持三种使用方式：

1. **静态实例**：写在 `conf/config.yaml` 的 `proxy.instances`，随服务启动；
2. **动态实例**：通过 REST API / `bbx-cli proxy` 创建，持久化到 SQLite，重启后自动恢复 `running` 状态的实例；
3. **poll 隧道**：无公网入口的内网 bbx 通过 poll 模式承接上游下发的 `proxy_session` 任务，把外部 HTTP 请求经 WebSocket 隧道反代到内网目标（见 [poll-mode.md](poll-mode.md) 的 proxy_session 章节）。

反代基于标准库 `httputil.ReverseProxy`，原生支持 WebSocket upgrade 透传。

## 配置

```yaml
proxy:
  enabled: true
  instances:
    - name: web-internal            # 反向代理实例
      listen: 127.0.0.1:8080        # 监听地址
      target: http://10.0.0.5:9000  # 上游目标，http(s)://host[:port]
      whitelist: ["10.0.0.0/8"]     # 目标白名单（见下）
      allow_private: true           # 允许回环/私网/链路本地目标（SSRF 防护，默认 false）
      max_body_mb: 0                # 请求体上限 MB，0 = 不限
      header_set: {X-Real-IP: ""}   # 注入到上游请求的额外头
      tls: {enabled: false, cert_path: "", key_path: ""}  # 监听侧 TLS 终止
    - name: egress                  # 正向代理实例
      mode: forward
      listen: 127.0.0.1:3128
      whitelist: ["*.corp.local", "10.0.0.0/8"]  # 正向模式必填，管控每个客户端指定的目的地
      allow_private: true
  tunnel:                           # poll 隧道策略
    enabled: false
    allowed_targets: []             # 隧道目标白名单（本机策略，上游不可扩大）；空 = 禁止隧道
    max_sessions: 8                 # 并发隧道会话上限
    max_duration_sec: 3600          # 单会话硬上限
    max_body_mb: 64
```

## 正向代理（mode: forward）

客户端把实例当作标准 HTTP 代理使用：

- **普通 HTTP**：请求行为绝对 URI（`GET http://目标/path`），代理校验白名单后转发；
- **HTTPS 及其他协议**：`CONNECT 目标:443`，代理建立到目标的 TCP 隧道后双向透传原始字节；
- **认证**：客户端必须通过标准 `Proxy-Authorization: Basic ...` 头提供 **config.yaml 里 `auth.username` / `auth.password` 超管账号**（仅校验超管，不查用户库），失败返回 407；`auth.enabled: false` 时正向实例拒绝一切请求（fail closed）；
- **白名单必填**：正向实例的目标由客户端指定，白名单是唯一授权边界，空白名单 = 拒绝一切（创建时直接报错）；
- `target` / `header_set` 在正向模式下不可用；`max_body_mb` 只约束普通 HTTP 请求体，`CONNECT` 隧道是原始字节流不受其限。

用法示例（curl 经正向代理访问，`-x` 指定代理地址，凭据放代理 URL 里）：

```bash
curl -x http://admin:password@127.0.0.1:3128 http://internal.corp/        # 普通转发
curl -x http://admin:password@127.0.0.1:3128 https://10.0.0.5:8443/ -k     # 自动走 CONNECT
```

正向实例同样支持 `--tls` 监听侧加密（客户端与代理之间 TLS）。

## 白名单语法与 SSRF 防护

白名单条目按字符串形态自动判别：

| 形态 | 例子 | 说明 |
|---|---|---|
| CIDR | `10.0.0.0/8` | 匹配解析后的 IP |
| 域名后缀 | `*.internal.corp` | 匹配该域名及其子域 |
| 精确主机 | `db01`、`db01:5432`、`1.2.3.4` | 不带端口则任意端口 |

规则要点：

- **默认拒绝**：反向模式白名单为空时仅允许 target 主机本身；正向模式白名单必填；隧道模式下空 `allowed_targets` = 完全禁止；
- **私网防护**：`allow_private: false`（默认）时回环、RFC1918、链路本地、ULA 地址一律拒绝，即使命中白名单条目；
- **云元数据永远拒绝**：`169.254.169.254` 与 `metadata.google.internal` 不受任何配置影响；
- **DNS rebinding 缓解**：连接建立后对实际连接的 IP 再校验一次；白名单全部为 CIDR 时，连接 IP 必须落在 CIDR 内。这只能缓解不能根除（TOCTOU），高安全场景建议白名单只使用 CIDR。

## REST API

所有端点位于 `/api/proxy/instances`，走全局 Basic Auth 与审计：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/proxy/instances` | 列出全部实例（含状态与统计） |
| POST | `/api/proxy/instances` | 创建，body `{"config": {...}, "autostart": true}` |
| GET | `/api/proxy/instances/{name}` | 查询单个实例 |
| PUT | `/api/proxy/instances/{name}` | 更新配置（运行中的实例会平滑重启） |
| DELETE | `/api/proxy/instances/{name}` | 删除（config 定义的实例返回 409） |
| POST | `/api/proxy/instances/{name}/start` | 启动 |
| POST | `/api/proxy/instances/{name}/stop` | 停止 |
| GET | `/api/proxy/instances/{name}/status` | 运行状态与统计 |

`proxy.enabled: false` 时上述端点不注册（404）；管理操作在功能未启用时返回 503。

## CLI

```bash
# 反向代理：创建并立即启动
bbx-cli proxy add --name web --listen 127.0.0.1:8080 \
  --target http://10.0.0.5:9000 --whitelist 10.0.0.0/8 \
  --allow-private --header 'X-Real-IP:' --start

# 正向代理：白名单必填，客户端用超管账号认证
bbx-cli proxy add --name egress --mode forward --listen 127.0.0.1:3128 \
  --whitelist '*.corp.local' --whitelist 10.0.0.0/8 --allow-private --start

bbx-cli proxy list            # 表格列出实例
bbx-cli proxy status web      # 状态与统计
bbx-cli proxy stop web
bbx-cli proxy start web
bbx-cli proxy delete web
```

## 审计与访问日志

- **管控操作**（create/update/delete/start/stop/list/get）由全局 `LoggingMiddleware` 记录到 `logs/audit/`；
- **白名单拒绝**记 `proxy_denied` 审计条目（正向模式记客户端请求的方法与目标）；
- **每条被转发请求的明细不进审计主日志**（量太大），写入 `logs/proxy/<instance>_YYYY-MM-DD.log`（JSON 行：client_ip、method、path、status、bytes_out、duration_ms、via=`listen|forward|tunnel`）；正向模式的 407 认证失败也记在这里。

## 实现说明

- 每个实例独立的 `http.Server` + listener，**不经过** API server 的 `LoggingMiddleware`（它会缓冲整个请求/响应体，大流量反代不可接受）；审计通过 proxy 自己的 `AccessHook` 完成；
- 实例的 WebSocket 升级由 `httputil.ReverseProxy` 原生处理；
- 动态实例持久化在共享 SQLite 库（`data/vigil.db` 的 `proxy_instances` 表），记录 `origin`（config/api）与 `desired_state`；重启时 config 实例覆盖同名 DB 记录，API 实例按 `desired_state=running` 恢复；
- config 定义的实例不能通过 API 删除（409），同名冲突时 config 优先。

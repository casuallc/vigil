# Poll Mode（出方向受限网络下的任务拉取模式）

当网络环境只允许出方向请求时，bbx-server 可开启 poll 模式：主动向多个相互独立的上游服务长轮询拉取任务，本地执行后回执结果。设计细节见 `tmp/poll-mode-design.md`。

## 配置（conf/config.yaml）

```yaml
poll:
  enabled: true
  agent_id: ""                        # 默认 hostname
  defaults:
    long_poll_wait: 25s               # 长轮询 hold 时长（< 网关空闲超时）
    busy_interval: 500ms              # BUSY 态拉取间隔
    idle_backoff_max: 10s             # 空闲/错误退避封顶
    busy_to_idle_empty_polls: 3       # 连续空拉次数，达到后 BUSY→IDLE
    task_timeout: 120s                # 任务级超时（任务可用 timeout_sec 覆盖）
    max_topics: 32                    # topic 队列上限，超出落 default 兜底队列
    queue_buffer: 64                  # 每 topic 队列缓冲，满则阻塞派发（反压）
    topic_idle_ttl: 10m               # topic 队列空闲回收时间
    tls:
      insecure_skip_verify: false
      ca_file: ""                     # 私有 CA 证书
  upstreams:
    - name: svc-a
      endpoint: https://10.0.0.11:9000  # http:// 需显式 allow_http: true
      auth: {username: bbx, password: xxx}
    - name: svc-b
      endpoint: https://10.0.0.12:8080
      auth: {username: bbx, password: yyy}
      tls: {insecure_skip_verify: true}
```

配置中**没有 topic 相关项**：topic 由上游在任务上打标，bbx 只作为串行调度键。

## 上游协议

```
GET  {endpoint}/poll?agent=<id>&wait=25s
     → 200 {"tasks": [Task, ...], "has_more": bool}

POST {endpoint}/ack
     {"id": "...", "status": "success|failed|timeout", "result": {...},
      "exit_code": 0, "duration_ms": 1234, "error": "..."}
```

- 任务从哪个上游拉到，ack 回哪个上游；任务可携带 `ack_url` 覆盖回执地址；
- ack 内嵌结果上限 64KB，超出截断并标记 `truncated: true`；
- bbx 按任务 `id` 幂等去重（每上游缓存最近 1024 条结果），上游重派时直接回缓存结果。

## 任务模型

```json
{
  "id": "task-uuid",
  "topic": "docker",
  "timeout_sec": 120,
  "action": { "...": "按 type 而定" },
  "ack_url": "",
  "created_at": "..."
}
```

### 任务类型

| type | 说明 | 数据面 | ack 内容 |
|---|---|---|---|
| `api`（默认） | 本地 API 回环调用，复用现有全部 handler | 无 | 内嵌结果（≤64KB） |
| `push_file` | 把本地文件流式推送到任务给定的 `push_url` | bbx → push_url | size / sha256 |
| `tail_file` | tail 本地日志，持续推增量到 `push_url`（chunked POST） | bbx → push_url | lines / bytes / end_reason |
| `ws_bridge` | bbx 主动外拨 WS（`connect_url`）并与本地 WS handler 桥接 | bbx 外拨 WS | duration_ms / end_reason |

`api` 示例：`{"type": "api", "method": "POST", "path": "/api/process/scan", "body": {"query": "java"}}`

回环调用使用 server 启动时生成的内部 token（`X-Vigil-Internal-Token` 头，仅 loopback 来源生效），不使用任何上游凭据。

## 执行模型

- 每个 topic 一条有界队列 + 一个专属 worker：**topic 内严格串行，topic 间并行**；
- 队列惰性创建，空闲超过 `topic_idle_ttl` 自动回收；
- 队列满时 poller 阻塞在派发上 → 停止拉新，反压自然传导回上游，任务不丢；
- 上游连接失败指数退避（封顶 `idle_backoff_max`），各上游互不影响；
- 优雅退出：停止入队，worker drain 10s，未开始的任务 NACK 回上游。

## 代码结构

```
poll/
  config.go      # PollConfig / UpstreamConfig，挂到 conf/config.yaml 的 poll 段
  upstream.go    # Agent + 每上游一个 poller：busy/idle 状态机、http.Client（auth/tls 装配）
  dispatcher.go  # topic → 队列 → 单 worker；惰性创建、上限、反压、回收
  executor.go    # api 回环调用 / push_file / tail_file / ws_bridge 四种执行器
  dedup.go       # 任务 id 幂等去重
```

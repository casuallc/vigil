# Docker 容器管理 API

Docker 管理是 `bbx-server` 的子功能，基于官方 Docker SDK 管理本地 Docker Daemon 上的容器，提供 REST 与 WebSocket 接口。

> 启用方式：默认启用（`conf/config.yaml` 中 `docker.enabled` 为 `true` 或省略）。若启动时无法连接 Docker Daemon，则相关接口返回 `503 docker manager not initialized`；显式设置 `docker.enabled: false` 可关闭。

## 认证

这些接口复用 **vigil 全局 Basic Auth**（`conf/config.yaml` 的 `auth`，即超管凭据或用户库中的注册用户），与其它 API 一致。

```
Authorization: Basic base64(username:password)
```

未携带或凭据错误返回 `401`，响应头带 `WWW-Authenticate: Basic realm="vigil"`。

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/docker/compose | POST | 部署 compose 项目 |
| /api/docker/compose/{project} | GET | 查看 compose 项目状态 |
| /api/docker/compose/{project} | DELETE | 删除 compose 项目 |
| /api/docker/ping | GET | 探测 Docker Daemon 连通性 |
| /api/docker/containers | GET | 列出容器 |
| /api/docker/containers | POST | 创建容器 |
| /api/docker/containers/{id} | GET | 查看容器详情 |
| /api/docker/containers/{id} | DELETE | 删除容器 |
| /api/docker/containers/{id}/start | POST | 启动容器 |
| /api/docker/containers/{id}/stop | POST | 停止容器 |
| /api/docker/containers/{id}/restart | POST | 重启容器 |
| /api/docker/containers/{id}/pause | POST | 暂停容器 |
| /api/docker/containers/{id}/unpause | POST | 恢复容器 |
| /api/docker/containers/{id}/exec | POST | 在容器内执行一次性命令 |
| /api/docker/containers/{id}/logs | GET | 流式获取容器日志 |
| /api/docker/containers/{id}/stats | GET | 流式获取容器指标 |
| /api/docker/containers/{id}/exec/ws | WebSocket | 交互式 exec |
| /api/docker/containers/{id}/logs/ws | WebSocket | 交互式日志流 |

> 响应体采用 vigil 原生裸 JSON（非 `{code,msg,data}` 包裹），错误统一为 `{"error": "..."}`。

---

## GET /api/docker/ping

**功能描述**：探测 Docker Daemon 是否可达。

**响应格式**：

```json
{
  "api_version": "1.47",
  "version": "27.5.1",
  "experimental": false
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/ping
```

---

## GET /api/docker/containers

**功能描述**：列出容器。

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| all | boolean | 否 | 是否包含已停止容器，默认 `false` |

**响应格式**：`[]ContainerSummary`

```json
[
  {
    "id": "a1b2c3d4e5f6...",
    "names": ["/my-app"],
    "image": "nginx:alpine",
    "command": "nginx -g 'daemon off;'",
    "created": 1751270400,
    "status": "Up 2 hours",
    "state": "running",
    "ports": [
      { "ip": "0.0.0.0", "private_port": 80, "public_port": 8080, "type": "tcp" }
    ],
    "labels": {}
  }
]
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  "http://localhost:57575/api/docker/containers?all=true"
```

---

## POST /api/docker/containers

**功能描述**：基于镜像创建容器（不自动启动）。

**请求体**：`CreateContainerRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| name | string | 否 | 容器名称 |
| image | string | 是 | 镜像，如 `nginx:alpine` |
| cmd | string[] | 否 | 启动命令 |
| env | string[] | 否 | 环境变量，如 `["FOO=bar"]` |
| ports | map[string]string | 否 | 端口映射，`{"容器端口": "主机端口"}`，如 `{"80": "8080"}` 或 `{"80": "127.0.0.1:8080"}` |

**请求示例**：

```json
{
  "name": "my-nginx",
  "image": "nginx:alpine",
  "cmd": ["nginx", "-g", "daemon off;"],
  "env": ["FOO=bar"],
  "ports": { "80": "8080" }
}
```

**响应格式**：

```json
{
  "id": "a1b2c3d4e5f6..."
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"my-nginx","image":"nginx:alpine","ports":{"80":"8080"}}' \
  http://localhost:57575/api/docker/containers
```

---

## GET /api/docker/containers/{id}

**功能描述**：查看容器详情。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 容器 ID 或名称 |

**响应格式**：Docker SDK 的 `ContainerJSON`，包含 `Config`、`HostConfig`、`NetworkSettings`、`State` 等完整字段。

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/containers/a1b2c3d4e5f6
```

---

## DELETE /api/docker/containers/{id}

**功能描述**：删除容器。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 容器 ID 或名称 |

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| force | boolean | 否 | 是否强制删除运行中容器，默认 `false` |

**响应格式**：

```json
{
  "status": "removed"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X DELETE \
  "http://localhost:57575/api/docker/containers/a1b2c3d4e5f6?force=true"
```

---

## POST /api/docker/containers/{id}/start

**功能描述**：启动容器。

**响应格式**：

```json
{
  "status": "started"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  http://localhost:57575/api/docker/containers/a1b2c3d4e5f6/start
```

---

## POST /api/docker/containers/{id}/stop

**功能描述**：停止容器。

**请求体**：

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| timeout | int | 否 | 优雅停止等待秒数 |

**响应格式**：

```json
{
  "status": "stopped"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"timeout":30}' \
  http://localhost:57575/api/docker/containers/a1b2c3d4e5f6/stop
```

---

## POST /api/docker/containers/{id}/restart

**功能描述**：重启容器。

**请求体**：

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| timeout | int | 否 | 优雅停止等待秒数 |

**响应格式**：

```json
{
  "status": "restarted"
}
```

---

## POST /api/docker/containers/{id}/pause

**功能描述**：暂停容器。

**响应格式**：

```json
{
  "status": "paused"
}
```

---

## POST /api/docker/containers/{id}/unpause

**功能描述**：恢复容器。

**响应格式**：

```json
{
  "status": "unpaused"
}
```

---

## POST /api/docker/containers/{id}/exec

**功能描述**：在容器内执行一次性命令，返回合并后的 stdout/stderr。

**请求体**：

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| command | string | 是 | shell 命令 |
| tty | boolean | 否 | 是否分配 TTY，默认 `false` |

**响应格式**：`text/plain; charset=utf-8`

```
root@a1b2c3d4:/# ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 nginx -g daemon off;
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"command":"ps aux"}' \
  http://localhost:57575/api/docker/containers/a1b2c3d4e5f6/exec
```

---

## GET /api/docker/containers/{id}/logs

**功能描述**：流式获取容器日志。

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| follow | boolean | 否 | 是否持续跟踪新日志，默认 `false` |
| tail | string | 否 | 返回最近若干条，如 `"100"` |
| since | string | 否 | 时间戳或相对时间，如 `"2025-07-01T00:00:00Z"` |

**响应格式**：`text/plain; charset=utf-8`，带时间戳。

**cURL 示例**：

```bash
curl -u <username>:<password> \
  "http://localhost:57575/api/docker/containers/a1b2c3d4e5f6/logs?follow=true&tail=100"
```

---

## GET /api/docker/containers/{id}/stats

**功能描述**：流式获取容器指标。

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| stream | boolean | 否 | 是否持续流式输出，默认 `true` |

**响应格式**：`application/x-ndjson`，每行一个 JSON 指标对象。

**cURL 示例**：

```bash
curl -u <username>:<password> \
  "http://localhost:57575/api/docker/containers/a1b2c3d4e5f6/stats?stream=false"
```

---

## WebSocket /api/docker/containers/{id}/exec/ws

**功能描述**：在容器内建立交互式 exec 会话。

**连接流程**：

1. 客户端通过 WebSocket 协议连接此端点；
2. 首条消息必须是 JSON：
   ```json
   {
     "command": "sh",
     "tty": true,
     "width": 120,
     "height": 40
   }
   ```
3. 后续客户端文本消息作为 stdin 写入容器；
4. 服务端二进制消息为容器 stdout/stderr；
5. 支持终端大小调整：客户端发送 `resize:{"cols":120,"rows":40}`。

---

## WebSocket /api/docker/containers/{id}/logs/ws

**功能描述**：通过 WebSocket 持续接收容器日志。

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| tail | string | 否 | 返回最近若干条 |
| since | string | 否 | 起始时间 |

**连接流程**：

1. 客户端通过 WebSocket 协议连接此端点；
2. 服务端持续以文本消息推送日志；
3. 客户端关闭连接或发送任意消息即可停止。

---

## Docker Compose 部署

vigil 提供无状态的 Docker Compose 部署能力：上传 `docker-compose.yml` 内容后，服务端解析并按项目创建容器、网络和卷。项目归属通过容器标签 `com.docker.compose.project` 和 `com.docker.compose.service` 实时查询，不做额外持久化。

### 支持的 Compose 字段

| 字段 | 说明 |
|------|------|
| `services` | 服务定义（必填） |
| `services.<name>.image` | 镜像（必填） |
| `services.<name>.command` | 启动命令，支持字符串或列表 |
| `services.<name>.environment` | 环境变量，支持 map 或 `KEY=VALUE` 列表 |
| `services.<name>.ports` | 端口映射，如 `"8080:80"` |
| `services.<name>.volumes` | 卷挂载，支持短语法或对象 |
| `services.<name>.networks` | 所属网络，支持列表或 map |
| `services.<name>.restart` | 重启策略：`no`、`always`、`unless-stopped`、`on-failure[:N]` |
| `services.<name>.labels` | 标签，支持 map 或列表 |
| `services.<name>.replicas` / `services.<name>.deploy.replicas` | 副本数 |
| `networks` | 顶层网络定义 |
| `volumes` | 顶层卷定义 |

暂不支持 `build`、`extends`、`profiles`、`secrets`、`configs`、健康检查等高级特性。

---

## POST /api/docker/compose

**功能描述**：基于 docker-compose.yml 内容部署一个项目。

**请求体**：`ComposeDeployRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| name | string | 是 | 项目名称，只能包含小写字母、数字、下划线和连字符 |
| content | string | 是 | compose YAML 内容 |
| start | boolean | 否 | 是否自动启动容器，默认 `true` |

**请求示例**：

```json
{
  "name": "demo",
  "content": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n"
}
```

**响应格式**：`ComposeProjectStatus`

```json
{
  "name": "demo",
  "services": [
    {
      "name": "web",
      "image": "nginx:alpine",
      "replicas": 1,
      "containers": [
        {
          "id": "a1b2c3d4e5f6...",
          "names": ["/demo_web_1"],
          "image": "nginx:alpine",
          "status": "Up 2 seconds",
          "state": "running",
          "ports": [{"ip": "0.0.0.0", "private_port": 80, "public_port": 8080, "type": "tcp"}],
          "labels": {
            "com.docker.compose.project": "demo",
            "com.docker.compose.service": "web"
          }
        }
      ]
    }
  ]
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"demo","content":"services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n"}' \
  http://localhost:57575/api/docker/compose
```

**错误码**：

- `409 Conflict`：项目已存在（已有同名项目容器）。

---

## GET /api/docker/compose/{project}

**功能描述**：查看指定 compose 项目的容器状态。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| project | string | 项目名称 |

**响应格式**：`ComposeProjectStatus`

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/compose/demo
```

---

## DELETE /api/docker/compose/{project}

**功能描述**：停止并删除项目下的所有容器、项目创建的网络和卷。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| project | string | 项目名称 |

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| force | boolean | 否 | 是否强制删除运行中容器，默认 `false` |

**响应格式**：

```json
{
  "status": "removed"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X DELETE \
  "http://localhost:57575/api/docker/compose/demo?force=true"
```

---

## 配置

`conf/config.yaml`：

```yaml
# Docker 管理配置。默认启用（省略或 true）；false 显式关闭。
docker:
  enabled: true
  host: ""                              # 可选 Docker daemon host 覆盖，如 "tcp://192.168.1.10:2376"
```

Docker 客户端初始化遵循标准环境变量：`DOCKER_HOST`、`DOCKER_TLS_VERIFY`、`DOCKER_CERT_PATH`。`docker.host` 若设置则优先于环境变量。

## 权限说明

v1 仅暴露容器生命周期与执行相关操作，不暴露以下敏感能力：

- 镜像构建/推送/拉取
- 卷/网络管理
- Daemon 配置修改
- 特权容器创建

所有操作均经过全局 Basic Auth 与审计日志记录。

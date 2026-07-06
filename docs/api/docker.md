# Docker 容器与镜像管理 API

Docker 管理是 `bbx-server` 的子功能，基于官方 Docker SDK 管理本地 Docker Daemon 上的容器与镜像，提供 REST 与 WebSocket 接口。

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
| /api/docker/compose/dir | POST | 从服务器本地目录部署 compose 项目 |
| /api/docker/compose/{project} | GET | 查看 compose 项目状态 |
| /api/docker/compose/{project} | DELETE | 删除 compose 项目 |
| /api/docker/compose-version | GET | 获取 docker-compose 版本信息 |
| /api/docker/ping | GET | 探测 Docker Daemon 连通性 |
| /api/docker/version | GET | 获取 Docker Daemon 版本信息 |
| /api/docker/images/load | POST | 异步下载并加载 Docker tar 镜像包 |
| /api/docker/images/load/{id}/status | GET | 查询镜像加载任务状态 |
| /api/docker/images | GET | 列出本地镜像 |
| /api/docker/images/{id} | GET | 查看镜像详情 |
| /api/docker/images/{id} | DELETE | 删除镜像 |
| /api/docker/images/{id}/history | GET | 查看镜像历史 |
| /api/docker/images/pull | POST | 拉取镜像 |
| /api/docker/images/tag | POST | 给镜像打标签 |
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
  "APIVersion": "1.44",
  "OSType": "linux",
  "Experimental": true,
  "BuilderVersion": "2",
  "SwarmStatus": {
    "NodeState": "inactive",
    "ControlAvailable": false
  }
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/ping
```

---

## GET /api/docker/version

**功能描述**：获取 Docker Daemon 的版本信息。

**响应格式**：Docker SDK 的 `Version`，包含 `Version`、`APIVersion`、`MinAPIVersion`、`GitCommit`、`GoVersion`、`Os`、`Arch`、`KernelVersion`、`BuildTime` 等字段。

```json
{
  "Version": "24.0.7",
  "APIVersion": "1.43",
  "MinAPIVersion": "1.12",
  "GitCommit": "afdd53b",
  "GoVersion": "go1.20.7",
  "Os": "linux",
  "Arch": "amd64",
  "KernelVersion": "5.15.0",
  "BuildTime": "2023-10-26T09:44:52.000000000+00:00"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/version
```

**状态码**：

- `200 OK`：成功
- `503 Service Unavailable`：Docker manager 未初始化

---

## POST /api/docker/images/load

**功能描述**：从指定 URL 下载 Docker tar 镜像包，并异步加载到本地 Docker daemon。若请求中提供了 `metadata.name` 与 `metadata.tag`，加载成功后还会调用 `docker tag` 将第一个加载的镜像重命名为该名称。

**请求体**：`LoadImageRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| url | string | 是 | Docker tar 包下载地址，仅支持 `http`/`https` |
| metadata.name | string | 否 | 镜像名称，与 `tag` 同时提供时会覆盖 tar 包内第一个镜像的标签 |
| metadata.tag | string | 否 | 镜像标签，与 `name` 同时提供时会覆盖 tar 包内第一个镜像的标签 |
| metadata.platform | string | 否 | 平台，如 `linux/amd64`（当前仅记录） |
| metadata.size | int64 | 否 | 期望文件大小（字节），非零时校验下载大小 |
| metadata.sha256 | string | 否 | 期望 SHA256 校验和，非空时校验文件 |
| metadata.labels | map[string]string | 否 | 附加标签（当前仅记录） |

**请求示例**：

```json
{
  "url": "https://example.com/app-v1.0.tar",
  "metadata": {
    "name": "myapp",
    "tag": "v1.0",
    "sha256": "abc123..."
  }
}
```

**响应格式**：

```json
{
  "task_id": "task_1751270400000000000"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/app-v1.0.tar","metadata":{"name":"myapp","tag":"v1.0"}}' \
  http://localhost:57575/api/docker/images/load
```

**状态码**：

- `202 Accepted`：任务已创建
- `400 Bad Request`：请求体非法或 `url` 为空
- `503 Service Unavailable`：Docker manager 未初始化

---

## GET /api/docker/images/load/{id}/status

**功能描述**：查询镜像加载任务的当前状态。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 任务 ID，即 `task_id` |

**响应格式**：`LoadImageTask`

| 字段 | 类型 | 描述 |
|------|------|------|
| id | string | 任务 ID |
| url | string | 下载 URL |
| metadata | ImageMetadata | 提交时的元数据 |
| state | string | `pending` / `downloading` / `loading` / `success` / `failed` |
| images | string[] | 成功加载或重命名后的镜像引用 |
| error_msg | string | `failed` 状态下的错误信息 |
| created_at | string | 创建时间（RFC3339） |
| updated_at | string | 最后更新时间（RFC3339） |

**响应示例（进行中）**：

```json
{
  "id": "task_1751270400000000000",
  "url": "https://example.com/app-v1.0.tar",
  "metadata": {"name":"myapp","tag":"v1.0"},
  "state": "downloading",
  "images": [],
  "created_at": "2025-07-05T10:00:00Z",
  "updated_at": "2025-07-05T10:00:01Z"
}
```

**响应示例（成功）**：

```json
{
  "id": "task_1751270400000000000",
  "url": "https://example.com/app-v1.0.tar",
  "metadata": {"name":"myapp","tag":"v1.0"},
  "state": "success",
  "images": ["origin:latest", "myapp:v1.0"],
  "created_at": "2025-07-05T10:00:00Z",
  "updated_at": "2025-07-05T10:00:05Z"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/images/load/task_1751270400000000000/status
```

**状态码**：

- `200 OK`：任务存在
- `404 Not Found`：任务不存在
- `503 Service Unavailable`：Docker manager 未初始化

---

## GET /api/docker/images

**功能描述**：列出本地镜像。

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| all | boolean | 否 | 是否包含中间层镜像，默认 `false` |
| dangling | boolean | 否 | 只显示悬空镜像，默认 `false` |
| label | string[] | 否 | 按标签过滤，如 `label=app=web`，可重复 |
| filter | string | 否 | 原始 JSON 过滤条件，会覆盖 `dangling`/`label` |

**响应格式**：`[]ImageSummary`

```json
[
  {
    "id": "sha256:abc123...",
    "containers": 2,
    "created": 1751270400,
    "labels": {},
    "parent_id": "sha256:parent...",
    "repo_digests": ["nginx@sha256:def..."],
    "repo_tags": ["nginx:alpine"],
    "shared_size": 0,
    "size": 43212800,
    "virtual_size": 43212800
  }
]
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  "http://localhost:57575/api/docker/images?all=true&dangling=true&label=app=web"
```

**状态码**：

- `200 OK`：成功
- `400 Bad Request`：`filter` JSON 格式非法
- `503 Service Unavailable`：Docker manager 未初始化

---

## GET /api/docker/images/{id}

**功能描述**：查看镜像详情。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 镜像 ID 或引用，如 `nginx:alpine` |

**响应格式**：Docker SDK 的 `ImageInspect`，包含 `Config`、`RootFS`、`RepoTags` 等完整字段。

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/images/nginx:alpine
```

**状态码**：

- `200 OK`：成功
- `503 Service Unavailable`：Docker manager 未初始化

---

## POST /api/docker/images/pull

**功能描述**：从镜像仓库拉取镜像到本地 Docker daemon。

**请求体**：`PullImageRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| image | string | 是 | 镜像引用，如 `nginx:alpine` |

**请求示例**：

```json
{
  "image": "alpine:latest"
}
```

**响应格式**：

```json
{
  "status": "pulled"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"image":"alpine:latest"}' \
  http://localhost:57575/api/docker/images/pull
```

**状态码**：

- `200 OK`：拉取成功
- `400 Bad Request`：请求体非法或 `image` 为空
- `503 Service Unavailable`：Docker manager 未初始化

---

## DELETE /api/docker/images/{id}

**功能描述**：删除本地镜像。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 镜像 ID 或引用 |

**查询参数**：

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| force | boolean | 否 | 是否强制删除，默认 `false` |
| noprune | boolean | 否 | 不删除未标记的父镜像，默认 `false` |

**响应格式**：`ImageRemoveResponse`

```json
{
  "deleted": [
    {"untagged": "nginx:alpine"},
    {"deleted": "sha256:abc123..."}
  ]
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X DELETE \
  "http://localhost:57575/api/docker/images/nginx:alpine?force=true"
```

**状态码**：

- `200 OK`：删除成功
- `503 Service Unavailable`：Docker manager 未初始化

---

## POST /api/docker/images/tag

**功能描述**：为已有镜像创建新标签。

**请求体**：`TagImageRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| source | string | 是 | 源镜像引用 |
| target | string | 是 | 目标标签 |

**请求示例**：

```json
{
  "source": "alpine:latest",
  "target": "myregistry/alpine:v1"
}
```

**响应格式**：

```json
{
  "status": "tagged"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"source":"alpine:latest","target":"myregistry/alpine:v1"}' \
  http://localhost:57575/api/docker/images/tag
```

**状态码**：

- `200 OK`：打标签成功
- `400 Bad Request`：`source` 或 `target` 为空
- `503 Service Unavailable`：Docker manager 未初始化

---

## GET /api/docker/images/{id}/history

**功能描述**：查看镜像构建历史。

**路径参数**：

| 参数 | 类型 | 描述 |
|------|------|------|
| id | string | 镜像 ID 或引用 |

**响应格式**：`[]HistoryResponseItem`

```json
[
  {
    "Id": "sha256:abc123...",
    "Created": 1751270400,
    "CreatedBy": "/bin/sh -c #(nop) CMD [\"nginx\"]",
    "Size": 0,
    "Comment": "",
    "Tags": ["nginx:alpine"]
  }
]
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/images/nginx:alpine/history
```

**状态码**：

- `200 OK`：成功
- `503 Service Unavailable`：Docker manager 未初始化

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

## GET /api/docker/compose-version

**功能描述**：获取本地安装的 Docker Compose 版本信息。

**响应格式**：`ComposeVersionResponse`

```json
{
  "version": "2.27.0",
  "raw_output": "2.27.0"
}
```

**cURL 示例**：

```bash
curl -u <username>:<password> \
  http://localhost:57575/api/docker/compose-version
```

**状态码**：

- `200 OK`：成功
- `500 Internal Server Error`：无法调用 `docker compose version` 或 `docker-compose version`
- `503 Service Unavailable`：Docker compose manager 未初始化

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

## POST /api/docker/compose/dir

**功能描述**：从 vigil server 本地的目录中读取 `docker-compose.yml` 并部署项目。目录必须位于 server 所在主机，且其中必须包含 `docker-compose.yml`。

**请求体**：`ComposeDeployFromDirRequest`

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| name | string | 否 | 项目名称，只能包含小写字母、数字、下划线和连字符；为空时使用目录 basename |
| dir | string | 是 | 服务器本地目录路径，该目录下需存在 `docker-compose.yml` |
| start | boolean | 否 | 是否自动启动容器，默认 `true` |

**请求示例**：

```json
{
  "name": "demo",
  "dir": "/opt/stacks/demo",
  "start": true
}
```

**响应格式**：`ComposeProjectStatus`，与 `POST /api/docker/compose` 相同。

**cURL 示例**：

```bash
curl -u <username>:<password> -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"demo","dir":"/opt/stacks/demo"}' \
  http://localhost:57575/api/docker/compose/dir
```

**状态码**：

- `201 Created`：部署成功
- `400 Bad Request`：`dir` 为空、目录不存在、路径不是目录、目录中缺少 `docker-compose.yml`，或推导的项目名非法
- `409 Conflict`：项目已存在
- `503 Service Unavailable`：Docker compose manager 未初始化

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

v1 暴露容器生命周期、执行、镜像列表/查看/拉取/删除/标签/历史以及通过 tar 包加载镜像等操作，不暴露以下敏感能力：

- 镜像构建/推送
- 卷/网络管理
- Daemon 配置修改
- 特权容器创建

所有操作均经过全局 Basic Auth 与审计日志记录。

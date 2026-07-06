# Docker 命令文档

## 概述

`docker` 命令组用于通过 vigil server 管理本地 Docker 容器、镜像和 Compose 项目，支持容器生命周期管理、镜像异步加载、交互式 exec / logs WebSocket 会话等操作。

## 命令结构

```
bbx-cli docker [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `ping` | 检查 Docker daemon 连通性 |
| `image` | Docker 镜像操作 |
| `container` | Docker 容器操作 |
| `compose` | Docker Compose 操作 |

## 命令详情

### docker ping

检查 vigil server 能否连接到本地 Docker daemon。

**语法：**

```
bbx-cli docker ping
```

**参数：**

无

**示例：**

```bash
# 检查 Docker daemon 连通性
./bbx-cli docker ping
```

**输出示例：**

```
Docker daemon is reachable
  API Version:  1.45
  OS Type:      linux
  Experimental: false
```

---

### docker image

Docker 镜像命令组，支持列出、查看、拉取、删除、打标签、查看历史以及从远程 tar 包异步加载镜像。

**语法：**

```
bbx-cli docker image [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `load` | 从远程 tar 包异步下载并加载镜像 |
| `list` | 列出本地镜像 |
| `inspect` | 查看镜像详情 |
| `pull` | 拉取镜像 |
| `rm` | 删除镜像 |
| `tag` | 给镜像打标签 |
| `history` | 查看镜像历史 |

#### docker image load

从远程 URL 下载 Docker tar 包并加载到本地 Docker daemon。

**语法：**

```
bbx-cli docker image load --url <url> --metadata <json>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--url` | `-u` | Docker tar 包下载地址 | 是 | - |
| `--metadata` | `-m` | 镜像元数据 JSON 字符串 | 否 | - |

**示例：**

```bash
# 从远程地址加载镜像
./bbx-cli docker image load -u https://example.com/images/nginx.tar.gz

# 加载镜像并附带元数据
./bbx-cli docker image load -u https://example.com/images/app.tar.gz -m '{"name":"myapp","tag":"v1.0"}'
```

#### docker image list

列出本地 Docker 镜像。

**语法：**

```
bbx-cli docker image list [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--all` | `-a` | 包含中间层镜像 | 否 | false |
| `--dangling` | - | 只显示悬空镜像 | 否 | false |
| `--filter-label` | - | 按标签过滤（如 `app=web`），可多次使用 | 否 | - |
| `--filter` | - | 原始 JSON 过滤条件 | 否 | - |

**示例：**

```bash
# 列出所有镜像
./bbx-cli docker image list

# 包含中间层镜像
./bbx-cli docker image list -a

# 只显示悬空镜像
./bbx-cli docker image list --dangling

# 按标签过滤
./bbx-cli docker image list --filter-label app=web
```

**输出示例：**

```
ID             REPOSITORY:TAG                 CREATED              SIZE
------------------------------------------------------------------------------------------------------------------------
sha256:abc123  alpine:latest                  2025-07-05 10:00:00  43212800
sha256:def456  nginx:alpine                   2025-07-04 18:30:00  67890123
```

#### docker image inspect

查看指定镜像的详细信息。

**语法：**

```
bbx-cli docker image inspect [id]
```

**参数：**

| 参数 | 描述 |
|------|------|
| `id` | 镜像 ID 或引用（必填） |

**示例：**

```bash
# 查看镜像详情
./bbx-cli docker image inspect alpine:latest
```

#### docker image pull

从镜像仓库拉取镜像。

**语法：**

```
bbx-cli docker image pull <image>
```

**参数：**

| 参数 | 描述 |
|------|------|
| `image` | 镜像引用（必填） |

**示例：**

```bash
./bbx-cli docker image pull alpine:latest
```

#### docker image rm

删除一个本地镜像。

**语法：**

```
bbx-cli docker image rm [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 镜像 ID 或引用（必填） | 是 | - |
| `--force` | `-f` | 强制删除 | 否 | false |
| `--no-prune` | - | 不删除未标记的父镜像 | 否 | false |

**示例：**

```bash
# 删除镜像
./bbx-cli docker image rm alpine:latest

# 强制删除
./bbx-cli docker image rm alpine:latest -f
```

#### docker image tag

为已有镜像创建新标签。

**语法：**

```
bbx-cli docker image tag <source> <target>
```

**参数：**

| 参数 | 描述 |
|------|------|
| `source` | 源镜像引用（必填） |
| `target` | 目标标签（必填） |

**示例：**

```bash
./bbx-cli docker image tag alpine:latest myregistry/alpine:v1
```

#### docker image history

查看镜像的构建历史。

**语法：**

```
bbx-cli docker image history [id]
```

**参数：**

| 参数 | 描述 |
|------|------|
| `id` | 镜像 ID 或引用（必填） |

**示例：**

```bash
./bbx-cli docker image history alpine:latest
```

**输出示例：**

```
IMAGE          CREATED              CREATED BY                                         SIZE
------------------------------------------------------------------------------------------------------------------------
sha256:abc123  2025-07-05 10:00:00  /bin/sh -c #(nop) CMD ["sh"]                       0
sha256:def456  2025-07-05 09:59:00  /bin/sh -c #(nop) ADD file:... in /                43212800
```

---

### docker container

Docker 容器命令组，支持容器的增删改查、状态控制和交互式操作。

**语法：**

```
bbx-cli docker container [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `list` | 列出容器 |
| `create` | 创建容器 |
| `inspect` | 查看容器详情 |
| `rm` | 删除容器 |
| `start` | 启动容器 |
| `stop` | 停止容器 |
| `restart` | 重启容器 |
| `pause` | 暂停容器 |
| `unpause` | 恢复容器 |
| `exec` | 在容器内执行一次性命令 |
| `logs` | 获取容器日志 |
| `stats` | 获取容器资源统计 |
| `exec-ws` | 通过 WebSocket 交互式执行命令 |
| `logs-ws` | 通过 WebSocket 实时获取日志 |

#### docker container list

列出 Docker 容器。

**语法：**

```
bbx-cli docker container list [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--all` | `-a` | 包含已停止的容器 | 否 | false |

**示例：**

```bash
# 列出运行中的容器
./bbx-cli docker container list

# 列出所有容器（含已停止）
./bbx-cli docker container list -a
```

**输出示例：**

```
ID             IMAGE                NAMES                     STATE        STATUS               PORTS
----------------------------------------------------------------------------------------------------------------------------------
abc123def456   nginx:latest         web                       running      Up 2 hours           80->8080/tcp
```

#### docker container create

创建一个新的 Docker 容器（不自动启动）。

**语法：**

```
bbx-cli docker container create [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | 容器名称 | 否 | - |
| `--image` | `-i` | 容器镜像 | 是 | - |
| `--cmd` | `-c` | 容器启动命令参数，可多次使用 | 否 | - |
| `--env` | `-e` | 环境变量 KEY=VALUE，可多次使用 | 否 | - |
| `--port` | `-p` | 端口映射，格式 `containerPort:hostPort` 或 `containerPort:hostIP:hostPort` | 否 | - |

**示例：**

```bash
# 创建 nginx 容器
./bbx-cli docker container create -n web -i nginx:latest -p 80:8080

# 创建带环境变量和启动命令的容器
./bbx-cli docker container create -n app -i alpine:latest -e "FOO=bar" -c sh -c "while true; do sleep 1; done"

# 创建带指定主机 IP 的端口映射
./bbx-cli docker container create -n db -i postgres:15 -p 5432:127.0.0.1:5432
```

#### docker container inspect

查看指定容器的详细信息。

**语法：**

```
bbx-cli docker container inspect [id]
```

**参数：**

| 参数 | 描述 |
|------|------|
| `id` | 容器 ID 或名称（必填） |

**示例：**

```bash
# 查看容器详情
./bbx-cli docker container inspect web
```

#### docker container rm

删除一个 Docker 容器。

**语法：**

```
bbx-cli docker container rm [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--force` | `-f` | 强制删除运行中的容器 | 否 | false |

**示例：**

```bash
# 删除容器
./bbx-cli docker container rm web

# 强制删除运行中的容器
./bbx-cli docker container rm web -f
```

#### docker container start

启动一个已停止的容器。

**语法：**

```
bbx-cli docker container start [id]
```

**示例：**

```bash
# 启动容器
./bbx-cli docker container start web
```

#### docker container stop

停止一个运行中的容器。

**语法：**

```
bbx-cli docker container stop [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--timeout` | `-t` | 优雅停止超时时间（秒），0 表示使用 daemon 默认值 | 否 | 0 |

**示例：**

```bash
# 停止容器
./bbx-cli docker container stop web

# 30 秒后强制停止
./bbx-cli docker container stop web -t 30
```

#### docker container restart

重启一个容器。

**语法：**

```
bbx-cli docker container restart [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--timeout` | `-t` | 优雅停止超时时间（秒），0 表示使用 daemon 默认值 | 否 | 0 |

**示例：**

```bash
# 重启容器
./bbx-cli docker container restart web

# 设置重启超时
./bbx-cli docker container restart web -t 30
```

#### docker container pause

暂停容器内的所有进程。

**语法：**

```
bbx-cli docker container pause [id]
```

**示例：**

```bash
# 暂停容器
./bbx-cli docker container pause web
```

#### docker container unpause

恢复容器内的所有进程。

**语法：**

```
bbx-cli docker container unpause [id]
```

**示例：**

```bash
# 恢复容器
./bbx-cli docker container unpause web
```

#### docker container exec

在容器内执行一次性命令并打印输出。

**语法：**

```
bbx-cli docker container exec [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--command` | `-c` | 要执行的命令 | 是 | - |
| `--tty` | `-t` | 分配伪终端 | 否 | false |

**示例：**

```bash
# 在容器中执行 ls
./bbx-cli docker container exec web -c "ls -la /etc/nginx"

# 分配 TTY 执行命令
./bbx-cli docker container exec web -c "bash" -t
```

#### docker container logs

获取或实时跟踪容器日志。

**语法：**

```
bbx-cli docker container logs [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--follow` | `-f` | 持续跟踪日志输出 | 否 | false |
| `--tail` | - | 只输出最后 N 行 | 否 | - |
| `--since` | - | 只输出指定时间戳之后的日志 | 否 | - |

**示例：**

```bash
# 获取容器全部日志
./bbx-cli docker container logs web

# 实时跟踪日志
./bbx-cli docker container logs web -f

# 获取最后 100 行日志
./bbx-cli docker container logs web --tail 100

# 获取 2024-01-01 之后的日志
./bbx-cli docker container logs web --since 2024-01-01T00:00:00Z
```

#### docker container stats

获取容器资源使用统计。

**语法：**

```
bbx-cli docker container stats [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--stream` | `-s` | 持续流式输出统计 | 否 | true |

**示例：**

```bash
# 持续查看容器资源统计
./bbx-cli docker container stats web

# 只获取一次统计
./bbx-cli docker container stats web -s=false
```

#### docker container exec-ws

通过 WebSocket 在容器内打开一个交互式命令会话（类似 `docker exec -it`）。

**语法：**

```
bbx-cli docker container exec-ws [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--command` | `-c` | 要交互执行的命令 | 是 | - |
| `--tty` | `-t` | 分配伪终端 | 否 | true |
| `--width` | `-W` | 终端宽度 | 否 | 120 |
| `--height` | `-H` | 终端高度 | 否 | 40 |

**示例：**

```bash
# 交互式进入容器 bash
./bbx-cli docker container exec-ws web -c "/bin/bash"

# 指定终端尺寸
./bbx-cli docker container exec-ws web -c "/bin/sh" -W 160 -H 50
```

#### docker container logs-ws

通过 WebSocket 实时流式获取容器日志，直到手动中断。

**语法：**

```
bbx-cli docker container logs-ws [id] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `id` | - | 容器 ID 或名称（必填） | 是 | - |
| `--tail` | - | 只输出最后 N 行 | 否 | - |
| `--since` | - | 只输出指定时间戳之后的日志 | 否 | - |

**示例：**

```bash
# 实时流式获取日志
./bbx-cli docker container logs-ws web

# 实时获取最后 50 行日志
./bbx-cli docker container logs-ws web --tail 50
```

---

### docker compose

Docker Compose 命令组，支持部署、查看状态和移除 Compose 项目。

**语法：**

```
bbx-cli docker compose [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `up` | 部署 Compose 项目 |
| `status` | 查看 Compose 项目状态 |
| `down` | 移除 Compose 项目 |

#### docker compose up

根据 docker-compose.yml 文件部署一个 Compose 项目。

**语法：**

```
bbx-cli docker compose up [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--file` | `-f` | docker-compose.yml 文件路径 | 否 | docker-compose.yml |
| `--name` | `-n` | 项目名称 | 是 | - |
| `--start` | - | 创建后是否启动容器 | 否 | true |

**示例：**

```bash
# 使用默认文件部署项目
./bbx-cli docker compose up -n myapp

# 指定 compose 文件
./bbx-cli docker compose up -n myapp -f ./stack/docker-compose.yml

# 只创建不启动
./bbx-cli docker compose up -n myapp --start=false
```

#### docker compose status

查看指定 Compose 项目中所有服务的状态。

**语法：**

```
bbx-cli docker compose status [project]
```

**参数：**

| 参数 | 描述 |
|------|------|
| `project` | 项目名称（必填） |

**示例：**

```bash
# 查看项目状态
./bbx-cli docker compose status myapp
```

**输出示例：**

```
Project: myapp
  Service: web (image: nginx:latest, replicas: 1)
    abc123def456   nginx:latest         running      Up 2 hours
  Service: db (image: postgres:15, replicas: 1)
    def789abc012   postgres:15          running      Up 2 hours
```

#### docker compose down

停止并移除指定 Compose 项目的容器、网络等资源。

**语法：**

```
bbx-cli docker compose down [project] [flags]
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `project` | - | 项目名称（必填） | 是 | - |
| `--force` | `-f` | 强制移除运行中的容器 | 否 | false |

**示例：**

```bash
# 移除项目
./bbx-cli docker compose down myapp

# 强制移除
./bbx-cli docker compose down myapp -f
```

## 注意事项

1. **server 依赖**：所有 Docker 命令都需要 vigil server 正在运行，并且 server 所在主机已安装 Docker。

2. **端口映射格式**：`docker container create` 支持两种端口映射格式：
   - `containerPort:hostPort`：绑定到所有接口
   - `containerPort:hostIP:hostPort`：绑定到指定 IP

3. **异步镜像加载**：`docker image load` 会创建一个后台任务，CLI 会自动轮询任务状态直到完成或失败。

4. **交互式会话**：`docker container exec-ws` 会将本地终端切换到 raw 模式，使用 `Ctrl+C` 或关闭连接即可退出。

5. **WebSocket 路径**：`exec-ws` 和 `logs-ws` 依赖 server 的 WebSocket 端点，请确保防火墙未阻止 WebSocket 连接。

## 相关配置

server 的 `conf/config.yaml` 中可配置 Docker 相关模块：

- `docker`：启用/禁用容器与 Compose 管理功能
- `docker_registry`：启用/禁用内嵌 Docker Registry V2 镜像仓库服务及存储路径

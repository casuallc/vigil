# Docker Registry（镜像仓库）API

`bbx-server` 内嵌了一个符合 **Docker Registry HTTP API V2** 协议的镜像仓库。配置启用后，可以直接使用 Docker CLI 对 `bbx-server` 执行 `docker login / tag / push / pull`。

> 启用方式：默认启用（`conf/config.yaml` 中 `docker_registry.enabled` 为 `true` 或省略）。设置 `docker_registry.enabled: false` 可关闭。
> 
> Registry 的存储目录默认位于 `./data/docker/registry`，可通过 `docker_registry.storage_path` 修改。

## 认证

Registry 接口复用 **vigil 全局 Basic Auth**（`conf/config.yaml` 的 `auth`，即超管凭据或用户库中的注册用户），与其它 API 一致。

```
Authorization: Basic base64(username:password)
```

未携带或凭据错误返回 `401`，响应头带 `WWW-Authenticate: Basic realm="vigil"`。

## HTTP/HTTPS 说明

Docker CLI 默认要求镜像仓库使用 HTTPS。若 `bbx-server` 以 HTTP 运行，需要把服务器地址加入 Docker daemon 的 `insecure-registries`：

```json
{
  "insecure-registries": ["127.0.0.1:57575"]
}
```

- Linux：`/etc/docker/daemon.json`
- Windows：`%USERPROFILE%\.docker\daemon.json`

修改后重启 Docker daemon 生效。

## 端点列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| `/v2/` | GET | API 版本检查 |
| `/v2/_catalog` | GET | 列出所有仓库 |
| `/v2/{name}/tags/list` | GET | 列出仓库标签 |
| `/v2/{name}/manifests/{reference}` | HEAD | 检查 manifest 是否存在 |
| `/v2/{name}/manifests/{reference}` | GET | 获取 manifest |
| `/v2/{name}/manifests/{reference}` | PUT | 上传 manifest（reference 为 tag） |
| `/v2/{name}/manifests/{reference}` | DELETE | 删除 manifest |
| `/v2/{name}/blobs/uploads/` | POST | 开始 blob 上传 |
| `/v2/{name}/blobs/uploads/{uuid}` | PATCH | 上传 blob 分块 |
| `/v2/{name}/blobs/uploads/{uuid}?digest=...` | PUT | 完成 blob 上传 |
| `/v2/{name}/blobs/uploads/{uuid}` | DELETE | 取消 blob 上传 |
| `/v2/{name}/blobs/{digest}` | HEAD | 检查 blob 是否存在 |
| `/v2/{name}/blobs/{digest}` | GET | 下载 blob |
| `/v2/{name}/blobs/{digest}` | DELETE | 删除 blob |

> `{name}` 支持带 namespace，例如 `library/nginx`。
> 
> 错误响应遵循 Docker Registry 规范：`{"errors":[{"code":"...","message":"..."}]}`。

## 使用示例

### 1. 配置

```yaml
# conf/config.yaml
docker_registry:
  enabled: true
  storage_path: ./data/docker/registry
```

### 2. 登录

```bash
docker login 127.0.0.1:57575 -u bbx -p Flzx3qL@ysyhl9t
```

### 3. 标记并推送镜像

```bash
docker tag hello-world 127.0.0.1:57575/library/hello-world:latest
docker push 127.0.0.1:57575/library/hello-world:latest
```

### 4. 拉取镜像

```bash
docker pull 127.0.0.1:57575/library/hello-world:latest
```

### 5. 查看仓库与标签

```bash
curl -u bbx:Flzx3qL@ysyhl9t http://127.0.0.1:57575/v2/_catalog
curl -u bbx:Flzx3qL@ysyhl9t http://127.0.0.1:57575/v2/library/hello-world/tags/list
```

## 存储布局

```
<storage_path>/
  repositories/
    <namespace>/<image>/
      _manifests/
        <tag>
        <tag>.mediaType
      _tags.json
      _index.json
  blobs/
    sha256/
      <hex>/
        data
        digest
  uploads/
    <uuid>/
      data
      state.json
```

- blob 按内容寻址，跨仓库共享。
- manifest 以 tag 文件和 `_index.json` 两种形式保存，支持按 tag 或 digest 读取。
- 上传中的 blob 先存放在 `uploads/<uuid>/`，完成校验 digest 后原子移动到 `blobs/`。

## 限制

- 第一版不支持镜像 GC，已上传但未被 manifest 引用的 blob 不会自动删除。
- 多租户、配额、签名（notary）等高级功能不在当前版本范围内。
- 并发上传同一 tag 的 manifest 采用 last-write-wins 策略。

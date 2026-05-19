# BBX 代理注册 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/v1/bbx/register | POST | BBX 代理向管控台注册 |

---

## POST /api/v1/bbx/register

**功能描述**：供 BBX 代理安装后调用，完成向管控台的自动注册。此接口不需要 JWT 认证，通过 `installToken` 自身完成鉴权。

**请求头**：

| 头部 | 值 | 说明 |
|------|-----|------|
| Content-Type | application/json | 必需 |

**请求参数**：JSON 对象

```json
{
  "installToken": "zL4WRvBijFNgepHEMpWxhhq33p1Llc6r",
  "hostname": "server-01",
  "arch": "amd64"
}
```

**字段说明**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| installToken | string | 是 | 安装令牌，用于鉴权 |
| hostname | string | 是 | BBX 代理所在服务器的主机名 |
| arch | string | 是 | 系统架构，如 `amd64`、`arm64` |

**响应格式**：JSON 对象

```json
{
  "code": 0,
  "msg": ""
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，`0` 表示成功，非 `0` 表示失败 |
| msg | string | 结果信息，失败时返回错误描述 |

**成功响应示例**：

```json
{
  "code": 0,
  "msg": ""
}
```

**失败响应示例**：

```json
{
  "code": 1,
  "msg": "Invalid install token"
}
```

## CLI 用法

```bash
# 使用默认配置文件（/etc/bbx/install-config.json）
bbx-cli register

# 指定配置文件
bbx-cli register -f /path/to/install-config.json

# HTTPS 自签名证书场景
bbx-cli register --insecure
```

**配置文件格式**（`install-config.json`）：

```json
{
  "managerUrl": "https://172.20.140.224:12306",
  "installToken": "zL4WRvBijFNgepHEMpWxhhq33p1Llc6r"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| managerUrl | string | 是 | 管控台地址 |
| installToken | string | 是 | 安装令牌 |

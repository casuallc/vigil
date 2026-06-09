# 网络诊断 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/network/probe | POST | 探测目标 IP 和端口是否可达 |

---

## POST /api/network/probe

**功能描述**：探测指定目标 IP 和端口的网络连通性，返回是否可达以及连接延迟。

**请求参数**：

| 参数 | 类型 | 必填 | 描述 | 默认值 |
|------|------|------|------|--------|
| targetIp | string | 是 | 目标 IP 地址或主机名 | - |
| port | int | 是 | 目标端口（1-65535） | - |
| protocol | string | 否 | 协议类型（tcp、udp） | tcp |
| timeoutMs | int | 否 | 超时时间（毫秒） | 5000 |

**请求示例**：

```json
{
  "targetIp": "192.168.1.100",
  "port": 22,
  "protocol": "tcp",
  "timeoutMs": 5000
}
```

**响应格式**：

```json
{
  "reachable": true,
  "latencyMs": 12,
  "error": ""
}
```

**响应字段说明**：

| 字段 | 类型 | 描述 |
|------|------|------|
| reachable | bool | 是否可达 |
| latencyMs | int64 | 连接延迟（毫秒），不可达时为 -1 |
| error | string | 错误信息，可达时为空字符串 |

**成功响应示例**（可达）：

```json
{
  "reachable": true,
  "latencyMs": 12,
  "error": ""
}
```

**成功响应示例**（不可达）：

```json
{
  "reachable": false,
  "latencyMs": -1,
  "error": "dial tcp 192.168.1.100:22: connect: connection refused"
}
```

**错误响应示例**（参数缺失）：

```json
{
  "error": "targetIp is required"
}
```

**错误响应示例**（端口无效）：

```json
{
  "error": "port must be between 1 and 65535"
}
```

**cURL 示例**：

```bash
# 探测 SSH 端口
curl -X POST http://localhost:57575/api/network/probe \
  -H "Content-Type: application/json" \
  -d '{"targetIp":"192.168.1.100","port":22}'

# 探测 HTTP 端口，使用 UDP 协议，超时 3 秒
curl -X POST http://localhost:57575/api/network/probe \
  -H "Content-Type: application/json" \
  -d '{"targetIp":"example.com","port":53,"protocol":"udp","timeoutMs":3000}'
```

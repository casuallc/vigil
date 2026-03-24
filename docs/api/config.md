# 配置管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/config | GET | 获取配置 |
| /api/config | PUT | 更新配置 |

---

## GET /api/config

**功能描述**：获取配置

**请求参数**：无

**响应格式**：
```json
{
  "addr": ":8080",
  "auth": {
    "enabled": true,
    "username": "username",
    "password": "password"
  },
  "log": {
    "level": "info"
  },
  "monitor": {
    "rate": 60
  },
  "process": {
    "pid_file": "/path/to/pid/file"
  },
  "security": {
    "encryption_key": "encryption-key"
  },
  "https": {
    "enabled": false,
    "cert_path": "",
    "key_path": ""
  }
}
```

---

## PUT /api/config

**功能描述**：更新配置

**请求参数**：
- 请求体：配置信息

**请求体示例**：同获取配置

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

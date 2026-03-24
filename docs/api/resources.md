# 资源监控 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/resources/system | GET | 获取系统资源信息 |
| /api/resources/process/{pid} | GET | 获取进程资源信息 |

---

## GET /api/resources/system

**功能描述**：获取系统资源信息

**请求参数**：无

**响应格式**：
```json
{
  "cpu_usage": 10.5,
  "memory_usage": 50.2,
  "disk_usage": 75.8,
  "network_stats": {
    "rx_bytes": 1024,
    "tx_bytes": 2048
  }
}
```

---

## GET /api/resources/process/{pid}

**功能描述**：获取进程资源信息

**请求参数**：
- `pid`：进程 ID（路径参数）

**响应格式**：
```json
{
  "pid": 1234,
  "cpu_usage": 5.2,
  "memory_usage": 20.5,
  "disk_usage": 10.1,
  "network_stats": {
    "rx_bytes": 512,
    "tx_bytes": 1024
  }
}
```

# 巡检检查 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/inspect | POST | 执行巡检检查 |

---

## POST /api/inspect

**功能描述**：执行巡检检查

**请求参数**：
- 请求体：巡检请求

**请求体示例**：
```json
{
  "type": "type",
  "target": "target",
  "parameters": {
    "key": "value"
  }
}
```

**响应格式**：
```json
{
  "type": "type",
  "target": "target",
  "status": "success",
  "result": {
    "key": "value"
  },
  "errors": ["error message"]
}
```

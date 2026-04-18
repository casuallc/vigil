# 文件管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/files/upload | POST | 上传文件（≤100MB） |
| /api/files/stream | POST | 流式上传文件（>100MB） |
| /api/files/download | POST | 下载文件 |
| /api/files/list | POST | 列出文件 |
| /api/files/delete | POST | 删除文件 |
| /api/files/copy | POST | 复制文件 |
| /api/files/move | POST | 移动文件 |

---

## POST /api/files/upload

**功能描述**：上传文件

**请求参数**：
- 表单数据：
  - `file`：文件内容
  - `target_path`：目标路径

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

---

## POST /api/files/stream

**功能描述**：大文件流式上传（>100MB 推荐使用，内存占用低）

**请求头**：
- `Content-Type: application/octet-stream`
- `X-Target-Path`: 目标路径

**请求体**：文件原始字节流

**响应格式**：
- 成功：200 OK
  ```json
  {
    "message": "File uploaded successfully"
  }
  ```
- 失败：400 Bad Request（缺少 X-Target-Path 头）或 500 Internal Server Error

**与 /api/files/upload 的区别**：
- `/api/files/upload`：multipart/form-data，服务端 `ParseMultipartForm(10MB)`，适合小文件（≤100MB）
- `/api/files/stream`：raw body 流式传输，服务端 `io.Copy` 直接写入，不限制文件大小，适合大文件

---

## POST /api/files/download

**功能描述**：下载文件

**请求参数**：
- 请求体：
  - `source_path`：源路径

**请求体示例**：
```json
{
  "source_path": "/path/to/file"
}
```

**响应格式**：
- 成功：文件内容
- 失败：400 Bad Request 或 500 Internal Server Error

---

## POST /api/files/list

**功能描述**：列出文件

**请求参数**：
- 请求体：
  - `path`：路径
  - `max_depth`：最大深度

**请求体示例**：
```json
{
  "path": "/path/to/dir",
  "max_depth": 1
}
```

**响应格式**：
```json
[
  {
    "name": "file-name",
    "path": "/path/to/file",
    "size": 1024,
    "is_dir": false,
    "mode": "-rw-r--r--",
    "mod_time": "2023-01-01T00:00:00Z",
    "depth": 1
  }
]
```

---

## POST /api/files/delete

**功能描述**：删除文件

**请求参数**：
- 请求体：
  - `path`：路径

**请求体示例**：
```json
{
  "path": "/path/to/file"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

---

## POST /api/files/copy

**功能描述**：复制文件

**请求参数**：
- 请求体：
  - `source_path`：源路径
  - `target_path`：目标路径

**请求体示例**：
```json
{
  "source_path": "/path/to/source",
  "target_path": "/path/to/target"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

---

## POST /api/files/move

**功能描述**：移动文件

**请求参数**：
- 请求体：
  - `source_path`：源路径
  - `target_path`：目标路径

**请求体示例**：
```json
{
  "source_path": "/path/to/source",
  "target_path": "/path/to/target"
}
```

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

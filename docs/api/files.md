# 文件管理 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/files/upload | POST | 上传文件（≤100MB） |
| /api/files/stream | POST | 流式上传文件（>100MB） |
| /api/files/logs/stream | GET | 实时流式查看日志（tail -f） |
| /api/files/download | POST | 下载文件 |
| /api/files/list | POST | 列出文件 |
| /api/files/delete | POST | 删除文件 |
| /api/files/copy | POST | 复制文件 |
| /api/files/move | POST | 移动文件 |
| /api/files/mkdir | POST | 创建目录 |

---

## POST /api/files/upload

**功能描述**：上传文件（≤100MB，multipart/form-data）

**请求参数**：
- 表单数据：
  - `file`：文件内容
  - `target_path`：目标路径（**支持中文路径**，非 ASCII 字符建议进行 [URL Path 编码](#编码规范)）

**响应格式**：
- 成功：200 OK
- 失败：400 Bad Request 或 500 Internal Server Error

---

## POST /api/files/stream

**功能描述**：大文件流式上传（>100MB 推荐使用，内存占用低）

**请求头**：
- `Content-Type: application/octet-stream`
- `X-Target-Path`: 目标路径（**支持中文路径**，非 ASCII 字符建议进行 [URL Path 编码](#编码规范)）

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
- 成功：文件内容，响应头 `Content-Disposition` 采用 [RFC 5987](#编码规范) 编码，同时提供 `filename`（原始值）和 `filename*=UTF-8''`（URL 编码）两个参数，兼容新旧浏览器
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

---

## POST /api/files/mkdir

**功能描述**：创建目录

**请求参数**：
- 请求体：
  - `path`：目录路径（必填）
  - `parents`：是否递归创建父目录（可选，默认 `false`）

**请求体示例**：
```json
{
  "path": "/132"
}
```

**请求体示例（递归创建）**：
```json
{
  "path": "/a/b/c",
  "parents": true
}
```

**响应格式**：
- 成功：200 OK
  ```json
  {
    "message": "Directory created successfully"
  }
  ```
- 失败：400 Bad Request 或 500 Internal Server Error

---

## GET /api/files/logs/stream

**功能描述**：实时流式查看日志文件（类似 `tail -f`），通过 SSE（Server-Sent Events）逐行推送。

**请求参数**（Query String）：
- `path`（必填）：日志文件路径
- `from_line`（可选）：起始行号策略
  - 省略：从文件末尾开始（默认 `tail -f` 行为）
  - `0`：从文件开头开始
  - 正数：从指定行号开始
  - 负数：从末尾向前偏移（如 `-100` 表示最后 100 行）

**请求示例**：
```
GET /api/files/logs/stream?path=/var/log/app.log&from_line=-100
```

**响应格式**：SSE（`Content-Type: text/event-stream`）

- 普通日志行事件：
  ```
  event: line
  data: {"line_number": 101, "content": "2025-04-18 log message..."}
  ```

- 错误事件（文件不存在、读取失败等）：
  ```
  event: error
  data: {"message": "failed to open file"}
  ```

**响应头**：
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**说明**：
- 服务端读取到文件末尾后，每 500ms 轮询检查是否有新内容写入
- 客户端断开连接时，服务端自动清理资源
- 支持跨平台换行符（`\n` 和 `\r\n`）

---

## 编码规范

文件管理 API 支持中文文件名和路径，对接第三方客户端时请遵循以下编码约定：

### 上传路径编码

- **Multipart 上传**（`/api/files/upload`）：`target_path` 表单字段支持 UTF-8 直接传递，也接受 **URL Path 编码**（`url.PathEscape` / `encodeURIComponent`）后的值。服务端会自动解码
- **Stream 上传**（`/api/files/stream`）：`X-Target-Path` 请求头中的非 ASCII 字符（如中文）**必须**进行 URL Path 编码，因为 HTTP Header 按规范只支持 ASCII 字符

**不同客户端示例**：

```bash
# Go: url.PathEscape("/tmp/测试.txt") → "/tmp/%E6%B5%8B%E8%AF%95.txt"
# JavaScript: encodeURIComponent("测试.txt") → "%E6%B5%8B%E8%AF%95.txt"
# Python: urllib.parse.quote("测试.txt") → "%E6%B5%8B%E8%AF%95.txt"
```

**向后兼容**：纯 ASCII 路径不需要编码，服务端 `PathUnescape` 会原样返回

### 下载文件名编码

下载接口的 `Content-Disposition` 响应头同时提供两个参数：

```
Content-Disposition: attachment; filename="测试.txt"; filename*=UTF-8''%E6%B5%8B%E8%AF%95.txt
```

- `filename="测试.txt"`：原始 UTF-8 值，供旧版浏览器兼容
- `filename*=UTF-8''%E6%B5%8B%E8%AF%95.txt`：RFC 5987 编码格式，供现代浏览器正确显示中文文件名

客户端解析 `Content-Disposition` 时，优先使用 `filename*`，回退到 `filename`

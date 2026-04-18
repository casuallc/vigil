# 大文件流式上传设计文档

## 背景

现有 `/api/files/upload` 使用 `ParseMultipartForm(10MB)`，整个文件在服务端和客户端都会加载到内存中，不适合大文件上传。需要新增流式上传端点，支持大文件通过 raw body 直接流式传输。

## 目标

- 新增本地文件和 VM 文件的大文件流式上传 API
- 客户端根据文件大小自动选择上传策略（≤100MB 走 multipart，>100MB 走 stream）
- 服务端和客户端内存占用均不受文件大小影响

## API 设计

### 1. 本地文件流式上传

```
POST /api/files/stream
Headers:
  X-Target-Path: /data/large.zip
  Content-Type: application/octet-stream
Body: <raw file bytes>

Response (200 OK):
  {"message": "File uploaded successfully"}
```

### 2. VM 文件流式上传

```
POST /api/vms/files/{name}/stream
Headers:
  X-Target-Path: /data/large.zip
  Content-Type: application/octet-stream
Body: <raw file bytes>

Response (200 OK):
  {"message": "File uploaded successfully"}
```

## 服务端实现

- `handleFileStreamUpload`: 读取 `r.Header.Get("X-Target-Path")`，使用 `io.Copy(r.Body, dstFile)` 直接流式写入
- `handleVmFileStreamUpload`: 同上，通过 VM 的 SSH/SFTP 通道流式传输
- 不调用 `ParseMultipartForm`，内存占用恒定

## 客户端实现

- `FileUpload`: 判断文件大小，>100MB 时走 `/api/files/stream`，`os.File` 直接作为 `http.Request.Body`
- `VMFileUpload`: 同上，走 `/api/vms/files/{name}/stream`
- 小文件保持现有 multipart 方式不变

## CLI

- 现有 `file upload` 命令行为不变，内部自动切换策略

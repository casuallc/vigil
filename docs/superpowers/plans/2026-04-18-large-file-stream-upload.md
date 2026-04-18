# 大文件流式上传实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增本地文件和VM文件的大文件流式上传API，客户端根据文件大小（100MB阈值）自动选择multipart或stream上传策略。

**Architecture:** 新增两个raw-body流式上传端点（`/api/files/stream` 和 `/api/vms/files/{name}/stream`），服务端直接 `io.Copy(r.Body, dstFile)` 写入，不调用 `ParseMultipartForm`。客户端在 `FileUpload` / `VMFileUpload` 中检查文件大小自动路由。

**Tech Stack:** Go, gorilla/mux, SFTP (vm/ssh.go), standard library `io`, `os`, `net/http`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `api/routes.go` | 注册两个新的stream路由 |
| `api/handlers_file.go` | 本地文件流式上传handler |
| `api/handlers_vm.go` | VM文件流式上传handler |
| `api/client_file.go` | 客户端自动切换上传策略 |
| `api/handlers_file_test.go` | stream上传的handler测试 |

---

## Task 1: 注册Stream路由

**Files:**
- Modify: `api/routes.go`

在现有File Management endpoints和VM File Management endpoints之间各增加一个stream端点。

- [ ] **Step 1: 添加本地文件stream路由**

在 `api/routes.go` 第53行（`// Execute command endpoint`注释上方）添加：

```go
  // File stream upload endpoint (for large files)
  r.HandleFunc("/api/files/stream", s.handleFileStreamUpload).Methods("POST")
```

- [ ] **Step 2: 添加VM文件stream路由**

在 `api/routes.go` 第86行（`// VM Permission endpoints`注释上方）添加：

```go
  // VM File Stream Upload endpoint (for large files)
  r.HandleFunc("/api/vms/files/{name}/stream", s.handleVmFileStreamUpload).Methods("POST")
```

- [ ] **Step 3: Commit**

```bash
git add api/routes.go
git commit -m "feat(api): add stream upload routes for large files"
```

---

## Task 2: 实现本地文件流式上传Handler

**Files:**
- Modify: `api/handlers_file.go`

- [ ] **Step 1: 实现 `handleFileStreamUpload`**

在 `api/handlers_file.go` 的 `handleFileUpload` 函数之后（第90行后）添加：

```go
// handleFileStreamUpload handles uploading large files via raw body stream
func (s *Server) handleFileStreamUpload(w http.ResponseWriter, r *http.Request) {
  // Get target path from header
  targetPath := r.Header.Get("X-Target-Path")
  if targetPath == "" {
    writeError(w, http.StatusBadRequest, "X-Target-Path header is required")
    return
  }

  // Create file manager
  fileManager := file.NewManager("")

  // Upload file from request body (streaming, no memory limit)
  if err := fileManager.UploadFile(r.Body, targetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./api/...
```

Expected: 编译成功，无错误。

- [ ] **Step 3: Commit**

```bash
git add api/handlers_file.go
git commit -m "feat(api): add local file stream upload handler"
```

---

## Task 3: 实现VM文件流式上传Handler

**Files:**
- Modify: `api/handlers_vm.go`

- [ ] **Step 1: 实现 `handleVmFileStreamUpload`**

在 `api/handlers_vm.go` 的 `handleVmFileUpload` 函数之后（第153行后）添加：

```go
// handleVmFileStreamUpload handles uploading large files to a VM via raw body stream
func (s *Server) handleVmFileStreamUpload(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  vmName := vars["name"]
  if vmName == "" {
    writeError(w, http.StatusBadRequest, "vm name is required in path")
    return
  }

  // Get target path from header
  targetPath := r.Header.Get("X-Target-Path")
  if targetPath == "" {
    writeError(w, http.StatusBadRequest, "X-Target-Path header is required")
    return
  }

  // Get VM info
  vmInfo, err := s.vmManager.GetVM(vmName)
  if err != nil {
    writeError(w, http.StatusNotFound, err.Error())
    return
  }

  // Create SSH client
  sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
    Host:     vmInfo.IP,
    Port:     vmInfo.Port,
    Username: vmInfo.Username,
    Password: vmInfo.Password,
    KeyPath:  vmInfo.KeyPath,
  })
  if err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  // Connect to SSH server
  if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }
  defer sshClient.Close()

  // Upload file from request body (streaming, no memory limit)
  if err := sshClient.UploadFile(r.Body, targetPath); err != nil {
    writeError(w, http.StatusInternalServerError, err.Error())
    return
  }

  writeJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./api/...
```

Expected: 编译成功，无错误。

- [ ] **Step 3: Commit**

```bash
git add api/handlers_vm.go
git commit -m "feat(api): add VM file stream upload handler"
```

---

## Task 4: 修改客户端实现自动切换策略

**Files:**
- Modify: `api/client_file.go`

在 `api/client_file.go` 顶部添加常量定义：

```go
const (
  streamUploadThreshold = 100 << 20 // 100MB
)
```

- [ ] **Step 1: 修改 `FileUpload` 支持自动切换**

将 `api/client_file.go` 中的 `FileUpload` 函数替换为：

```go
// FileUpload uploads a file to the server, automatically choosing multipart or stream based on file size
func (c *Client) FileUpload(sourcePath, targetPath string) error {
  // Get file info to determine upload strategy
  fileInfo, err := os.Stat(sourcePath)
  if err != nil {
    return fmt.Errorf("failed to stat source file: %v", err)
  }

  // For large files, use stream upload
  if fileInfo.Size() > streamUploadThreshold {
    return c.fileStreamUpload(sourcePath, targetPath, "/api/files/stream")
  }

  // For small files, use multipart upload (existing logic)
  // Open local file
  sourceFile, err := os.Open(sourcePath)
  if err != nil {
    return fmt.Errorf("failed to open source file: %v", err)
  }
  defer sourceFile.Close()

  // Create multipart/form-data request
  body := &bytes.Buffer{}
  writer := multipart.NewWriter(body)

  // Add file field
  fileField, err := writer.CreateFormFile("file", filepath.Base(sourcePath))
  if err != nil {
    return err
  }

  // Copy file content to multipart writer
  if _, err := io.Copy(fileField, sourceFile); err != nil {
    return err
  }

  // Add target path field
  if err := writer.WriteField("target_path", targetPath); err != nil {
    return err
  }

  // Finish multipart writer
  contentType := writer.FormDataContentType()
  writer.Close()

  // Create request
  req, err := http.NewRequest("POST", c.host+"/api/files/upload", body)
  if err != nil {
    return err
  }

  // Set request headers
  req.Header.Set("Content-Type", contentType)

  // If Basic Auth is configured, attach to request
  if c.basicUser != "" && c.basicPass != "" {
    req.SetBasicAuth(c.basicUser, c.basicPass)
  }

  // Send request
  resp, err := c.httpClient.Do(req)
  if err != nil {
    return err
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return c.errorFromResponse(resp)
  }

  return nil
}
```

- [ ] **Step 2: 添加 `fileStreamUpload` 辅助方法**

在 `FileUpload` 函数之后添加：

```go
// fileStreamUpload uploads a file via raw body stream
func (c *Client) fileStreamUpload(sourcePath, targetPath, endpoint string) error {
  // Open local file
  sourceFile, err := os.Open(sourcePath)
  if err != nil {
    return fmt.Errorf("failed to open source file: %v", err)
  }
  defer sourceFile.Close()

  // Create request with file as body (streaming)
  req, err := http.NewRequest("POST", c.host+endpoint, sourceFile)
  if err != nil {
    return err
  }

  // Set request headers
  req.Header.Set("Content-Type", "application/octet-stream")
  req.Header.Set("X-Target-Path", targetPath)

  // If Basic Auth is configured, attach to request
  if c.basicUser != "" && c.basicPass != "" {
    req.SetBasicAuth(c.basicUser, c.basicPass)
  }

  // Send request
  resp, err := c.httpClient.Do(req)
  if err != nil {
    return err
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return c.errorFromResponse(resp)
  }

  return nil
}
```

- [ ] **Step 3: 修改 `VMFileUpload` 支持自动切换**

将 `api/client_file.go` 中的 `VMFileUpload` 函数替换为：

```go
// VMFileUpload uploads a file to a VM (through server), automatically choosing multipart or stream based on file size
func (c *Client) VMFileUpload(vmName, sourcePath, targetPath string) error {
  // Get file info to determine upload strategy
  fileInfo, err := os.Stat(sourcePath)
  if err != nil {
    return fmt.Errorf("failed to stat source file: %v", err)
  }

  // For large files, use stream upload
  if fileInfo.Size() > streamUploadThreshold {
    return c.fileStreamUpload(sourcePath, targetPath, fmt.Sprintf("/api/vms/files/%s/stream", vmName))
  }

  // For small files, use multipart upload (existing logic)
  // Open local file
  sourceFile, err := os.Open(sourcePath)
  if err != nil {
    return fmt.Errorf("failed to open source file: %v", err)
  }
  defer sourceFile.Close()

  // Create multipart/form-data request
  body := &bytes.Buffer{}
  writer := multipart.NewWriter(body)

  // Add file field
  fileField, err := writer.CreateFormFile("file", filepath.Base(sourcePath))
  if err != nil {
    return err
  }

  // Copy file content to multipart writer
  if _, err := io.Copy(fileField, sourceFile); err != nil {
    return err
  }

  // Add target path field
  if err := writer.WriteField("target_path", targetPath); err != nil {
    return err
  }

  // Finish multipart writer
  contentType := writer.FormDataContentType()
  writer.Close()

  // Create request with new route format
  req, err := http.NewRequest("POST", c.host+fmt.Sprintf("/api/vms/files/%s/upload", vmName), body)
  if err != nil {
    return err
  }

  // Set request headers
  req.Header.Set("Content-Type", contentType)

  // If Basic Auth is configured, attach to request
  if c.basicUser != "" && c.basicPass != "" {
    req.SetBasicAuth(c.basicUser, c.basicPass)
  }

  // Send request
  resp, err := c.httpClient.Do(req)
  if err != nil {
    return err
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return c.errorFromResponse(resp)
  }

  return nil
}
```

- [ ] **Step 4: 编译验证**

```bash
go build ./api/...
```

Expected: 编译成功，无错误。

- [ ] **Step 5: Commit**

```bash
git add api/client_file.go
git commit -m "feat(api): client auto-switches between multipart and stream upload based on file size"
```

---

## Task 5: 编写Handler测试

**Files:**
- Create: `api/handlers_file_test.go`

- [ ] **Step 1: 创建测试文件**

```go
package api

import (
  "bytes"
  "net/http"
  "net/http/httptest"
  "os"
  "path/filepath"
  "testing"
)

func TestHandleFileStreamUpload(t *testing.T) {
  server := &Server{}

  // Create a temporary directory for uploads
  tempDir := t.TempDir()
  targetPath := filepath.Join(tempDir, "test-upload.txt")

  // Create request with file content in body
  content := []byte("hello, this is a stream upload test")
  req := httptest.NewRequest(http.MethodPost, "/api/files/stream", bytes.NewReader(content))
  req.Header.Set("X-Target-Path", targetPath)

  rr := httptest.NewRecorder()
  server.handleFileStreamUpload(rr, req)

  if rr.Code != http.StatusOK {
    t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
  }

  // Verify file was written
  written, err := os.ReadFile(targetPath)
  if err != nil {
    t.Fatalf("failed to read uploaded file: %v", err)
  }
  if !bytes.Equal(written, content) {
    t.Errorf("expected %q, got %q", content, written)
  }
}

func TestHandleFileStreamUploadMissingHeader(t *testing.T) {
  server := &Server{}

  req := httptest.NewRequest(http.MethodPost, "/api/files/stream", bytes.NewReader([]byte("test")))
  rr := httptest.NewRecorder()
  server.handleFileStreamUpload(rr, req)

  if rr.Code != http.StatusBadRequest {
    t.Errorf("expected status 400, got %d", rr.Code)
  }
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./api/... -run TestHandleFileStreamUpload -v
```

Expected: 两个测试都通过。

- [ ] **Step 3: Commit**

```bash
git add api/handlers_file_test.go
git commit -m "test(api): add tests for file stream upload handler"
```

---

## Task 6: 全量编译和测试验证

- [ ] **Step 1: 编译整个项目**

```bash
go build ./...
```

Expected: 编译成功，无错误。

- [ ] **Step 2: 运行API包测试**

```bash
go test ./api/...
```

Expected: 所有测试通过。

- [ ] **Step 3: Commit（如有未提交的变更）**

---

## Self-Review Checklist

1. **Spec coverage:**
   - [x] 新增本地文件stream端点 — Task 1 + Task 2
   - [x] 新增VM文件stream端点 — Task 1 + Task 3
   - [x] 客户端自动切换策略（100MB阈值）— Task 4
   - [x] 服务端流式写入不占用大量内存 — Task 2 + Task 3
   - [x] CLI行为不变 — 无需修改CLI，客户端内部自动切换

2. **Placeholder scan:** 无TBD/TODO/"implement later"/"add appropriate error handling"等占位符

3. **Type consistency:** `fileStreamUpload` 接收 `sourcePath, targetPath, endpoint string` 在所有调用处一致；`streamUploadThreshold` 常量在两个判断处一致

/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package file

import (
  "context"
  "crypto/md5"
  "crypto/sha1"
  "crypto/sha256"
  "encoding/hex"
  "fmt"
  "hash"
  "io"
  "io/ioutil"
  "net/http"
  "net/url"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "time"
)

// Manager 表示文件管理器
type Manager struct {
  BasePath string
}

// NewManager 创建一个新的文件管理器
func NewManager(basePath string) *Manager {
  if basePath == "" {
    basePath = "."
  }
  return &Manager{BasePath: basePath}
}

// UploadFile 上传文件
func (m *Manager) UploadFile(src io.Reader, dstPath string) error {
  // 确保目标目录存在
  dstDir := filepath.Dir(dstPath)
  if err := os.MkdirAll(dstDir, 0755); err != nil {
    return fmt.Errorf("failed to create directory: %v", err)
  }

  // 创建目标文件
  dstFile, err := os.Create(dstPath)
  if err != nil {
    return fmt.Errorf("failed to create file: %v", err)
  }
  defer dstFile.Close()

  // 复制文件内容
  if _, err := io.Copy(dstFile, src); err != nil {
    return fmt.Errorf("failed to copy file content: %v", err)
  }

  return nil
}

// DownloadFile 下载文件
func (m *Manager) DownloadFile(srcPath string) ([]byte, error) {
  // 读取文件内容
  content, err := ioutil.ReadFile(srcPath)
  if err != nil {
    return nil, fmt.Errorf("failed to read file: %v", err)
  }

  return content, nil
}

// ListFiles 列出文件
func (m *Manager) ListFiles(path string, maxDepth int) ([]Info, error) {
  var files []Info

  // 遍历目录
  err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    // 跳过指定的路径本身，只处理其子文件和子目录
    if filePath == path {
      return nil
    }

    // 计算深度
    depth := strings.Count(strings.TrimPrefix(filePath, path), string(os.PathSeparator))
    if depth > maxDepth {
      return filepath.SkipDir
    }

    // 创建文件信息
    fileInfo := Info{
      Name:    info.Name(),
      Path:    filePath,
      Size:    info.Size(),
      IsDir:   info.IsDir(),
      Mode:    info.Mode().String(),
      ModTime: info.ModTime().Format(time.RFC3339),
      Depth:   depth,
    }

    files = append(files, fileInfo)

    return nil
  })

  if err != nil {
    return nil, fmt.Errorf("failed to walk directory: %v", err)
  }

  return files, nil
}

// DeleteFile 删除文件
func (m *Manager) DeleteFile(path string) error {
  return os.RemoveAll(path)
}

// CopyFile 复制文件
func (m *Manager) CopyFile(srcPath, dstPath string) error {
  // 读取源文件
  srcContent, err := m.DownloadFile(srcPath)
  if err != nil {
    return err
  }

  // 写入目标文件
  dstFile, err := os.Create(dstPath)
  if err != nil {
    return fmt.Errorf("failed to create file: %v", err)
  }
  defer dstFile.Close()

  if _, err := dstFile.Write(srcContent); err != nil {
    return fmt.Errorf("failed to write file: %v", err)
  }

  return nil
}

// MoveFile 移动文件
func (m *Manager) MoveFile(srcPath, dstPath string) error {
  // 复制文件
  if err := m.CopyFile(srcPath, dstPath); err != nil {
    return err
  }

  // 删除源文件
  if err := m.DeleteFile(srcPath); err != nil {
    return err
  }

  return nil
}

// Mkdir 创建目录
func (m *Manager) Mkdir(path string, parents bool) error {
  if parents {
    return os.MkdirAll(path, 0755)
  }
  return os.Mkdir(path, 0755)
}

// FetchFile 根据元数据从远程 URL 下载文件到本地，并可选地校验校验码、设置权限和所属用户。
// 成功时返回下载结果（路径、大小、实际校验码），失败时返回 error。
func (m *Manager) FetchFile(ctx context.Context, req *FetchRequest) (*FetchResult, error) {
  if req.URL == "" {
    return nil, fmt.Errorf("url is required")
  }

  parsed, err := url.Parse(req.URL)
  if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
    return nil, fmt.Errorf("invalid url: only http/https schemes are supported")
  }

  // 解析保存的文件名
  name := req.Name
  if name == "" {
    name = filepath.Base(parsed.Path)
    if name == "" || name == "." || name == "/" {
      return nil, fmt.Errorf("cannot determine file name from url, please set name explicitly")
    }
  }
  if strings.ContainsAny(name, `/\`) {
    return nil, fmt.Errorf("invalid file name: %q", name)
  }

  dir := req.Dir
  if dir == "" {
    dir = m.BasePath
  }
  if err := os.MkdirAll(dir, 0755); err != nil {
    return nil, fmt.Errorf("failed to create directory: %v", err)
  }
  dstPath := filepath.Join(dir, name)

  // 解析文件权限
  var mode os.FileMode
  if req.Mode != "" {
    v, err := strconv.ParseUint(req.Mode, 8, 32)
    if err != nil {
      return nil, fmt.Errorf("invalid mode %q: expected octal string like \"0755\"", req.Mode)
    }
    mode = os.FileMode(v)
  }

  // 发起下载请求
  httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
  if err != nil {
    return nil, fmt.Errorf("failed to create request: %v", err)
  }
  for k, v := range req.Headers {
    httpReq.Header.Set(k, v)
  }

  // 应用下载认证（可选，预留其他认证类型的扩展点）
  if req.Auth != nil {
    if err := applyFetchAuth(httpReq, req.Auth); err != nil {
      return nil, err
    }
  }

  client := &http.Client{} // 不设置整体超时，跟随调用方上下文取消
  resp, err := client.Do(httpReq)
  if err != nil {
    return nil, fmt.Errorf("failed to download: %v", err)
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  // 下载到同目录临时文件，成功后再重命名，避免留下残缺文件
  tmpFile, err := os.CreateTemp(dir, ".fetch-*")
  if err != nil {
    return nil, fmt.Errorf("failed to create temp file: %v", err)
  }
  tmpPath := tmpFile.Name()
  defer func() {
    tmpFile.Close()
    os.Remove(tmpPath)
  }()

  // 边下载边计算校验码
  md5Hash := md5.New()
  sha1Hash := sha1.New()
  sha256Hash := sha256.New()
  writer := io.MultiWriter(tmpFile, md5Hash, sha1Hash, sha256Hash)

  written, err := io.Copy(writer, resp.Body)
  if err != nil {
    return nil, fmt.Errorf("failed to write file content: %v", err)
  }
  if err := tmpFile.Close(); err != nil {
    return nil, fmt.Errorf("failed to close temp file: %v", err)
  }

  // 校验码比对（提供了才校验）
  checksums := map[string]hash.Hash{"md5": md5Hash, "sha1": sha1Hash, "sha256": sha256Hash}
  expected := map[string]string{"md5": req.MD5, "sha1": req.SHA1, "sha256": req.SHA256}
  for algo, want := range expected {
    if want == "" {
      continue
    }
    got := hex.EncodeToString(checksums[algo].Sum(nil))
    if !strings.EqualFold(got, want) {
      return nil, fmt.Errorf("%s checksum mismatch: expected %s, got %s", algo, want, got)
    }
  }

  // 重命名到目标路径
  if err := os.Rename(tmpPath, dstPath); err != nil {
    return nil, fmt.Errorf("failed to move file to destination: %v", err)
  }

  // 设置文件权限
  if req.Mode != "" {
    if err := os.Chmod(dstPath, mode); err != nil {
      return nil, fmt.Errorf("failed to chmod: %v", err)
    }
  }

  // 设置文件所属用户
  if req.Owner != "" {
    if err := chownPath(dstPath, req.Owner); err != nil {
      return nil, fmt.Errorf("failed to chown: %v", err)
    }
  }

  return &FetchResult{
    URL:    req.URL,
    Path:   dstPath,
    Size:   written,
    MD5:    hex.EncodeToString(md5Hash.Sum(nil)),
    SHA1:   hex.EncodeToString(sha1Hash.Sum(nil)),
    SHA256: hex.EncodeToString(sha256Hash.Sum(nil)),
  }, nil
}

// applyFetchAuth 按认证类型为下载请求设置认证信息。
// 新增认证类型时在此扩展（如 bearer token、digest 等）。
func applyFetchAuth(httpReq *http.Request, auth *FetchAuth) error {
  switch strings.ToLower(auth.Type) {
  case "basic":
    httpReq.SetBasicAuth(auth.Username, auth.Password)
    return nil
  default:
    return fmt.Errorf("unsupported auth type: %q", auth.Type)
  }
}

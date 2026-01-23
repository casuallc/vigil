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
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
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
func (m *Manager) ListFiles(path string, maxDepth int) ([]FileInfo, error) {
  var files []FileInfo

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
    fileInfo := FileInfo{
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

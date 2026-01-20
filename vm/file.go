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

package vm

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// FormatFileSize 将文件大小转换为人类可读的格式
func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// FileManager 表示文件管理功能
type FileManager struct {
	// 不再需要SSHClient，直接操作本地文件
}

// NewFileManager 创建一个新的文件管理器
func NewFileManager() *FileManager {
	return &FileManager{}
}

// UploadFile 上传文件到VM
func (fm *FileManager) UploadFile(sourcePath, targetPath string) error {
	// 打开源文件
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// 调用从Reader上传的方法
	return fm.UploadFileFromReader(sourceFile, targetPath)
}

// UploadFileFromReader 从io.Reader上传文件到VM
func (fm *FileManager) UploadFileFromReader(reader io.Reader, targetPath string) error {
	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// 创建目标文件
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// 传输文件内容
	_, err = io.Copy(targetFile, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// 设置文件权限
	if err := targetFile.Chmod(0644); err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	log.Printf("File uploaded: %s", targetPath)
	return nil
}

// DownloadFile 从VM下载文件
func (fm *FileManager) DownloadFile(sourcePath, targetPath string) error {
	// 打开源文件
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// 获取源文件信息
	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// 创建目标文件
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// 传输文件内容
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// 设置文件权限
	if err := targetFile.Chmod(fileInfo.Mode()); err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	log.Printf("File downloaded: %s -> %s", sourcePath, targetPath)
	return nil
}

// ListFiles 列出VM上的文件列表
func (fm *FileManager) ListFiles(path string, maxDepth int) ([]*FileInfo, error) {
	// 定义一个递归函数来列出文件和子目录
	var listFilesRecursive func(string, int) ([]*FileInfo, error)
	listFilesRecursive = func(currentPath string, currentDepth int) ([]*FileInfo, error) {
		// 读取目录内容
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}

		var files []*FileInfo
		for _, entry := range entries {
			// 获取文件信息
			info, err := entry.Info()
			if err != nil {
				log.Printf("Warning: failed to get file info for %s: %v", entry.Name(), err)
				continue
			}

			// 构建完整路径
			fullPath := filepath.Join(currentPath, entry.Name())

			// 构建FileInfo对象
			file := &FileInfo{
				Name:    entry.Name(),
				Path:    fullPath,
				Size:    info.Size(),
				IsDir:   entry.IsDir(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
				Depth:   currentDepth,
			}

			files = append(files, file)

			// 如果是目录且还未达到最大深度，递归列出子目录
			if entry.IsDir() && currentDepth < maxDepth {
				subFiles, err := listFilesRecursive(fullPath, currentDepth+1)
				if err != nil {
					log.Printf("Warning: failed to list subdirectory %s: %v", fullPath, err)
					continue
				}
				files = append(files, subFiles...)
			}
		}

		return files, nil
	}

	// 从深度0开始递归列出文件
	return listFilesRecursive(path, 0)
}

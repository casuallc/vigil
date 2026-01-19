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
	"strings"

	"github.com/pkg/sftp"
)

// FileManager 表示文件管理功能
type FileManager struct {
	sshClient *SSHClient
}

// NewFileManager 创建一个新的文件管理器
func NewFileManager(sshClient *SSHClient) *FileManager {
	return &FileManager{
		sshClient: sshClient,
	}
}

// UploadFile 上传文件到VM
func (fm *FileManager) UploadFile(localPath, remotePath string) error {
	if fm.sshClient.client == nil {
		return fmt.Errorf("SSH client not connected")
	}

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	// 获取本地文件信息
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// 在VM上创建文件
	session, err := fm.sshClient.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 获取SFTP客户端
	sftpClient, err := sftp.NewClient(fm.sshClient.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	remoteDir := filepath.Dir(remotePath)
	_, err = fm.sshClient.ExecuteCommand(fmt.Sprintf("mkdir -p %s", remoteDir))
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %v", err)
	}
	defer remoteFile.Close()

	// 传输文件内容
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	// 设置文件权限
	if err := remoteFile.Chmod(fileInfo.Mode()); err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	log.Printf("File uploaded: %s -> %s", localPath, remotePath)
	return nil
}

// DownloadFile 从VM下载文件
func (fm *FileManager) DownloadFile(remotePath, localPath string) error {
	if fm.sshClient.client == nil {
		return fmt.Errorf("SSH client not connected")
	}

	// 获取SFTP客户端
	sftpClient, err := sftp.NewClient(fm.sshClient.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer remoteFile.Close()

	// 获取远程文件信息
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get remote file info: %v", err)
	}

	// 确保本地目录存在
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// 传输文件内容
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	// 设置文件权限
	if err := localFile.Chmod(fileInfo.Mode()); err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	log.Printf("File downloaded: %s -> %s", remotePath, localPath)
	return nil
}

// ListFiles 列出VM上的文件列表
func (fm *FileManager) ListFiles(remotePath string) ([]*FileInfo, error) {
	if fm.sshClient.client == nil {
		return nil, fmt.Errorf("SSH client not connected")
	}

	// 执行ls命令获取文件列表
	cmd := fmt.Sprintf("ls -la %s", remotePath)
	output, err := fm.sshClient.ExecuteCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	// 解析ls命令输出
	files := parseLSOutput(output, remotePath)
	return files, nil
}

// parseLSOutput 解析ls命令的输出
func parseLSOutput(output, basePath string) []*FileInfo {
	var files []*FileInfo
	lines := strings.Split(output, "\n")

	// 跳过前两行（总大小和当前目录）
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 解析ls -la输出的每一行
		// 格式：-rw-r--r-- 1 user group 12345 Jan 1 00:00 filename
		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}

		mode := parts[0]
		sizeStr := parts[4]
		modTime := strings.Join(parts[5:8], " ")
		name := strings.Join(parts[8:], " ")

		// 跳过当前目录和父目录
		if name == "." || name == ".." {
			continue
		}

		// 解析文件大小
		size := int64(0)
		fmt.Sscanf(sizeStr, "%d", &size)

		// 确定是否为目录
		isDir := strings.HasPrefix(mode, "d")

		// 构建完整路径
		path := filepath.Join(basePath, name)

		file := &FileInfo{
			Name:    name,
			Path:    path,
			Size:    size,
			IsDir:   isDir,
			Mode:    mode,
			ModTime: modTime,
		}

		files = append(files, file)
	}

	return files
}

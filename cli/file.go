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

package cli

import (
  "fmt"
  "os"
  "strings"

  "github.com/spf13/cobra"
)

// setupFileCommand 设置file命令组
func (c *CLI) setupFileCommand() *cobra.Command {
  fileCmd := &cobra.Command{
    Use:   "file",
    Short: "File operations",
    Long:  "Perform file operations",
  }

  fileCmd.AddCommand(c.setupFileUploadCommand())
  fileCmd.AddCommand(c.setupFileDownloadCommand())
  fileCmd.AddCommand(c.setupFileListCommand())
  fileCmd.AddCommand(c.setupFileDeleteCommand())
  fileCmd.AddCommand(c.setupFileCopyCommand())
  fileCmd.AddCommand(c.setupFileMoveCommand())

  return fileCmd
}

// setupFileUploadCommand 设置file upload命令
func (c *CLI) setupFileUploadCommand() *cobra.Command {
  var sourcePath, targetPath string

  uploadCmd := &cobra.Command{
    Use:   "upload",
    Short: "Upload a file",
    Long:  "Upload a local file",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileUpload(sourcePath, targetPath)
    },
  }

  uploadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
  uploadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

  uploadCmd.MarkFlagRequired("source")
  uploadCmd.MarkFlagRequired("target")

  return uploadCmd
}

// setupFileDownloadCommand 设置file download命令
func (c *CLI) setupFileDownloadCommand() *cobra.Command {
  var sourcePath, targetPath string

  downloadCmd := &cobra.Command{
    Use:   "download",
    Short: "Download a file",
    Long:  "Download a file to local",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileDownload(sourcePath, targetPath)
    },
  }

  downloadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
  downloadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

  downloadCmd.MarkFlagRequired("source")
  downloadCmd.MarkFlagRequired("target")

  return downloadCmd
}

// setupFileListCommand 设置file list命令
func (c *CLI) setupFileListCommand() *cobra.Command {
  var path string
  var maxDepth int

  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List files",
    Long:  "List files in a directory",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileList(path, maxDepth)
    },
  }

  listCmd.Flags().StringVarP(&path, "path", "p", "/", "Directory path")
  listCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "Maximum depth for recursive listing (0 means no recursion)")

  return listCmd
}

// setupFileDeleteCommand 设置file delete命令
func (c *CLI) setupFileDeleteCommand() *cobra.Command {
  var path string

  deleteCmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete a file",
    Long:  "Delete a file",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileDelete(path)
    },
  }

  deleteCmd.Flags().StringVarP(&path, "path", "p", "", "File path")
  deleteCmd.MarkFlagRequired("path")

  return deleteCmd
}

// setupFileCopyCommand 设置file copy命令
func (c *CLI) setupFileCopyCommand() *cobra.Command {
  var sourcePath, targetPath string

  copyCmd := &cobra.Command{
    Use:   "copy",
    Short: "Copy a file",
    Long:  "Copy a file from source to target",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileCopy(sourcePath, targetPath)
    },
  }

  copyCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
  copyCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

  copyCmd.MarkFlagRequired("source")
  copyCmd.MarkFlagRequired("target")

  return copyCmd
}

// setupFileMoveCommand 设置file move命令
func (c *CLI) setupFileMoveCommand() *cobra.Command {
  var sourcePath, targetPath string

  moveCmd := &cobra.Command{
    Use:   "move",
    Short: "Move a file",
    Long:  "Move a file from source to target",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleFileMove(sourcePath, targetPath)
    },
  }

  moveCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
  moveCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

  moveCmd.MarkFlagRequired("source")
  moveCmd.MarkFlagRequired("target")

  return moveCmd
}

// handleFileUpload 处理file upload命令
func (c *CLI) handleFileUpload(sourcePath, targetPath string) error {
  // 检查源文件是否存在
  if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
    return fmt.Errorf("source file does not exist: %s", sourcePath)
  }

  // 上传文件
  if err := c.client.FileUpload(sourcePath, targetPath); err != nil {
    return fmt.Errorf("failed to upload file: %v", err)
  }

  fmt.Printf("File uploaded successfully: %s -> %s\n", sourcePath, targetPath)
  return nil
}

// handleFileDownload 处理file download命令
func (c *CLI) handleFileDownload(sourcePath, targetPath string) error {
  // 下载文件
  if err := c.client.FileDownload(sourcePath, targetPath); err != nil {
    return fmt.Errorf("failed to download file: %v", err)
  }

  fmt.Printf("File downloaded successfully: %s -> %s\n", sourcePath, targetPath)
  return nil
}

// handleFileList 处理file list命令
func (c *CLI) handleFileList(path string, maxDepth int) error {
  // 获取文件列表
  files, err := c.client.FileList(path, maxDepth)
  if err != nil {
    return fmt.Errorf("failed to list files: %v", err)
  }

  // 打印文件列表
  fmt.Printf("%-10s %-10s %-20s %-30s\n", "Type", "Size", "Modified Time", "Name")
  fmt.Println("----------------------------------------------------------------------------------")
  for _, file := range files {
    fileType := "-"
    if file.IsDir {
      fileType = "dir"
      file.Name += "/"
    } else {
      fileType = "file"
    }
    // 添加缩进，限制文件名长度
    indent := strings.Repeat("  ", file.Depth)
    fmt.Printf("%-10s %-10s %-20s %s%-30s\n", fileType, FormatFileSize(file.Size), file.ModTime, indent, file.Name)
  }

  return nil
}

// handleFileDelete 处理file delete命令
func (c *CLI) handleFileDelete(path string) error {
  // 删除文件
  if err := c.client.FileDelete(path); err != nil {
    return fmt.Errorf("failed to delete file: %v", err)
  }

  fmt.Printf("File deleted successfully: %s\n", path)
  return nil
}

// handleFileCopy 处理file copy命令
func (c *CLI) handleFileCopy(sourcePath, targetPath string) error {
  // 复制文件
  if err := c.client.FileCopy(sourcePath, targetPath); err != nil {
    return fmt.Errorf("failed to copy file: %v", err)
  }

  fmt.Printf("File copied successfully: %s -> %s\n", sourcePath, targetPath)
  return nil
}

// handleFileMove 处理file move命令
func (c *CLI) handleFileMove(sourcePath, targetPath string) error {
  // 移动文件
  if err := c.client.FileMove(sourcePath, targetPath); err != nil {
    return fmt.Errorf("failed to move file: %v", err)
  }

  fmt.Printf("File moved successfully: %s -> %s\n", sourcePath, targetPath)
  return nil
}

// FormatFileSize 格式化文件大小
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

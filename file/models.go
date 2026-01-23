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

import "time"

// TransferRequest 表示文件传输请求
type TransferRequest struct {
  SourcePath      string `json:"source_path"`
  DestinationPath string `json:"destination_path"`
  Overwrite       bool   `json:"overwrite"`
}

// ListRequest 表示文件列表请求
type ListRequest struct {
  Path     string `json:"path"`
  MaxDepth int    `json:"max_depth"`
}

// Info 表示文件或目录的信息
type Info struct {
  Name    string `json:"name"`
  Path    string `json:"path"`
  Size    int64  `json:"size"`
  IsDir   bool   `json:"is_dir"`
  Mode    string `json:"mode"`
  ModTime string `json:"mod_time"`
  Depth   int    `json:"depth"`
}

// TransferResponse 表示文件传输响应
type TransferResponse struct {
  Success bool   `json:"success"`
  Message string `json:"message"`
  Error   string `json:"error,omitempty"`
}

// OperationLog 表示文件操作日志
type OperationLog struct {
  Operation       string    `json:"operation"` // upload, download, list
  SourcePath      string    `json:"source_path"`
  DestinationPath string    `json:"destination_path"`
  Username        string    `json:"username"`
  Timestamp       time.Time `json:"timestamp"`
  Success         bool      `json:"success"`
  Error           string    `json:"error,omitempty"`
}

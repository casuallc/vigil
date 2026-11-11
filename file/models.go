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

// FetchAuth 表示下载认证配置（可选）。目前仅支持 basic，预留其他认证类型的扩展点
type FetchAuth struct {
  Type     string `json:"type"`               // 认证类型：basic（必填，若提供 auth）
  Username string `json:"username,omitempty"` // basic 认证用户名
  Password string `json:"password,omitempty"` // basic 认证密码
}

// FetchRequest 表示基于元数据从远程 URL 下载文件的请求
type FetchRequest struct {
  URL     string            `json:"url"`               // 下载地址（必填，仅支持 http/https）
  Dir     string            `json:"dir"`               // 目标目录（可选，默认服务端工作目录）
  Name    string            `json:"name"`              // 保存的文件名（可选，默认取 URL 路径的最后一段）
  Mode    string            `json:"mode"`              // 文件权限，八进制字符串，如 "0755"（可选）
  Owner   string            `json:"owner"`             // 文件所属用户，格式 "user" 或 "user:group"（可选，仅 Unix）
  MD5     string            `json:"md5"`               // MD5 校验码（可选）
  SHA1    string            `json:"sha1"`              // SHA1 校验码（可选）
  SHA256  string            `json:"sha256"`            // SHA256 校验码（可选）
  Auth    *FetchAuth        `json:"auth,omitempty"`    // 下载认证配置（可选）
  Headers map[string]string `json:"headers,omitempty"` // 额外的下载请求头（可选）
}

// FetchResult 表示单个文件的下载结果
type FetchResult struct {
  URL     string `json:"url"`
  Path    string `json:"path,omitempty"`
  Size    int64  `json:"size,omitempty"`
  MD5     string `json:"md5,omitempty"`
  SHA1    string `json:"sha1,omitempty"`
  SHA256  string `json:"sha256,omitempty"`
  Success bool   `json:"success"`
  Error   string `json:"error,omitempty"`
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

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

package audit

import (
  "time"
)

// ActionType 审计操作类型
type ActionType string

// 审计操作类型常量
const (
  ActionLogin            ActionType = "login"
  ActionLogout           ActionType = "logout"
  ActionVMAdd            ActionType = "vm_add"
  ActionVMDelete         ActionType = "vm_delete"
  ActionVMUpdate         ActionType = "vm_update"
  ActionVMGet            ActionType = "vm_get"
  ActionVMList           ActionType = "vm_list"
  ActionVMSSH            ActionType = "vm_ssh"
  ActionFileUpload       ActionType = "file_upload"
  ActionFileDownload     ActionType = "file_download"
  ActionFileList         ActionType = "file_list"
  ActionGroupAdd         ActionType = "group_add"
  ActionGroupDelete      ActionType = "group_delete"
  ActionGroupUpdate      ActionType = "group_update"
  ActionGroupGet         ActionType = "group_get"
  ActionGroupList        ActionType = "group_list"
  ActionPermissionAdd    ActionType = "permission_add"
  ActionPermissionRemove ActionType = "permission_remove"
  ActionPermissionList   ActionType = "permission_list"
  ActionProcessManage    ActionType = "process_manage"
  ActionResourceMonitor  ActionType = "resource_monitor"
  ActionConfigManage     ActionType = "config_manage"
  ActionCommandExecute   ActionType = "command_exec"
)

// StatusType 审计操作状态
type StatusType string

// 审计操作状态常量
const (
  StatusSuccess StatusType = "success"
  StatusFailed  StatusType = "failed"
)

// LogEntry 审计日志条目
type LogEntry struct {
  ID        string      `json:"id"`
  Timestamp time.Time   `json:"timestamp"`
  User      string      `json:"user"`
  IP        string      `json:"ip"`
  Action    ActionType  `json:"action"`
  Resource  string      `json:"resource"`
  Status    StatusType  `json:"status"`
  Message   string      `json:"message"`
  Details   interface{} `json:"details,omitempty"`
}

// NewLogEntry 创建新的审计日志条目
func NewLogEntry(user, ip string, action ActionType, resource string, status StatusType, message string, details interface{}) *LogEntry {
  return &LogEntry{
    ID:        generateID(),
    Timestamp: time.Now(),
    User:      user,
    IP:        ip,
    Action:    action,
    Resource:  resource,
    Status:    status,
    Message:   message,
    Details:   details,
  }
}

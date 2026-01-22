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
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "syscall"
  "time"

  "github.com/casuallc/vigil/vm"
  "github.com/gorilla/websocket"
  "github.com/spf13/cobra"
  "golang.org/x/term"
)

// setupVMCommands 设置所有VM相关的命令
func (c *CLI) setupVMCommands() *cobra.Command {
  // VM command - 作为父命令来组织所有VM相关的子命令
  vmCmd := &cobra.Command{
    Use:   "vm",
    Short: "VM management operations",
    Long:  "Manage and interact with virtual machines",
  }

  // 添加子命令
  vmCmd.AddCommand(c.setupVMAddCommand())
  vmCmd.AddCommand(c.setupVMListCommand())
  vmCmd.AddCommand(c.setupVMGetCommand())
  vmCmd.AddCommand(c.setupVMDeleteCommand())
  vmCmd.AddCommand(c.setupVMSSHCommand())
  vmCmd.AddCommand(c.setupVMFileCommand())
  vmCmd.AddCommand(c.setupVMUpdateCommand())
  vmCmd.AddCommand(c.setupVMGroupCommand())
  vmCmd.AddCommand(c.setupVMPermissionCommand())

  return vmCmd
}

// setupVMListCommand 设置vm list命令
func (c *CLI) setupVMListCommand() *cobra.Command {
  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List all VMs",
    Long:  "List all virtual machines",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMList()
    },
  }

  return listCmd
}

// setupVMAddCommand 设置vm add命令
func (c *CLI) setupVMAddCommand() *cobra.Command {
  var name, ip, username, password, keyPath string
  var port int

  addCmd := &cobra.Command{
    Use:   "add",
    Short: "Add a new VM",
    Long:  "Add a new virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMAdd(name, ip, port, username, password, keyPath)
    },
  }

  addCmd.Flags().StringVarP(&name, "name", "n", "", "VM name")
  addCmd.Flags().StringVarP(&ip, "ip", "i", "", "VM IP address")
  addCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
  addCmd.Flags().StringVarP(&username, "username", "u", "root", "SSH username")
  addCmd.Flags().StringVarP(&password, "password", "P", "", "SSH password")
  addCmd.Flags().StringVarP(&keyPath, "key-path", "k", "", "SSH private key path")

  addCmd.MarkFlagRequired("name")
  addCmd.MarkFlagRequired("ip")

  return addCmd
}

// setupVMDeleteCommand 设置vm delete命令
func (c *CLI) setupVMDeleteCommand() *cobra.Command {
  deleteCmd := &cobra.Command{
    Use:   "delete [name]",
    Short: "Delete a VM",
    Long:  "Delete a virtual machine",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMDelete(args[0])
    },
  }

  return deleteCmd
}

// setupVMGetCommand 设置vm get命令
func (c *CLI) setupVMGetCommand() *cobra.Command {
  getCmd := &cobra.Command{
    Use:   "get [name]",
    Short: "Get VM details",
    Long:  "Get details of a virtual machine",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGet(args[0])
    },
  }

  return getCmd
}

// setupVMSSHCommand 设置vm ssh命令
func (c *CLI) setupVMSSHCommand() *cobra.Command {
  var vmName string

  sshCmd := &cobra.Command{
    Use:   "ssh",
    Short: "SSH into VM",
    Long:  "SSH into a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMSSH(vmName)
    },
  }

  sshCmd.Flags().StringVarP(&vmName, "vm", "v", "", "VM name")

  return sshCmd
}

// setupVMFileCommand 设置vm file命令组
func (c *CLI) setupVMFileCommand() *cobra.Command {
  fileCmd := &cobra.Command{
    Use:   "file",
    Short: "VM file operations",
    Long:  "Perform file operations on a virtual machine",
  }

  fileCmd.AddCommand(c.setupVMFileUploadCommand())
  fileCmd.AddCommand(c.setupVMFileDownloadCommand())
  fileCmd.AddCommand(c.setupVMFileListCommand())

  return fileCmd
}

// setupVMFileUploadCommand 设置vm file upload命令
func (c *CLI) setupVMFileUploadCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var sourcePath, targetPath string

  uploadCmd := &cobra.Command{
    Use:   "upload",
    Short: "Upload a file to VM via SFTP",
    Long:  "Upload a local file to VM using SFTP",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileUpload(vmNames, groupNames, sourcePath, targetPath)
    },
  }

  uploadCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  uploadCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  uploadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
  uploadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path on VM")

  uploadCmd.MarkFlagRequired("source")
  uploadCmd.MarkFlagRequired("target")

  return uploadCmd
}

// setupVMFileDownloadCommand 设置vm file download命令
func (c *CLI) setupVMFileDownloadCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var sourcePath, targetPath string

  downloadCmd := &cobra.Command{
    Use:   "download",
    Short: "Download a file from VM via SFTP",
    Long:  "Download a file from VM to local using SFTP",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileDownload(vmNames, groupNames, sourcePath, targetPath)
    },
  }

  downloadCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  downloadCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  downloadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path on VM")
  downloadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

  downloadCmd.MarkFlagRequired("source")
  downloadCmd.MarkFlagRequired("target")

  return downloadCmd
}

// setupVMFileListCommand 设置vm file list命令
func (c *CLI) setupVMFileListCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string
  var maxDepth int

  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List files on VM via SFTP",
    Long:  "List files in a directory on VM using SFTP",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileList(vmNames, groupNames, path, maxDepth)
    },
  }

  listCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  listCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  listCmd.Flags().StringVarP(&path, "path", "p", "/", "Directory path on VM")
  listCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "Maximum depth for recursive listing (0 means no recursion)")

  return listCmd
}

// setupVMUpdateCommand 设置vm update命令
func (c *CLI) setupVMUpdateCommand() *cobra.Command {
  var name, password, keyPath string

  updateCmd := &cobra.Command{
    Use:   "update",
    Short: "Update VM credentials",
    Long:  "Update password and key path for a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMUpdate(name, password, keyPath)
    },
  }

  updateCmd.Flags().StringVarP(&name, "name", "n", "", "VM name")
  updateCmd.Flags().StringVarP(&password, "password", "P", "", "SSH password")
  updateCmd.Flags().StringVarP(&keyPath, "key-path", "k", "", "SSH private key path")

  updateCmd.MarkFlagRequired("name")

  return updateCmd
}

// setupVMGroupCommand 设置vm group命令组
func (c *CLI) setupVMGroupCommand() *cobra.Command {
  groupCmd := &cobra.Command{
    Use:   "group",
    Short: "VM group operations",
    Long:  "Manage VM groups",
  }

  groupCmd.AddCommand(c.setupVMGroupAddCommand())
  groupCmd.AddCommand(c.setupVMGroupListCommand())
  groupCmd.AddCommand(c.setupVMGroupGetCommand())
  groupCmd.AddCommand(c.setupVMGroupUpdateCommand())
  groupCmd.AddCommand(c.setupVMGroupDeleteCommand())

  return groupCmd
}

// setupVMGroupAddCommand 设置vm group add命令
func (c *CLI) setupVMGroupAddCommand() *cobra.Command {
  var name, description string
  var vms []string

  addCmd := &cobra.Command{
    Use:   "add",
    Short: "Add a VM group",
    Long:  "Add a new VM group",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGroupAdd(name, description, vms)
    },
  }

  addCmd.Flags().StringVarP(&name, "name", "n", "", "Group name")
  addCmd.Flags().StringVarP(&description, "description", "d", "", "Group description")
  addCmd.Flags().StringArrayVarP(&vms, "vms", "v", []string{}, "VM names (can be used multiple times)")

  addCmd.MarkFlagRequired("name")
  addCmd.MarkFlagRequired("vms")

  return addCmd
}

// setupVMGroupListCommand 设置vm group list命令
func (c *CLI) setupVMGroupListCommand() *cobra.Command {
  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List VM groups",
    Long:  "List all VM groups",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGroupList()
    },
  }

  return listCmd
}

// setupVMGroupGetCommand 设置vm group get命令
func (c *CLI) setupVMGroupGetCommand() *cobra.Command {
  var name string

  getCmd := &cobra.Command{
    Use:   "get",
    Short: "Get VM group details",
    Long:  "Get details of a VM group",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGroupGet(name)
    },
  }

  getCmd.Flags().StringVarP(&name, "name", "n", "", "Group name")
  getCmd.MarkFlagRequired("name")

  return getCmd
}

// setupVMGroupUpdateCommand 设置vm group update命令
func (c *CLI) setupVMGroupUpdateCommand() *cobra.Command {
  var name, description string
  var vms []string

  updateCmd := &cobra.Command{
    Use:   "update",
    Short: "Update a VM group",
    Long:  "Update details of a VM group",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGroupUpdate(name, description, vms)
    },
  }

  updateCmd.Flags().StringVarP(&name, "name", "n", "", "Group name")
  updateCmd.Flags().StringVarP(&description, "description", "d", "", "Group description")
  updateCmd.Flags().StringArrayVarP(&vms, "vms", "v", []string{}, "VM names (can be used multiple times)")

  updateCmd.MarkFlagRequired("name")

  return updateCmd
}

// setupVMGroupDeleteCommand 设置vm group delete命令
func (c *CLI) setupVMGroupDeleteCommand() *cobra.Command {
  var name string

  deleteCmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete a VM group",
    Long:  "Delete a VM group",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGroupDelete(name)
    },
  }

  deleteCmd.Flags().StringVarP(&name, "name", "n", "", "Group name")
  deleteCmd.MarkFlagRequired("name")

  return deleteCmd
}

// setupVMPermissionCommand 设置vm permission命令组
func (c *CLI) setupVMPermissionCommand() *cobra.Command {
  permissionCmd := &cobra.Command{
    Use:   "permission",
    Short: "VM permission operations",
    Long:  "Manage permissions for virtual machines",
  }

  permissionCmd.AddCommand(c.setupVMPermissionAddCommand())
  permissionCmd.AddCommand(c.setupVMPermissionRemoveCommand())
  permissionCmd.AddCommand(c.setupVMPermissionListCommand())
  permissionCmd.AddCommand(c.setupVMPermissionCheckCommand())

  return permissionCmd
}

// setupVMPermissionAddCommand 设置vm permission add命令
func (c *CLI) setupVMPermissionAddCommand() *cobra.Command {
  var vmName, username string
  var permissions []string

  addCmd := &cobra.Command{
    Use:   "add",
    Short: "Add permissions to a VM",
    Long:  "Add permissions for a user on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMPermissionAdd(vmName, username, permissions)
    },
  }

  addCmd.Flags().StringVarP(&vmName, "vm", "n", "", "VM name")
  addCmd.Flags().StringVarP(&username, "user", "u", "", "Username")
  addCmd.Flags().StringArrayVarP(&permissions, "permissions", "p", []string{}, "Permissions")

  addCmd.MarkFlagRequired("vm")
  addCmd.MarkFlagRequired("user")
  addCmd.MarkFlagRequired("permissions")

  return addCmd
}

// setupVMPermissionRemoveCommand 设置vm permission remove命令
func (c *CLI) setupVMPermissionRemoveCommand() *cobra.Command {
  var vmName, username string
  var permissions []string

  removeCmd := &cobra.Command{
    Use:   "remove",
    Short: "Remove permissions from a VM",
    Long:  "Remove permissions for a user on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMPermissionRemove(vmName, username, permissions)
    },
  }

  removeCmd.Flags().StringVarP(&vmName, "vm", "n", "", "VM name")
  removeCmd.Flags().StringVarP(&username, "user", "u", "", "Username")
  removeCmd.Flags().StringArrayVarP(&permissions, "permissions", "p", []string{}, "Permissions")

  removeCmd.MarkFlagRequired("vm")
  removeCmd.MarkFlagRequired("user")
  removeCmd.MarkFlagRequired("permissions")

  return removeCmd
}

// setupVMPermissionListCommand 设置vm permission list命令
func (c *CLI) setupVMPermissionListCommand() *cobra.Command {
  var vmName string

  listCmd := &cobra.Command{
    Use:   "list",
    Short: "List permissions on a VM",
    Long:  "List all permissions for a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMPermissionList(vmName)
    },
  }

  listCmd.Flags().StringVarP(&vmName, "vm", "n", "", "VM name")

  listCmd.MarkFlagRequired("vm")

  return listCmd
}

// setupVMPermissionCheckCommand 设置vm permission check命令
func (c *CLI) setupVMPermissionCheckCommand() *cobra.Command {
  var vmName, username, permission string

  checkCmd := &cobra.Command{
    Use:   "check",
    Short: "Check permissions on a VM",
    Long:  "Check if a user has a specific permission on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMPermissionCheck(vmName, username, permission)
    },
  }

  checkCmd.Flags().StringVarP(&vmName, "vm", "n", "", "VM name")
  checkCmd.Flags().StringVarP(&username, "user", "u", "", "Username")
  checkCmd.Flags().StringVarP(&permission, "permission", "p", "", "Permission to check")

  checkCmd.MarkFlagRequired("vm")
  checkCmd.MarkFlagRequired("user")
  checkCmd.MarkFlagRequired("permission")

  return checkCmd
}

// ------------------------- Command Handlers -------------------------

// handleVMList 处理vm list命令
func (c *CLI) handleVMList() error {
  vms, err := c.client.ListVMs()
  if err != nil {
    return fmt.Errorf("failed to list VMs: %v", err)
  }

  // 打印VM列表
  fmt.Printf("%-20s %-15s %-10s %-15s\n", "Name", "IP", "Port", "Username")
  fmt.Println("--------------------------------------------------------")
  for _, vm := range vms {
    fmt.Printf("%-20s %-15s %-10d %-15s\n", vm.Name, vm.IP, vm.Port, vm.Username)
  }

  return nil
}

// handleVMAdd 处理vm add命令
func (c *CLI) handleVMAdd(name, ip string, port int, username, password, keyPath string) error {
  // 创建VM
  vm, err := c.client.AddVM(name, ip, port, username, password, keyPath)
  if err != nil {
    return fmt.Errorf("failed to add VM: %v", err)
  }

  fmt.Printf("VM added successfully: %s\n", vm.Name)
  return nil
}

// handleVMGet 处理vm get命令
func (c *CLI) handleVMGet(name string) error {
  // 获取VM
  vm, err := c.client.GetVM(name)
  if err != nil {
    return fmt.Errorf("failed to get VM: %v", err)
  }

  // 打印VM详情
  fmt.Printf("Name:        %s\n", vm.Name)
  fmt.Printf("IP:          %s\n", vm.IP)
  fmt.Printf("Port:        %d\n", vm.Port)
  fmt.Printf("Username:    %s\n", vm.Username)

  // 不显示明文密码，显示占位符
  password := "[REDACTED]"
  if vm.Password == "" {
    password = ""
  }
  fmt.Printf("Password:    %s\n", password)

  // 不显示明文密钥路径，显示占位符
  keyPath := "[REDACTED]"
  if vm.KeyPath == "" {
    keyPath = ""
  }
  fmt.Printf("KeyPath:     %s\n", keyPath)

  fmt.Printf("Status:      %s\n", vm.Status)
  fmt.Printf("CreatedAt:   %s\n", vm.CreatedAt)
  fmt.Printf("UpdatedAt:   %s\n", vm.UpdatedAt)

  return nil
}

// handleVMUpdate 处理vm update命令
func (c *CLI) handleVMUpdate(name, password, keyPath string) error {
  // 更新VM
  if err := c.client.UpdateVM(name, password, keyPath); err != nil {
    return fmt.Errorf("failed to update VM: %v", err)
  }

  fmt.Printf("VM updated successfully: %s\n", name)
  return nil
}

// handleVMGroupAdd 处理vm group add命令
// AI Modified
func (c *CLI) handleVMGroupAdd(name, description string, vms []string) error {
  // 添加组
  if err := c.client.AddGroup(name, description, vms); err != nil {
    return fmt.Errorf("failed to add group: %v", err)
  }

  fmt.Printf("Group added successfully: %s\n", name)
  return nil
}

// handleVMGroupList 处理vm group list命令
// AI Modified
func (c *CLI) handleVMGroupList() error {
  // 列出所有组
  groups, err := c.client.ListGroups()
  if err != nil {
    return fmt.Errorf("failed to list groups: %v", err)
  }

  // 打印组列表
  fmt.Printf("%s\n", strings.Repeat("-", 80))
  fmt.Printf("%s\n", strings.Repeat("-", 80))
  for _, group := range groups {
    fmt.Printf("Group Name: %s\n", group.Name)
    fmt.Printf("Description: %s\n", group.Description)
    fmt.Printf("VMs: %v\n", group.VMs)
    fmt.Printf("Created At: %s\n", group.CreatedAt.Format(time.RFC3339))
    fmt.Printf("Updated At: %s\n", group.UpdatedAt.Format(time.RFC3339))
    fmt.Printf("%s\n", strings.Repeat("-", 80))
  }

  return nil
}

// handleVMGroupGet 处理vm group get命令
// AI Modified
func (c *CLI) handleVMGroupGet(name string) error {
  // 获取组
  group, err := c.client.GetGroup(name)
  if err != nil {
    return fmt.Errorf("failed to get group: %v", err)
  }

  // 打印组详情
  fmt.Printf("Group Name: %s\n", group.Name)
  fmt.Printf("Description: %s\n", group.Description)
  fmt.Printf("VMs: %v\n", group.VMs)
  fmt.Printf("Created At: %s\n", group.CreatedAt.Format(time.RFC3339))
  fmt.Printf("Updated At: %s\n", group.UpdatedAt.Format(time.RFC3339))

  return nil
}

// handleVMGroupUpdate 处理vm group update命令
// AI Modified
func (c *CLI) handleVMGroupUpdate(name, description string, vms []string) error {
  // 更新组
  if err := c.client.UpdateGroup(name, description, vms); err != nil {
    return fmt.Errorf("failed to update group: %v", err)
  }

  fmt.Printf("Group updated successfully: %s\n", name)
  return nil
}

// handleVMGroupDelete 处理vm group delete命令
// AI Modified
func (c *CLI) handleVMGroupDelete(name string) error {
  // 删除组
  if err := c.client.DeleteGroup(name); err != nil {
    return fmt.Errorf("failed to delete group: %v", err)
  }

  fmt.Printf("Group deleted successfully: %s\n", name)
  return nil
}

// handleVMDelete 处理vm delete命令
func (c *CLI) handleVMDelete(name string) error {
  // 删除VM
  if err := c.client.DeleteVM(name); err != nil {
    return fmt.Errorf("failed to delete VM: %v", err)
  }

  fmt.Printf("VM deleted successfully: %s\n", name)
  return nil
}

// handleVMSSH 处理vm ssh命令
func (c *CLI) handleVMSSH(vmName string) error {
  var selectedVM *vm.VM
  var err error

  // 如果指定了VM名称，直接获取VM信息
  if vmName != "" {
    selectedVM, err = c.client.GetVM(vmName)
    if err != nil {
      return fmt.Errorf("failed to get VM: %v", err)
    }
  } else {
    // 从配置文件加载已添加的VM信息
    vms, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(vms) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 显示VM列表
    fmt.Printf("Available VMs:\n")
    for i, vm := range vms {
      fmt.Printf("  %d. %s@%s:%d\n", i+1, vm.Username, vm.IP, vm.Port)
    }

    // 提示用户选择
    fmt.Printf("\nSelect a VM (1-%d): ", len(vms))

    // 读取用户输入
    var selection int
    _, err = fmt.Scanln(&selection)
    if err != nil {
      return fmt.Errorf("invalid selection: %v", err)
    }

    // 验证选择
    if selection < 1 || selection > len(vms) {
      return fmt.Errorf("invalid selection: must be between 1 and %d", len(vms))
    }

    // 获取选中的VM
    selectedVM = &vms[selection-1]
  }

  // 使用存储的密码，如果没有则提示用户输入
  password := selectedVM.Password
  if password == "" {
    // 提示用户输入密码
    fmt.Printf("Password for %s@%s: ", selectedVM.Username, selectedVM.IP)

    // 隐藏密码输入
    var err error
    password, err = readPassword()
    if err != nil {
      return fmt.Errorf("failed to read password: %v", err)
    }
    fmt.Printf("\n")
  }

  // 建立WebSocket SSH连接
  conn, err := c.client.SSHWebSocket(selectedVM.Name, password)
  if err != nil {
    return fmt.Errorf("failed to establish WebSocket SSH connection: %v", err)
  }
  defer conn.Close()

  // 设置终端为原始模式
  oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
  if err != nil {
    return fmt.Errorf("failed to set terminal to raw mode: %v", err)
  }
  defer term.Restore(int(os.Stdin.Fd()), oldState)

  // 获取当前终端大小
  w, h, err := term.GetSize(int(os.Stdout.Fd()))
  if err != nil {
    return fmt.Errorf("failed to get terminal size: %v", err)
  }

  // 发送窗口大小调整消息
  resizeMsg := map[string]int{"cols": w, "rows": h}
  resizeJSON, _ := json.Marshal(resizeMsg)
  conn.WriteMessage(websocket.TextMessage, []byte("resize:"+string(resizeJSON)))

  // 启动协程处理终端大小变化
  go func() {
    for {
      w, h, err := term.GetSize(int(os.Stdout.Fd()))
      if err != nil {
        continue
      }
      resizeMsg := map[string]int{"cols": w, "rows": h}
      resizeJSON, _ := json.Marshal(resizeMsg)
      conn.WriteMessage(websocket.TextMessage, []byte("resize:"+string(resizeJSON)))
      // 在Windows上，我们简单地休眠一段时间后再次检查
      // 在Unix/Linux上，我们可以使用SIGWINCH信号
      // 这里我们使用一个通用的方法，每100毫秒检查一次
      time.Sleep(100 * time.Millisecond)
    }
  }()

  // 启动协程处理WebSocket消息
  go func() {
    for {
      _, message, err := conn.ReadMessage()
      if err != nil {
        return
      }
      // 输出WebSocket消息到终端
      os.Stdout.Write(message)
    }
  }()

  // 从终端读取输入并发送到WebSocket
  buffer := make([]byte, 1024)
  for {
    n, err := os.Stdin.Read(buffer)
    if err != nil {
      return fmt.Errorf("failed to read from terminal: %v", err)
    }
    if n > 0 {
      err := conn.WriteMessage(websocket.TextMessage, buffer[:n])
      if err != nil {
        return fmt.Errorf("failed to write to WebSocket: %v", err)
      }
    }
  }
}

// readPassword 读取密码输入（隐藏输入内容）
func readPassword() (string, error) {
  bytePassword, err := term.ReadPassword(int(syscall.Stdin))
  if err != nil {
    return "", err
  }
  return string(bytePassword), nil
}

// getTargetVMs 获取目标VM列表，包括直接指定的VM和通过组指定的VM
// AI Modified
func (c *CLI) getTargetVMs(vmNames, groupNames []string) ([]*vm.VM, error) {
  // 检查是否指定了VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    return nil, fmt.Errorf("no VMs or groups specified")
  }

  // 存储所有目标VM的名称，使用map去重
  vmNameMap := make(map[string]bool)

  // 添加直接指定的VM
  for _, vmName := range vmNames {
    vmNameMap[vmName] = true
  }

  // 添加通过组指定的VM
  for _, groupName := range groupNames {
    group, err := c.client.GetGroup(groupName)
    if err != nil {
      return nil, fmt.Errorf("failed to get group %s: %v", groupName, err)
    }

    for _, vmName := range group.VMs {
      vmNameMap[vmName] = true
    }
  }

  // 获取所有VM的详细信息
  var vms []*vm.VM
  for vmName := range vmNameMap {
    vm, err := c.client.GetVM(vmName)
    if err != nil {
      return nil, fmt.Errorf("failed to get VM %s: %v", vmName, err)
    }
    vms = append(vms, vm)
  }

  return vms, nil
}

// handleVMFileUpload 处理vm file upload命令
// AI Modified
func (c *CLI) handleVMFileUpload(vmNames, groupNames []string, sourcePath, targetPath string) error {
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    return fmt.Errorf("you must specify at least one VM or group using --vm or --group flag")
  }

  // 获取目标VM列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有VM并上传文件
  for _, selectedVM := range vms {
    fmt.Printf("Uploading to VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)

    // 创建SSH客户端
    sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
      Host:     selectedVM.IP,
      Port:     selectedVM.Port,
      Username: selectedVM.Username,
      Password: selectedVM.Password,
      KeyPath:  selectedVM.KeyPath,
    })
    if err != nil {
      fmt.Printf("Failed to create SSH client for %s: %v\n", selectedVM.Name, err)
      continue
    }

    // 连接到SSH服务器
    if err := sshClient.Connect(selectedVM.IP, selectedVM.Port); err != nil {
      fmt.Printf("Failed to connect to %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 打开本地文件
    sourceFile, err := os.Open(sourcePath)
    if err != nil {
      fmt.Printf("Failed to open source file: %v\n", err)
      sshClient.Close()
      continue
    }
    defer sourceFile.Close()

    // 上传文件
    if err := sshClient.UploadFile(sourceFile, targetPath); err != nil {
      fmt.Printf("Failed to upload file to %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    fmt.Printf("File uploaded successfully to %s: %s -> %s\n", selectedVM.Name, sourcePath, targetPath)
    sshClient.Close()
  }

  return nil
}

// handleVMFileDownload 处理vm file download命令
// AI Modified
func (c *CLI) handleVMFileDownload(vmNames, groupNames []string, sourcePath, targetPath string) error {
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    return fmt.Errorf("you must specify at least one VM or group using --vm or --group flag")
  }

  // 获取目标VM列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有VM并下载文件
  for _, selectedVM := range vms {
    fmt.Printf("Downloading from VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)

    // 为每个VM创建目标路径，避免文件冲突
    var vmTargetPath string
    if len(vms) > 1 {
      // 如果下载多个VM的文件，为每个VM创建子目录
      vmTargetPath = filepath.Join(targetPath, selectedVM.Name)
      // 创建目录
      if err := os.MkdirAll(vmTargetPath, 0755); err != nil {
        fmt.Printf("Failed to create directory for %s: %v\n", selectedVM.Name, err)
        continue
      }
      // 获取源文件的文件名并添加到目标路径
      _, fileName := filepath.Split(sourcePath)
      vmTargetPath = filepath.Join(vmTargetPath, fileName)
    } else {
      // 如果只下载一个VM的文件，直接使用指定的目标路径
      vmTargetPath = targetPath
    }

    // 创建SSH客户端
    sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
      Host:     selectedVM.IP,
      Port:     selectedVM.Port,
      Username: selectedVM.Username,
      Password: selectedVM.Password,
      KeyPath:  selectedVM.KeyPath,
    })
    if err != nil {
      fmt.Printf("Failed to create SSH client for %s: %v\n", selectedVM.Name, err)
      continue
    }

    // 连接到SSH服务器
    if err := sshClient.Connect(selectedVM.IP, selectedVM.Port); err != nil {
      fmt.Printf("Failed to connect to %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 下载文件
    content, err := sshClient.DownloadFile(sourcePath)
    if err != nil {
      fmt.Printf("Failed to download file from %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 确保目标目录存在
    targetDir := filepath.Dir(vmTargetPath)
    if err := os.MkdirAll(targetDir, 0755); err != nil {
      fmt.Printf("Failed to create target directory %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 写入文件
    if err := os.WriteFile(vmTargetPath, content, 0644); err != nil {
      fmt.Printf("Failed to write file: %v\n", err)
      sshClient.Close()
      continue
    }

    fmt.Printf("File downloaded successfully from %s: %s -> %s\n", selectedVM.Name, sourcePath, vmTargetPath)
    sshClient.Close()
  }

  return nil
}

// handleVMFileList 处理vm file list命令
// AI Modified
func (c *CLI) handleVMFileList(vmNames, groupNames []string, path string, maxDepth int) error {
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    return fmt.Errorf("you must specify at least one VM or group using --vm or --group flag")
  }

  // 获取目标VM列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有VM并列出文件
  for _, selectedVM := range vms {
    fmt.Printf("\nListing files on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)
    fmt.Println(strings.Repeat("=", 80))

    // 创建SSH客户端
    sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
      Host:     selectedVM.IP,
      Port:     selectedVM.Port,
      Username: selectedVM.Username,
      Password: selectedVM.Password,
      KeyPath:  selectedVM.KeyPath,
    })
    if err != nil {
      fmt.Printf("Failed to create SSH client for %s: %v\n", selectedVM.Name, err)
      continue
    }

    // 连接到SSH服务器
    if err := sshClient.Connect(selectedVM.IP, selectedVM.Port); err != nil {
      fmt.Printf("Failed to connect to %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 获取文件列表
    files, err := sshClient.ListFiles(path, maxDepth)
    if err != nil {
      fmt.Printf("Failed to list files on %s: %v\n", selectedVM.Name, err)
      sshClient.Close()
      continue
    }

    // 生成缩进字符串的辅助函数
    getIndent := func(depth int) string {
      if depth == 0 {
        return ""
      }
      return strings.Repeat("  ", depth-1) + "└── "
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
      indent := getIndent(file.Depth + 1)
      fmt.Printf("%-10s %-10s %-20s %s%-30s\n", fileType, vm.FormatFileSize(file.Size), file.ModTime, indent, file.Name)
    }

    sshClient.Close()
    fmt.Println(strings.Repeat("=", 80))
  }

  return nil
}

// handleVMPermissionAdd 处理vm permission add命令
func (c *CLI) handleVMPermissionAdd(vmName, username string, permissions []string) error {
  // 添加权限
  if err := c.client.AddPermission(vmName, username, permissions); err != nil {
    return fmt.Errorf("failed to add permission: %v", err)
  }

  fmt.Printf("Permissions added successfully for user %s on VM %s\n", username, vmName)
  return nil
}

// handleVMPermissionRemove 处理vm permission remove命令
func (c *CLI) handleVMPermissionRemove(vmName, username string, permissions []string) error {
  // 移除权限
  if err := c.client.RemovePermission(vmName, username, permissions); err != nil {
    return fmt.Errorf("failed to remove permission: %v", err)
  }

  fmt.Printf("Permissions removed successfully for user %s on VM %s\n", username, vmName)
  return nil
}

// handleVMPermissionList 处理vm permission list命令
func (c *CLI) handleVMPermissionList(vmName string) error {
  // 获取权限列表
  permissions, err := c.client.ListPermissions(vmName)
  if err != nil {
    return fmt.Errorf("failed to list permissions: %v", err)
  }

  // 打印权限列表
  fmt.Printf("%-20s %-20s\n", "Username", "Permissions")
  fmt.Println("--------------------------------------------------------")
  for _, p := range permissions {
    fmt.Printf("%-20s %-20s\n", p.Username, p.Permissions)
  }

  return nil
}

// handleVMPermissionCheck 处理vm permission check命令
func (c *CLI) handleVMPermissionCheck(vmName, username, permission string) error {
  // 检查权限
  hasPermission, err := c.client.CheckPermission(vmName, username, permission)
  if err != nil {
    return fmt.Errorf("failed to check permission: %v", err)
  }

  if hasPermission {
    fmt.Printf("User %s has permission %s on VM %s\n", username, permission, vmName)
  } else {
    fmt.Printf("User %s does not have permission %s on VM %s\n", username, permission, vmName)
  }

  return nil
}

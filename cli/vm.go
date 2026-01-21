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

	// 添加各个子命令
	vmCmd.AddCommand(c.setupVMListCommand())
	vmCmd.AddCommand(c.setupVMAddCommand())
	vmCmd.AddCommand(c.setupVMDeleteCommand())
	vmCmd.AddCommand(c.setupVMGetCommand())
	vmCmd.AddCommand(c.setupVMSSHCommand())
	vmCmd.AddCommand(c.setupVMSSHExecuteCommand())
	vmCmd.AddCommand(c.setupVMFileCommand())
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
	var name, ip, username string
	var port int

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new VM",
		Long:  "Add a new virtual machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleVMAdd(name, ip, port, username)
		},
	}

	addCmd.Flags().StringVarP(&name, "name", "n", "", "VM name")
	addCmd.Flags().StringVarP(&ip, "ip", "i", "", "VM IP address")
	addCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
	addCmd.Flags().StringVarP(&username, "username", "u", "root", "SSH username")

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
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "SSH into VM interactively",
		Long:  "Interactively select a VM and SSH into it",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleVMSSH()
		},
	}

	return sshCmd
}

// setupVMSSHExecuteCommand 设置vm ssh-execute命令
func (c *CLI) setupVMSSHExecuteCommand() *cobra.Command {
	sshExecuteCmd := &cobra.Command{
		Use:   "ssh-execute <vm-name> <command>",
		Short: "Execute SSH command on VM",
		Long:  "Execute SSH command on a specific VM",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			command := args[1]
			return c.handleVMSSHExecute(vmName, command)
		},
	}

	return sshExecuteCmd
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
	var sourcePath, targetPath string

	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to VM",
		Long:  "Upload a local file to VM",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleVMFileUpload(sourcePath, targetPath)
		},
	}

	uploadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path")
	uploadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

	uploadCmd.MarkFlagRequired("source")
	uploadCmd.MarkFlagRequired("target")

	return uploadCmd
}

// setupVMFileDownloadCommand 设置vm file download命令
func (c *CLI) setupVMFileDownloadCommand() *cobra.Command {
	var sourcePath, targetPath string

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download a file from VM",
		Long:  "Download a file from VM to local",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleVMFileDownload(sourcePath, targetPath)
		},
	}

	downloadCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path on VM")
	downloadCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target file path")

	downloadCmd.MarkFlagRequired("source")
	downloadCmd.MarkFlagRequired("target")

	return downloadCmd
}

// setupVMFileListCommand 设置vm file list命令
func (c *CLI) setupVMFileListCommand() *cobra.Command {
	var path string
	var maxDepth int

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List files on VM",
		Long:  "List files in a directory on VM",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleVMFileList(path, maxDepth)
		},
	}

	listCmd.Flags().StringVarP(&path, "path", "p", "/", "Directory path on VM")
	listCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "Maximum depth for recursive listing (0 means no recursion)")

	return listCmd
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
func (c *CLI) handleVMAdd(name, ip string, port int, username string) error {
	// 创建VM
	vm, err := c.client.AddVM(name, ip, port, username)
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
	fmt.Printf("Status:      %s\n", vm.Status)
	fmt.Printf("CreatedAt:   %s\n", vm.CreatedAt)
	fmt.Printf("UpdatedAt:   %s\n", vm.UpdatedAt)

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
func (c *CLI) handleVMSSH() error {
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
	selectedVM := vms[selection-1]

	// 提示用户输入密码
	fmt.Printf("Password for %s@%s: ", selectedVM.Username, selectedVM.IP)

	// 隐藏密码输入
	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %v", err)
	}
	fmt.Printf("\n")

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

// handleVMSSHExecute 处理vm ssh-execute命令
func (c *CLI) handleVMSSHExecute(vmName, command string) error {
	// 执行命令（通过API）
	output, err := c.client.SSHExecute(vmName, command)
	if err != nil {
		return fmt.Errorf("failed to execute command on VM %s: %v", vmName, err)
	}

	// 输出命令执行结果
	fmt.Println(output)

	return nil
}

// readPassword 读取密码输入（隐藏输入内容）
func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

// handleVMFileUpload 处理vm file upload命令
func (c *CLI) handleVMFileUpload(sourcePath, targetPath string) error {
	// 上传文件
	if err := c.client.FileUpload(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	fmt.Printf("File uploaded successfully: %s -> %s\n", sourcePath, targetPath)
	return nil
}

// handleVMFileDownload 处理vm file download命令
func (c *CLI) handleVMFileDownload(sourcePath, targetPath string) error {
	// 下载文件
	if err := c.client.FileDownload(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	fmt.Printf("File downloaded successfully: %s -> %s\n", sourcePath, targetPath)
	return nil
}

// handleVMFileList 处理vm file list命令
func (c *CLI) handleVMFileList(path string, maxDepth int) error {
	// 获取文件列表
	files, err := c.client.FileList(path, maxDepth)
	if err != nil {
		return fmt.Errorf("failed to list files: %v", err)
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

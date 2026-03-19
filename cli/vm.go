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
  "github.com/casuallc/vigil/common"
  "os"
  "path/filepath"
  "strings"
  "sync"
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
  vmCmd.AddCommand(c.setupVMExecCommand())
  vmCmd.AddCommand(c.setupVMPingCommand())
  vmCmd.AddCommand(c.setupVMSSHConnectionsCommand())
  vmCmd.AddCommand(c.setupVMSSHDisconnectCommand())

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
  var groupNames []string

  addCmd := &cobra.Command{
    Use:   "add",
    Short: "Add a new VM",
    Long:  "Add a new virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMAdd(name, ip, port, username, password, keyPath, groupNames)
    },
  }

  addCmd.Flags().StringVarP(&name, "name", "n", "", "VM name")
  addCmd.Flags().StringVarP(&ip, "ip", "i", "", "VM IP address")
  addCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
  addCmd.Flags().StringVarP(&username, "username", "u", "root", "SSH username")
  addCmd.Flags().StringVarP(&password, "password", "P", "", "SSH password")
  addCmd.Flags().StringVarP(&keyPath, "key-path", "k", "", "SSH private key path")
  addCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")

  addCmd.MarkFlagRequired("name")
  addCmd.MarkFlagRequired("ip")

  return addCmd
}

// setupVMDeleteCommand 设置vm delete命令
func (c *CLI) setupVMDeleteCommand() *cobra.Command {
  var vmName string

  deleteCmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete a VM",
    Long:  "Delete a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMDelete(vmName)
    },
  }

  deleteCmd.Flags().StringVarP(&vmName, "vm", "v", "", "VM name")

  return deleteCmd
}

// setupVMGetCommand 设置vm get命令
func (c *CLI) setupVMGetCommand() *cobra.Command {
  var vmName string

  getCmd := &cobra.Command{
    Use:   "get",
    Short: "Get VM details",
    Long:  "Get details of a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMGet(vmName)
    },
  }

  getCmd.Flags().StringVarP(&vmName, "vm", "v", "", "VM name")

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
  fileCmd.AddCommand(c.setupVMFileCatCommand())
  fileCmd.AddCommand(c.setupVMFileMkdirCommand())
  fileCmd.AddCommand(c.setupVMFileRmCommand())
  fileCmd.AddCommand(c.setupVMFileChmodCommand())
  fileCmd.AddCommand(c.setupVMFileChownCommand())
  fileCmd.AddCommand(c.setupVMFileDeleteCommand())
  fileCmd.AddCommand(c.setupVMFileTouchCommand())
  fileCmd.AddCommand(c.setupVMFileRmdirCommand())

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

// setupVMFileCatCommand 设置 vm file cat 命令
func (c *CLI) setupVMFileCatCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string

  catCmd := &cobra.Command{
    Use:   "cat",
    Short: "View file contents on VM",
    Long:  "View contents of a file on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileCat(vmNames, groupNames, path)
    },
  }

  catCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  catCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  catCmd.Flags().StringVarP(&path, "path", "p", "", "File path on VM")
  catCmd.MarkFlagRequired("path")

  return catCmd
}

// setupVMFileMkdirCommand 设置 vm file mkdir 命令
func (c *CLI) setupVMFileMkdirCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string
  var parents bool

  mkdirCmd := &cobra.Command{
    Use:   "mkdir",
    Short: "Create directory on VM",
    Long:  "Create a directory on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileMkdir(vmNames, groupNames, path, parents)
    },
  }

  mkdirCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  mkdirCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  mkdirCmd.Flags().StringVarP(&path, "path", "p", "", "Directory path on VM")
  mkdirCmd.Flags().BoolVarP(&parents, "parents", "P", false, "Create parent directories if needed")
  mkdirCmd.MarkFlagRequired("path")

  return mkdirCmd
}

// setupVMFileRmCommand 设置 vm file rm 命令
func (c *CLI) setupVMFileRmCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string
  var recursive bool

  rmCmd := &cobra.Command{
    Use:   "rm",
    Short: "Remove file/directory on VM",
    Long:  "Remove a file or directory from a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileRm(vmNames, groupNames, path, recursive)
    },
  }

  rmCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  rmCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  rmCmd.Flags().StringVarP(&path, "path", "p", "", "File/Directory path on VM")
  rmCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Remove directories recursively")
  rmCmd.MarkFlagRequired("path")

  return rmCmd
}

// setupVMFileChmodCommand 设置 vm file chmod 命令
func (c *CLI) setupVMFileChmodCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path, mode string

  chmodCmd := &cobra.Command{
    Use:   "chmod",
    Short: "Change file permissions on VM",
    Long:  "Change permissions of a file/directory on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileChmod(vmNames, groupNames, path, mode)
    },
  }

  chmodCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  chmodCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  chmodCmd.Flags().StringVarP(&path, "path", "p", "", "File/Directory path on VM")
  chmodCmd.Flags().StringVarP(&mode, "mode", "m", "", "Permission mode (e.g., 755, u+x)")
  chmodCmd.MarkFlagRequired("path")
  chmodCmd.MarkFlagRequired("mode")

  return chmodCmd
}

// setupVMFileChownCommand 设置 vm file chown 命令
func (c *CLI) setupVMFileChownCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path, owner string

  chownCmd := &cobra.Command{
    Use:   "chown",
    Short: "Change file owner on VM",
    Long:  "Change owner/group of a file/directory on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileChown(vmNames, groupNames, path, owner)
    },
  }

  chownCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  chownCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  chownCmd.Flags().StringVarP(&path, "path", "p", "", "File/Directory path on VM")
  chownCmd.Flags().StringVarP(&owner, "owner", "o", "", "Owner/group (e.g., user:group)")
  chownCmd.MarkFlagRequired("path")
  chownCmd.MarkFlagRequired("owner")

  return chownCmd
}

// setupVMFileDeleteCommand 设置 vm file delete 命令
func (c *CLI) setupVMFileDeleteCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string

  deleteCmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete file on VM",
    Long:  "Delete a file from a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileDelete(vmNames, groupNames, path)
    },
  }

  deleteCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  deleteCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  deleteCmd.Flags().StringVarP(&path, "path", "p", "", "File path on VM")
  deleteCmd.MarkFlagRequired("path")

  return deleteCmd
}

// setupVMFileTouchCommand 设置 vm file touch 命令
func (c *CLI) setupVMFileTouchCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string

  touchCmd := &cobra.Command{
    Use:   "touch",
    Short: "Create file on VM",
    Long:  "Create an empty file on a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileTouch(vmNames, groupNames, path)
    },
  }

  touchCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  touchCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  touchCmd.Flags().StringVarP(&path, "path", "p", "", "File path on VM")
  touchCmd.MarkFlagRequired("path")

  return touchCmd
}

// setupVMFileRmdirCommand 设置 vm file rmdir 命令
func (c *CLI) setupVMFileRmdirCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var path string
  var recursive bool

  rmdirCmd := &cobra.Command{
    Use:   "rmdir",
    Short: "Remove directory on VM",
    Long:  "Remove a directory from a virtual machine",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMFileRmdir(vmNames, groupNames, path, recursive)
    },
  }

  rmdirCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  rmdirCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  rmdirCmd.Flags().StringVarP(&path, "path", "p", "", "Directory path on VM")
  rmdirCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Remove directories recursively")
  rmdirCmd.MarkFlagRequired("path")

  return rmdirCmd
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

// setupVMExecCommand 设置 vm exec 命令
func (c *CLI) setupVMExecCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string
  var command string
  var timeout int

  execCmd := &cobra.Command{
    Use:   "exec",
    Short: "Execute command on VM(s)",
    Long:  "Execute a command on one or more VMs",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMExec(vmNames, groupNames, command, timeout)
    },
  }

  execCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  execCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")
  execCmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute")
  execCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Command timeout in seconds")

  execCmd.MarkFlagRequired("command")

  return execCmd
}

// setupVMPingCommand 设置 vm ping 命令
func (c *CLI) setupVMPingCommand() *cobra.Command {
  var vmNames []string
  var groupNames []string

  pingCmd := &cobra.Command{
    Use:   "ping",
    Short: "Ping VM(s)",
    Long:  "Test TCP connection to one or more VMs",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMPing(vmNames, groupNames)
    },
  }

  pingCmd.Flags().StringArrayVarP(&vmNames, "vm", "v", []string{}, "VM names (can be used multiple times)")
  pingCmd.Flags().StringArrayVarP(&groupNames, "group", "g", []string{}, "Group names (can be used multiple times)")

  return pingCmd
}

// setupVMSSHConnectionsCommand 设置 vm ssh connections-by-user 命令
func (c *CLI) setupVMSSHConnectionsCommand() *cobra.Command {
  var vmName, username, clientIP string

  connectionsCmd := &cobra.Command{
    Use:   "ssh connections",
    Short: "List active SSH connections filtered by VM, user or client IP",
    Long:  "List active SSH connections to VMs filtered by VM name, username, or client IP",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMSSHConnections(vmName, username, clientIP)
    },
  }

  connectionsCmd.Flags().StringVarP(&vmName, "vm", "v", "", "Filter connections by VM name")
  connectionsCmd.Flags().StringVarP(&username, "user", "u", "", "Filter connections by username")
  connectionsCmd.Flags().StringVarP(&clientIP, "client-ip", "i", "", "Filter connections by client IP")

  return connectionsCmd
}

// setupVMSSHDisconnectCommand 设置 vm ssh disconnect 命令
func (c *CLI) setupVMSSHDisconnectCommand() *cobra.Command {
  var connID string
  var all bool

  disconnectCmd := &cobra.Command{
    Use:   "ssh disconnect",
    Short: "Close SSH connections",
    Long:  "Close specific or all SSH connections",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleVMSSHDisconnect(connID, all)
    },
  }

  disconnectCmd.Flags().StringVarP(&connID, "id", "i", "", "Connection ID to close")
  disconnectCmd.Flags().BoolVarP(&all, "all", "a", false, "Close all connections")

  return disconnectCmd
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
func (c *CLI) handleVMAdd(name, ip string, port int, username, password, keyPath string, groupNames []string) error {
  // 创建VM
  vm, err := c.client.AddVM(name, ip, port, username, password, keyPath)
  if err != nil {
    return fmt.Errorf("failed to add VM: %v", err)
  }

  // 将VM添加到指定的组中
  if len(groupNames) > 0 {
    for _, groupName := range groupNames {
      // 检查组是否存在
      group, err := c.client.GetGroup(groupName)
      if err != nil {
        // 组不存在，创建新组
        if err := c.client.AddGroup(groupName, "", []string{name}); err != nil {
          fmt.Printf("Warning: Failed to create group %s: %v\n", groupName, err)
          continue
        }
        fmt.Printf("Created group %s and added VM %s to it\n", groupName, name)
      } else {
        // 组存在，检查VM是否已在组中
        vmExists := false
        for _, existingVM := range group.VMs {
          if existingVM == name {
            vmExists = true
            break
          }
        }
        if !vmExists {
          // 将VM添加到组中
          group.VMs = append(group.VMs, name)
          group.UpdatedAt = time.Now()
          if err := c.client.UpdateGroup(groupName, group.Description, group.VMs); err != nil {
            fmt.Printf("Warning: Failed to add VM %s to group %s: %v\n", name, groupName, err)
            continue
          }
          fmt.Printf("Added VM %s to group %s\n", name, groupName)
        }
      }
    }
  }

  fmt.Printf("VM added successfully: %s\n", vm.Name)
  return nil
}

// handleVMGet 处理vm get命令
func (c *CLI) handleVMGet(name string) error {
  var selectedVM *vm.VM
  var err error

  // 如果指定了VM名称，直接获取VM信息
  if name != "" {
    selectedVM, err = c.client.GetVM(name)
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

    // 准备选择项
    var options []SelectOption
    for _, vm := range vms {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    // 获取选中的VM
    selectedVM, err = c.client.GetVM(selectedVMName)
    if err != nil {
      return fmt.Errorf("failed to get VM: %v", err)
    }
  }

  // 打印VM详情
  fmt.Printf("Name:        %s\n", selectedVM.Name)
  fmt.Printf("IP:          %s\n", selectedVM.IP)
  fmt.Printf("Port:        %d\n", selectedVM.Port)
  fmt.Printf("Username:    %s\n", selectedVM.Username)

  // 不显示明文密码，显示占位符
  password := "[REDACTED]"
  if selectedVM.Password == "" {
    password = ""
  }
  fmt.Printf("Password:    %s\n", password)

  // 不显示明文密钥路径，显示占位符
  keyPath := "[REDACTED]"
  if selectedVM.KeyPath == "" {
    keyPath = ""
  }
  fmt.Printf("KeyPath:     %s\n", keyPath)

  fmt.Printf("Status:      %s\n", selectedVM.Status)
  fmt.Printf("CreatedAt:   %s\n", selectedVM.CreatedAt)
  fmt.Printf("UpdatedAt:   %s\n", selectedVM.UpdatedAt)

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
func (c *CLI) handleVMGroupAdd(name, description string, vms []string) error {
  // 添加组
  if err := c.client.AddGroup(name, description, vms); err != nil {
    return fmt.Errorf("failed to add group: %v", err)
  }

  fmt.Printf("Group added successfully: %s\n", name)
  return nil
}

// handleVMGroupList 处理vm group list命令
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
func (c *CLI) handleVMGroupUpdate(name, description string, vms []string) error {
  // 更新组
  if err := c.client.UpdateGroup(name, description, vms); err != nil {
    return fmt.Errorf("failed to update group: %v", err)
  }

  fmt.Printf("Group updated successfully: %s\n", name)
  return nil
}

// handleVMGroupDelete 处理vm group delete命令
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
  var selectedVMName string

  // 如果指定了VM名称，直接使用
  if name != "" {
    selectedVMName = name
  } else {
    // 从配置文件加载已添加的VM信息
    vms, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(vms) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range vms {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err = Select(SelectConfig{
      Label:    "Select a VM to delete",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }
  }

  // 删除VM
  if err := c.client.DeleteVM(selectedVMName); err != nil {
    return fmt.Errorf("failed to delete VM: %v", err)
  }

  fmt.Printf("VM deleted successfully: %s\n", selectedVMName)
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

    // 准备选择项
    var options []SelectOption
    for _, vm := range vms {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    // 获取选中的VM
    selectedVM, err = c.client.GetVM(selectedVMName)
    if err != nil {
      return fmt.Errorf("failed to get VM: %v", err)
    }
  }

  // 不提供密码，让服务器自己使用存储的密码
  // 建立WebSocket SSH连接
  conn, err := c.client.SSHWebSocket(selectedVM.Name)
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

  // 连接状态管理
  var (
    connClosed bool
    mu         sync.Mutex
  )

  // 获取当前终端大小
  w, h, err := term.GetSize(int(os.Stdout.Fd()))
  if err != nil {
    return fmt.Errorf("failed to get terminal size: %v", err)
  }

  // 发送窗口大小调整消息
  resizeMsg := map[string]int{"cols": w, "rows": h}
  resizeJSON, _ := json.Marshal(resizeMsg)
  if err := writeWebSocketMessage(conn, &connClosed, &mu, websocket.TextMessage, []byte("resize:"+string(resizeJSON))); err != nil {
    fmt.Println("close")
    return nil
  }

  // 启动协程处理终端大小变化
  go func() {
    for {
      // 检查连接是否关闭
      mu.Lock()
      if connClosed {
        mu.Unlock()
        return
      }
      mu.Unlock()

      w, h, err := term.GetSize(int(os.Stdout.Fd()))
      if err != nil {
        continue
      }
      resizeMsg := map[string]int{"cols": w, "rows": h}
      resizeJSON, _ := json.Marshal(resizeMsg)
      if err := writeWebSocketMessage(conn, &connClosed, &mu, websocket.TextMessage, []byte("resize:"+string(resizeJSON))); err != nil {
        // 连接已关闭，退出协程
        return
      }
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
        // 连接关闭，不报错
        fmt.Println("close")
        // 标记连接为关闭
        mu.Lock()
        connClosed = true
        mu.Unlock()
        // 恢复终端状态
        term.Restore(int(os.Stdin.Fd()), oldState)
        // 退出程序
        os.Exit(0)
      }
      // 输出WebSocket消息到终端
      os.Stdout.Write(message)
    }
  }()

  // 从终端读取输入并发送到WebSocket
  buffer := make([]byte, 1024)
  for {
    // 检查连接是否关闭
    mu.Lock()
    if connClosed {
      mu.Unlock()
      return nil
    }
    mu.Unlock()

    n, err := os.Stdin.Read(buffer)
    if err != nil {
      // 终端读取错误，可能是用户按下了Ctrl+C
      fmt.Println("close")
      return nil
    }
    if n > 0 {
      if err := writeWebSocketMessage(conn, &connClosed, &mu, websocket.TextMessage, buffer[:n]); err != nil {
        // 写入WebSocket错误，可能是连接已关闭
        fmt.Println("close")
        return nil
      }
    }
  }
}

// writeWebSocketMessage 安全地写入WebSocket消息，避免并发写入
func writeWebSocketMessage(conn *websocket.Conn, connClosed *bool, mu *sync.Mutex, messageType int, data []byte) error {
  mu.Lock()
  defer mu.Unlock()

  if *connClosed {
    return fmt.Errorf("connection closed")
  }

  err := conn.WriteMessage(messageType, data)
  if err != nil {
    // 标记连接为关闭
    *connClosed = true
    return err
  }

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

// getTargetVMs 获取目标VM列表，包括直接指定的VM和通过组指定的VM
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
func (c *CLI) handleVMFileUpload(vmNames, groupNames []string, sourcePath, targetPath string) error {
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to upload to",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    // 添加到VM列表
    vmNames = append(vmNames, selectedVMName)
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

  // 遍历所有VM并上传文件（通过服务器中转）
  for _, selectedVM := range vms {
    fmt.Printf("Uploading to VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)

    // 通过服务器中转上传文件
    if err := c.client.VMFileUpload(selectedVM.Name, sourcePath, targetPath); err != nil {
      fmt.Printf("Failed to upload file to %s: %v\n", selectedVM.Name, err)
      continue
    }

    fmt.Printf("File uploaded successfully to %s: %s -> %s\n", selectedVM.Name, sourcePath, targetPath)
  }

  return nil
}

// handleVMFileDownload 处理vm file download命令
func (c *CLI) handleVMFileDownload(vmNames, groupNames []string, sourcePath, targetPath string) error {
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to download from",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    // 添加到VM列表
    vmNames = append(vmNames, selectedVMName)
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

  // 遍历所有VM并下载文件（通过服务器中转）
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

    // 通过服务器中转下载文件
    if err := c.client.VMFileDownload(selectedVM.Name, sourcePath, vmTargetPath); err != nil {
      fmt.Printf("Failed to download file from %s: %v\n", selectedVM.Name, err)
      continue
    }

    fmt.Printf("File downloaded successfully from %s: %s -> %s\n", selectedVM.Name, sourcePath, vmTargetPath)
  }

  return nil
}

// handleVMFileList 处理vm file list命令
func (c *CLI) handleVMFileList(vmNames, groupNames []string, path string, maxDepth int) error {
  if !strings.HasSuffix(path, "/") {
    path = path + "/"
  }
  // 检查是否提供了目标VM或组
  if len(vmNames) == 0 && len(groupNames) == 0 {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to list files from",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    // 添加到VM列表
    vmNames = append(vmNames, selectedVMName)
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

  // 遍历所有VM并列出文件（通过服务器中转）
  for _, selectedVM := range vms {
    fmt.Printf("\nListing files on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)
    fmt.Println(strings.Repeat("=", 80))

    // 通过服务器中转获取文件列表
    files, err := c.client.VMFileList(selectedVM.Name, path, maxDepth)
    if err != nil {
      fmt.Printf("Failed to list files on %s: %v\n", selectedVM.Name, err)
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
      fmt.Printf("%-10s %-10s %-20s %s%-30s\n", fileType, common.FormatFileSize(file.Size), common.FormatFileTime(file.ModTime), indent, file.Name)
    }

    fmt.Println(strings.Repeat("=", 80))
  }

  return nil
}

// handleVMPermissionAdd 处理vm permission add命令
func (c *CLI) handleVMPermissionAdd(vmName, username string, permissions []string) error {
  // 如果没有指定VM名称，让用户选择
  if vmName == "" {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to add permission to",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    vmName = selectedVMName
  }

  // 添加权限
  if err := c.client.AddPermission(vmName, username, permissions); err != nil {
    return fmt.Errorf("failed to add permission: %v", err)
  }

  fmt.Printf("Permissions added successfully for user %s on VM %s\n", username, vmName)
  return nil
}

// handleVMPermissionRemove 处理vm permission remove命令
func (c *CLI) handleVMPermissionRemove(vmName, username string, permissions []string) error {
  // 如果没有指定VM名称，让用户选择
  if vmName == "" {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to remove permission from",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    vmName = selectedVMName
  }

  // 移除权限
  if err := c.client.RemovePermission(vmName, username, permissions); err != nil {
    return fmt.Errorf("failed to remove permission: %v", err)
  }

  fmt.Printf("Permissions removed successfully for user %s on VM %s\n", username, vmName)
  return nil
}

// handleVMPermissionList 处理vm permission list命令
func (c *CLI) handleVMPermissionList(vmName string) error {
  // 如果没有指定VM名称，让用户选择
  if vmName == "" {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to list permissions for",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    vmName = selectedVMName
  }

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
  // 如果没有指定VM名称，让用户选择
  if vmName == "" {
    // 从配置文件加载已添加的VM信息
    allVMs, err := c.client.ListVMs()
    if err != nil {
      return fmt.Errorf("failed to list VMs: %v", err)
    }

    if len(allVMs) == 0 {
      return fmt.Errorf("no VMs available. Please add VMs first using 'bbx-cli vm add'")
    }

    // 准备选择项
    var options []SelectOption
    for _, vm := range allVMs {
      options = append(options, SelectOption{
        Value: vm.Name,
        Label: fmt.Sprintf("%s@%s:%d", vm.Username, vm.IP, vm.Port),
      })
    }

    // 使用Select函数让用户选择
    _, selectedVMName, err := Select(SelectConfig{
      Label:    "Select a VM to check permission for",
      Items:    options,
      PageSize: 10,
    })
    if err != nil {
      return fmt.Errorf("failed to select VM: %v", err)
    }

    vmName = selectedVMName
  }

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

// handleVMExec 处理 vm exec 命令
func (c *CLI) handleVMExec(vmNames, groupNames []string, command string, timeout int) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并执行命令
  for _, selectedVM := range vms {
    fmt.Printf("Executing on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Command: %s (timeout: %ds)\n", command, timeout)

    // 执行命令
    output, err := c.client.VMExec(selectedVM.Name, command, timeout)
    if err != nil {
      fmt.Printf("Failed to execute command on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("Output from %s:\n%s\n", selectedVM.Name, output)
    fmt.Println(strings.Repeat("-", 80))
  }

  return nil
}

// handleVMPing 处理 vm ping 命令
func (c *CLI) handleVMPing(vmNames, groupNames []string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并执行 ping 测试
  for _, selectedVM := range vms {
    fmt.Printf("Pinging VM: %s (%s:%d)...\n", selectedVM.Name, selectedVM.IP, selectedVM.Port)

    // 执行 ping 测试
    result, err := c.client.VMPing(selectedVM.Name)
    if err != nil {
      fmt.Printf("Ping failed: %v\n", err)
      continue
    }

    if result.Success {
      fmt.Printf("Ping successful: latency=%v, status=%s\n", result.Latency, result.Status)
    } else {
      fmt.Printf("Ping failed: status=%s, message=%s\n", result.Status, result.Message)
    }
  }

  return nil
}

// handleVMFileCat 处理 vm file cat 命令
func (c *CLI) handleVMFileCat(vmNames, groupNames []string, path string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并查看文件内容
  for _, selectedVM := range vms {
    fmt.Printf("Viewing file on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)
    fmt.Println(strings.Repeat("=", 80))

    // 执行 cat 命令
    output, err := c.client.VMExec(selectedVM.Name, fmt.Sprintf("cat %s", path), 30)
    if err != nil {
      fmt.Printf("Failed to view file on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("%s\n", output)
    fmt.Println(strings.Repeat("=", 80))
  }

  return nil
}

// handleVMFileMkdir 处理 vm file mkdir 命令
func (c *CLI) handleVMFileMkdir(vmNames, groupNames []string, path string, parents bool) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 构建命令
  cmd := "mkdir"
  if parents {
    cmd += " -p"
  }
  cmd += fmt.Sprintf(" %s", path)

  // 遍历所有 VM 并创建目录
  for _, selectedVM := range vms {
    fmt.Printf("Creating directory on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Command: %s\n", cmd)

    // 执行命令
    output, err := c.client.VMExec(selectedVM.Name, cmd, 30)
    if err != nil {
      fmt.Printf("Failed to create directory on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("Directory created successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileRm 处理 vm file rm 命令
func (c *CLI) handleVMFileRm(vmNames, groupNames []string, path string, recursive bool) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 构建命令
  cmd := "rm"
  if recursive {
    cmd += " -rf"
  }
  cmd += fmt.Sprintf(" %s", path)

  // 遍历所有 VM 并删除文件
  for _, selectedVM := range vms {
    fmt.Printf("Removing on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Command: %s\n", cmd)

    // 执行命令
    output, err := c.client.VMExec(selectedVM.Name, cmd, 30)
    if err != nil {
      fmt.Printf("Failed to remove on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("Removed successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileChmod 处理 vm file chmod 命令
func (c *CLI) handleVMFileChmod(vmNames, groupNames []string, path, mode string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 构建命令
  cmd := fmt.Sprintf("chmod %s %s", mode, path)

  // 遍历所有 VM 并修改权限
  for _, selectedVM := range vms {
    fmt.Printf("Changing permissions on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Command: %s\n", cmd)

    // 执行命令
    output, err := c.client.VMExec(selectedVM.Name, cmd, 30)
    if err != nil {
      fmt.Printf("Failed to change permissions on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("Permissions changed successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileChown 处理 vm file chown 命令
func (c *CLI) handleVMFileChown(vmNames, groupNames []string, path, owner string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 构建命令
  cmd := fmt.Sprintf("chown %s %s", owner, path)

  // 遍历所有 VM 并修改所有者
  for _, selectedVM := range vms {
    fmt.Printf("Changing owner on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Command: %s\n", cmd)

    // 执行命令
    output, err := c.client.VMExec(selectedVM.Name, cmd, 30)
    if err != nil {
      fmt.Printf("Failed to change owner on %s: %v\n", selectedVM.Name, err)
      fmt.Printf("Output: %s\n", output)
      continue
    }

    fmt.Printf("Owner changed successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileDelete 处理 vm file delete 命令
func (c *CLI) handleVMFileDelete(vmNames, groupNames []string, path string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并删除文件
  for _, selectedVM := range vms {
    fmt.Printf("Deleting file on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)

    // 使用 API 客户端删除文件
    err := c.client.VMFileDelete(selectedVM.Name, path)
    if err != nil {
      fmt.Printf("Failed to delete file on %s: %v\n", selectedVM.Name, err)
      continue
    }

    fmt.Printf("File deleted successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileTouch 处理 vm file touch 命令
func (c *CLI) handleVMFileTouch(vmNames, groupNames []string, path string) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并创建文件
  for _, selectedVM := range vms {
    fmt.Printf("Creating file on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)

    // 使用 API 客户端创建文件
    err := c.client.VMFileTouch(selectedVM.Name, path)
    if err != nil {
      fmt.Printf("Failed to create file on %s: %v\n", selectedVM.Name, err)
      continue
    }

    fmt.Printf("File created successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMFileRmdir 处理 vm file rmdir 命令
func (c *CLI) handleVMFileRmdir(vmNames, groupNames []string, path string, recursive bool) error {
  // 获取目标 VM 列表
  vms, err := c.getTargetVMs(vmNames, groupNames)
  if err != nil {
    return err
  }

  // 检查是否找到了目标 VM
  if len(vms) == 0 {
    return fmt.Errorf("no VMs found for the specified names or groups")
  }

  // 遍历所有 VM 并删除目录
  for _, selectedVM := range vms {
    fmt.Printf("Removing directory on VM: %s (%s)\n", selectedVM.Name, selectedVM.IP)
    fmt.Printf("Path: %s\n", path)
    fmt.Printf("Recursive: %v\n", recursive)

    // 使用 API 客户端删除目录
    err := c.client.VMFileRmdir(selectedVM.Name, path, recursive)
    if err != nil {
      fmt.Printf("Failed to remove directory on %s: %v\n", selectedVM.Name, err)
      continue
    }

    fmt.Printf("Directory removed successfully on %s\n", selectedVM.Name)
  }

  return nil
}

// handleVMSSHConnections 处理 vm ssh connections-by-user 命令
func (c *CLI) handleVMSSHConnections(vmName, username, clientIP string) error {
  connections, err := c.client.ListSSHConnections(vmName, username, clientIP)
  if err != nil {
    return fmt.Errorf("failed to list SSH connections: %v", err)
  }

  if len(connections) == 0 {
    fmt.Printf("No active SSH connections found")
    if vmName != "" {
      fmt.Printf(" for VM: %s", vmName)
    }
    if username != "" {
      fmt.Printf(" for user: %s", username)
    }
    if clientIP != "" {
      fmt.Printf(" for client IP: %s", clientIP)
    }
    fmt.Println(".")
    return nil
  }

  fmt.Printf("%-20s %-20s %-20s %-20s %-15s\n", "ID", "VM Name", "Client IP", "Username", "Duration")
  fmt.Println(strings.Repeat("-", 100))

  for _, conn := range connections {
    fmt.Printf("%-20s %-20s %-20s %-20s %-15s\n",
      conn.ID,
      conn.VMName,
      conn.ClientIP,
      conn.Username,
      conn.Duration)
  }

  return nil
}

// handleVMSSHDisconnect 处理 vm ssh disconnect 命令
func (c *CLI) handleVMSSHDisconnect(connID string, all bool) error {
  if all {
    // Close all connections
    count, err := c.client.CloseAllSSHConnections()
    if err != nil {
      return fmt.Errorf("failed to close all SSH connections: %v", err)
    }
    fmt.Printf("Closed %d SSH connections successfully.\n", count)
    return nil
  }

  if connID == "" {
    return fmt.Errorf("connection ID is required when not using --all flag")
  }

  // Close specific connection
  err := c.client.CloseSSHConnection(connID)
  if err != nil {
    return fmt.Errorf("failed to close SSH connection: %v", err)
  }

  fmt.Printf("SSH connection %s closed successfully.\n", connID)
  return nil
}

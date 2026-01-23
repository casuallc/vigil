# VM 命令文档

## 概述

`vm` 命令组用于管理和与虚拟机交互，支持添加、列出、获取、删除虚拟机，以及执行 SSH 登录、文件操作、权限管理等功能。

## 命令结构

```
bbx-cli vm [command]
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `add` | 添加新虚拟机 |
| `list` | 列出所有虚拟机 |
| `get` | 获取虚拟机详情 |
| `delete` | 删除虚拟机 |
| `ssh` | SSH 登录到虚拟机 |
| `file` | 虚拟机文件操作 |
| `update` | 更新虚拟机凭据 |
| `group` | 虚拟机组管理 |
| `permission` | 虚拟机权限管理 |

## 命令详情

### vm add

添加新虚拟机。

**语法：**

```
bbx-cli vm add --name <name> --ip <ip> --port <port> --username <username> --password <password> --key-path <key_path> --group <group_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | VM 名称 | 是 | - |
| `--ip` | `-i` | VM IP 地址 | 是 | - |
| `--port` | `-p` | SSH 端口 | 否 | 22 |
| `--username` | `-u` | SSH 用户名 | 否 | `root` |
| `--password` | `-P` | SSH 密码 | 否 | - |
| `--key-path` | `-k` | SSH 私钥路径 | 否 | - |
| `--group` | `-g` | 组名称（可多次使用） | 否 | - |

**示例：**

```bash
# 添加新虚拟机
./bbx-cli vm add --name vm1 --ip 192.168.1.100 --port 22 --username root --password password123

# 使用密钥文件添加虚拟机
./bbx-cli vm add --name vm2 --ip 192.168.1.101 --username root --key-path ~/.ssh/id_rsa

# 添加虚拟机到指定组
./bbx-cli vm add --name vm3 --ip 192.168.1.102 --username root --password password123 --group web

# 添加虚拟机到多个组
./bbx-cli vm add --name vm4 --ip 192.168.1.103 --username root --password password123 --group web --group db
```

### vm list

列出所有虚拟机。

**语法：**

```
bbx-cli vm list
```

**参数：**

无

**示例：**

```bash
# 列出所有虚拟机
./bbx-cli vm list
```

**输出示例：**

```
Name                 IP              Port       Username
--------------------------------------------------------
vm1                  192.168.1.100   22         root
vm2                  192.168.1.101   22         root
```

### vm get

获取虚拟机详情。

**语法：**

```
bbx-cli vm get --vm <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称 | 否 | - |

**示例：**

```bash
# 获取指定虚拟机详情
./bbx-cli vm get --vm vm1

# 交互式选择虚拟机获取详情
./bbx-cli vm get
```

**输出示例：**

```
Name:        vm1
IP:          192.168.1.100
Port:        22
Username:    root
Password:    [REDACTED]
KeyPath:     [REDACTED]
Status:      active
CreatedAt:   2025-01-01T12:00:00Z
UpdatedAt:   2025-01-01T12:00:00Z
```

### vm delete

删除虚拟机。

**语法：**

```
bbx-cli vm delete --vm <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称 | 否 | - |

**示例：**

```bash
# 删除指定虚拟机
./bbx-cli vm delete --vm vm1

# 交互式选择虚拟机删除
./bbx-cli vm delete
```

### vm ssh

SSH 登录到虚拟机。

**语法：**

```
bbx-cli vm ssh --vm <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称 | 否 | - |

**示例：**

```bash
# SSH 登录到指定虚拟机
./bbx-cli vm ssh --vm vm1

# 交互式选择虚拟机登录
./bbx-cli vm ssh
```

### vm file

虚拟机文件操作命令组，支持上传、下载和列出文件。

**语法：**

```
bbx-cli vm file [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `upload` | 上传文件到虚拟机 |
| `download` | 从虚拟机下载文件 |
| `list` | 列出虚拟机上的文件 |

#### vm file upload

上传文件到虚拟机。

**语法：**

```
bbx-cli vm file upload --vm <vm_name> --group <group_name> --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称（可多次使用） | 否 | - |
| `--group` | `-g` | 组名称（可多次使用） | 否 | - |
| `--source` | `-s` | 源文件路径 | 是 | - |
| `--target` | `-t` | 虚拟机上的目标文件路径 | 是 | - |

**示例：**

```bash
# 上传文件到指定虚拟机
./bbx-cli vm file upload --vm vm1 --source local.txt --target /home/remote.txt

# 上传文件到多个虚拟机
./bbx-cli vm file upload --vm vm1 --vm vm2 --source local.txt --target /home/remote.txt

# 上传文件到组中的所有虚拟机
./bbx-cli vm file upload --group web --source local.txt --target /home/remote.txt
```

#### vm file download

从虚拟机下载文件。

**语法：**

```
bbx-cli vm file download --vm <vm_name> --group <group_name> --source <source_path> --target <target_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称（可多次使用） | 否 | - |
| `--group` | `-g` | 组名称（可多次使用） | 否 | - |
| `--source` | `-s` | 虚拟机上的源文件路径 | 是 | - |
| `--target` | `-t` | 本地目标文件路径 | 是 | - |

**示例：**

```bash
# 从指定虚拟机下载文件
./bbx-cli vm file download --vm vm1 --source /home/remote.txt --target local.txt

# 从多个虚拟机下载文件
./bbx-cli vm file download --vm vm1 --vm vm2 --source /home/remote.txt --target ./downloads/
```

#### vm file list

列出虚拟机上的文件。

**语法：**

```
bbx-cli vm file list --vm <vm_name> --group <group_name> --path <path> --max-depth <depth>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-v` | VM 名称（可多次使用） | 否 | - |
| `--group` | `-g` | 组名称（可多次使用） | 否 | - |
| `--path` | `-p` | 虚拟机上的目录路径 | 否 | `/` |
| `--max-depth` | `-d` | 递归列出的最大深度（0 表示不递归） | 否 | 0 |

**示例：**

```bash
# 列出指定虚拟机上的文件
./bbx-cli vm file list --vm vm1 --path /home

# 递归列出虚拟机上的文件
./bbx-cli vm file list --vm vm1 --path /home --max-depth 2
```

### vm update

更新虚拟机凭据。

**语法：**

```
bbx-cli vm update --name <name> --password <password> --key-path <key_path>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | VM 名称 | 是 | - |
| `--password` | `-P` | SSH 密码 | 否 | - |
| `--key-path` | `-k` | SSH 私钥路径 | 否 | - |

**示例：**

```bash
# 更新虚拟机密码
./bbx-cli vm update --name vm1 --password newpassword123

# 更新虚拟机密钥文件
./bbx-cli vm update --name vm1 --key-path ~/.ssh/new_id_rsa
```

### vm group

虚拟机组管理命令组，支持添加、列出、获取、更新和删除组。

**语法：**

```
bbx-cli vm group [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `add` | 添加虚拟机组 |
| `list` | 列出所有虚拟机组 |
| `get` | 获取虚拟机组详情 |
| `update` | 更新虚拟机组 |
| `delete` | 删除虚拟机组 |

#### vm group add

添加虚拟机组。

**语法：**

```
bbx-cli vm group add --name <name> --description <description> --vms <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | 组名称 | 是 | - |
| `--description` | `-d` | 组描述 | 否 | - |
| `--vms` | `-v` | VM 名称（可多次使用） | 是 | - |

**示例：**

```bash
# 添加虚拟机组
./bbx-cli vm group add --name web --description "Web servers" --vms vm1 --vms vm2
```

#### vm group list

列出所有虚拟机组。

**语法：**

```
bbx-cli vm group list
```

**参数：**

无

**示例：**

```bash
# 列出所有虚拟机组
./bbx-cli vm group list
```

#### vm group get

获取虚拟机组详情。

**语法：**

```
bbx-cli vm group get --name <name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | 组名称 | 是 | - |

**示例：**

```bash
# 获取虚拟机组详情
./bbx-cli vm group get --name web
```

#### vm group update

更新虚拟机组。

**语法：**

```
bbx-cli vm group update --name <name> --description <description> --vms <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | 组名称 | 是 | - |
| `--description` | `-d` | 组描述 | 否 | - |
| `--vms` | `-v` | VM 名称（可多次使用） | 否 | - |

**示例：**

```bash
# 更新虚拟机组描述
./bbx-cli vm group update --name web --description "Web servers group"

# 更新虚拟机组成员
./bbx-cli vm group update --name web --vms vm1 --vms vm2 --vms vm3
```

#### vm group delete

删除虚拟机组。

**语法：**

```
bbx-cli vm group delete --name <name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | 组名称 | 是 | - |

**示例：**

```bash
# 删除虚拟机组
./bbx-cli vm group delete --name web
```

### vm permission

虚拟机权限管理命令组，支持添加、移除、列出和检查权限。

**语法：**

```
bbx-cli vm permission [command]
```

**子命令：**

| 命令 | 描述 |
|------|------|
| `add` | 添加虚拟机权限 |
| `remove` | 移除虚拟机权限 |
| `list` | 列出虚拟机权限 |
| `check` | 检查虚拟机权限 |

#### vm permission add

添加虚拟机权限。

**语法：**

```
bbx-cli vm permission add --vm <vm_name> --user <username> --permissions <permission>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-n` | VM 名称 | 是 | - |
| `--user` | `-u` | 用户名 | 是 | - |
| `--permissions` | `-p` | 权限（可多次使用） | 是 | - |

**示例：**

```bash
# 添加虚拟机权限
./bbx-cli vm permission add --vm vm1 --user user1 --permissions read --permissions write
```

#### vm permission remove

移除虚拟机权限。

**语法：**

```
bbx-cli vm permission remove --vm <vm_name> --user <username> --permissions <permission>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-n` | VM 名称 | 是 | - |
| `--user` | `-u` | 用户名 | 是 | - |
| `--permissions` | `-p` | 权限（可多次使用） | 是 | - |

**示例：**

```bash
# 移除虚拟机权限
./bbx-cli vm permission remove --vm vm1 --user user1 --permissions write
```

#### vm permission list

列出虚拟机权限。

**语法：**

```
bbx-cli vm permission list --vm <vm_name>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-n` | VM 名称 | 是 | - |

**示例：**

```bash
# 列出虚拟机权限
./bbx-cli vm permission list --vm vm1
```

#### vm permission check

检查虚拟机权限。

**语法：**

```
bbx-cli vm permission check --vm <vm_name> --user <username> --permission <permission>
```

**参数：**

| 参数 | 缩写 | 描述 | 必填 | 默认值 |
|------|------|------|------|--------|
| `--vm` | `-n` | VM 名称 | 是 | - |
| `--user` | `-u` | 用户名 | 是 | - |
| `--permission` | `-p` | 要检查的权限 | 是 | - |

**示例：**

```bash
# 检查虚拟机权限
./bbx-cli vm permission check --vm vm1 --user user1 --permission read
```
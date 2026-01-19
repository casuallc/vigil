# VM管理功能概述

Vigil 提供了强大的 VM（虚拟机）管理功能，支持 VM 生命周期管理、SSH 连接、文件管理和权限控制等核心功能。

## 功能特性

### 1. VM生命周期管理
- **创建VM**：基于模板或配置创建新的VM实例
- **启动/停止**：控制VM的运行状态
- **重启**：重启VM实例
- **删除**：删除不再需要的VM实例
- **状态监控**：查看VM的运行状态和资源使用情况

### 2. SSH连接
- **直接连接**：通过SSH直接连接到VM实例
- **密钥认证**：支持使用SSH密钥进行安全认证
- **密码认证**：支持使用密码进行认证

### 3. 文件管理
- **文件上传**：将本地文件上传到VM
- **文件下载**：从VM下载文件到本地
- **文件列表**：查看VM上的文件和目录列表
- **文件权限**：设置文件的访问权限

### 4. 权限控制
- **用户管理**：管理VM的用户账户
- **权限分配**：为用户分配不同级别的访问权限
- **访问控制**：控制用户对VM资源的访问权限

## 命令列表

| 命令 | 描述 |
|------|------|
| `vm list` | 列出所有VM实例 |
| `vm create` | 创建新的VM实例 |
| `vm start` | 启动VM实例 |
| `vm stop` | 停止VM实例 |
| `vm restart` | 重启VM实例 |
| `vm delete` | 删除VM实例 |
| `vm status` | 查看VM实例状态 |
| `vm ssh` | SSH连接到VM实例 |
| `vm file upload` | 上传文件到VM |
| `vm file download` | 从VM下载文件 |
| `vm file list` | 查看VM文件列表 |
| `vm permission add` | 添加VM访问权限 |
| `vm permission remove` | 移除VM访问权限 |
| `vm permission list` | 查看VM访问权限 |

## 基本使用方法

### 列出所有VM

```bash
./bbx-cli vm list
```

### 创建VM

```bash
# 基于默认模板创建VM
./bbx-cli vm create --name my-vm --template default

# 基于自定义配置创建VM
./bbx-cli vm create --name my-vm --cpu 2 --memory 4G --disk 50G
```

### 启动VM

```bash
./bbx-cli vm start my-vm
```

### SSH连接到VM

```bash
# 使用密码认证
./bbx-cli vm ssh my-vm --user admin --password pass123

# 使用密钥认证
./bbx-cli vm ssh my-vm --user admin --key ~/.ssh/id_rsa
```

### 文件上传

```bash
./bbx-cli vm file upload my-vm --local-file /path/to/local/file --remote-path /path/on/vm
```

### 文件下载

```bash
./bbx-cli vm file download my-vm --remote-file /path/on/vm --local-path /path/to/local
```

### 查看文件列表

```bash
./bbx-cli vm file list my-vm --path /path/on/vm
```

### 权限管理

```bash
# 添加用户权限
./bbx-cli vm permission add my-vm --user user1 --permissions read,write,execute

# 列出权限
./bbx-cli vm permission list my-vm

# 移除用户权限
./bbx-cli vm permission remove my-vm --user user1
```

## 配置文件

VM管理功能支持通过配置文件进行高级配置，配置文件默认路径为 `conf/vm_config.yaml`。

配置示例：

```yaml
# VM管理配置
global:
  default_template: "ubuntu-22.04"
  default_cpu: 2
  default_memory: "4G"
  default_disk: "50G"

ssh:
  default_port: 22
  timeout: 30s
  keepalive: 60s

file:
  chunk_size: 10M
  timeout: 300s

permissions:
  default_role: "user"
  roles:
    admin: ["read", "write", "execute", "manage"]
    user: ["read", "write", "execute"]
    guest: ["read"]
```

## 安全注意事项

1. **SSH认证**：建议使用SSH密钥认证而非密码认证，以提高安全性
2. **权限控制**：遵循最小权限原则，只分配必要的权限
3. **文件传输**：确保传输的文件内容安全，避免传输敏感信息
4. **VM隔离**：确保VM之间的网络隔离，防止未授权访问
5. **定期审计**：定期检查VM的访问日志和权限配置

## 故障排除

### SSH连接失败

1. 检查VM是否处于运行状态：
   ```bash
   ./bbx-cli vm status my-vm
   ```

2. 检查SSH配置是否正确：
   ```bash
   ./bbx-cli vm ssh my-vm --verbose
   ```

### 文件传输失败

1. 检查文件路径是否正确
2. 检查用户是否有足够的权限访问目标路径
3. 检查网络连接是否稳定

### 权限错误

1. 检查用户是否存在：
   ```bash
   ./bbx-cli vm permission list my-vm
   ```

2. 检查用户权限是否正确：
   ```bash
   ./bbx-cli vm permission list my-vm --user user1
   ```
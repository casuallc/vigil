# VM SSH命令

`vm ssh` 命令用于通过SSH连接到VM实例，支持多种认证方式和配置选项。

## 命令语法

```bash
./bbx-cli vm ssh [VM名称] [选项]
```

## 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | SSH用户名 | 字符串 | `root` |
| `--password` | `-p` | SSH密码 | 字符串 | 无 |
| `--key` | `-k` | SSH私钥文件路径 | 字符串 | `~/.ssh/id_rsa` |
| `--port` | `-P` | SSH端口 | 整数 | 22 |
| `--timeout` | `-t` | SSH连接超时时间 | 字符串 | `30s` |
| `--command` | `-c` | 在VM上执行的命令（非交互式） | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的连接日志 | 布尔值 | `false` |
| `--config` | `-C` | 指定配置文件路径 | 字符串 | 无 |
| `--host` | `-H` | 服务器地址（如果未使用默认配置） | 字符串 | 无 |

## 认证方式

### 1. 密码认证

使用密码进行SSH认证：

```bash
./bbx-cli vm ssh my-vm --user admin --password pass123
```

### 2. 密钥认证

使用SSH密钥进行安全认证：

```bash
# 使用默认密钥路径
./bbx-cli vm ssh my-vm --user admin

# 使用指定密钥路径
./bbx-cli vm ssh my-vm --user admin --key ~/.ssh/my-key.pem
```

### 3. 配置文件认证

可以在配置文件中存储SSH认证信息，避免每次输入：

```yaml
# vm_ssh_config.yaml
ssh:
  default_user: "admin"
  default_key: "~/.ssh/my-key.pem"
  default_timeout: "60s"
```

使用配置文件：

```bash
./bbx-cli vm ssh my-vm --config vm_ssh_config.yaml
```

## 使用示例

### 1. 交互式SSH连接

```bash
# 基本连接
./bbx-cli vm ssh my-vm

# 指定用户名和端口
./bbx-cli vm ssh my-vm --user ubuntu --port 2222
```

### 2. 执行单个命令

在VM上执行单个命令并返回结果：

```bash
# 查看VM的IP地址
./bbx-cli vm ssh my-vm --command "ip addr show"

# 查看VM的CPU使用率
./bbx-cli vm ssh my-vm --command "top -bn1 | grep load"
```

### 3. 使用环境变量

```bash
# 设置环境变量
VM_NAME="my-vm"
VM_USER="admin"

# 使用环境变量连接
./bbx-cli vm ssh $VM_NAME --user $VM_USER --key ~/.ssh/id_rsa
```

### 4. 高级选项

```bash
# 增加超时时间并显示详细日志
./bbx-cli vm ssh my-vm --timeout 60s --verbose

# 使用非默认服务器地址
./bbx-cli vm ssh my-vm --user admin --host http://192.168.1.100:8181
```

## 配置文件

SSH命令支持通过配置文件进行高级配置，配置文件格式为YAML。

### 配置示例

```yaml
# SSH配置示例
ssh:
  # 默认认证信息
  default_user: "admin"
  default_password: ""  # 不建议在配置文件中存储密码
  default_key: "~/.ssh/id_rsa"
  default_port: 22
  
  # 连接选项
  timeout: "30s"
  keepalive_interval: "30s"
  keepalive_count_max: 3
  
  # 安全选项
  strict_host_key_checking: true
  user_known_hosts_file: "~/.ssh/known_hosts"
  
  # 特定VM的配置
  hosts:
    my-vm:
      user: "ubuntu"
      port: 2222
      key: "~/.ssh/my-vm-key.pem"
    test-vm:
      user: "testuser"
      password: "testpass"  # 仅在测试环境使用
```

### 使用配置文件

```bash
# 使用默认配置文件（~/.vigil/vm_ssh_config.yaml）
./bbx-cli vm ssh my-vm

# 使用指定配置文件
./bbx-cli vm ssh my-vm --config /path/to/ssh_config.yaml
```

## 安全最佳实践

1. **优先使用密钥认证**：避免使用密码认证，减少被暴力破解的风险

2. **使用强密钥**：生成足够长度的SSH密钥（推荐4096位）

   ```bash
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/my-key.pem
   ```

3. **保护私钥**：确保SSH私钥的权限设置正确（仅所有者可读写）

   ```bash
   chmod 600 ~/.ssh/my-key.pem
   ```

4. **定期更换密钥**：定期生成新的SSH密钥并更新VM上的公钥

5. **启用严格的主机密钥检查**：避免中间人攻击

6. **使用非默认端口**：更改默认SSH端口（22）可以减少自动扫描和攻击

7. **限制登录尝试次数**：在VM的SSH配置中限制登录尝试次数

## 故障排除

### 连接超时

```bash
# 增加超时时间并查看详细日志
./bbx-cli vm ssh my-vm --timeout 60s --verbose
```

**可能的原因**：
- VM未运行：使用`./bbx-cli vm status my-vm`检查VM状态
- 网络问题：检查网络连接和防火墙设置
- SSH服务未启动：在VM上执行`systemctl status ssh`检查SSH服务状态

### 认证失败

```bash
# 查看详细认证日志
./bbx-cli vm ssh my-vm --user admin --verbose
```

**可能的原因**：
- 用户名错误：确认VM上存在该用户
- 密码错误：检查密码是否正确
- 密钥错误：确认密钥文件路径和权限正确
- 公钥未添加到VM：将公钥添加到VM的`~/.ssh/authorized_keys`文件

### 权限被拒绝

**可能的原因**：
- 用户没有SSH登录权限：检查`/etc/ssh/sshd_config`中的`AllowUsers`或`DenyUsers`配置
- SELinux或AppArmor限制：检查安全模块的配置
- 主目录权限设置不当：确保用户主目录权限为700，~/.ssh目录权限为700，~/.ssh/authorized_keys权限为600

### 端口错误

```bash
# 检查VM的SSH端口配置
./bbx-cli vm ssh my-vm --port 2222
```

**可能的原因**：
- SSH服务使用了非默认端口：检查VM上的`/etc/ssh/sshd_config`中的`Port`配置
- 防火墙阻止了端口访问：检查防火墙规则

## 高级用法

### 1. 端口转发

```bash
# 本地端口转发
./bbx-cli vm ssh my-vm --command "ssh -L 8080:localhost:80 -N admin@my-vm"

# 远程端口转发
./bbx-cli vm ssh my-vm --command "ssh -R 8080:localhost:80 -N admin@my-vm"
```

### 2. 文件传输

虽然有专门的`vm file`命令，但也可以通过SSH使用scp：

```bash
# 上传文件
./bbx-cli vm ssh my-vm --command "scp local-file admin@my-vm:/remote/path"

# 下载文件
./bbx-cli vm ssh my-vm --command "scp admin@my-vm:/remote/file local-path"
```

### 3. 批量操作

结合shell脚本进行批量VM操作：

```bash
# 批量在多个VM上执行命令
for vm in vm1 vm2 vm3; do
  echo "=== $vm ==="
  ./bbx-cli vm ssh $vm --command "uptime"
done
```

## 相关命令

- `vm status`：查看VM的运行状态
- `vm start`：启动VM实例
- `vm file upload/download`：上传/下载文件
- `vm permission`：管理VM的访问权限
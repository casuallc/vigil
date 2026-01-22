# VM文件管理命令

`vm file` 命令用于在本地和VM之间进行文件传输，以及查看VM上的文件列表。

## 命令列表

### 1. `vm file upload` - 上传文件到VM

### 2. `vm file download` - 从VM下载文件

### 3. `vm file list` - 查看VM上的文件列表

## 1. 文件上传命令

### 命令语法

```bash
./bbx-cli vm file upload [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--local-file` | `-l` | 本地文件路径 | 字符串 | 无（必填） |
| `--remote-path` | `-r` | VM上的目标路径 | 字符串 | 无（必填） |
| `--user` | `-u` | SSH用户名 | 字符串 | `root` |
| `--password` | `-p` | SSH密码 | 字符串 | 无 |
| `--key` | `-k` | SSH私钥文件路径 | 字符串 | `~/.ssh/id_rsa` |
| `--port` | `-P` | SSH端口 | 整数 | 22 |
| `--timeout` | `-t` | 传输超时时间 | 字符串 | `300s` |
| `--chunk-size` | `-c` | 传输块大小 | 字符串 | `10M` |
| `--verbose` | `-v` | 显示详细的传输日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 上传单个文件
./bbx-cli vm file upload my-vm --local-file /local/path/file.txt --remote-path /vms/path/

# 上传文件并指定用户名和密钥
./bbx-cli vm file upload my-vm --local-file /local/path/file.txt --remote-path /vms/path/ --user admin --key ~/.ssh/my-key.pem

# 上传大文件并调整块大小
./bbx-cli vm file upload my-vm --local-file /local/path/large.iso --remote-path /vms/path/ --chunk-size 50M --timeout 1800s
```

## 2. 文件下载命令

### 命令语法

```bash
./bbx-cli vm file download [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--remote-file` | `-r` | VM上的文件路径 | 字符串 | 无（必填） |
| `--local-path` | `-l` | 本地目标路径 | 字符串 | 无（必填） |
| `--user` | `-u` | SSH用户名 | 字符串 | `root` |
| `--password` | `-p` | SSH密码 | 字符串 | 无 |
| `--key` | `-k` | SSH私钥文件路径 | 字符串 | `~/.ssh/id_rsa` |
| `--port` | `-P` | SSH端口 | 整数 | 22 |
| `--timeout` | `-t` | 传输超时时间 | 字符串 | `300s` |
| `--chunk-size` | `-c` | 传输块大小 | 字符串 | `10M` |
| `--verbose` | `-v` | 显示详细的传输日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 下载单个文件
./bbx-cli vm file download my-vm --remote-file /vms/path/file.txt --local-path /local/path/

# 下载文件并指定用户名和密码
./bbx-cli vm file download my-vm --remote-file /vms/path/file.txt --local-path /local/path/ --user admin --password pass123

# 下载大文件并调整超时时间
./bbx-cli vm file download my-vm --remote-file /vms/path/large.iso --local-path /local/path/ --timeout 1800s
```

## 3. 文件列表命令

### 命令语法

```bash
./bbx-cli vm file list [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--path` | `-p` | VM上的目录路径 | 字符串 | `/` |
| `--user` | `-u` | SSH用户名 | 字符串 | `root` |
| `--password` | `-P` | SSH密码 | 字符串 | 无 |
| `--key` | `-k` | SSH私钥文件路径 | 字符串 | `~/.ssh/id_rsa` |
| `--port` | `-o` | SSH端口 | 整数 | 22 |
| `--recursive` | `-r` | 递归列出子目录 | 布尔值 | `false` |
| `--long` | `-l` | 显示详细信息（权限、大小、修改时间） | 布尔值 | `false` |
| `--sort` | `-s` | 排序方式（name, size, time） | 字符串 | `name` |
| `--reverse` | `-R` | 反向排序 | 布尔值 | `false` |
| `--filter` | `-f` | 文件过滤模式（支持通配符） | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的操作日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 列出根目录文件
./bbx-cli vm file list my-vm

# 列出指定目录的详细信息
./bbx-cli vm file list my-vm --path /home/admin --long

# 递归列出目录
./bbx-cli vm file list my-vm --path /var --recursive

# 按大小排序列出文件
./bbx-cli vm file list my-vm --path /tmp --sort size

# 使用过滤模式列出文件
./bbx-cli vm file list my-vm --path /home --filter "*.txt"

# 详细列出并反向排序
./bbx-cli vm file list my-vm --path /etc --long --sort time --reverse
```

## 通用配置

可以在配置文件中设置文件传输的默认参数：

```yaml
# vm_file_config.yaml
file:
  default_user: "admin"
  default_key: "~/.ssh/my-key.pem"
  chunk_size: "20M"
  timeout: "600s"
  transfer_buffer_size: "64K"
  ssh:
    port: 22
    connection_timeout: "30s"
```

使用配置文件：

```bash
./bbx-cli vm file upload my-vm --local-file /local/path/file.txt --remote-path /vms/path/ --config vm_file_config.yaml
```

## 安全注意事项

1. **文件权限**：上传文件后，注意检查VM上的文件权限设置

```bash
# 设置文件权限
./bbx-cli vm ssh my-vm --command "chmod 644 /vms/path/file.txt"
```

2. **敏感数据**：避免传输包含密码、密钥等敏感信息的文件

3. **文件完整性**：传输重要文件后，建议验证文件完整性

```bash
# 计算本地文件哈希值
sha256sum /local/path/file.txt

# 计算VM上文件哈希值
./bbx-cli vm ssh my-vm --command "sha256sum /vms/path/file.txt"
```

## 高级功能

### 1. 批量文件上传

可以使用shell脚本批量上传文件：

```bash
# 上传目录中的所有.txt文件
for file in /local/path/*.txt; do
  ./bbx-cli vm file upload my-vm --local-file "$file" --remote-path /vms/path/
done
```

### 2. 大文件传输优化

对于大文件传输，可以调整块大小和超时时间：

```bash
# 上传大文件，使用更大的块大小和超时时间
./bbx-cli vm file upload my-vm --local-file /local/path/large.iso --remote-path /vms/path/ --chunk-size 50M --timeout 3600s
```

### 3. 目录同步

虽然没有专门的同步命令，但可以结合多个命令实现目录同步：

```bash
# 上传整个目录（需要先在VM上创建目录）
./bbx-cli vm ssh my-vm --command "mkdir -p /vms/path/my-directory"
for file in /local/path/my-directory/*; do
  filename=$(basename "$file")
  ./bbx-cli vm file upload my-vm --local-file "$file" --remote-path /vms/path/my-directory/$filename
done
```

## 故障排除

### 上传失败

```bash
# 查看详细日志
./bbx-cli vm file upload my-vm --local-file /local/path/file.txt --remote-path /vms/path/ --verbose
```

**可能的原因**：
- 本地文件不存在：检查本地文件路径是否正确
- 目标目录不存在：在VM上创建目录后重试
- 权限不足：确保用户有足够权限写入目标路径
- 网络问题：检查网络连接和防火墙设置

### 下载失败

```bash
# 查看详细日志
./bbx-cli vm file download my-vm --remote-file /vms/path/file.txt --local-path /local/path/ --verbose
```

**可能的原因**：
- 远程文件不存在：检查VM上的文件路径是否正确
- 目标目录不存在：创建本地目录后重试
- 权限不足：确保用户有足够权限读取远程文件
- 磁盘空间不足：检查本地磁盘空间

### 文件列表命令失败

```bash
# 查看详细日志
./bbx-cli vm file list my-vm --path /vms/path/ --verbose
```

**可能的原因**：
- 目录不存在：检查VM上的目录路径是否正确
- 权限不足：确保用户有足够权限读取目录
- SSH连接问题：检查SSH配置和网络连接

## 性能优化

1. **调整块大小**：根据网络情况调整块大小，网络好可以使用更大的块大小

2. **并行传输**：对于多个小文件，可以使用并行传输提高效率

3. **压缩传输**：对于文本文件，可以先压缩再传输（需要在VM上解压）

```bash
# 压缩文件
gzip /local/path/file.txt

# 上传压缩文件
./bbx-cli vm file upload my-vm --local-file /local/path/file.txt.gz --remote-path /vms/path/

# 在VM上解压
./bbx-cli vm ssh my-vm --command "gunzip /vms/path/file.txt.gz"
```

## 相关命令

- `vm ssh`：通过SSH连接到VM进行文件操作
- `vm status`：检查VM的运行状态
- `vm permission`：管理VM的访问权限
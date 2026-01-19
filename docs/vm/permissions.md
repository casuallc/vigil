# VM权限控制命令

`vm permission` 命令用于管理VM的访问权限，包括用户管理、权限分配和访问控制等功能。

## 命令列表

### 1. `vm permission add` - 添加VM访问权限

### 2. `vm permission remove` - 移除VM访问权限

### 3. `vm permission list` - 查看VM访问权限

### 4. `vm permission update` - 更新VM访问权限

### 5. `vm permission check` - 检查用户对VM的权限

## 1. 添加权限命令

### 命令语法

```bash
./bbx-cli vm permission add [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | 用户名 | 字符串 | 无（必填） |
| `--permissions` | `-p` | 权限列表（逗号分隔） | 字符串 | 无（必填） |
| `--expires` | `-e` | 权限过期时间（如：1h, 2d, 3w, 4m, 5y） | 字符串 | 永不过期 |
| `--description` | `-d` | 权限描述 | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的操作日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 支持的权限

| 权限 | 描述 |
|------|------|
| `read` | 查看VM信息和状态 |
| `write` | 修改VM配置和状态 |
| `execute` | 执行命令和文件操作 |
| `manage` | 管理VM的权限和生命周期 |
| `ssh` | SSH连接到VM |
| `file_upload` | 上传文件到VM |
| `file_download` | 从VM下载文件 |
| `file_list` | 查看VM文件列表 |
| `all` | 所有权限 |

### 使用示例

```bash
# 为用户添加基本权限
./bbx-cli vm permission add my-vm --user alice --permissions read,ssh

# 为用户添加管理权限
./bbx-cli vm permission add my-vm --user bob --permissions all

# 为用户添加文件操作权限
./bbx-cli vm permission add my-vm --user charlie --permissions file_upload,file_download,file_list

# 添加带有过期时间的权限
./bbx-cli vm permission add my-vm --user dave --permissions read,ssh --expires 24h --description "临时访问权限"
```

## 2. 移除权限命令

### 命令语法

```bash
./bbx-cli vm permission remove [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | 用户名 | 字符串 | 无（必填） |
| `--permissions` | `-p` | 要移除的权限列表（逗号分隔） | 字符串 | 所有权限 |
| `--verbose` | `-v` | 显示详细的操作日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 移除用户的所有权限
./bbx-cli vm permission remove my-vm --user alice

# 移除用户的特定权限
./bbx-cli vm permission remove my-vm --user bob --permissions write,execute

# 移除用户的文件操作权限
./bbx-cli vm permission remove my-vm --user charlie --permissions file_upload,file_download
```

## 3. 列出权限命令

### 命令语法

```bash
./bbx-cli vm permission list [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | 用户名（可选，只显示该用户的权限） | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的权限信息 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 列出VM的所有权限
./bbx-cli vm permission list my-vm

# 列出特定用户的权限
./bbx-cli vm permission list my-vm --user alice

# 列出详细的权限信息
./bbx-cli vm permission list my-vm --verbose
```

## 4. 更新权限命令

### 命令语法

```bash
./bbx-cli vm permission update [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | 用户名 | 字符串 | 无（必填） |
| `--permissions` | `-p` | 新的权限列表（逗号分隔） | 字符串 | 无 |
| `--expires` | `-e` | 新的过期时间 | 字符串 | 无 |
| `--description` | `-d` | 新的权限描述 | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的操作日志 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 更新用户的权限列表
./bbx-cli vm permission update my-vm --user alice --permissions read,write,ssh

# 更新权限的过期时间
./bbx-cli vm permission update my-vm --user bob --expires 7d

# 更新权限描述
./bbx-cli vm permission update my-vm --user charlie --description "开发人员访问权限"

# 同时更新多个权限属性
./bbx-cli vm permission update my-vm --user dave --permissions read,execute --expires 30d --description "测试人员临时权限"
```

## 5. 检查权限命令

### 命令语法

```bash
./bbx-cli vm permission check [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--user` | `-u` | 用户名 | 字符串 | 无（必填） |
| `--permission` | `-p` | 要检查的权限 | 字符串 | 无（可选，检查所有权限） |
| `--verbose` | `-v` | 显示详细的检查结果 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 检查用户对VM的所有权限
./bbx-cli vm permission check my-vm --user alice

# 检查用户是否有特定权限
./bbx-cli vm permission check my-vm --user bob --permission ssh

# 检查用户是否有文件操作权限
./bbx-cli vm permission check my-vm --user charlie --permission file_upload

# 详细检查用户权限
./bbx-cli vm permission check my-vm --user dave --verbose
```

## 权限组管理

### 1. 预定义权限组

系统提供了一些预定义的权限组，方便快速分配权限：

| 权限组 | 包含的权限 | 适用角色 |
|--------|------------|----------|
| `viewer` | `read` | 只读用户 |
| `operator` | `read,write,execute,ssh` | 操作员 |
| `developer` | `read,write,execute,ssh,file_upload,file_download,file_list` | 开发人员 |
| `admin` | `all` | 管理员 |

### 2. 使用权限组

```bash
# 为用户分配viewer权限组
./bbx-cli vm permission add my-vm --user alice --permissions viewer

# 为用户分配admin权限组
./bbx-cli vm permission add my-vm --user bob --permissions admin

# 为用户分配developer权限组并添加额外权限
./bbx-cli vm permission add my-vm --user charlie --permissions developer,manage
```

## 权限继承

权限可以从父资源继承到子资源：

```bash
# 创建VM组
./bbx-cli vm group create dev-group --vms vm1,vm2,vm3

# 为用户分配组权限
./bbx-cli vm permission add dev-group --user alice --permissions developer
```

## 审计日志

可以查看权限操作的审计日志：

```bash
# 查看权限审计日志
./bbx-cli vm audit --type permission --vm my-vm

# 查看特定用户的权限变更日志
./bbx-cli vm audit --type permission --vm my-vm --user alice
```

## 安全最佳实践

1. **最小权限原则**：只分配必要的权限给用户

```bash
# 正确：只为用户分配所需权限
./bbx-cli vm permission add my-vm --user alice --permissions read,ssh

# 错误：分配过多权限
./bbx-cli vm permission add my-vm --user alice --permissions all
```

2. **定期审查权限**：定期检查和更新用户权限

```bash
# 列出所有权限并审查
./bbx-cli vm permission list my-vm --verbose > permissions.txt
```

3. **使用临时权限**：对于临时访问需求，设置过期时间

```bash
# 为临时用户分配24小时权限
./bbx-cli vm permission add my-vm --user temp-user --permissions read,ssh --expires 24h
```

4. **权限隔离**：不同角色的用户分配不同的权限

```bash
# 为管理员分配所有权限
./bbx-cli vm permission add my-vm --user admin --permissions all

# 为普通用户分配有限权限
./bbx-cli vm permission add my-vm --user user --permissions read,file_list
```

5. **使用权限组**：使用预定义的权限组简化权限管理

```bash
# 使用预定义权限组
./bbx-cli vm permission add my-vm --user developer --permissions developer
```

## 故障排除

### 权限不足错误

```bash
# 检查用户权限
./bbx-cli vm permission check my-vm --user alice

# 如果权限不足，添加相应权限
./bbx-cli vm permission add my-vm --user alice --permissions ssh
```

### 权限冲突

```bash
# 查看用户的所有权限
./bbx-cli vm permission list my-vm --user bob --verbose

# 更新权限解决冲突
./bbx-cli vm permission update my-vm --user bob --permissions developer
```

### 过期权限

```bash
# 检查权限是否过期
./bbx-cli vm permission check my-vm --user charlie --verbose

# 延长权限过期时间
./bbx-cli vm permission update my-vm --user charlie --expires 30d
```

## 相关命令

- `vm list`：列出所有VM
- `vm status`：查看VM状态
- `vm ssh`：SSH连接到VM
- `vm file`：文件管理命令
- `vm group`：VM组管理命令
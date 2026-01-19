# VM生命周期管理命令

`vm` 命令用于管理VM的完整生命周期，包括创建、启动、停止、重启和删除等操作。

## 命令列表

### 1. `vm create` - 创建新的VM实例

### 2. `vm start` - 启动VM实例

### 3. `vm stop` - 停止VM实例

### 4. `vm restart` - 重启VM实例

### 5. `vm delete` - 删除VM实例

### 6. `vm status` - 查看VM实例状态

### 7. `vm list` - 列出所有VM实例

## 1. 创建VM命令

### 命令语法

```bash
./bbx-cli vm create [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--name` | `-n` | VM名称 | 字符串 | 无（必填） |
| `--template` | `-t` | VM模板名称 | 字符串 | `default` |
| `--cpu` | `-c` | CPU核心数 | 整数 | 2 |
| `--memory` | `-m` | 内存大小（如：2G, 4G） | 字符串 | `4G` |
| `--disk` | `-d` | 磁盘大小（如：50G, 100G） | 字符串 | `50G` |
| `--image` | `-i` | 操作系统镜像名称 | 字符串 | 无 |
| `--network` | `-N` | 网络配置（如：bridge, nat） | 字符串 | `nat` |
| `--ip` | `-I` | 静态IP地址 | 字符串 | 动态分配 |
| `--user-data` | `-U` | 用户数据文件路径（用于初始化配置） | 字符串 | 无 |
| `--tags` | `-T` | 标签列表（逗号分隔） | 字符串 | 无 |
| `--description` | `-D` | VM描述 | 字符串 | 无 |
| `--verbose` | `-v` | 显示详细的创建过程 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 基于默认模板创建VM
./bbx-cli vm create --name my-vm --template default

# 指定CPU、内存和磁盘创建VM
./bbx-cli vm create --name my-vm --cpu 4 --memory 8G --disk 100G

# 使用指定镜像创建VM
./bbx-cli vm create --name my-vm --image ubuntu-22.04-lts

# 创建VM并指定静态IP
./bbx-cli vm create --name my-vm --network bridge --ip 192.168.1.100

# 创建VM并使用用户数据文件
./bbx-cli vm create --name my-vm --user-data cloud-config.yaml

# 创建VM并添加标签
./bbx-cli vm create --name my-vm --tags "env=dev,project=test"
```

## 2. 启动VM命令

### 命令语法

```bash
./bbx-cli vm start [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--verbose` | `-v` | 显示详细的启动过程 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 启动VM
./bbx-cli vm start my-vm

# 详细启动VM
./bbx-cli vm start my-vm --verbose
```

## 3. 停止VM命令

### 命令语法

```bash
./bbx-cli vm stop [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--force` | `-f` | 强制停止VM（相当于断电） | 布尔值 | `false` |
| `--timeout` | `-t` | 等待VM正常停止的超时时间 | 字符串 | `30s` |
| `--verbose` | `-v` | 显示详细的停止过程 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 正常停止VM
./bbx-cli vm stop my-vm

# 强制停止VM
./bbx-cli vm stop my-vm --force

# 设置停止超时时间
./bbx-cli vm stop my-vm --timeout 60s

# 详细停止VM
./bbx-cli vm stop my-vm --verbose
```

## 4. 重启VM命令

### 命令语法

```bash
./bbx-cli vm restart [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--force` | `-f` | 强制重启VM | 布尔值 | `false` |
| `--timeout` | `-t` | 等待VM正常停止的超时时间 | 字符串 | `30s` |
| `--verbose` | `-v` | 显示详细的重启过程 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 正常重启VM
./bbx-cli vm restart my-vm

# 强制重启VM
./bbx-cli vm restart my-vm --force

# 设置重启超时时间
./bbx-cli vm restart my-vm --timeout 60s

# 详细重启VM
./bbx-cli vm restart my-vm --verbose
```

## 5. 删除VM命令

### 命令语法

```bash
./bbx-cli vm delete [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--force` | `-f` | 强制删除VM（忽略错误） | 布尔值 | `false` |
| `--keep-disk` | `-k` | 删除VM但保留磁盘 | 布尔值 | `false` |
| `--verbose` | `-v` | 显示详细的删除过程 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 删除VM
./bbx-cli vm delete my-vm

# 强制删除VM
./bbx-cli vm delete my-vm --force

# 删除VM但保留磁盘
./bbx-cli vm delete my-vm --keep-disk

# 详细删除VM
./bbx-cli vm delete my-vm --verbose
```

## 6. 查看VM状态命令

### 命令语法

```bash
./bbx-cli vm status [VM名称] [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--verbose` | `-v` | 显示详细的VM状态信息 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 查看VM状态
./bbx-cli vm status my-vm

# 查看详细的VM状态信息
./bbx-cli vm status my-vm --verbose
```

### 状态码说明

| 状态码 | 描述 |
|--------|------|
| `running` | VM正在运行 |
| `stopped` | VM已停止 |
| `paused` | VM已暂停 |
| `suspended` | VM已挂起 |
| `creating` | VM正在创建 |
| `deleting` | VM正在删除 |
| `error` | VM出现错误 |

## 7. 列出VM命令

### 命令语法

```bash
./bbx-cli vm list [选项]
```

### 选项

| 选项 | 简写 | 描述 | 类型 | 默认值 |
|------|------|------|------|--------|
| `--filter` | `-f` | 过滤条件（如：status=running,tag=dev） | 字符串 | 无 |
| `--sort` | `-s` | 排序字段（如：name, cpu, memory） | 字符串 | `name` |
| `--reverse` | `-r` | 反向排序 | 布尔值 | `false` |
| `--verbose` | `-v` | 显示详细的VM信息 | 布尔值 | `false` |
| `--host` | `-H` | 服务器地址 | 字符串 | 无 |

### 使用示例

```bash
# 列出所有VM
./bbx-cli vm list

# 列出正在运行的VM
./bbx-cli vm list --filter "status=running"

# 按CPU核心数排序列出VM
./bbx-cli vm list --sort cpu

# 反向排序列出VM
./bbx-cli vm list --sort memory --reverse

# 列出详细的VM信息
./bbx-cli vm list --verbose

# 按标签过滤列出VM
./bbx-cli vm list --filter "tag=env:dev"
```

## VM模板管理

### 查看可用模板

```bash
./bbx-cli vm template list
```

### 查看模板详情

```bash
./bbx-cli vm template show [模板名称]
```

### 创建自定义模板

```bash
./bbx-cli vm template create --name my-template --image ubuntu-22.04-lts --cpu 4 --memory 8G --disk 100G
```

## VM组管理

### 创建VM组

```bash
./bbx-cli vm group create [组名称] --vms vm1,vm2,vm3
```

### 列出VM组

```bash
./bbx-cli vm group list
```

### 管理组中的VM

```bash
# 添加VM到组
./bbx-cli vm group add [组名称] --vms vm4,vm5

# 从组中移除VM
./bbx-cli vm group remove [组名称] --vms vm3

# 删除VM组
./bbx-cli vm group delete [组名称]
```

### 对组中的VM执行批量操作

```bash
# 启动组中的所有VM
./bbx-cli vm group start [组名称]

# 停止组中的所有VM
./bbx-cli vm group stop [组名称]

# 列出组中的VM
./bbx-cli vm group list [组名称]
```

## 配置文件

可以使用配置文件创建VM：

```yaml
# vm_create_config.yaml
name: "my-vm"
template: "default"
cpu: 4
memory: "8G"
disk: "100G"
network: "bridge"
ip: "192.168.1.100"
tags:
  - "env:dev"
  - "project:test"
description: "开发测试VM"
```

使用配置文件创建VM：

```bash
./bbx-cli vm create --config vm_create_config.yaml
```

## 最佳实践

1. **命名规范**：使用清晰的命名规范标识VM的用途和环境

```bash
# 良好的命名示例
./bbx-cli vm create --name "dev-web-01" --tags "env:dev,role:web"

# 避免使用模糊的命名
./bbx-cli vm create --name "myvm"  # 不推荐
```

2. **资源规划**：根据实际需求合理规划CPU、内存和磁盘资源

```bash
# 为Web服务器分配适当资源
./bbx-cli vm create --name "web-server" --cpu 4 --memory 8G --disk 100G

# 为数据库服务器分配更多资源
./bbx-cli vm create --name "db-server" --cpu 8 --memory 16G --disk 500G
```

3. **使用模板**：使用预定义模板确保VM配置的一致性

```bash
# 创建标准模板
./bbx-cli vm template create --name "web-template" --cpu 4 --memory 8G --disk 100G --image ubuntu-22.04-lts

# 使用模板创建VM
./bbx-cli vm create --name "web-02" --template "web-template"
```

4. **使用标签**：使用标签对VM进行分类和管理

```bash
# 使用标签标记VM
./bbx-cli vm create --name "web-03" --tags "env:prod,role:web,version:v1.0"

# 按标签过滤VM
./bbx-cli vm list --filter "tag=env:prod,tag=role:web"
```

5. **定期备份**：定期备份VM配置和数据

6. **监控资源**：监控VM的资源使用情况，及时调整配置

## 故障排除

### VM创建失败

```bash
# 查看详细的创建日志
./bbx-cli vm create --name my-vm --verbose
```

**可能的原因**：
- 资源不足：检查主机的CPU、内存和磁盘资源
- 模板不存在：使用`./bbx-cli vm template list`检查可用模板
- 网络配置错误：检查网络设置和IP地址配置

### VM启动失败

```bash
# 查看VM状态和错误信息
./bbx-cli vm status my-vm --verbose
```

**可能的原因**：
- 磁盘空间不足：检查VM的磁盘空间
- 启动配置错误：检查VM的启动配置
- 硬件兼容性问题：检查硬件兼容性

### VM停止失败

```bash
# 强制停止VM
./bbx-cli vm stop my-vm --force
```

**可能的原因**：
- 进程无法正常终止：使用强制停止选项
- 系统死锁：重启主机或使用强制停止

## 相关命令

- `vm ssh`：SSH连接到VM
- `vm file`：文件管理命令
- `vm permission`：权限管理命令
- `vm template`：VM模板管理命令
- `vm group`：VM组管理命令
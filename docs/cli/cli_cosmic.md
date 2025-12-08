# Cosmic 命令

Cosmic命令用于管理Cosmic相关功能。

## 命令格式

```
bbx-cli cosmic [command] [flags]
```

## 命令列表

### list - 列出Cosmic资源

列出Cosmic资源。

**用法：**
```
bbx-cli cosmic list [flags]
```

**参数：**
- `-t, --type string`：资源类型

### get - 获取Cosmic资源

获取Cosmic资源详情。

**用法：**
```
bbx-cli cosmic get [name] [flags]
```

**参数：**
- `name`：资源名称
- `-t, --type string`：资源类型

### create - 创建Cosmic资源

创建Cosmic资源。

**用法：**
```
bbx-cli cosmic create [flags]
```

**参数：**
- `-t, --type string`：资源类型
- `-c, --config string`：资源配置文件

### delete - 删除Cosmic资源

删除Cosmic资源。

**用法：**
```
bbx-cli cosmic delete [name] [flags]
```

**参数：**
- `name`：资源名称
- `-t, --type string`：资源类型

## 示例

```bash
# 列出所有Cosmic资源
./bbx-cli cosmic list

# 获取特定类型资源
./bbx-cli cosmic list -t instance

# 获取资源详情
./bbx-cli cosmic get my-instance -t instance

# 创建资源
./bbx-cli cosmic create -t instance -c config.yaml

# 删除资源
./bbx-cli cosmic delete my-instance -t instance
```

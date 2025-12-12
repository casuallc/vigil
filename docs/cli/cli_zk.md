# Zookeeper 命令

Zookeeper命令用于与Zookeeper集群进行交互，支持基本操作。

## 命令格式

```
bbx-cli zk [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--servers` | `-s` | Zookeeper服务器地址列表 | 127.0.0.1:2181 |
| `--timeout` | `-t` | 连接超时时间（秒） | 10 |

## 命令列表

### create - 创建节点

创建Zookeeper节点。

**用法：**
```
bbx-cli zk create [path] [data] [flags]
```

**参数：**
- `path`：节点路径
- `data`：节点数据
- `-e, --ephemeral`：临时节点标志
- `-s, --sequential`：顺序节点标志

### get - 获取节点数据

获取Zookeeper节点数据。

**用法：**
```
bbx-cli zk get [path] [flags]
```

**参数：**
- `path`：节点路径

### set - 设置节点数据

设置Zookeeper节点数据。

**用法：**
```
bbx-cli zk set [path] [data] [flags]
```

**参数：**
- `path`：节点路径
- `data`：节点数据

### delete - 删除节点

删除Zookeeper节点。

**用法：**
```
bbx-cli zk delete [path] [flags]
```

**参数：**
- `path`：节点路径

### ls - 列出节点子节点

列出Zookeeper节点的子节点。

**用法：**
```
bbx-cli zk ls [path] [flags]
```

**参数：**
- `path`：节点路径

## 示例

```bash
# 创建节点
./bbx-cli zk create /test "hello zookeeper" -s 127.0.0.1:2181

# 获取节点数据
./bbx-cli zk get /test -s 127.0.0.1:2181

# 设置节点数据
./bbx-cli zk set /test "new data" -s 127.0.0.1:2181

# 列出子节点
./bbx-cli zk ls / -s 127.0.0.1:2181

# 删除节点
./bbx-cli zk delete /test -s 127.0.0.1:2181
```

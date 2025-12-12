# Redis 命令

Redis命令用于与Redis服务器进行交互，支持各种Redis操作。

## 命令格式

```
bbx-cli redis [command] [flags]
```

## 全局参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--server` | `-s` | Redis服务器地址 | 127.0.0.1 |
| `--port` | `-p` | Redis服务器端口 | 6379 |
| `--password` | `-P` | Redis密码 |  |
| `--db` | `-d` | Redis数据库 | 0 |

## 命令列表

### get - 获取Redis值

从Redis中获取指定键的值。

**用法：**
```
bbx-cli redis get [flags]
```

**参数：**
- `-k, --key string`：要获取的键名

### set - 设置Redis值

设置Redis中指定键的值。

**用法：**
```
bbx-cli redis set [flags]
```

**参数：**
- `-k, --key string`：要设置的键名
- `-v, --value string`：要设置的值

### delete - 删除Redis键

删除Redis中的指定键。

**用法：**
```
bbx-cli redis delete [flags]
```

**参数：**
- `-k, --key string`：要删除的键名

### info - 获取Redis信息

获取Redis服务器的信息。

**用法：**
```
bbx-cli redis info
```

## 示例

```bash
# 连接到Redis并获取值
./bbx-cli redis get -k mykey -s 127.0.0.1 -p 6379

# 设置Redis值
./bbx-cli redis set -k mykey -v myvalue -s 127.0.0.1 -p 6379

# 删除Redis键
./bbx-cli redis delete -k mykey -s 127.0.0.1 -p 6379

# 获取Redis信息
./bbx-cli redis info -s 127.0.0.1 -p 6379
```

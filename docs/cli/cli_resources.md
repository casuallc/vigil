# 资源管理命令

资源管理命令用于查看系统和进程的资源使用情况。

## 命令格式

```
bbx-cli resources [command]
```

## 命令列表

### system - 获取系统资源

获取系统资源使用信息。

**用法：**
```
bbx-cli resources system
```

**示例：**
```bash
# 查看系统资源
./bbx-cli resources system
```

### process - 获取进程资源

获取特定进程的资源使用信息。

**用法：**
```
bbx-cli resources process [pid]
```

**参数：**
- `pid`：进程ID（可选）

**示例：**
```bash
# 查看指定进程资源
./bbx-cli resources process 1234

# 交互式选择进程查看资源
./bbx-cli resources process
```

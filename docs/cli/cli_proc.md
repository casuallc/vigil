# 进程管理命令

进程管理命令用于管理和监控系统进程，提供了丰富的操作选项。

## 命令格式

```
bbx-cli proc [command]
```

## 命令列表

### scan - 扫描进程

扫描系统进程并可选择注册到管理系统。

**用法：**
```
bbx-cli proc scan [flags]
```

**参数：**
- `-q, --query string`：搜索查询字符串或正则表达式；支持前缀：script://, file://（必填）
- `-r, --register`：扫描后注册进程
- `-n, --namespace string`：进程命名空间（默认：default）
- `-c, --config string`：批量扫描的配置文件（默认：conf/scan.yaml）
- `-b, --batch`：启用批量扫描模式

**示例：**
```bash
# 扫描包含 "java" 的进程
./bbx-cli proc scan -q "java" -H http://127.0.0.1:8181

# 扫描并注册进程
./bbx-cli proc scan -q "java" -r -H http://127.0.0.1:8181

# 批量扫描
./bbx-cli proc scan -b -c conf/scan.yaml -r -H http://127.0.0.1:8181
```

### create - 创建进程

创建一个新的托管进程。

**用法：**
```
bbx-cli proc create [name] [flags]
```

**参数：**
- `name`：进程名称（可选，可通过 --name 参数指定）
- `-N, --name string`：进程名称（替代位置参数）
- `-c, --command string`：命令路径
- `-n, --namespace string`：进程命名空间（默认：default）

### start - 启动进程

启动一个托管进程。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc start [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

### stop - 停止进程

停止一个托管进程。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc stop [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

### restart - 重启进程

重启一个托管进程。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc restart [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

### delete - 删除进程

从托管列表中删除一个进程。如果进程正在运行，会先停止它。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc delete [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

### list - 列出进程

列出所有托管进程。

**用法：**
```
bbx-cli proc list [flags]
```

**参数：**
- `-n, --namespace string`：进程命名空间（默认：default）

### status - 检查进程状态

检查托管进程的状态。

**用法：**
```
bbx-cli proc status [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）

### edit - 编辑进程定义

使用vim编辑器编辑托管进程的定义。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc edit [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

### get - 获取进程详情

获取托管进程的详细信息。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc get [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-f, --format string`：输出格式（yaml|text）（默认：yaml）
- `-n, --namespace string`：进程命名空间（默认：default）

## 挂载管理命令

挂载管理命令用于为进程添加、移除和列出挂载点。

### mount add - 添加挂载点

为进程添加一个挂载点。

**用法：**
```
bbx-cli proc mount add [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-t, --type string`：挂载类型（bind|tmpfs|volume）（默认：bind）
- `-N, --name string`：挂载标识符（每个进程唯一）（必填）
- `-T, --target string`：进程内的目标路径（必填）
- `-s, --source string`：bind挂载的源路径
- `-v, --volume string`：volume挂载的卷名
- `-r, --read-only`：以只读方式挂载
- `-o, --option stringArray`：附加挂载选项（可重复）
- `-n, --namespace string`：进程命名空间（默认：default）

### mount remove - 移除挂载点

从进程中移除一个挂载点。

**用法：**
```
bbx-cli proc mount remove [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-T, --target string`：要移除的目标路径
- `-i, --index int`：要移除的挂载索引

### mount list - 列出挂载点

列出进程的所有挂载点。

**用法：**
```
bbx-cli proc mount list [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）

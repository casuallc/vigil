# 进程管理命令

进程管理命令用于管理和监控系统进程，提供了从进程扫描、创建到启动、停止、重启、删除等全生命周期管理功能，以及资源监控和挂载管理。

## 命令格式

```
bbx-cli proc [command] [flags]
```

## 命令列表

### scan - 扫描进程

扫描系统进程并可选择注册到管理系统。支持多种扫描模式：
- 字符串匹配
- 正则表达式
- 脚本扫描（通过 script:// 前缀）
- 文件扫描（通过 file:// 前缀）

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
./bbx-cli proc scan -q "java"

# 使用正则表达式扫描Java进程
./bbx-cli proc scan -q "java.*"

# 扫描并注册进程到指定命名空间
./bbx-cli proc scan -q "java" -r -n production

# 批量扫描（使用默认配置文件 conf/scan.yaml）
./bbx-cli proc scan -b -r

# 使用自定义配置文件进行批量扫描
./bbx-cli proc scan -b -c /path/to/custom/scan.yaml -r

# 使用脚本扫描进程
./bbx-cli proc scan -q "script://path/to/scan-script.sh" -r

# 使用文件内容作为扫描查询
./bbx-cli proc scan -q "file://path/to/query.txt" -r
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
- `-c, --command string`：命令路径（必填）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-d, --dir string`：工作目录（可选）
- `-e, --env stringArray`：环境变量，格式 KEY=VALUE（可选，可重复）
- `-t, --timeout int`：启动超时（秒，默认：10）

**示例：**
```bash
# 创建一个简单的进程
./bbx-cli proc create test-process -c /bin/echo -e "TEST=value"

# 使用命名空间创建进程
./bbx-cli proc create -N test-process -c /bin/echo -n staging

# 创建带工作目录和环境变量的进程
./bbx-cli proc create web-server -c /usr/bin/node -d /path/to/app -e "PORT=3000" -e "NODE_ENV=production"
```

### start - 启动进程

启动一个托管进程。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc start [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-t, --timeout int`：启动超时（秒，默认：10）

**示例：**
```bash
# 启动指定进程
./bbx-cli proc start test-process

# 从指定命名空间启动进程
./bbx-cli proc start test-process -n production

# 交互式选择要启动的进程
./bbx-cli proc start

# 启动进程并设置超时
./bbx-cli proc start web-server -t 30
```

### stop - 停止进程

停止一个托管进程。如果没有提供名称，将显示交互式选择。支持优雅停止和强制停止。

**用法：**
```
bbx-cli proc stop [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-t, --timeout int`：停止超时（秒，默认：30）

**示例：**
```bash
# 停止指定进程
./bbx-cli proc stop test-process

# 从指定命名空间停止进程
./bbx-cli proc stop test-process -n production

# 交互式选择要停止的进程
./bbx-cli proc stop

# 停止进程并设置超时
./bbx-cli proc stop web-server -t 60
```

### restart - 重启进程

重启一个托管进程。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc restart [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-t, --timeout int`：启动超时（秒，默认：10）

**示例：**
```bash
# 重启指定进程
./bbx-cli proc restart test-process

# 从指定命名空间重启进程
./bbx-cli proc restart test-process -n production

# 交互式选择要重启的进程
./bbx-cli proc restart

# 重启进程并设置超时
./bbx-cli proc restart web-server -t 30
```

### delete - 删除进程

从托管列表中删除一个进程。如果进程正在运行，会先停止它。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc delete [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-t, --timeout int`：停止超时（秒，默认：30）

**示例：**
```bash
# 删除指定进程
./bbx-cli proc delete test-process

# 从指定命名空间删除进程
./bbx-cli proc delete test-process -n production

# 交互式选择要删除的进程
./bbx-cli proc delete

# 删除进程并设置停止超时
./bbx-cli proc delete web-server -t 60
```

### list - 列出进程

列出所有托管进程。

**用法：**
```
bbx-cli proc list [flags]
```

**参数：**
- `-n, --namespace string`：进程命名空间（默认：default）

**示例：**
```bash
# 列出所有进程
./bbx-cli proc list

# 列出指定命名空间的进程
./bbx-cli proc list -n production
```

### status - 检查进程状态

检查托管进程的状态。

**用法：**
```
bbx-cli proc status [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）

**示例：**
```bash
# 检查指定进程状态
./bbx-cli proc status test-process

# 检查指定命名空间的进程状态
./bbx-cli proc status test-process -n production
```

### edit - 编辑进程定义

使用系统默认编辑器编辑托管进程的定义。如果没有提供名称，将显示交互式选择。

**用法：**
```
bbx-cli proc edit [name] [flags]
```

**参数：**
- `name`：进程名称（可选）
- `-n, --namespace string`：进程命名空间（默认：default）

**示例：**
```bash
# 编辑指定进程
./bbx-cli proc edit test-process

# 编辑指定命名空间的进程
./bbx-cli proc edit test-process -n production

# 交互式选择要编辑的进程
./bbx-cli proc edit
```

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

**示例：**
```bash
# 获取进程详情（默认YAML格式）
./bbx-cli proc get test-process

# 获取进程详情（文本格式）
./bbx-cli proc get test-process -f text

# 获取指定命名空间的进程详情
./bbx-cli proc get test-process -n production

# 交互式选择要获取详情的进程
./bbx-cli proc get
```

## 挂载管理命令

挂载管理命令用于为进程添加、移除和列出挂载点。支持三种挂载类型：
- bind：绑定挂载本地目录
- tmpfs：临时文件系统挂载
- volume：命名卷挂载

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
- `-s, --source string`：bind挂载的源路径（bind类型必填）
- `-v, --volume string`：volume挂载的卷名（volume类型必填）
- `-r, --read-only`：以只读方式挂载
- `-o, --option stringArray`：附加挂载选项（可重复）
- `-n, --namespace string`：进程命名空间（默认：default）

**示例：**
```bash
# 添加bind挂载
./bbx-cli proc mount add test-process -t bind -N config -T /etc/app/config -s /host/path/to/config

# 添加只读bind挂载
./bbx-cli proc mount add test-process -t bind -N data -T /app/data -s /host/path/to/data -r

# 添加tmpfs挂载
./bbx-cli proc mount add test-process -t tmpfs -N cache -T /app/cache -o size=100m

# 添加volume挂载
./bbx-cli proc mount add test-process -t volume -N db -T /app/db -v app-db
```

### mount remove - 移除挂载点

从进程中移除一个挂载点。

**用法：**
```
bbx-cli proc mount remove [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）
- `-T, --target string`：要移除的目标路径（与--index二选一）
- `-i, --index int`：要移除的挂载索引（与--target二选一）

**示例：**
```bash
# 根据目标路径移除挂载点
./bbx-cli proc mount remove test-process -T /etc/app/config

# 根据索引移除挂载点
./bbx-cli proc mount remove test-process -i 0

# 移除指定命名空间进程的挂载点
./bbx-cli proc mount remove test-process -n production -T /etc/app/config
```

### mount list - 列出挂载点

列出进程的所有挂载点。

**用法：**
```
bbx-cli proc mount list [name] [flags]
```

**参数：**
- `name`：进程名称（必填）
- `-n, --namespace string`：进程命名空间（默认：default）

**示例：**
```bash
# 列出进程的所有挂载点
./bbx-cli proc mount list test-process

# 列出指定命名空间进程的挂载点
./bbx-cli proc mount list test-process -n production
```

## 注意事项

1. **进程命名空间**：所有进程都属于一个命名空间，默认命名空间为 "default"。可以通过 `-n, --namespace` 参数指定。

2. **交互式命令**：当不提供进程名称时，命令会进入交互式模式，允许用户从列表中选择要操作的进程。

3. **超时控制**：`start`、`stop` 和 `restart` 命令都支持 `-t, --timeout` 参数，用于设置操作超时时间。

4. **环境变量**：`create` 命令支持通过 `-e, --env` 参数设置环境变量，格式为 `KEY=VALUE`，可多次使用。

5. **挂载权限**：挂载操作可能需要管理员权限，特别是在Linux系统上。

6. **配置文件**：批量扫描配置文件（如 `conf/scan.yaml`）采用YAML格式，定义了要扫描的进程规则。

## 进程管理架构

进程管理系统采用以下架构：

- **进程管理器**：核心组件，负责进程的创建、启动、停止、监控等生命周期管理
- **进程监控器**：每个被管理进程都有独立的监控协程，定期检查进程状态
- **资源监控**：监控进程的CPU、内存、磁盘、网络等资源使用情况
- **事件系统**：捕获和处理进程事件，如启动、停止、崩溃等

这种架构确保了进程管理的可靠性和高效性，能够实时监控和响应进程状态变化。

# 实时日志流式查看设计文档

## 背景

Vigil 是一个进程管理和运维工具，用户需要实时查看服务器上的日志文件。需要提供一个类似 `tail -f` 的 API，支持通过 SSE 实时推送日志新增内容。

## 目标

- 提供基于 SSE 的实时日志流式查看 API
- 支持从指定位置开始读取（行号、文件开头、文件末尾）
- 客户端断开时自动清理资源
- 提供 CLI 命令 `bbx-cli logs tail`

## API 设计

### 端点

```
GET /api/files/logs/stream?path=/var/log/app.log&from_line=-100
```

路由注册在 `api/routes.go` 的 File Management endpoints 组中。

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 日志文件路径 |
| from_line | int | 否 | 起始行号策略 |

**from_line 语义：**
- 不传/省略：从文件末尾开始（默认 `tail -f` 行为）
- `0`：从文件开头开始
- 正数：从指定行号开始（如 `100` 表示从第100行）
- 负数：从末尾向前偏移（如 `-100` 表示最后100行）

### SSE 响应格式

```
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: line
data: {"line_number": 101, "content": "2025-04-18 log message..."}

event: line
data: {"line_number": 102, "content": "next log line..."}

event: error
data: {"message": "file not found"}
```

### 错误事件

服务端遇到以下情况时推送 `event: error` 并关闭连接：
- 文件不存在或无法访问
- 文件被删除或移动
- 读取文件时发生 IO 错误

## 服务端实现

### 核心逻辑

1. 解析 `path` 和 `from_line` 参数
2. 打开文件，根据 `from_line` 定位起始读取位置
3. 使用 `bufio.Scanner` 逐行读取
4. 读取到文件末尾后，进入等待循环
5. 通过 `time.Tick(500ms)` 轮询文件大小变化
6. 文件大小增加时，继续读取并推送新行
7. 客户端断开（HTTP context done）时，关闭文件并退出

### 文件定位策略

```go
func seekToLine(file *os.File, fromLine int) error {
    switch {
    case fromLine == 0:
        // 从文件开头开始
        _, err := file.Seek(0, io.SeekStart)
        return err
    case fromLine > 0:
        // 从指定行号开始
        scanner := bufio.NewScanner(file)
        currentLine := 0
        for scanner.Scan() {
            currentLine++
            if currentLine >= fromLine {
                // 回退当前行的起始位置
                offset, _ := file.Seek(0, io.SeekCurrent)
                // 减去当前行的长度和换行符长度
                lineLen := len(scanner.Bytes()) + 1
                file.Seek(offset-int64(lineLen), io.SeekStart)
                return nil
            }
        }
        return scanner.Err()
    case fromLine < 0:
        // 从末尾向前偏移
        // 先读取全部行，缓存最后 |from_line| 行
        // 或使用 Seek 到文件末尾，然后反向扫描
        // 简化方案：顺序读取，只保留最后 |from_line| 行
    default:
        // 从文件末尾开始
        _, err := file.Seek(0, io.SeekEnd)
        return err
    }
}
```

对于负数情况，更高效的方案：
- 先 `Seek` 到文件末尾
- 反向读取块，找到第 `|from_line|` 个换行符
- 从该位置开始正向读取

为了简单起见，可以先顺序读取整个文件，用环形缓冲区保留最后 N 行（N 不大时可行，如最多 10000 行）。或者如果文件很大，使用 Seek 反向查找。

实际上，更实用的方案：
- 对于负数 from_line，可以先顺序扫描统计总行数，然后从 `总行数 + from_line` 行开始。但这需要两次扫描。
- 更高效的方案：分块从末尾反向读取。

为了简化实现，可以限制 from_line 负数的绝对值不超过 10000，这样用环形缓冲区存储最后 10000 行是可行的。

## 客户端实现

### Go Client

```go
// LogLine represents a single log line
type LogLine struct {
    LineNumber int    `json:"line_number"`
    Content    string `json:"content"`
}

// StreamLogs opens an SSE connection and streams log lines
func (c *Client) StreamLogs(path string, fromLine int, handler func(line LogLine)) error {
    // 构造 URL
    // 设置 Accept: text/event-stream
    // 解析 SSE 事件流
    // 调用 handler 处理每一行
}
```

### CLI

```bash
# 默认从末尾开始，实时推送
bbx-cli logs tail -p /var/log/app.log

# 从最后100行开始
bbx-cli logs tail -p /var/log/app.log -n 100

# 从第100行开始
bbx-cli logs tail -p /var/log/app.log --from-line 100

# 从文件开头开始
bbx-cli logs tail -p /var/log/app.log --from-line 0
```

## 文件结构

| File | Responsibility |
|------|---------------|
| `api/routes.go` | 注册 `/api/files/logs/stream` 路由 |
| `api/handlers_log.go` | SSE 日志流 handler |
| `api/client_log.go` | 客户端 SSE 连接和解析 |
| `cli/log.go` | CLI `logs tail` 命令 |

## 测试策略

- Handler 测试：用 `httptest.NewRecorder` 模拟 SSE 连接，验证事件格式
- 需要测试 from_line 的四种语义
- 需要测试文件不存在时的错误响应

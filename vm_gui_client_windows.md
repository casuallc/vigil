# Windows GUI 客户端方案

## 1. 方案概述

本方案旨在为 VM 管理系统开发一个可以在 Windows 平台运行的 GUI 客户端，提供直观的图形界面，方便用户进行 VM 连接、文件上传下载等操作。

## 2. 技术选型

### 2.1 GUI 框架选择

| 框架 | 技术栈 | 优点 | 缺点 | 推荐指数 |
|------|--------|------|------|----------|
| **Fyne** | Go + Fyne UI | 跨平台、纯 Go 实现、设计简洁、开发效率高 | 功能相对简单、定制性有限 | ⭐⭐⭐⭐⭐ |
| **Wails** | Go + Web 技术（HTML/CSS/JS） | 跨平台、Web 技术成熟、UI 定制性强 | 需要 Web 开发经验、打包体积较大 | ⭐⭐⭐⭐ |
| **QT** | Go + QT | 功能强大、成熟稳定、跨平台 | 学习曲线陡、依赖复杂、构建麻烦 | ⭐⭐⭐ |
| **Windows API** | Go + Windows API | 性能最佳、深度集成 Windows | 开发难度大、非跨平台、维护成本高 | ⭐⭐ |

**推荐选择：Fyne**

理由：
- 纯 Go 实现，无需额外依赖，构建简单
- 跨平台支持，可在 Windows、Linux、macOS 上运行
- 设计简洁，开发效率高
- 提供现代化的 UI 组件
- 内置支持主题、国际化等功能

### 2.2 核心库

- **Fyne**：GUI 框架
- **github.com/casuallc/vigil/api**：现有 API 客户端
- **golang.org/x/crypto/ssh**：SSH 连接管理
- **github.com/gorilla/websocket**：WebSocket 支持
- **github.com/pkg/sftp**：SFTP 文件传输

## 3. 功能设计

### 3.1 主界面设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 标题栏：VM 客户端                                                      │
├───────────────┬─────────────────────────────────────────────────────────┤
│ 侧边栏         │ 主内容区                                              │
│ ┌───────────┐ │ ┌─────────────────────────────────────────────────────┐ │
│ │ 连接管理   │ │ │ 连接列表                                           │ │
│ ├───────────┤ │ │ ┌─────────────────┐ ┌─────────────────────────────┐ │ │
│ │ SSH 终端   │ │ │ │ VM1            │ │ 连接详情                     │ │ │
│ ├───────────┤ │ │ ├─────────────────┤ │ 名称：VM1                   │ │ │
│ │ 文件管理   │ │ │ │ VM2            │ │ IP：172.20.140.158          │ │ │
│ ├───────────┤ │ │ └─────────────────┘ │ 状态：在线                   │ │ │
│ │ 配置管理   │ │ │                     │ 操作：[连接] [编辑] [删除]  │ │ │
│ └───────────┘ │ └─────────────────────────────────────────────────────┘ │
├───────────────┴─────────────────────────────────────────────────────────┤
│ 状态栏：显示连接状态、当前操作等                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 功能模块

#### 3.2.1 连接管理
- 保存和管理多个 VM 连接配置
- 支持添加、编辑、删除连接
- 连接配置包括：名称、IP、端口、用户名、密码/密钥文件
- 支持测试连接
- 支持连接分组管理

#### 3.2.2 SSH 终端
- 提供交互式 SSH 终端
- 支持终端窗口大小调整
- 支持复制粘贴功能
- 支持终端历史记录
- 支持多种终端主题

#### 3.2.3 文件管理
- 图形化文件浏览器
- 支持文件上传、下载
- 支持文件拖放操作
- 支持文件权限查看和修改
- 支持文件搜索
- 支持文件夹创建、删除、重命名

#### 3.2.4 VM 状态监控
- 显示 VM 的运行状态
- 显示 CPU、内存、磁盘等资源使用情况
- 支持实时刷新

#### 3.2.5 配置管理
- 客户端配置，如默认连接、日志设置等
- 支持主题切换（浅色/深色）
- 支持语言切换
- 支持日志级别设置

## 4. 实现方案

### 4.1 目录结构

```
vm-gui-client/
├── cmd/
│   └── vm-gui-client/          # 主入口
│       └── main.go             # 主函数
├── internal/
│   ├── app/                    # 应用核心
│   │   ├── app.go              # 应用初始化和管理
│   │   └── theme.go            # 主题管理
│   ├── config/                 # 配置管理
│   │   └── config.go           # 配置加载和保存
│   ├── connection/             # 连接管理
│   │   └── manager.go          # 连接配置管理
│   ├── ssh/                    # SSH 功能
│   │   ├── client.go           # SSH 客户端
│   │   └── terminal.go         # SSH 终端实现
│   ├── file/                   # 文件管理
│   │   └── manager.go          # 文件操作管理
│   ├── ui/                     # UI 组件
│   │   ├── main_window.go      # 主窗口
│   │   ├── connection_panel.go # 连接管理面板
│   │   ├── terminal_panel.go   # SSH 终端面板
│   │   └── file_panel.go       # 文件管理面板
│   └── utils/                  # 工具函数
│       └── utils.go            # 通用工具
├── go.mod                      # Go 模块定义
└── go.sum                      # 依赖校验
```

### 4.2 核心代码示例

#### 4.2.1 主函数（main.go）

```go
package main

import (
	"github.com/casuallc/vigil/vm-gui-client/internal/app"
	"github.com/casuallc/vigil/vm-gui-client/internal/config"
	"github.com/fyne-io/fyne/v2"
	"github.com/fyne-io/fyne/v2/app"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		// 使用默认配置
		cfg = config.DefaultConfig()
	}

	// 创建 Fyne 应用
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(app.NewTheme(cfg.Theme))

	// 创建应用实例
	myApp := app.NewApp(fyneApp, cfg)

	// 启动应用
	myApp.Run()
}
```

#### 4.2.2 SSH 终端实现（terminal.go）

```go
package ssh

import (
	"bytes"
	"github.com/casuallc/vigil/api"
	"github.com/fyne-io/fyne/v2"
	"github.com/fyne-io/fyne/v2/widget"
	"golang.org/x/crypto/ssh"
)

// TerminalWidget 实现 SSH 终端组件
type TerminalWidget struct {
	widget.TextGrid
	client   *api.Client
	vmName   string
	conn     *ssh.Client
	buffer   bytes.Buffer
	// 其他字段
}

// NewTerminalWidget 创建新的终端组件
func NewTerminalWidget(client *api.Client, vmName string) *TerminalWidget {
	return &TerminalWidget{
		client: client,
		vmName: vmName,
		// 初始化其他字段
	}
}

// Connect 连接到 SSH 服务器
func (t *TerminalWidget) Connect() error {
	// 使用 api.Client 建立 WebSocket SSH 连接
	conn, err := t.client.SSHWebSocket(t.vmName)
	if err != nil {
		return err
	}
	// 处理 WebSocket 通信
	// ...
	return nil
}

// 其他终端相关方法
// ...
```

#### 4.2.3 文件管理实现（file_manager.go）

```go
package file

import (
	"github.com/casuallc/vigil/api"
	"github.com/casuallc/vigil/file"
)

// FileManager 实现文件管理功能
type FileManager struct {
	client *api.Client
	vmName string
}

// NewFileManager 创建新的文件管理器
func NewFileManager(client *api.Client, vmName string) *FileManager {
	return &FileManager{
		client: client,
		vmName: vmName,
	}
}

// ListFiles 列出文件
func (fm *FileManager) ListFiles(path string, maxDepth int) ([]file.FileInfo, error) {
	return fm.client.VMFileList(fm.vmName, path, maxDepth)
}

// UploadFile 上传文件
func (fm *FileManager) UploadFile(srcPath, dstPath string) error {
	return fm.client.VMFileUpload(fm.vmName, srcPath, dstPath)
}

// DownloadFile 下载文件
func (fm *FileManager) DownloadFile(srcPath, dstPath string) error {
	return fm.client.VMFileDownload(fm.vmName, srcPath, dstPath)
}

// 其他文件操作方法
// ...
```

## 5. 构建和部署

### 5.1 开发环境

- **Go**：1.20+（推荐 1.21）
- **Fyne**：v2.4+（可通过 `go install fyne.io/fyne/v2/cmd/fyne@latest` 安装）
- **Windows 开发工具**：
  - Windows 11 SDK
  - Visual Studio Build Tools（或 MinGW-w64）
  - Git

### 5.2 构建步骤

1. **安装依赖**：
   ```bash
go mod tidy
```

2. **本地开发运行**：
   ```bash
go run ./cmd/vm-gui-client
```

3. **构建 Windows 可执行文件**：
   ```bash
fyne package -os windows -icon ./assets/icon.png -name "VM Client"
```

4. **构建安装包**：
   ```bash
fyne package -os windows -icon ./assets/icon.png -name "VM Client" -release
```

### 5.3 部署方式

- **绿色版**：直接提供可执行文件和必要的资源文件
- **安装包**：使用 Inno Setup 或 NSIS 等工具打包成 Windows 安装包
- **MSIX 包**：为 Windows 10/11 系统提供现代安装体验

## 6. 性能和兼容性考虑

### 6.1 性能优化

1. **异步操作**：文件上传下载等耗时操作使用异步方式，避免阻塞 UI
2. **缓存机制**：对常用数据进行缓存，减少 API 调用
3. **资源管理**：及时释放不再使用的资源，如 SSH 连接、文件句柄等
4. **UI 渲染优化**：避免频繁更新 UI，使用批量更新机制

### 6.2 兼容性

1. **Windows 版本支持**：支持 Windows 10/11（32 位和 64 位）
2. **高 DPI 支持**：适配不同分辨率和缩放比例
3. **Windows 主题支持**：支持 Windows 浅色/深色主题
4. **防火墙适配**：提供防火墙配置建议

## 7. 安全性考虑

1. **连接加密**：所有通信使用 HTTPS/WebSocket 加密
2. **认证安全**：支持密码和密钥认证，密钥文件本地加密存储
3. **访问控制**：客户端本地存储的连接配置加密保存
4. **日志安全**：避免在日志中记录敏感信息
5. **防注入**：对用户输入进行严格验证，防止命令注入等攻击

## 8. 测试和维护

### 8.1 测试策略

1. **单元测试**：对核心功能进行单元测试
2. **集成测试**：测试不同模块之间的交互
3. **UI 测试**：使用 Fyne 提供的测试工具进行 UI 测试
4. **兼容性测试**：在不同 Windows 版本上进行测试
5. **性能测试**：测试大文件传输、多连接并发等场景

### 8.2 维护计划

1. **版本更新**：定期发布更新，修复 bug 和添加新功能
2. **日志管理**：实现详细的日志记录，便于问题排查
3. **错误报告**：添加自动错误报告功能
4. **文档更新**：保持文档与代码同步

## 9. 预期效果

1. **用户体验提升**：提供直观的图形界面，降低用户学习成本
2. **操作效率提高**：通过图形化操作，减少命令行输入，提高操作效率
3. **功能完整性**：支持 VM 管理的核心功能，如 SSH 连接、文件传输等
4. **跨平台支持**：除了 Windows，还可以在 Linux、macOS 上运行
5. **易于扩展**：模块化设计，便于添加新功能

## 10. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| GUI 库稳定性问题 | 客户端崩溃、功能异常 | 选择成熟的 GUI 库，进行充分测试 |
| 性能问题 | 客户端卡顿、响应慢 | 优化代码，使用异步操作，避免阻塞 UI |
| 兼容性问题 | 在某些 Windows 版本上无法运行 | 进行充分的兼容性测试，提供多个版本的安装包 |
| 安全性问题 | 数据泄露、未授权访问 | 实现加密通信、安全认证、访问控制等措施 |
| 开发周期延长 | 项目延期 | 采用模块化设计，优先实现核心功能，分阶段开发 |

## 11. 结论

本方案提供了一个完整的 Windows GUI 客户端设计，使用 Fyne 作为 GUI 框架，具有开发效率高、跨平台支持好、用户体验佳等优点。通过模块化设计，便于后续扩展和维护。建议采用本方案进行开发，优先实现核心功能，分阶段发布，逐步完善客户端功能。
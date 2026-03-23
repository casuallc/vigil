/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ------------------------- AI Assistant Handlers -------------------------

// handleAIGenerateCommand handles POST /api/ai/generate-command
func (s *Server) handleAIGenerateCommand(w http.ResponseWriter, r *http.Request) {
	var req AIGenerateCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// TODO: Integrate with actual LLM service
	// For now, return mock responses based on common patterns
	resp := generateMockCommand(req)
	writeJSON(w, http.StatusOK, resp)
}

// handleAIExplainCommand handles POST /api/ai/explain-command
func (s *Server) handleAIExplainCommand(w http.ResponseWriter, r *http.Request) {
	var req AIExplainCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}

	// TODO: Integrate with actual LLM service
	resp := explainMockCommand(req.Command)
	writeJSON(w, http.StatusOK, resp)
}

// handleAIFixCommand handles POST /api/ai/fix-command
func (s *Server) handleAIFixCommand(w http.ResponseWriter, r *http.Request) {
	var req AIFixCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Command == "" || req.Error == "" {
		writeError(w, http.StatusBadRequest, "command and error are required")
		return
	}

	// TODO: Integrate with actual LLM service
	resp := fixMockCommand(req)
	writeJSON(w, http.StatusOK, resp)
}

// ------------------------- Mock AI Functions -------------------------
// These functions provide mock responses for development.
// In production, these should be replaced with actual LLM API calls.

func generateMockCommand(req AIGenerateCommandRequest) AIGenerateCommandResponse {
	prompt := strings.ToLower(req.Prompt)

	// Pattern matching for common commands
	switch {
	case containsAny(prompt, []string{"nginx", "error", "log"}):
		return AIGenerateCommandResponse{
			Command:     "tail -n 100 /var/log/nginx/error.log",
			Explanation: "查看 nginx 错误日志的最后 100 行",
			Alternatives: []AICommandAlternative{
				{
					Command:     "journalctl -u nginx -p err -n 100",
					Explanation: "通过 systemd 查看nginx错误日志",
				},
				{
					Command:     "cat /var/log/nginx/error.log | grep -i error | tail -20",
					Explanation: "过滤并显示最近的错误行",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"disk", "space", "磁盘", "空间"}):
		return AIGenerateCommandResponse{
			Command:     "df -h",
			Explanation: "显示磁盘使用情况（人类可读格式）",
			Alternatives: []AICommandAlternative{
				{
					Command:     "du -sh /* 2>/dev/null | sort -hr | head -20",
					Explanation: "查看根目录下最大的20个目录",
				},
				{
					Command:     "df -i",
					Explanation: "显示 inode 使用情况",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"memory", "mem", "内存"}):
		return AIGenerateCommandResponse{
			Command:     "free -h",
			Explanation: "显示内存使用情况（人类可读格式）",
			Alternatives: []AICommandAlternative{
				{
					Command:     "ps aux --sort=-%mem | head -20",
					Explanation: "显示内存占用最高的20个进程",
				},
				{
					Command:     "cat /proc/meminfo",
					Explanation: "显示详细的内存信息",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"cpu", "load", "负载"}):
		return AIGenerateCommandResponse{
			Command:     "top -bn1 | head -20",
			Explanation: "显示系统进程和CPU使用情况",
			Alternatives: []AICommandAlternative{
				{
					Command:     "mpstat -P ALL 1 1",
					Explanation: "显示每个CPU核心的使用情况",
				},
				{
					Command:     "ps aux --sort=-%cpu | head -20",
					Explanation: "显示CPU占用最高的20个进程",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"process", "进程", "kill"}):
		return AIGenerateCommandResponse{
			Command:     "ps aux | grep <process_name>",
			Explanation: "查找指定名称的进程（请将 <process_name> 替换为实际的进程名）",
			Alternatives: []AICommandAlternative{
				{
					Command:     "pgrep -a <process_name>",
					Explanation: "使用 pgrep 查找进程ID和名称",
				},
				{
					Command:     "pidof <process_name>",
					Explanation: "获取进程的PID",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"port", "端口", "listen", "listening"}):
		return AIGenerateCommandResponse{
			Command:     "ss -tlnp",
			Explanation: "显示所有监听的TCP端口和关联的进程",
			Alternatives: []AICommandAlternative{
				{
					Command:     "netstat -tlnp",
					Explanation: "使用 netstat 显示监听端口（需要安装 net-tools）",
				},
				{
					Command:     "lsof -i -P -n | grep LISTEN",
					Explanation: "使用 lsof 显示监听端口",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"clean", "cleanup", "清理", "删除", "delete"}):
		return AIGenerateCommandResponse{
			Command:     "find /var/log -name '*.log' -mtime +7",
			Explanation: "查找7天前的日志文件（先查看，不删除）",
			Alternatives: []AICommandAlternative{
				{
					Command:     "find /var/log -name '*.log' -mtime +7 -delete",
					Explanation: "删除7天前的日志文件（谨慎使用）",
				},
				{
					Command:     "find /tmp -type f -atime +7 -delete",
					Explanation: "删除7天未访问的临时文件",
				},
			},
			IsDangerous: true,
		}

	case containsAny(prompt, []string{"docker", "container", "容器"}):
		return AIGenerateCommandResponse{
			Command:     "docker ps -a",
			Explanation: "显示所有容器（包括已停止的）",
			Alternatives: []AICommandAlternative{
				{
					Command:     "docker system df",
					Explanation: "显示 Docker 磁盘使用情况",
				},
				{
					Command:     "docker container ls --filter 'status=exited'",
					Explanation: "显示已退出的容器",
				},
			},
			IsDangerous: false,
		}

	case containsAny(prompt, []string{"mysql", "database", "db", "数据库"}):
		return AIGenerateCommandResponse{
			Command:     "mysql -u root -p -e 'SHOW PROCESSLIST;'",
			Explanation: "显示 MySQL 当前正在执行的查询",
			Alternatives: []AICommandAlternative{
				{
					Command:     "mysql -u root -p -e 'SHOW VARIABLES LIKE \"max_connections\";';",
					Explanation: "查看 MySQL 最大连接数设置",
				},
				{
					Command:     "mysqladmin -u root -p status",
					Explanation: "显示 MySQL 服务器状态",
				},
			},
			IsDangerous: false,
		}

	default:
		// Generic response for unrecognized prompts
		return AIGenerateCommandResponse{
			Command:     "echo 'Command not recognized. Please provide more specific details.'",
			Explanation: "无法根据描述生成命令，请提供更多详细信息",
			Alternatives: []AICommandAlternative{
				{
					Command:     "man <command>",
					Explanation: "查看命令的手册页",
				},
				{
					Command:     "<command> --help",
					Explanation: "查看命令的帮助信息",
				},
			},
			IsDangerous: false,
		}
	}
}

func explainMockCommand(command string) AIExplainCommandResponse {
	cmd := strings.ToLower(command)

	// Check for dangerous commands
	isDangerous := containsAny(cmd, []string{"rm -rf", "> /dev/null", ":(){", "mkfs", "dd if=/dev/zero"})
	warnings := []string{}
	if isDangerous {
		warnings = append(warnings, "此命令可能具有破坏性，请谨慎使用")
	}

	// Build breakdown based on command parts
	breakdown := []AICommandPart{}
	parts := strings.Fields(command)

	if len(parts) > 0 {
		breakdown = append(breakdown, AICommandPart{
			Part:    parts[0],
			Meaning: getCommandMeaning(parts[0]),
		})
	}

	// Analyze common patterns
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		switch {
		case strings.HasPrefix(part, "-"):
			breakdown = append(breakdown, AICommandPart{
				Part:    part,
				Meaning: getFlagMeaning(parts[0], part),
			})
		case part == "|":
			breakdown = append(breakdown, AICommandPart{
				Part:    part,
				Meaning: "管道操作符：将前一个命令的输出传递给后一个命令",
			})
		case part == ">" || part == ">>":
			breakdown = append(breakdown, AICommandPart{
				Part:    part,
				Meaning: "重定向操作符：将输出保存到文件",
			})
		case strings.Contains(part, "/"):
			breakdown = append(breakdown, AICommandPart{
				Part:    part,
				Meaning: "文件路径: " + part,
			})
		default:
			breakdown = append(breakdown, AICommandPart{
				Part:    part,
				Meaning: "参数或值",
			})
		}
	}

	// Generate overall explanation
	explanation := generateExplanation(command, parts)

	return AIExplainCommandResponse{
		Explanation: explanation,
		Breakdown:   breakdown,
		Warnings:    warnings,
		IsDangerous: isDangerous,
	}
}

func fixMockCommand(req AIFixCommandRequest) AIFixCommandResponse {
	cmd := strings.ToLower(req.Command)
	errMsg := strings.ToLower(req.Error)

	// Common fixes
	switch {
	case strings.Contains(errMsg, "permission denied"):
		return AIFixCommandResponse{
			FixedCommand: "sudo " + req.Command,
			Explanation:  "命令需要管理员权限，已添加 sudo 前缀",
		}

	case strings.Contains(errMsg, "command not found"):
		return AIFixCommandResponse{
			FixedCommand: req.Command, // Can't fix command not found
			Explanation:  "命令未找到，请检查是否已安装相关软件包",
		}

	case strings.Contains(errMsg, "unknown shorthand flag: 'a' in -a") && strings.Contains(cmd, "docker ps"):
		return AIFixCommandResponse{
			FixedCommand: "docker container ls -a",
			Explanation:  "原命令在部分 Docker 版本中语法不兼容，已修复为兼容性更好的写法",
		}

	case strings.Contains(errMsg, "no such file or directory"):
		return AIFixCommandResponse{
			FixedCommand: req.Command, // Can't fix missing files
			Explanation:  "文件或目录不存在，请检查路径是否正确",
		}

	case strings.Contains(errMsg, "xargs: docker rm") && strings.Contains(cmd, "xargs"):
		return AIFixCommandResponse{
			FixedCommand: strings.Replace(req.Command, "xargs docker rm", "xargs -r docker rm", -1),
			Explanation:  "添加 -r 参数防止空输入时执行命令",
		}

	case strings.Contains(errMsg, "is a directory"):
		return AIFixCommandResponse{
			FixedCommand: "rm -rf " + req.Command[strings.LastIndex(req.Command, " ")+1:],
			Explanation:  "需要递归删除目录，已添加 -r 参数",
		}

	default:
		return AIFixCommandResponse{
			FixedCommand: req.Command,
			Explanation:  "无法自动修复该错误，请检查命令语法或查看相关文档",
		}
	}
}

// Helper functions

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func getCommandMeaning(cmd string) string {
	meanings := map[string]string{
		"ls":      "列出目录内容",
		"cd":      "切换当前目录",
		"pwd":     "显示当前工作目录",
		"cat":     "显示文件内容",
		"grep":    "搜索文本内容",
		"find":    "查找文件",
		"ps":      "显示进程状态",
		"kill":    "终止进程",
		"tar":     "归档文件",
		"chmod":   "修改文件权限",
		"chown":   "修改文件所有者",
		"df":      "显示磁盘空间使用情况",
		"du":      "显示目录/文件大小",
		"top":     "实时显示进程",
		"free":    "显示内存使用情况",
		"echo":    "输出文本",
		"mkdir":   "创建目录",
		"rm":      "删除文件或目录",
		"cp":      "复制文件",
		"mv":      "移动/重命名文件",
		"tail":    "显示文件末尾内容",
		"head":    "显示文件开头内容",
		"curl":    "发送HTTP请求",
		"wget":    "下载文件",
		"ssh":     "远程登录",
		"scp":     "安全复制文件",
		"docker":  "Docker容器管理",
		"kubectl": "Kubernetes管理",
		"systemctl": "systemd服务管理",
		"journalctl": "查看系统日志",
		"ss":      "查看套接字统计",
		"netstat": "查看网络连接",
		"lsof":    "列出打开的文件",
		"ping":    "测试网络连通性",
		"traceroute": "追踪路由路径",
		"iptables": "防火墙规则管理",
	}

	if meaning, ok := meanings[cmd]; ok {
		return meaning
	}
	return "执行命令"
}

func getFlagMeaning(cmd, flag string) string {
	// Common flag meanings
	commonFlags := map[string]string{
		"-a": "显示所有项目（包括隐藏文件）",
		"-l": "使用长格式列表",
		"-h": "人类可读格式",
		"-r": "递归处理",
		"-f": "强制执行",
		"-v": "显示详细信息",
		"-n": "显示行号",
		"-i": "忽略大小写",
		"-c": "计数",
		"-t": "按时间排序",
		"-s": "安静模式",
		"-p": "保留权限",
		"-z": "压缩",
		"-9": "最大压缩",
		"-b": "备份",
		"-u": "更新",
		"-d": "指定目录",
		"-e": "指定表达式",
		"-E": "扩展正则表达式",
		"-o": "指定输出",
	}

	if meaning, ok := commonFlags[flag]; ok {
		return meaning
	}

	// Command-specific flags
	if cmd == "docker" || cmd == "docker-compose" {
		switch flag {
		case "-d":
			return "后台运行容器"
		case "--rm":
			return "容器停止后自动删除"
		case "-p":
			return "端口映射"
		case "-v":
			return "挂载卷"
		case "-e":
			return "设置环境变量"
		}
	}

	if cmd == "tar" {
		switch flag {
		case "-c":
			return "创建归档"
		case "-x":
			return "解压归档"
		case "-f":
			return "指定文件名"
		case "-z":
			return "使用 gzip 压缩"
		case "-j":
			return "使用 bzip2 压缩"
		case "-v":
			return "显示进度"
		}
	}

	return "命令选项"
}

func generateExplanation(command string, parts []string) string {
	if len(parts) == 0 {
		return "空命令"
	}

	cmd := parts[0]

	// Common command patterns
	patterns := map[string]string{
		"ls -la":    "列出当前目录的所有文件（包括隐藏文件）及其详细信息",
		"df -h":     "以人类可读格式显示磁盘空间使用情况",
		"free -h":   "以人类可读格式显示内存使用情况",
		"ps aux":    "显示所有用户的所有进程",
		"top":       "实时显示系统进程和资源使用情况",
		"netstat":   "显示网络连接、路由表、接口统计等信息",
		"ping":      "测试与目标主机的网络连通性",
		"grep":      "在文本中搜索匹配指定模式的行",
		"find":      "在文件系统中查找文件",
		"tar":       "创建或解压归档文件",
		"chmod":     "修改文件或目录的权限",
		"chown":     "修改文件或目录的所有者",
		"docker ps": "列出 Docker 容器",
		"kubectl":   "管理 Kubernetes 集群",
	}

	// Check for exact pattern match
	for pattern, explanation := range patterns {
		if strings.HasPrefix(command, pattern) {
			return explanation
		}
	}

	// Generate generic explanation
	return fmt.Sprintf("执行 %s 命令", getCommandMeaning(cmd))
}

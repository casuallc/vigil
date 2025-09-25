package process

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// 定义扫描类型常量
const (
	// 特殊前缀用于识别脚本
	ScriptPrefix = "script://"
	// 特殊前缀用于识别文件脚本
	FileScriptPrefix = "file://"
)

// ScanProcesses implements ProcessManager interface to scan system processes
func (m *Manager) ScanProcesses(query string) ([]ManagedProcess, error) {
	// 根据查询类型选择不同的扫描方法
	if strings.HasPrefix(query, ScriptPrefix) {
		// 直接执行内联脚本
		scriptContent := strings.TrimPrefix(query, ScriptPrefix)
		return m.scanWithScript(scriptContent)
	} else if strings.HasPrefix(query, FileScriptPrefix) {
		// 从文件加载脚本并执行
		scriptPath := strings.TrimPrefix(query, FileScriptPrefix)
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read script file: %v", err)
		}
		return m.scanWithScript(string(content))
	} else {
		// 使用标准的Unix进程扫描
		return m.scanUnixProcesses(query)
	}
}

// scanWithScript scans processes using a custom script
func (m *Manager) scanWithScript(script string) ([]ManagedProcess, error) {
	// 创建一个临时脚本文件
	// 在实际实现中，应该使用更安全的方式处理临时文件
	// 这里为了简化示例，直接执行脚本内容

	// 执行脚本
	cmd := exec.Command("sh", "-c", script)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %v, output: %s", err, output.String())
	}

	// 解析脚本输出，期望每行包含一个PID
	var processes []ManagedProcess
	lines := strings.Split(output.String(), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试将每行解析为PID
		pid, err := strconv.Atoi(line)
		if err != nil {
			// 如果不是纯PID，则忽略该行或记录警告
			continue
		}

		// 通过PID获取进程信息
		process, err := m.getProcessByPID(pid)
		if err != nil {
			// 如果无法获取进程信息，则忽略该PID或记录警告
			continue
		}

		processes = append(processes, process)
	}

	return processes, nil
}

// getProcessByPID 获取指定PID的进程详细信息
func (m *Manager) getProcessByPID(pid int) (ManagedProcess, error) {
	// 使用ps命令获取特定PID的详细信息
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid,user,comm,args")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ManagedProcess{}, fmt.Errorf("failed to get process info for PID %d: %v, output: %s", pid, err, string(output))
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return ManagedProcess{}, fmt.Errorf("No process info found for PID %d", pid)
	}

	// 解析第二行（第一行是表头）
	line := strings.TrimSpace(lines[1])
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return ManagedProcess{}, fmt.Errorf("Invalid process info for PID %d", pid)
	}

	// 提取进程信息
	username := fields[1]
	commandName := fields[2]
	var args []string
	if len(fields) > 3 {
		args = fields[3:]
	}

	// 创建ManagedProcess对象
	process := ManagedProcess{
		PID:    pid,
		Name:   commandName,
		Status: StatusRunning,
		// Use CommandConfig
		Command: CommandConfig{
			Command: commandName,
			Args:    args,
		},
		// Initialize EnvironmentVariable array
		Env: []EnvironmentVariable{},
		// Initialize other fields
		StartCommand:  CommandConfig{},
		StopCommand:   CommandConfig{},
		RestartPolicy: RestartPolicyNever,
	}

	return process, nil
}

// scanUnixProcesses scans processes on Unix/Linux/macOS systems
func (m *Manager) scanUnixProcesses(query string) ([]ManagedProcess, error) {
	// Use ps command to get process information
	cmd := exec.Command("ps", "-eo", "pid,user,comm,args,etime")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Failed to scan Unix processes: %v, output: %s", err, string(output))
	}

	var processes []ManagedProcess

	// Split output lines
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("Invalid process output")
	}

	// Compile regex for query matching
	queryRegex, err := regexp.Compile(query)
	if err != nil {
		// If not a valid regex, use as plain string match
		queryRegex, _ = regexp.Compile(regexp.QuoteMeta(query))
	}

	// Parse each line (skip header)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Split the line into fields
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Parse PID
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// Get username, command name, and arguments
		username := fields[1]
		commandName := fields[2]

		// Join all arguments
		var args []string
		if len(fields) > 3 {
			args = fields[3:]
		}

		// Check if matches query
		if !queryRegex.MatchString(line) {
			continue
		}

		// Create ManagedProcess object
		process := ManagedProcess{
			PID:    pid,
			Name:   commandName,
			Status: StatusRunning,
			// Use CommandConfig
			Command: CommandConfig{
				Command: commandName,
				Args:    args,
			},
			// Initialize EnvironmentVariable array
			Env: []EnvironmentVariable{},
			// Initialize other fields
			StartCommand:  CommandConfig{},
			StopCommand:   CommandConfig{},
			RestartPolicy: RestartPolicyNever,
		}

		processes = append(processes, process)
	}

	return processes, nil
}

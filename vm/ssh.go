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

package vm

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// SSHClient 表示模拟的SSH客户端
type SSHClient struct {
	username   string
	authorized bool
}

// CommandLog 表示命令执行日志
type CommandLog struct {
	Command   string    `json:"command"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output"`
	IsAllowed bool      `json:"is_allowed"`
}

// 命令限制列表
var restrictedCommands = map[string]bool{
	"rm": true, "rmdir": true, "mkdir": true, "chmod": true, "chown": true,
	"kill": true, "reboot": true, "shutdown": true, "init": true, "poweroff": true,
}

// NewSSHClient 创建一个新的SSH客户端
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	// 这里可以添加用户认证逻辑
	// 由于是模拟SSH，暂时只检查用户名
	if config.Username == "" {
		return nil, fmt.Errorf("username must be provided")
	}

	return &SSHClient{
		username:   config.Username,
		authorized: true,
	}, nil
}

// Connect 建立SSH连接（模拟）
func (c *SSHClient) Connect(host string, port int) error {
	// 模拟SSH连接，实际不需要建立网络连接
	if !c.authorized {
		return fmt.Errorf("unauthorized user")
	}

	log.Printf("SSH connected (simulated) as user: %s", c.username)
	return nil
}

// ExecuteCommand 执行命令（直接在本地执行）
func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	if !c.authorized {
		return "", fmt.Errorf("unauthorized user")
	}

	// 检查命令是否被限制
	cmdName := strings.Fields(cmd)[0]
	if restrictedCommands[cmdName] {
		log.Printf("Command execution denied: %s (restricted command)", cmd)
		return "", fmt.Errorf("command not allowed: %s", cmdName)
	}

	// 记录命令执行
	log.Printf("Executing command: %s (user: %s)", cmd, c.username)

	// 执行命令
	var stdout, stderr bytes.Buffer
	cmdObj := exec.Command("cmd", "/C", cmd) // Windows系统
	// cmdObj := exec.Command("sh", "-c", cmd) // Linux系统
	cmdObj.Stdout = &stdout
	cmdObj.Stderr = &stderr

	err := cmdObj.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", fmt.Errorf("failed to execute command: %v", err)
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}

	// 记录命令执行结果
	commandLog := &CommandLog{
		Command:   cmd,
		Username:  c.username,
		Timestamp: time.Now(),
		ExitCode:  exitCode,
		Output:    output,
		IsAllowed: true,
	}

	// 打印命令日志
	log.Printf("Command executed: %+v", commandLog)

	return output, nil
}

// Close 关闭SSH连接（模拟）
func (c *SSHClient) Close() error {
	log.Printf("SSH connection closed (simulated) for user: %s", c.username)
	return nil
}

// AddRestrictedCommand 添加被限制的命令
func AddRestrictedCommand(cmd string) {
	restrictedCommands[cmd] = true
}

// RemoveRestrictedCommand 移除被限制的命令
func RemoveRestrictedCommand(cmd string) {
	delete(restrictedCommands, cmd)
}

// IsCommandRestricted 检查命令是否被限制
func IsCommandRestricted(cmd string) bool {
	cmdName := strings.Fields(cmd)[0]
	return restrictedCommands[cmdName]
}

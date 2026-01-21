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
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient 表示真正的SSH客户端
type SSHClient struct {
	username   string
	client     *ssh.Client
	config     *ssh.ClientConfig
	authorized bool
	session    *ssh.Session
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
	if config.Username == "" {
		return nil, fmt.Errorf("username must be provided")
	}

	// 创建SSH客户端配置
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// 忽略主机密钥验证（生产环境中应该验证）
			return nil
		},
	}

	// 添加认证方式
	if config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	}

	if config.KeyPath != "" {
		// 支持私钥认证
		// 读取私钥文件内容
		privateKeyBytes, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %v", err)
		}

		// 解析私钥
		privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(privateKey))
	}

	return &SSHClient{
		username:   config.Username,
		config:     sshConfig,
		authorized: true,
	}, nil
}

// Connect 建立SSH连接
func (c *SSHClient) Connect(host string, port int) error {
	if !c.authorized {
		return fmt.Errorf("unauthorized user")
	}

	// 建立SSH连接
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	c.client = client
	log.Printf("SSH connected to %s as user: %s", addr, c.username)
	return nil
}

// ExecuteCommand 执行命令（通过SSH连接）
func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	if !c.authorized || c.client == nil {
		return "", fmt.Errorf("not connected to SSH server")
	}

	// 检查命令是否被限制
	cmdName := strings.Fields(cmd)[0]
	if restrictedCommands[cmdName] {
		log.Printf("Command execution denied: %s (restricted command)", cmd)
		return "", fmt.Errorf("command not allowed: %s", cmdName)
	}

	// 记录命令执行
	log.Printf("Executing command: %s (user: %s)", cmd, c.username)

	// 创建SSH会话
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// 执行命令
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(cmd)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
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

// CreateSession 创建一个新的SSH会话
func (c *SSHClient) CreateSession() (*ssh.Session, error) {
	if !c.authorized || c.client == nil {
		return nil, fmt.Errorf("not connected to SSH server")
	}

	// 创建SSH会话
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %v", err)
	}

	c.session = session
	return session, nil
}

// WindowChangeCallback 返回一个窗口大小变化的回调函数
func (c *SSHClient) WindowChangeCallback() func(int, int) {
	return func(w, h int) {
		if c.session != nil {
			c.session.WindowChange(h, w)
		}
	}
}

// Close 关闭SSH连接
func (c *SSHClient) Close() error {
	if c.session != nil {
		c.session.Close()
		log.Printf("SSH session closed for user: %s", c.username)
	}

	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			return fmt.Errorf("failed to close SSH connection: %v", err)
		}
		log.Printf("SSH connection closed for user: %s", c.username)
	}
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

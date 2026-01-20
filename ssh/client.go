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

package ssh

import (
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// ClientConfig 表示SSH客户端配置
type ClientConfig struct {
	Username   string
	Password   string
	PrivateKey string
}

// Client 表示SSH客户端
type Client struct {
	config *ssh.ClientConfig
	client *ssh.Client
	addr   string
}

// NewClient 创建一个新的SSH客户端
func NewClient(host string, port int, config *ClientConfig) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	// 构建SSH客户端配置
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// 忽略主机密钥验证（生产环境中应该验证）
			return nil
		},
	}

	// 添加认证方式
	if config.PrivateKey != "" {
		// 使用私钥认证
		key, err := parsePrivateKey(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(key))
	}

	if config.Password != "" {
		// 使用密码认证
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	return &Client{
		config: sshConfig,
		client: client,
		addr:   addr,
	}, nil
}

// CreateSession 创建一个新的SSH会话
func (c *Client) CreateSession() (*Session, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return &Session{
		session: sess,
	}, nil
}

// Close 关闭SSH连接
func (c *Client) Close() error {
	return c.client.Close()
}

// Session 表示SSH会话
type Session struct {
	session *ssh.Session
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

// SetStdin 设置会话的标准输入
func (s *Session) SetStdin(r io.Reader) {
	s.stdin = r
	s.session.Stdin = r
}

// SetStdout 设置会话的标准输出
func (s *Session) SetStdout(w io.Writer) {
	s.stdout = w
	s.session.Stdout = w
}

// SetStderr 设置会话的标准错误
func (s *Session) SetStderr(w io.Writer) {
	s.stderr = w
	s.session.Stderr = w
}

// Pty 请求PTY
func (s *Session) Pty(term string, h, w, hp, wp int) error {
	return s.session.RequestPty(term, h, w, ssh.TerminalModes{
		ssh.ECHO:          1,     // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	})
}

// WindowChange 处理窗口大小变化
func (s *Session) WindowChange(h, w int) error {
	return s.session.WindowChange(h, w)
}

// Shell 启动交互式shell
func (s *Session) Shell() error {
	return s.session.Shell()
}

// Wait 等待会话结束
func (s *Session) Wait() error {
	return s.session.Wait()
}

// Close 关闭会话
func (s *Session) Close() error {
	return s.session.Close()
}

// parsePrivateKey 解析私钥
func parsePrivateKey(privateKeyStr string) (ssh.Signer, error) {
	// 检查是否是文件路径
	if _, err := os.Stat(privateKeyStr); err == nil {
		// 读取私钥文件
		privateKeyData, err := os.ReadFile(privateKeyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %v", err)
		}
		return ssh.ParsePrivateKey(privateKeyData)
	}

	// 尝试直接解析私钥字符串
	return ssh.ParsePrivateKey([]byte(privateKeyStr))
}

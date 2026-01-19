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
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient 表示SSH客户端
type SSHClient struct {
	config *ssh.ClientConfig
	client *ssh.Client
}

// NewSSHClient 创建一个新的SSH客户端
func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// 为了简化，暂时接受所有主机密钥
			return nil
		},
		Timeout: 30 * time.Second,
	}

	// 配置认证方式
	if config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	} else if config.KeyPath != "" {
		privateKey, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %v", err)
		}

		key, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}

		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(key))
	} else {
		return nil, fmt.Errorf("either password or private key must be provided")
	}

	return &SSHClient{
			config: sshConfig,
		},
		nil
}

// Connect 建立SSH连接
func (c *SSHClient) Connect(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	c.client = client
	log.Printf("SSH connected to %s:%d", host, port)
	return nil
}

// ExecuteCommand 执行SSH命令
func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("SSH client not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Close 关闭SSH连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		log.Printf("Closing SSH connection")
		return c.client.Close()
	}
	return nil
}

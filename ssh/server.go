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
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/casuallc/vigil/vm"
	"github.com/gliderlabs/ssh"
)

// VMInfo 表示虚拟机信息
type VMInfo struct {
	Name     string
	IP       string
	Port     int
	Username string
}

// ServerConfig 表示SSH服务器配置
type ServerConfig struct {
	Host             string
	Port             int
	AuthorizedKeys   string
	AuditLogPath     string
	VMs              []VMInfo // 从CLI传递的VM列表
	TargetHost       string
	TargetPort       int
	TargetUsername   string
	TargetPassword   string
	TargetPrivateKey string
}

// Server 表示SSH服务器
type Server struct {
	config    ServerConfig
	ssh       *ssh.Server
	log       io.WriteCloser
	vmManager *vm.Manager
}

// NewServer 创建一个新的SSH服务器
func NewServer(config ServerConfig) (*Server, error) {
	// 验证配置
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 2222
	}

	// 打开审计日志文件
	logFile, err := os.OpenFile(config.AuditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %v", err)
	}

	// 创建VM管理器
	vmManager := vm.NewManager()

	// 将从CLI传递的VM信息添加到VM管理器中
	for _, vmInfo := range config.VMs {
		vm := vm.NewVM(vmInfo.Name, vmInfo.IP, vmInfo.Port, vmInfo.Username)
		vmManager.AddVM(vm)
	}

	// 创建SSH服务器配置
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler: func(sess ssh.Session) {
			// 处理SSH会话
			handleSession(sess, config, logFile, vmManager)
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			// 这里可以添加公钥认证逻辑
			// 暂时允许所有公钥
			return true
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			// 这里可以添加密码认证逻辑
			// 暂时允许所有密码
			return true
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": func(sess ssh.Session) {
				// 处理SFTP子系统（可选）
				log.Println("SFTP subsystem not implemented")
			},
		},
	}

	return &Server{
		config:    config,
		ssh:       sshServer,
		log:       logFile,
		vmManager: vmManager,
	}, nil
}

// Start 启动SSH服务器
func (s *Server) Start() error {
	log.Printf("Starting SSH server on %s:%d", s.config.Host, s.config.Port)
	return s.ssh.ListenAndServe()
}

// Stop 停止SSH服务器
func (s *Server) Stop() error {
	log.Printf("Stopping SSH server on %s:%d", s.config.Host, s.config.Port)
	s.log.Close()
	return s.ssh.Shutdown(context.Background())
}

// selectHost 显示VM列表并让用户选择要连接的主机
func selectHost(sess ssh.Session, vmManager *vm.Manager) (*vm.VM, error) {
	// 显示欢迎信息
	sess.Write([]byte("\nWelcome to Go Bastion\n\n"))

	// 获取所有VM
	vms := vmManager.ListVMs()
	if len(vms) == 0 {
		sess.Write([]byte("No hosts available\n"))
		return nil, fmt.Errorf("no hosts available")
	}

	// 显示VM列表
	for i, vm := range vms {
		line := fmt.Sprintf("[%d] %s@%s:%d\n", i+1, vm.Username, vm.IP, vm.Port)
		sess.Write([]byte(line))
	}

	// 提示用户选择
	sess.Write([]byte("\nSelect host > "))

	// 读取用户输入
	reader := bufio.NewReader(sess)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// 处理输入
	input = strings.TrimSpace(input)
	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(vms) {
		sess.Write([]byte("Invalid selection\n"))
		return nil, fmt.Errorf("invalid selection")
	}

	// 返回用户选择的VM
	return vms[index-1], nil
}

// handleSession 处理SSH会话
func handleSession(sess ssh.Session, config ServerConfig, auditLog io.Writer, vmManager *vm.Manager) {
	// 获取客户端信息
	clientAddr := sess.RemoteAddr()
	username := sess.User()

	log.Printf("New SSH connection from %s as user %s", clientAddr, username)

	// 获取PTY信息（如果有）
	ptyInfo, resizeChan, ok := sess.Pty()
	if !ok {
		fmt.Fprintln(sess, "PTY not requested")
		sess.Exit(1)
		return
	}

	// 让用户选择要连接的VM
	selectedVM, err := selectHost(sess, vmManager)
	if err != nil {
		fmt.Fprintf(sess, "Failed to select host: %v\n", err)
		sess.Exit(1)
		return
	}

	// 提示用户输入密码
	sess.Write([]byte(fmt.Sprintf("\nPassword for %s@%s: ", selectedVM.Username, selectedVM.IP)))

	// 读取用户输入的密码
	reader := bufio.NewReader(sess)
	password, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(sess, "Failed to read password: %v\n", err)
		sess.Exit(1)
		return
	}
	password = strings.TrimSpace(password)

	// 连接到目标主机
	client, err := NewClient(selectedVM.IP, selectedVM.Port, &ClientConfig{
		Username:   selectedVM.Username,
		Password:   password,                // 使用用户输入的密码
		PrivateKey: config.TargetPrivateKey, // 暂时使用配置中的私钥，可以扩展为每个VM都有自己的私钥
	})
	if err != nil {
		fmt.Fprintf(sess, "Failed to connect to target host: %v\n", err)
		sess.Exit(1)
		return
	}
	defer client.Close()

	// 在目标主机上创建会话
	targetSess, err := client.CreateSession()
	if err != nil {
		fmt.Fprintf(sess, "Failed to create session on target host: %v\n", err)
		sess.Exit(1)
		return
	}
	defer targetSess.Close()

	// 设置IO重定向
	targetSess.SetStdin(sess)
	targetSess.SetStdout(io.MultiWriter(sess, auditLog))
	targetSess.SetStderr(io.MultiWriter(sess.Stderr(), auditLog))

	// 请求PTY
	err = targetSess.Pty(ptyInfo.Term, ptyInfo.Window.Height, ptyInfo.Window.Width, 0, 0)
	if err != nil {
		fmt.Fprintf(sess, "Failed to request PTY on target host: %v\n", err)
		sess.Exit(1)
		return
	}

	// 处理窗口resize
	go func() {
		for resize := range resizeChan {
			if err := targetSess.WindowChange(resize.Height, resize.Width); err != nil {
				log.Printf("Failed to resize window on target host: %v", err)
				break
			}
		}
	}()

	// 启动shell
	log.Printf("Starting shell for %s@%s to %s@%s:%d", username, clientAddr, selectedVM.Username, selectedVM.IP, selectedVM.Port)
	if err := targetSess.Shell(); err != nil {
		fmt.Fprintf(sess, "Failed to start shell on target host: %v\n", err)
		sess.Exit(1)
		return
	}

	// 等待shell退出
	if err := targetSess.Wait(); err != nil {
		log.Printf("Shell exited with error: %v", err)
	}

	log.Printf("SSH connection from %s closed", clientAddr)
}

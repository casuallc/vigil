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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/inspection"
	"github.com/casuallc/vigil/proc"
	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/websocket"
)

// Client represents the HTTP client for the Vigil API
type Client struct {
	httpClient *http.Client
	host       string
	baseURL    string
	basicUser  string
	basicPass  string
}

// NewClient creates a new API client
func NewClient(host string) *Client {
	// 确保host包含协议和端口
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	// 解析URL
	parsedURL, err := url.Parse(host)
	if err != nil {
		return &Client{
			httpClient: &http.Client{},
			host:       host,
			baseURL:    host,
		}
	}

	// 构建baseURL
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	c := &Client{
		httpClient: &http.Client{},
		host:       host,
		baseURL:    baseURL,
	}

	// 从 conf/app.conf 加载 Basic Auth 凭据
	confPath := filepath.Join("conf", "app.conf")
	if kv, err := common.LoadKeyValues(confPath); err == nil {
		c.basicUser = kv["BASIC_AUTH_USER"]
		c.basicPass = kv["BASIC_AUTH_PASS"]
	}

	return c
}

// Helper methods for HTTP requests
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.host, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 如果已配置 Basic Auth，则附加到请求
	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	return c.httpClient.Do(req)
}

// 新增：从非 2xx 响应构造详细错误
func (c *Client) errorFromResponse(resp *http.Response) error {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	msg := string(b)

	var obj map[string]interface{}
	if len(b) > 0 && json.Unmarshal(b, &obj) == nil {
		if s, ok := obj["error"].(string); ok && s != "" {
			msg = s
		} else if s, ok := obj["message"].(string); ok && s != "" {
			msg = s
		}
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
}

func (c *Client) getJSONResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// ScanProcesses Process management methods
func (c *Client) ScanProcesses(query string) ([]proc.ManagedProcess, error) {
	// 对 query 参数进行 URL 编码，避免空格等特殊字符导致请求错误
	encodedQuery := url.QueryEscape(query)
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/processes/scan?query=%s", encodedQuery), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var processes []proc.ManagedProcess
	if err := c.getJSONResponse(resp, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

func (c *Client) CreateProcess(process proc.ManagedProcess) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/add",
		process.Metadata.Namespace, process.Metadata.Name), process)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return c.errorFromResponse(resp)
	}

	return nil
}

// StartProcess starts a managed proc
func (c *Client) StartProcess(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	// 对路径参数进行 URL 编码
	encodedNamespace := url.QueryEscape(namespace)
	encodedName := url.QueryEscape(name)
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/start", encodedNamespace, encodedName), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// StopProcess stops a managed proc
func (c *Client) StopProcess(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	// 对路径参数进行 URL 编码
	encodedNamespace := url.QueryEscape(namespace)
	encodedName := url.QueryEscape(name)
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/stop", encodedNamespace, encodedName), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// RestartProcess restarts a managed proc
func (c *Client) RestartProcess(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	// 对路径参数进行 URL 编码
	encodedNamespace := url.QueryEscape(namespace)
	encodedName := url.QueryEscape(name)
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/restart", encodedNamespace, encodedName), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// GetProcess gets detailed information about a proc
func (c *Client) GetProcess(namespace, name string) (proc.ManagedProcess, error) {
	var process proc.ManagedProcess
	if namespace == "" {
		namespace = "default"
	}
	// 对路径参数进行 URL 编码
	encodedNamespace := url.QueryEscape(namespace)
	encodedName := url.QueryEscape(name)
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/namespaces/%s/processes/%s", encodedNamespace, encodedName), nil)
	if err != nil {
		return process, err
	}

	if resp.StatusCode != http.StatusOK {
		return process, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &process); err != nil {
		return process, err
	}

	return process, nil
}

// ListProcesses lists all managed processes
func (c *Client) ListProcesses(namespace string) ([]proc.ManagedProcess, error) {
	var url string
	if namespace == "" {
		url = fmt.Sprintf("/api/processes")
	} else {
		url = fmt.Sprintf("/api/namespaces/%s/processes", namespace)
	}

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var processes []proc.ManagedProcess
	if err := c.getJSONResponse(resp, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

// DeleteProcess deletes a managed proc
func (c *Client) DeleteProcess(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	// 对路径参数进行 URL 编码
	encodedNamespace := url.QueryEscape(namespace)
	encodedName := url.QueryEscape(name)
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/namespaces/%s/processes/%s", encodedNamespace, encodedName), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// UpdateProcess updates the configuration of a managed proc
func (c *Client) UpdateProcess(process proc.ManagedProcess) error {
	if process.Metadata.Namespace == "" {
		process.Metadata.Namespace = "default"
	}
	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/namespaces/%s/processes/%s",
		process.Metadata.Namespace, process.Metadata.Name), process)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// InspectProcess inspect a process
func (c *Client) InspectProcess(namespace, name string, inspectRequest inspection.Request) (inspection.Result, error) {
	var result inspection.Result

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/inspect", namespace, name), inspectRequest)
	if err != nil {
		return result, err
	}

	if resp.StatusCode != http.StatusOK {
		return result, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (c *Client) GetSystemResources() (proc.ResourceStats, error) {
	var resources proc.ResourceStats
	resp, err := c.doRequest("GET", "/api/resources/system", nil)
	if err != nil {
		return resources, err
	}

	if resp.StatusCode != http.StatusOK {
		return resources, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &resources); err != nil {
		return resources, err
	}

	return resources, nil
}

func (c *Client) GetProcessResources(pid int) (proc.ResourceStats, error) {
	var resources proc.ResourceStats
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/resources/process/%d", pid), nil)
	if err != nil {
		return resources, err
	}

	if resp.StatusCode != http.StatusOK {
		return resources, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &resources); err != nil {
		return resources, err
	}

	return resources, nil
}

func (c *Client) GetConfig() (config.Config, error) {
	var config config.Config
	resp, err := c.doRequest("GET", "/api/config", nil)
	if err != nil {
		return config, err
	}

	if resp.StatusCode != http.StatusOK {
		return config, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &config); err != nil {
		return config, err
	}

	return config, nil
}

func (c *Client) UpdateConfig(config config.Config) error {
	resp, err := c.doRequest("PUT", "/api/config", config)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

func (c *Client) CheckHealth() (bool, error) {
	resp, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

// ExecuteCommand executes a command or script on the server
func (c *Client) ExecuteCommand(command string, isFile bool, envVars []string) (string, error) {
	reqBody := map[string]interface{}{
		"command": command,
		"env":     envVars,
	}

	resp, err := c.doRequest("POST", "/api/exec", reqBody)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", c.errorFromResponse(resp)
	}

	// 读取响应内容
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ExecuteInspection 执行巡检检查
func (c *Client) ExecuteInspection(request inspection.Request) (inspection.Result, error) {
	var result inspection.Result

	resp, err := c.doRequest("POST", "/api/inspect", request)
	if err != nil {
		return result, err
	}

	if resp.StatusCode != http.StatusOK {
		return result, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &result); err != nil {
		return result, err
	}

	return result, nil
}

// ------------------------- VM Management Methods -------------------------

// ListVMs 列出所有VM
func (c *Client) ListVMs() ([]vm.VM, error) {
	var vms []vm.VM
	resp, err := c.doRequest("GET", "/api/vms", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &vms); err != nil {
		return nil, err
	}

	return vms, nil
}

// AddVM 添加一个新的VM
func (c *Client) AddVM(name, ip string, port int, username, password, keyPath string) (*vm.VM, error) {
	var newVM vm.VM
	vmData := vm.VM{
		Name:     name,
		IP:       ip,
		Port:     port,
		Username: username,
		Password: password,
		KeyPath:  keyPath,
	}

	resp, err := c.doRequest("POST", "/api/vms", vmData)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &newVM); err != nil {
		return nil, err
	}

	return &newVM, nil
}

// GetVM 获取VM详情
func (c *Client) GetVM(name string) (*vm.VM, error) {
	var vm vm.VM
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/%s", name), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &vm); err != nil {
		return nil, err
	}

	return &vm, nil
}

// DeleteVM 删除VM
func (c *Client) DeleteVM(name string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/vms/%s", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// SSHConnect SSH连接到VM
func (c *Client) SSHConnect(vmName, password string) error {
	reqBody := map[string]interface{}{
		"vm_name":  vmName,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/api/vms/ssh/connect", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// SSHExecute 在VM上执行SSH命令
func (c *Client) SSHExecute(vmName, command string) (string, error) {
	reqBody := map[string]interface{}{
		"vm_name": vmName,
		"command": command,
	}

	resp, err := c.doRequest("POST", "/api/vms/ssh/execute", reqBody)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", c.errorFromResponse(resp)
	}

	// 读取响应内容
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应JSON
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil // 如果无法解析JSON，直接返回原始响应
	}

	return result["output"], nil
}

// FileUpload 上传文件到VM
func (c *Client) FileUpload(vmName, sourcePath, targetPath, password string) error {
	// 打开本地文件
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// 创建multipart/form-data请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件字段
	fileField, err := writer.CreateFormFile("file", filepath.Base(sourcePath))
	if err != nil {
		return err
	}

	// 复制文件内容到multipart writer
	if _, err := io.Copy(fileField, sourceFile); err != nil {
		return err
	}

	// 添加目标路径字段
	if err := writer.WriteField("target_path", targetPath); err != nil {
		return err
	}

	// 添加VM名称字段
	if err := writer.WriteField("vm_name", vmName); err != nil {
		return err
	}

	// 添加密码字段
	if err := writer.WriteField("password", password); err != nil {
		return err
	}

	// 完成multipart writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// 创建请求
	req, err := http.NewRequest("POST", c.host+"/api/vms/file/upload", body)
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Content-Type", contentType)

	// 如果已配置 Basic Auth，则附加到请求
	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// FileDownload 从VM下载文件
func (c *Client) FileDownload(vmName, sourcePath, targetPath, password string) error {
	reqBody := map[string]interface{}{
		"vm_name":     vmName,
		"source_path": sourcePath,
		"password":    password,
	}

	// 发送请求
	resp, err := c.doRequest("POST", "/api/vms/file/download", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// 创建目标文件
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// 保存文件内容
	if _, err := io.Copy(targetFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save file content: %v", err)
	}

	return nil
}

// FileList 列出VM上的文件
func (c *Client) FileList(vmName, path string, maxDepth int, password string) ([]vm.FileInfo, error) {
	var files []vm.FileInfo

	reqBody := map[string]interface{}{
		"vm_name":   vmName,
		"path":      path,
		"max_depth": maxDepth,
		"password":  password,
	}

	resp, err := c.doRequest("POST", "/api/vms/file/list", reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &files); err != nil {
		return nil, err
	}

	return files, nil
}

// AddPermission 添加VM权限
func (c *Client) AddPermission(vmName, username string, permissions []string) error {
	permission := vm.Permission{
		VMName:      vmName,
		Username:    username,
		Permissions: permissions,
	}

	resp, err := c.doRequest("POST", "/api/vms/permissions", permission)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// RemovePermission 移除VM权限
func (c *Client) RemovePermission(vmName, username string, permissions []string) error {
	permission := vm.Permission{
		VMName:      vmName,
		Username:    username,
		Permissions: permissions,
	}

	resp, err := c.doRequest("DELETE", "/api/vms/permissions", permission)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// ListPermissions 列出VM权限
func (c *Client) ListPermissions(vmName string) ([]vm.Permission, error) {
	var permissions []vm.Permission
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/%s/permissions", vmName), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

// CheckPermission 检查VM权限
func (c *Client) CheckPermission(vmName, username, permission string) (bool, error) {
	reqBody := map[string]interface{}{
		"vm_name":    vmName,
		"username":   username,
		"permission": permission,
	}

	resp, err := c.doRequest("POST", "/api/vms/permissions/check", reqBody)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, c.errorFromResponse(resp)
	}

	// 读取响应内容
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// 解析响应JSON
	var result map[string]bool
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	return result["has_permission"], nil
}

// SSHWebSocket 建立WebSocket SSH连接
func (c *Client) SSHWebSocket(vmName, password string) (*websocket.Conn, error) {
	// 构建WebSocket URL
	wsScheme := "ws"
	if strings.HasPrefix(c.baseURL, "https://") {
		wsScheme = "wss"
	}

	// 构建基础URL（去掉http://或https://）
	baseURL := strings.TrimPrefix(c.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")

	// 构建完整的WebSocket URL
	wsURL := fmt.Sprintf("%s://%s/api/vms/ssh/ws?vm_name=%s&password=%s",
		wsScheme, baseURL, url.QueryEscape(vmName), url.QueryEscape(password))

	// 构建HTTP头部
	headers := http.Header{}

	// 如果已配置 Basic Auth，则添加到HTTP头部
	if c.basicUser != "" && c.basicPass != "" {
		// 手动构建BasicAuth头
		auth := c.basicUser + ":" + c.basicPass
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}

	// 建立WebSocket连接
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket SSH endpoint: %v", err)
	}

	return conn, nil
}

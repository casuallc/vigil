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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/websocket"
)

// ------------------------- VM Management Methods -------------------------

// ListVMs lists all VMs
func (c *Client) ListVMs() ([]vm.VM, error) {
	var vms []vm.VM
	resp, err := c.doRequest("GET", "/api/vms/servers", nil)
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

// AddVM adds a new VM
func (c *Client) AddVM(name, ip string, port int, username, password, keyPath string) (*vm.VM, error) {
	var newVM vm.VM
	vmData := vm.VM{
		IP:       ip,
		Port:     port,
		Username: username,
		Password: password,
		KeyPath:  keyPath,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/servers/%s", name), vmData)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, c.errorFromResponse(resp)
	}

	// 手动设置name字段，因为响应中可能不包含
	newVM.Name = name
	// 从响应中获取其他字段
	if err := c.getJSONResponse(resp, &newVM); err != nil {
		return nil, err
	}

	return &newVM, nil
}

// GetVM gets VM details
func (c *Client) GetVM(name string) (*vm.VM, error) {
	var vm vm.VM
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/servers/%s", name), nil)
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

// DeleteVM deletes a VM
func (c *Client) DeleteVM(name string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/vms/servers/%s", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// UpdateVM updates VM password and key path
func (c *Client) UpdateVM(name, password, keyPath string) error {
	reqBody := map[string]interface{}{
		"password": password,
		"key_path": keyPath,
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/vms/servers/%s", name), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// VMExec executes a command on a VM via SSH
func (c *Client) VMExec(vmName, command string, timeout int) (string, error) {
	reqBody := map[string]interface{}{
		"command": command,
		"timeout": timeout,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/servers/%s/exec", vmName), reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.errorFromResponse(resp)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// VMPing tests TCP connection to a VM
func (c *Client) VMPing(vmName string) (*PingResult, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/servers/%s/ping", vmName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var result PingResult
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SSHWebSocket establishes a WebSocket SSH connection
func (c *Client) SSHWebSocket(vmName string) (*websocket.Conn, error) {
	// 构建WebSocket URL
	wsScheme := "ws"
	if strings.HasPrefix(c.baseURL, "https://") {
		wsScheme = "wss"
	}

	// 构建基础URL（去掉http://或https://）
	baseURL := strings.TrimPrefix(c.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")

	// 构建完整的WebSocket URL
	wsURL := fmt.Sprintf("%s://%s/api/vms/ssh/ws?vm_name=%s",
		wsScheme, baseURL, url.QueryEscape(vmName))

	// 构建HTTP头部
	headers := http.Header{}

	// 如果已配置 Basic Auth，则添加到HTTP头部
	if c.basicUser != "" && c.basicPass != "" {
		// 手动构建BasicAuth头
		auth := c.basicUser + ":" + c.basicPass
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}

	// 创建自定义WebSocket拨号器，支持跳过证书验证
	dialer := c.internalWebSocketDialer()

	// 建立WebSocket连接
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket SSH endpoint: %v", err)
	}

	return conn, nil
}

// ListSSHConnections lists active SSH connections with filters
func (c *Client) ListSSHConnections(vmName, userName, clientIP string) ([]*SSHConnectionInfo, error) {
	var connections []*SSHConnectionInfo

	// Build query parameters
	params := []string{}
	if vmName != "" {
		params = append(params, "vm_name="+vmName)
	}
	if userName != "" {
		params = append(params, "user_name="+userName)
	}
	if clientIP != "" {
		params = append(params, "client_ip="+clientIP)
	}

	// Build URL with parameters
	connectionUrl := "/api/vms/ssh/connections"
	if len(params) > 0 {
		connectionUrl += "?" + strings.Join(params, "&")
	}

	resp, err := c.doRequest("GET", connectionUrl, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &connections); err != nil {
		return nil, err
	}

	return connections, nil
}

// CloseSSHConnection closes a specific SSH connection
func (c *Client) CloseSSHConnection(id string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/vms/ssh/connections/%s", id), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// CloseAllSSHConnections closes all SSH connections
func (c *Client) CloseAllSSHConnections() (int, error) {
	resp, err := c.doRequest("DELETE", "/api/vms/ssh/connections", nil)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, c.errorFromResponse(resp)
	}

	// Parse the response to get the count
	var result map[string]interface{}
	if err := c.getJSONResponse(resp, &result); err != nil {
		return 0, err
	}

	count, ok := result["count"].(float64)
	if !ok {
		return 0, fmt.Errorf("could not parse count from response")
	}

	return int(count), nil
}

// ------------------------- Group Management Methods -------------------------

// AddGroup adds a VM group
func (c *Client) AddGroup(name, description string, vms []string) error {
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"vms":         vms,
	}

	resp, err := c.doRequest("POST", "/api/vms/groups", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// ListGroups lists all VM groups
func (c *Client) ListGroups() ([]*vm.Group, error) {
	resp, err := c.doRequest("GET", "/api/vms/groups", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var groups []*vm.Group
	if err := c.getJSONResponse(resp, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroup gets VM group details
func (c *Client) GetGroup(name string) (*vm.Group, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/groups/%s", name), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var group *vm.Group
	if err := c.getJSONResponse(resp, &group); err != nil {
		return nil, err
	}

	return group, nil
}

// UpdateGroup updates a VM group
func (c *Client) UpdateGroup(name, description string, vms []string) error {
	reqBody := map[string]interface{}{
		"description": description,
		"vms":         vms,
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/vms/groups/%s", name), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// DeleteGroup deletes a VM group
func (c *Client) DeleteGroup(name string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/vms/groups/%s", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// ------------------------- Permission Management Methods -------------------------

// AddPermission adds VM permission
func (c *Client) AddPermission(vmName, username string, permissions []string) error {
	permission := map[string]interface{}{
		"username":    username,
		"permissions": permissions,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/permissions/%s", vmName), permission)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// RemovePermission removes VM permission
func (c *Client) RemovePermission(vmName, username string, permissions []string) error {
	permission := map[string]interface{}{
		"username":    username,
		"permissions": permissions,
	}

	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/vms/permissions/%s", vmName), permission)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// ListPermissions lists VM permissions
func (c *Client) ListPermissions(vmName string) ([]vm.Permission, error) {
	var permissions []vm.Permission
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/vms/servers/%s/permissions", vmName), nil)
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

// CheckPermission checks VM permission
func (c *Client) CheckPermission(vmName, username, permission string) (bool, error) {
	reqBody := map[string]interface{}{
		"username":   username,
		"permission": permission,
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/permissions/%s/check", vmName), reqBody)
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

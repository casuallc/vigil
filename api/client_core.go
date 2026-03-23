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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/config"
	"github.com/gorilla/websocket"
)

// Client represents the HTTP client for the Vigil API
type Client struct {
	httpClient         *http.Client
	host               string
	baseURL            string
	basicUser          string
	basicPass          string
	insecureSkipVerify bool
}

// NewClient creates a new API client
func NewClient(host string, insecureSkipVerify ...bool) *Client {
	// 确保host包含协议和端口
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	// 解析URL
	parsedURL, err := url.Parse(host)
	if err != nil {
		return &Client{
			httpClient:         &http.Client{},
			host:               host,
			baseURL:            host,
			insecureSkipVerify: len(insecureSkipVerify) > 0 && insecureSkipVerify[0],
		}
	}

	// 构建baseURL
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// 创建HTTP客户端，支持跳过证书验证
	client := &Client{
		host:               host,
		baseURL:            baseURL,
		insecureSkipVerify: len(insecureSkipVerify) > 0 && insecureSkipVerify[0],
	}

	// 配置HTTP客户端的TLS选项
	client.httpClient = &http.Client{}
	if parsedURL.Scheme == "https" && client.insecureSkipVerify {
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// 从 conf/app.conf 加载 Basic Auth 凭据
	confPath := filepath.Join("conf", "app.conf")
	if kv, err := common.LoadKeyValues(confPath); err == nil {
		client.basicUser = kv["BASIC_AUTH_USER"]
		client.basicPass = kv["BASIC_AUTH_PASS"]
	}

	return client
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

// PingResult represents the result of a ping test
type PingResult struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Message string `json:"message,omitempty"`
}

// User represents a user in the system (for API client use)
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Profile fields
	Avatar   string `json:"avatar,omitempty"`   // User avatar URL
	Nickname string `json:"nickname,omitempty"` // User nickname
	Region   string `json:"region,omitempty"`   // User region/location
	Configs  string `json:"configs,omitempty"`  // User configuration (JSON string)
}

// internalWebSocketDialer returns a WebSocket dialer with TLS config
func (c *Client) internalWebSocketDialer() *websocket.Dialer {
	dialer := &websocket.Dialer{}
	if c.insecureSkipVerify {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return dialer
}

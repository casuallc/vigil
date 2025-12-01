package api

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/inspection"
  "github.com/casuallc/vigil/proc"
  "io"
  "net/http"
  "net/url"
  "path/filepath"
)

// Client represents the HTTP client for the Vigil API
type Client struct {
  httpClient *http.Client
  host       string
  basicUser  string
  basicPass  string
}

// NewClient creates a new API client
func NewClient(host string) *Client {
  c := &Client{
    httpClient: &http.Client{},
    host:       host,
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

package client

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/proc"
  "io"
  "net/http"
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

func (c *Client) getJSONResponse(resp *http.Response, v interface{}) error {
  defer resp.Body.Close()
  return json.NewDecoder(resp.Body).Decode(v)
}

// ScanProcesses Process management methods
func (c *Client) ScanProcesses(query string) ([]proc.ManagedProcess, error) {
  resp, err := c.doRequest("GET", fmt.Sprintf("/api/processes/scan?query=%s", query), nil)
  if err != nil {
    return nil, err
  }

  if resp.StatusCode != http.StatusOK {
    return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

// StartProcess starts a managed proc
func (c *Client) StartProcess(namespace, name string) error {
  if namespace == "" {
    namespace = "default"
  }
  resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/start", namespace, name), nil)
  if err != nil {
    return err
  }

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

// StopProcess stops a managed proc
func (c *Client) StopProcess(namespace, name string) error {
  if namespace == "" {
    namespace = "default"
  }
  resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/stop", namespace, name), nil)
  if err != nil {
    return err
  }

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

// RestartProcess restarts a managed proc
func (c *Client) RestartProcess(namespace, name string) error {
  if namespace == "" {
    namespace = "default"
  }
  resp, err := c.doRequest("POST", fmt.Sprintf("/api/namespaces/%s/processes/%s/restart", namespace, name), nil)
  if err != nil {
    return err
  }

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

// GetProcess gets detailed information about a proc
func (c *Client) GetProcess(namespace, name string) (proc.ManagedProcess, error) {
  var process proc.ManagedProcess
  if namespace == "" {
    namespace = "default"
  }
  resp, err := c.doRequest("GET", fmt.Sprintf("/api/namespaces/%s/processes/%s", namespace, name), nil)
  if err != nil {
    return process, err
  }

  if resp.StatusCode != http.StatusOK {
    return process, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
  resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/namespaces/%s/processes/%s", namespace, name), nil)
  if err != nil {
    return err
  }

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

func (c *Client) GetSystemResources() (proc.ResourceStats, error) {
  var resources proc.ResourceStats
  resp, err := c.doRequest("GET", "/api/resources/system", nil)
  if err != nil {
    return resources, err
  }

  if resp.StatusCode != http.StatusOK {
    return resources, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return resources, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return config, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
    return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  // 读取响应内容
  defer resp.Body.Close()
  body, err := io.ReadAll(resp.Body)
  if err != nil {
    return "", err
  }

  return string(body), nil
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
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
  }

  return nil
}

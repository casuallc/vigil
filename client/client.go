package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/process"
	"io"
	"net/http"
)

// Client represents the HTTP client for the Vigil API

type Client struct {
	httpClient *http.Client
	host       string
}

// NewClient creates a new API client
func NewClient(host string) *Client {
	return &Client{
		httpClient: &http.Client{},
		host:       host,
	}
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

	return c.httpClient.Do(req)
}

func (c *Client) getJSONResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// ScanProcesses Process management methods
func (c *Client) ScanProcesses(query string) ([]process.ManagedProcess, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/processes/scan?query=%s", query), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var processes []process.ManagedProcess
	if err := c.getJSONResponse(resp, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

func (c *Client) ManageProcess(process process.ManagedProcess) error {
	resp, err := c.doRequest("POST", "/api/processes/manage", process)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) StartProcess(name string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/processes/%s/start", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) StopProcess(name string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/processes/%s/stop", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) RestartProcess(name string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/processes/%s/restart", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetProcess(name string) (process.ManagedProcess, error) {
	var process process.ManagedProcess
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/processes/%s", name), nil)
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

func (c *Client) ListProcesses() ([]process.ManagedProcess, error) {
	resp, err := c.doRequest("GET", "/api/processes", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var processes []process.ManagedProcess
	if err := c.getJSONResponse(resp, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

func (c *Client) GetSystemResources() (process.ResourceStats, error) {
	var resources process.ResourceStats
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

func (c *Client) GetProcessResources(pid int) (process.ResourceStats, error) {
	var resources process.ResourceStats
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

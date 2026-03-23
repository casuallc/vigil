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
	"fmt"
	"net/http"
	"net/url"

	"github.com/casuallc/vigil/inspection"
	"github.com/casuallc/vigil/models"
)

// ScanProcesses scans processes matching the query
func (c *Client) ScanProcesses(query string) ([]models.ManagedProcess, error) {
	// 对 query 参数进行 URL 编码，避免空格等特殊字符导致请求错误
	encodedQuery := url.QueryEscape(query)
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/processes/scan?query=%s", encodedQuery), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var processes []models.ManagedProcess
	if err := c.getJSONResponse(resp, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

func (c *Client) CreateProcess(process models.ManagedProcess) error {
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
func (c *Client) GetProcess(namespace, name string) (models.ManagedProcess, error) {
	var process models.ManagedProcess
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
func (c *Client) ListProcesses(namespace string) ([]models.ManagedProcess, error) {
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

	var processes []models.ManagedProcess
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
func (c *Client) UpdateProcess(process models.ManagedProcess) error {
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

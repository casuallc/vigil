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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/casuallc/vigil/filetransfer"
)

// doTransferRequest issues a request to a file-transfer agent endpoint. These
// endpoints share vigil's global Basic Auth, so it reuses the same credentials
// as every other API call.
func (c *Client) doTransferRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(data)
	}
	req, err := http.NewRequest(method, c.host+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}
	return c.httpClient.Do(req)
}

// FsList lists a directory on the agent.
func (c *Client) FsList(path string) ([]filetransfer.FsItem, error) {
	resp, err := c.doTransferRequest(http.MethodGet, "/api/fs/list?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var items []filetransfer.FsItem
	return items, c.getJSONResponse(resp, &items)
}

// FsStat returns stat info for a path on the agent.
func (c *Client) FsStat(path string) (map[string]interface{}, error) {
	resp, err := c.doTransferRequest(http.MethodGet, "/api/fs/stat?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var stat map[string]interface{}
	return stat, c.getJSONResponse(resp, &stat)
}

// TransferCreateTask creates a transfer task and returns its id.
func (c *Client) TransferCreateTask(config filetransfer.TaskConfig) (int64, error) {
	resp, err := c.doTransferRequest(http.MethodPost, "/api/transfer/tasks", config)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, c.errorFromResponse(resp)
	}
	var result struct {
		TaskID int64 `json:"taskId"`
	}
	if err := c.getJSONResponse(resp, &result); err != nil {
		return 0, err
	}
	return result.TaskID, nil
}

// TransferListTasks returns all task configs.
func (c *Client) TransferListTasks() ([]filetransfer.TaskConfig, error) {
	resp, err := c.doTransferRequest(http.MethodGet, "/api/transfer/tasks", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var tasks []filetransfer.TaskConfig
	return tasks, c.getJSONResponse(resp, &tasks)
}

// TransferGetTask returns one task config.
func (c *Client) TransferGetTask(id int64) (filetransfer.TaskConfig, error) {
	var task filetransfer.TaskConfig
	resp, err := c.doTransferRequest(http.MethodGet, fmt.Sprintf("/api/transfer/tasks/%d", id), nil)
	if err != nil {
		return task, err
	}
	if resp.StatusCode != http.StatusOK {
		return task, c.errorFromResponse(resp)
	}
	return task, c.getJSONResponse(resp, &task)
}

// TransferDeleteTask deletes a task.
func (c *Client) TransferDeleteTask(id int64) error {
	return c.transferSimple(http.MethodDelete, fmt.Sprintf("/api/transfer/tasks/%d", id))
}

// TransferStart/Pause/Resume/Cancel drive the task lifecycle.
func (c *Client) TransferStart(id int64) error {
	return c.transferSimple(http.MethodPost, fmt.Sprintf("/api/transfer/tasks/%d/start", id))
}
func (c *Client) TransferPause(id int64) error {
	return c.transferSimple(http.MethodPost, fmt.Sprintf("/api/transfer/tasks/%d/pause", id))
}
func (c *Client) TransferResume(id int64) error {
	return c.transferSimple(http.MethodPost, fmt.Sprintf("/api/transfer/tasks/%d/resume", id))
}
func (c *Client) TransferCancel(id int64) error {
	return c.transferSimple(http.MethodPost, fmt.Sprintf("/api/transfer/tasks/%d/cancel", id))
}

// TransferStatus returns the aggregated status of a task.
func (c *Client) TransferStatus(id int64) (filetransfer.TaskStatus, error) {
	var status filetransfer.TaskStatus
	resp, err := c.doTransferRequest(http.MethodGet, fmt.Sprintf("/api/transfer/tasks/%d/status", id), nil)
	if err != nil {
		return status, err
	}
	if resp.StatusCode != http.StatusOK {
		return status, c.errorFromResponse(resp)
	}
	return status, c.getJSONResponse(resp, &status)
}

// TransferProgress returns per-file progress of a task.
func (c *Client) TransferProgress(id int64) ([]filetransfer.FileProgress, error) {
	resp, err := c.doTransferRequest(http.MethodGet, fmt.Sprintf("/api/transfer/tasks/%d/progress", id), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var progress []filetransfer.FileProgress
	return progress, c.getJSONResponse(resp, &progress)
}

func (c *Client) transferSimple(method, path string) error {
	resp, err := c.doTransferRequest(method, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

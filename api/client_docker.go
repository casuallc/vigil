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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/casuallc/vigil/docker"
	"github.com/docker/docker/api/types"
	"github.com/gorilla/websocket"
)

// DockerListContainers lists all Docker containers.
func (c *Client) DockerListContainers(all bool) ([]docker.ContainerSummary, error) {
	path := "/api/docker/containers?all=" + strconv.FormatBool(all)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var containers []docker.ContainerSummary
	if err := c.getJSONResponse(resp, &containers); err != nil {
		return nil, err
	}
	return containers, nil
}

// DockerInspectContainer inspects a container.
func (c *Client) DockerInspectContainer(id string) (types.ContainerJSON, error) {
	var info types.ContainerJSON
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/docker/containers/%s", id), nil)
	if err != nil {
		return info, err
	}

	if resp.StatusCode != http.StatusOK {
		return info, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &info); err != nil {
		return info, err
	}
	return info, nil
}

// DockerCreateContainer creates a new container.
func (c *Client) DockerCreateContainer(name, image string, cmd, env []string, ports map[string]string) (string, error) {
	reqBody := docker.CreateContainerRequest{
		Name:  name,
		Image: image,
		Cmd:   cmd,
		Env:   env,
		Ports: ports,
	}
	resp, err := c.doRequest("POST", "/api/docker/containers", reqBody)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", c.errorFromResponse(resp)
	}

	var result map[string]string
	if err := c.getJSONResponse(resp, &result); err != nil {
		return "", err
	}
	return result["id"], nil
}

// DockerStartContainer starts a container.
func (c *Client) DockerStartContainer(id string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/start", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerStopContainer stops a container with an optional timeout.
func (c *Client) DockerStopContainer(id string, timeout *int) error {
	reqBody := map[string]interface{}{}
	if timeout != nil {
		reqBody["timeout"] = *timeout
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/stop", id), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerRestartContainer restarts a container with an optional timeout.
func (c *Client) DockerRestartContainer(id string, timeout *int) error {
	reqBody := map[string]interface{}{}
	if timeout != nil {
		reqBody["timeout"] = *timeout
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/restart", id), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerPauseContainer pauses a container.
func (c *Client) DockerPauseContainer(id string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/pause", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerUnpauseContainer unpauses a container.
func (c *Client) DockerUnpauseContainer(id string) error {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/unpause", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerRemoveContainer removes a container.
func (c *Client) DockerRemoveContainer(id string, force bool) error {
	path := fmt.Sprintf("/api/docker/containers/%s", id)
	if force {
		path += "?force=true"
	}
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// DockerExecContainer executes a one-shot command in a container.
func (c *Client) DockerExecContainer(id, command string, tty bool) (string, error) {
	reqBody := docker.ExecContainerRequest{
		Command: command,
		Tty:     tty,
	}
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/docker/containers/%s/exec", id), reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.errorFromResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// DockerStreamLogs streams container logs over HTTP.
func (c *Client) DockerStreamLogs(id string, follow bool, tail, since string) (io.ReadCloser, error) {
	q := url.Values{}
	q.Set("follow", strconv.FormatBool(follow))
	if tail != "" {
		q.Set("tail", tail)
	}
	if since != "" {
		q.Set("since", since)
	}
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/docker/containers/%s/logs?%s", id, q.Encode()), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	return resp.Body, nil
}

// DockerStreamStats streams container stats over HTTP.
func (c *Client) DockerStreamStats(id string, stream bool) (io.ReadCloser, error) {
	q := url.Values{}
	q.Set("stream", strconv.FormatBool(stream))
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/docker/containers/%s/stats?%s", id, q.Encode()), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	return resp.Body, nil
}

// DockerExecWebSocket opens an interactive exec WebSocket for a container.
func (c *Client) DockerExecWebSocket(id, command string, tty bool, width, height int) (*websocket.Conn, error) {
	wsScheme := "ws"
	if strings.HasPrefix(c.baseURL, "https://") {
		wsScheme = "wss"
	}
	baseURL := strings.TrimPrefix(c.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")

	wsURL := fmt.Sprintf("%s://%s/api/docker/containers/%s/exec/ws", wsScheme, baseURL, id)

	headers := http.Header{}
	if c.basicUser != "" && c.basicPass != "" {
		auth := c.basicUser + ":" + c.basicPass
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}

	dialer := c.internalWebSocketDialer()
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	initMsg := docker.WSExecRequest{
		Command: command,
		Tty:     tty,
		Width:   width,
		Height:  height,
	}
	if err := conn.WriteJSON(initMsg); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// DockerLogsWebSocket opens a log streaming WebSocket for a container.
func (c *Client) DockerLogsWebSocket(id, tail, since string) (*websocket.Conn, error) {
	wsScheme := "ws"
	if strings.HasPrefix(c.baseURL, "https://") {
		wsScheme = "wss"
	}
	baseURL := strings.TrimPrefix(c.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")

	q := url.Values{}
	if tail != "" {
		q.Set("tail", tail)
	}
	if since != "" {
		q.Set("since", since)
	}

	wsURL := fmt.Sprintf("%s://%s/api/docker/containers/%s/logs/ws?%s", wsScheme, baseURL, id, q.Encode())

	headers := http.Header{}
	if c.basicUser != "" && c.basicPass != "" {
		auth := c.basicUser + ":" + c.basicPass
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}

	dialer := c.internalWebSocketDialer()
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// DockerComposeDeploy deploys a compose project.
func (c *Client) DockerComposeDeploy(req docker.ComposeDeployRequest) (*docker.ComposeProjectStatus, error) {
	resp, err := c.doRequest("POST", "/api/docker/compose", req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, c.errorFromResponse(resp)
	}

	var status docker.ComposeProjectStatus
	if err := c.getJSONResponse(resp, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// DockerComposeGet returns the status of a compose project.
func (c *Client) DockerComposeGet(project string) (*docker.ComposeProjectStatus, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/docker/compose/%s", project), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	var status docker.ComposeProjectStatus
	if err := c.getJSONResponse(resp, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// DockerComposeRemove removes a compose project.
func (c *Client) DockerComposeRemove(project string, force bool) error {
	path := fmt.Sprintf("/api/docker/compose/%s", project)
	if force {
		path += "?force=true"
	}
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

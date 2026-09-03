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

	"github.com/casuallc/vigil/proxy"
)

// ProxyList returns all proxy instances.
func (c *Client) ProxyList() ([]proxy.InstanceStatus, error) {
	resp, err := c.doRequest("GET", "/api/proxy/instances", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var result []proxy.InstanceStatus
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ProxyCreate registers a new proxy instance.
func (c *Client) ProxyCreate(req ProxyCreateRequest) (*proxy.InstanceStatus, error) {
	resp, err := c.doRequest("POST", "/api/proxy/instances", req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, c.errorFromResponse(resp)
	}
	var result proxy.InstanceStatus
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ProxyGet returns one proxy instance.
func (c *Client) ProxyGet(name string) (*proxy.InstanceStatus, error) {
	resp, err := c.doRequest("GET", "/api/proxy/instances/"+name, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var result proxy.InstanceStatus
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ProxyUpdate replaces an instance's configuration.
func (c *Client) ProxyUpdate(name string, cfg proxy.InstanceConfig) (*proxy.InstanceStatus, error) {
	resp, err := c.doRequest("PUT", "/api/proxy/instances/"+name, cfg)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var result proxy.InstanceStatus
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ProxyDelete removes an instance.
func (c *Client) ProxyDelete(name string) error {
	resp, err := c.doRequest("DELETE", "/api/proxy/instances/"+name, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	return nil
}

// ProxyStart starts a stopped instance.
func (c *Client) ProxyStart(name string) (*proxy.InstanceStatus, error) {
	return c.proxyLifecycle(name, "start")
}

// ProxyStop stops a running instance.
func (c *Client) ProxyStop(name string) (*proxy.InstanceStatus, error) {
	return c.proxyLifecycle(name, "stop")
}

func (c *Client) proxyLifecycle(name, action string) (*proxy.InstanceStatus, error) {
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/proxy/instances/%s/%s", name, action), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	var result proxy.InstanceStatus
	if err := c.getJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

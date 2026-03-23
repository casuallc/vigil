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

	"github.com/casuallc/vigil/inspection"
	"github.com/casuallc/vigil/models"
)

// GetSystemResources gets system resource usage
func (c *Client) GetSystemResources() (models.ResourceStats, error) {
	var resources models.ResourceStats
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

// GetProcessResources gets resource usage for a specific process
func (c *Client) GetProcessResources(pid int) (models.ResourceStats, error) {
	var resources models.ResourceStats
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

// ExecuteInspection executes a cosmic inspection
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

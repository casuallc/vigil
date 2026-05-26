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

package upgrade

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HealthChecker performs self-health checks for the new process
type HealthChecker struct {
	addr string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(addr string) *HealthChecker {
	return &HealthChecker{addr: addr}
}

// Check performs a health check against the internal address
func (hc *HealthChecker) Check() error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://%s/health", hc.addr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

// WaitForReady repeatedly checks health until success or timeout
func (hc *HealthChecker) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := hc.Check(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("health check timed out after %v", timeout)
}

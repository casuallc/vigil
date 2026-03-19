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

package proc

import (
	"testing"
	"time"
)

func TestResourceCache(t *testing.T) {
	cache := NewResourceCache(5 * time.Second) // 5 second TTL

	// Test initial state - should be expired
	if !cache.IsExpired("system") {
		t.Error("System cache should initially be expired")
	}

	// Create sample stats
	stats := ResourceStats{
		CPUUsage:    10.5,
		MemoryUsage: 1024,
	}

	// Set system resources
	cache.SetSystemResources(stats)

	// Should not be expired immediately after setting
	if cache.IsExpired("system") {
		t.Error("System cache should not be expired right after setting")
	}

	// Get cached system resources
	retrievedStats, found := cache.GetSystemResources()
	if !found {
		t.Error("Should be able to retrieve system resources after setting them")
	}

	if retrievedStats.CPUUsage != stats.CPUUsage {
		t.Errorf("Retrieved CPU usage %f doesn't match set value %f", retrievedStats.CPUUsage, stats.CPUUsage)
	}

	// Test process cache
	pid := 1234
	if !cache.IsExpired("process_1234") {
		t.Error("Process cache should initially be expired")
	}

	// Set process resources
	cache.SetProcessResources(pid, stats)

	// Should not be expired immediately after setting
	if cache.IsExpired("process_1234") {
		t.Error("Process cache should not be expired right after setting")
	}

	// Get cached process resources
	retrievedProcessStats, found := cache.GetProcessResources(pid)
	if !found {
		t.Error("Should be able to retrieve process resources after setting them")
	}

	if retrievedProcessStats.MemoryUsage != stats.MemoryUsage {
		t.Errorf("Retrieved memory usage %d doesn't match set value %d", retrievedProcessStats.MemoryUsage, stats.MemoryUsage)
	}
}

func TestResourceMonitor(t *testing.T) {
	// Create a dummy manager (could be nil for this test)
	manager := &Manager{}

	// Create resource monitor with short intervals for testing
	monitor := NewResourceMonitor(manager, 1*time.Second, 100*time.Millisecond, false, false)

	// Start the monitor
	monitor.Start()

	// Give it a moment to run
	time.Sleep(150 * time.Millisecond)

	// Stop the monitor
	monitor.Stop()

	// At this point, we've tested that the monitor can start and stop without error
	t.Log("ResourceMonitor start/stop test passed")
}
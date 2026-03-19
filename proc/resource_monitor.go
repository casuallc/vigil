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
	"strconv"
	"sync"
	"time"
)

// ResourceCache holds cached resource information with TTL
type ResourceCache struct {
	mu           sync.RWMutex
	systemData   ResourceStats
	processData  map[int]ResourceStats
	lastUpdated  map[string]time.Time
	ttl          time.Duration
}

// NewResourceCache creates a new resource cache with specified TTL
func NewResourceCache(ttl time.Duration) *ResourceCache {
	return &ResourceCache{
		processData: make(map[int]ResourceStats),
		lastUpdated: make(map[string]time.Time),
		ttl:         ttl,
	}
}

// IsExpired checks if the cached data is expired
func (rc *ResourceCache) IsExpired(key string) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if lastUpdate, exists := rc.lastUpdated[key]; exists {
		return time.Now().Sub(lastUpdate) > rc.ttl
	}
	return true
}

// GetSystemResources returns cached system resources if not expired
func (rc *ResourceCache) GetSystemResources() (ResourceStats, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	key := "system"
	if time.Now().Sub(rc.lastUpdated[key]) <= rc.ttl {
		return rc.systemData, true
	}
	return ResourceStats{}, false
}

// SetSystemResources updates the cached system resources
func (rc *ResourceCache) SetSystemResources(stats ResourceStats) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.systemData = stats
	rc.lastUpdated["system"] = time.Now()
}

// GetProcessResources returns cached process resources if not expired
func (rc *ResourceCache) GetProcessResources(pid int) (ResourceStats, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	key := "process_" + strconv.Itoa(pid)
	if time.Now().Sub(rc.lastUpdated[key]) <= rc.ttl {
		if stats, exists := rc.processData[pid]; exists {
			return stats, true
		}
	}
	return ResourceStats{}, false
}

// SetProcessResources updates the cached process resources
func (rc *ResourceCache) SetProcessResources(pid int, stats ResourceStats) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.processData[pid] = stats
	rc.lastUpdated["process_"+strconv.Itoa(pid)] = time.Now()
}

// ResourceMonitor manages scheduled collection and caching of resource metrics
type ResourceMonitor struct {
	cache          *ResourceCache
	manager        *Manager
	interval       time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	collectSystem  bool
	collectProcess bool
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(manager *Manager, cacheTTL, collectionInterval time.Duration, collectSystem, collectProcess bool) *ResourceMonitor {
	return &ResourceMonitor{
		cache:          NewResourceCache(cacheTTL),
		manager:        manager,
		interval:       collectionInterval,
		stopCh:         make(chan struct{}),
		collectSystem:  collectSystem,
		collectProcess: collectProcess,
	}
}

// Start begins the scheduled collection of resource metrics
func (rm *ResourceMonitor) Start() {
	rm.wg.Add(1)
	go rm.runCollectionLoop()
}

// Stop stops the scheduled collection
func (rm *ResourceMonitor) Stop() {
	close(rm.stopCh)
	rm.wg.Wait()
}

// runCollectionLoop runs the periodic collection of resource metrics
func (rm *ResourceMonitor) runCollectionLoop() {
	defer rm.wg.Done()

	// Collect immediately on startup
	rm.collectOnce()

	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.collectOnce()
		case <-rm.stopCh:
			return
		}
	}
}

// collectOnce performs a single collection cycle
func (rm *ResourceMonitor) collectOnce() {
	if rm.collectSystem {
		if stats, err := GetSystemResourceUsage(); err == nil {
			rm.cache.SetSystemResources(stats)
		}
	}

	if rm.collectProcess && rm.manager != nil {
		// Get all managed processes and collect their stats
		for _, process := range rm.manager.GetProcesses() {
			if process.Status.PID > 0 {
				if stats, err := GetUnixProcessResourceUsage(process.Status.PID); err == nil {
					rm.cache.SetProcessResources(process.Status.PID, *stats)
				}
			}
		}
	}
}

// GetCachedSystemResources returns the latest cached system resources
func (rm *ResourceMonitor) GetCachedSystemResources() (ResourceStats, bool) {
	return rm.cache.GetSystemResources()
}

// GetCachedProcessResources returns the latest cached process resources
func (rm *ResourceMonitor) GetCachedProcessResources(pid int) (ResourceStats, bool) {
	return rm.cache.GetProcessResources(pid)
}
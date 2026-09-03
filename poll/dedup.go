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

package poll

import "sync"

// dedupCache remembers the ack payloads of recently completed task ids so
// that a re-dispatched task (upstream timeout / agent restart) is answered
// immediately with the cached result instead of being executed again.
type dedupCache struct {
	mu      sync.Mutex
	max     int
	order   []string // oldest first
	results map[string]AckPayload
}

func newDedupCache(max int) *dedupCache {
	if max <= 0 {
		max = dedupCacheSize
	}
	return &dedupCache{
		max:     max,
		results: make(map[string]AckPayload),
	}
}

// Get returns the cached ack payload for a previously completed task id.
func (d *dedupCache) Get(id string) (AckPayload, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.results[id]
	return p, ok
}

// Put records the ack payload of a completed task, evicting the oldest
// entries when the cache exceeds its capacity.
func (d *dedupCache) Put(id string, p AckPayload) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.results[id]; exists {
		d.results[id] = p
		return
	}
	d.results[id] = p
	d.order = append(d.order, id)
	for len(d.order) > d.max {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.results, oldest)
	}
}

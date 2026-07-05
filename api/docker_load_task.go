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
	"sync"
	"time"

	"github.com/casuallc/vigil/docker"
)

// loadImageTaskStore holds in-memory state for asynchronous docker image load
// operations. State is lost when the server process exits.
type loadImageTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*docker.LoadImageTask
}

// newLoadImageTaskStore creates an empty task store.
func newLoadImageTaskStore() *loadImageTaskStore {
	return &loadImageTaskStore{
		tasks: make(map[string]*docker.LoadImageTask),
	}
}

// create stores a new task and returns its ID.
func (s *loadImageTaskStore) create(task *docker.LoadImageTask) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	s.tasks[task.ID] = task
	return task.ID
}

// get retrieves a task by ID.
func (s *loadImageTaskStore) get(id string) (*docker.LoadImageTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// update applies an updater function to the task with the given ID.
func (s *loadImageTaskStore) update(id string, updater func(*docker.LoadImageTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return false
	}
	updater(task)
	task.UpdatedAt = time.Now()
	return true
}

// taskState constants for docker image load operations.
const (
	taskStatePending     = "pending"
	taskStateDownloading = "downloading"
	taskStateLoading     = "loading"
	taskStateSuccess     = "success"
	taskStateFailed      = "failed"
)

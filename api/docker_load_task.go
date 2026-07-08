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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/casuallc/vigil/docker"
)

const defaultLoadImageTaskStorePath = "./data/docker_load_tasks.json"

// loadImageTaskStore holds asynchronous docker image load operations. Tasks are
// kept in memory and mirrored to a local JSON file so they survive server restarts.
type loadImageTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*docker.LoadImageTask
	path  string
}

// newLoadImageTaskStore creates a task store backed by the given file path. If
// path is empty, it defaults to ./data/docker_load_tasks.json. Existing tasks
// are loaded from the file and non-terminal tasks are marked failed.
func newLoadImageTaskStore(path string) (*loadImageTaskStore, error) {
	if path == "" {
		path = defaultLoadImageTaskStorePath
	}
	s := &loadImageTaskStore{
		tasks: make(map[string]*docker.LoadImageTask),
		path:  path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads tasks from the backing file. Missing files are treated as empty
// stores; malformed files log a warning and start empty.
func (s *loadImageTaskStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read load task store %s: %v", s.path, err)
	}

	var loaded map[string]*docker.LoadImageTask
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("Warning: load task store %s is corrupt, starting fresh: %v", s.path, err)
		return nil
	}

	now := time.Now()
	for id, task := range loaded {
		if task == nil {
			continue
		}
		if task.ID == "" {
			task.ID = id
		}
		// Tasks that were running when the server stopped cannot be resumed
		// because the download goroutine is gone.
		if task.State != taskStateSuccess && task.State != taskStateFailed {
			task.State = taskStateFailed
			task.ErrorMsg = "interrupted by server restart"
			task.UpdatedAt = now
		}
		s.tasks[id] = task
	}
	return nil
}

// save writes the in-memory tasks to the backing file atomically.
func (s *loadImageTaskStore) save() error {
	data, err := json.MarshalIndent(s.tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal load tasks: %v", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create task store directory %s: %v", dir, err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary task store: %v", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("failed to rename task store: %v", err)
	}
	return nil
}

// create stores a new task, persists it, and returns its ID.
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

	if err := s.save(); err != nil {
		log.Printf("Warning: failed to persist load task %s: %v", task.ID, err)
	}
	return task.ID
}

// get retrieves a task by ID.
func (s *loadImageTaskStore) get(id string) (*docker.LoadImageTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// update applies an updater function to the task with the given ID and persists
// the change.
func (s *loadImageTaskStore) update(id string, updater func(*docker.LoadImageTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return false
	}
	updater(task)
	task.UpdatedAt = time.Now()

	if err := s.save(); err != nil {
		log.Printf("Warning: failed to persist load task update %s: %v", id, err)
	}
	return true
}

// list returns a snapshot of all tasks sorted by CreatedAt descending.
func (s *loadImageTaskStore) list() []*docker.LoadImageTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*docker.LoadImageTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
	return tasks
}

// listByState returns tasks filtered by state, sorted by CreatedAt descending.
func (s *loadImageTaskStore) listByState(state string) []*docker.LoadImageTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*docker.LoadImageTask, 0)
	for _, task := range s.tasks {
		if task.State == state {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
	return tasks
}

// taskState constants for docker image load operations.
const (
	taskStatePending     = "pending"
	taskStateDownloading = "downloading"
	taskStateLoading     = "loading"
	taskStateSuccess     = "success"
	taskStateFailed      = "failed"
)

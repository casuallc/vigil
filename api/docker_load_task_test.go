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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/casuallc/vigil/docker"
)

func TestLoadImageTaskStore_CreateAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("newLoadImageTaskStore failed: %v", err)
	}

	task := &docker.LoadImageTask{
		URL:   "http://example.com/img.tar",
		State: taskStatePending,
	}
	id := store.create(task)
	if id == "" {
		t.Fatal("expected task id")
	}

	// Verify file exists and contains the task.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read task file: %v", err)
	}
	var stored map[string]*docker.LoadImageTask
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("failed to decode task file: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 task, got %d", len(stored))
	}
	if stored[id].URL != task.URL {
		t.Fatalf("unexpected url: %s", stored[id].URL)
	}

	// Re-open the store and verify recovery.
	store2, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	recovered, ok := store2.get(id)
	if !ok {
		t.Fatal("expected task to be recovered")
	}
	if recovered.URL != task.URL {
		t.Fatalf("unexpected recovered url: %s", recovered.URL)
	}
	// Pending tasks should be marked failed on recovery.
	if recovered.State != taskStateFailed {
		t.Fatalf("expected recovered pending task to be failed, got %s", recovered.State)
	}
	if recovered.ErrorMsg != "interrupted by server restart" {
		t.Fatalf("unexpected error msg: %s", recovered.ErrorMsg)
	}
}

func TestLoadImageTaskStore_Update(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("newLoadImageTaskStore failed: %v", err)
	}

	id := store.create(&docker.LoadImageTask{URL: "http://example.com/img.tar", State: taskStatePending})
	ok := store.update(id, func(task *docker.LoadImageTask) {
		task.State = taskStateSuccess
		task.Images = []string{"myimg:v1"}
	})
	if !ok {
		t.Fatal("expected update to succeed")
	}

	store2, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	recovered, ok := store2.get(id)
	if !ok {
		t.Fatal("expected task to be recovered")
	}
	if recovered.State != taskStateSuccess {
		t.Fatalf("expected success state, got %s", recovered.State)
	}
	if len(recovered.Images) != 1 || recovered.Images[0] != "myimg:v1" {
		t.Fatalf("unexpected images: %v", recovered.Images)
	}
}

func TestLoadImageTaskStore_List(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("newLoadImageTaskStore failed: %v", err)
	}

	now := time.Now()
	store.create(&docker.LoadImageTask{ID: "task-1", URL: "http://a.tar", State: taskStateSuccess, CreatedAt: now.Add(-2 * time.Hour)})
	store.create(&docker.LoadImageTask{ID: "task-2", URL: "http://b.tar", State: taskStateFailed, CreatedAt: now.Add(-1 * time.Hour)})
	store.create(&docker.LoadImageTask{ID: "task-3", URL: "http://c.tar", State: taskStateSuccess, CreatedAt: now})

	tasks := store.list()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "task-3" || tasks[1].ID != "task-2" || tasks[2].ID != "task-1" {
		t.Fatalf("tasks not sorted by CreatedAt desc: %v", tasks)
	}

	failed := store.listByState(taskStateFailed)
	if len(failed) != 1 || failed[0].ID != "task-2" {
		t.Fatalf("unexpected failed tasks: %v", failed)
	}
}

func TestLoadImageTaskStore_Delete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("newLoadImageTaskStore failed: %v", err)
	}

	id := store.create(&docker.LoadImageTask{URL: "http://example.com/img.tar", State: taskStateSuccess})
	if len(store.list()) != 1 {
		t.Fatal("expected 1 task")
	}

	if !store.delete(id) {
		t.Fatal("expected delete to succeed")
	}
	if len(store.list()) != 0 {
		t.Fatal("expected 0 tasks after delete")
	}

	// Deleting a non-existent task should return false.
	if store.delete(id) {
		t.Fatal("expected delete of missing task to fail")
	}

	// Verify persistence after reopening.
	store2, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	if len(store2.list()) != 0 {
		t.Fatal("expected store to remain empty after reopen")
	}
}

func TestLoadImageTaskStore_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("expected corrupt file to be ignored, got error: %v", err)
	}
	if len(store.list()) != 0 {
		t.Fatal("expected empty store after corrupt file")
	}
}

func TestLoadImageTaskStore_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := newLoadImageTaskStore(path)
	if err != nil {
		t.Fatalf("newLoadImageTaskStore failed: %v", err)
	}
	if len(store.list()) != 0 {
		t.Fatal("expected empty store for missing file")
	}
}

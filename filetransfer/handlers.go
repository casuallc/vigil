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

package filetransfer

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers the file-transfer agent endpoints on r.
// Authentication is the embedding server's responsibility: these endpoints go
// through vigil's global Basic Auth just like every other API route.
func (m *Manager) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/fs/list", m.handleFsList).Methods(http.MethodGet)
	r.HandleFunc("/api/fs/stat", m.handleFsStat).Methods(http.MethodGet)

	r.HandleFunc("/api/transfer/tasks", m.handleCreateTask).Methods(http.MethodPost)
	r.HandleFunc("/api/transfer/tasks", m.handleListTasks).Methods(http.MethodGet)
	r.HandleFunc("/api/transfer/tasks/{id}", m.handleGetTask).Methods(http.MethodGet)
	r.HandleFunc("/api/transfer/tasks/{id}", m.handleDeleteTask).Methods(http.MethodDelete)
	r.HandleFunc("/api/transfer/tasks/{id}/start", m.handleStart).Methods(http.MethodPost)
	r.HandleFunc("/api/transfer/tasks/{id}/pause", m.handlePause).Methods(http.MethodPost)
	r.HandleFunc("/api/transfer/tasks/{id}/resume", m.handleResume).Methods(http.MethodPost)
	r.HandleFunc("/api/transfer/tasks/{id}/cancel", m.handleCancel).Methods(http.MethodPost)
	r.HandleFunc("/api/transfer/tasks/{id}/status", m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/transfer/tasks/{id}/progress", m.handleProgress).Methods(http.MethodGet)
	r.HandleFunc("/api/transfer/tasks/{id}/chunks", m.handleChunks).Methods(http.MethodPost)
}

// ===================== FS =====================

func (m *Manager) handleFsList(w http.ResponseWriter, r *http.Request) {
	items, err := m.fs.list(r.URL.Query().Get("path"))
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, items)
}

func (m *Manager) handleFsStat(w http.ResponseWriter, r *http.Request) {
	stat, err := m.fs.stat(r.URL.Query().Get("path"))
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, stat)
}

// ===================== task CRUD =====================

func (m *Manager) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var config TaskConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := m.CreateTask(config); err != nil {
		ftWriteError(w, http.StatusConflict, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, map[string]interface{}{"taskId": config.TaskID})
}

func (m *Manager) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ftWriteJSON(w, http.StatusOK, m.ListTasks())
}

func (m *Manager) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	config, err := m.GetConfig(id)
	if err != nil {
		ftWriteError(w, http.StatusNotFound, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, config)
}

func (m *Manager) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := m.DeleteTask(id); err != nil {
		ftWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ===================== lifecycle =====================

func (m *Manager) handleStart(w http.ResponseWriter, r *http.Request)  { m.lifecycle(w, r, m.Start) }
func (m *Manager) handlePause(w http.ResponseWriter, r *http.Request)  { m.lifecycle(w, r, m.Pause) }
func (m *Manager) handleResume(w http.ResponseWriter, r *http.Request) { m.lifecycle(w, r, m.Resume) }
func (m *Manager) handleCancel(w http.ResponseWriter, r *http.Request) { m.lifecycle(w, r, m.Cancel) }

func (m *Manager) lifecycle(w http.ResponseWriter, r *http.Request, action func(int64) error) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := action(id); err != nil {
		ftWriteError(w, http.StatusConflict, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ===================== status =====================

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	status, err := m.GetStatus(id)
	if err != nil {
		ftWriteError(w, http.StatusNotFound, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, status)
}

func (m *Manager) handleProgress(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	progress, err := m.GetProgress(id)
	if err != nil {
		ftWriteError(w, http.StatusNotFound, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, progress)
}

// ===================== chunk receive =====================

func (m *Manager) handleChunks(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDFromPath(r)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()

	// Optional recvToken check: enforced only when the RECV task defines one.
	if config, cfgErr := m.GetConfig(id); cfgErr == nil && config.RecvToken != "" {
		if q.Get("recvToken") != config.RecvToken {
			ftWriteError(w, http.StatusForbidden, "invalid recvToken")
			return
		}
	}

	offset, _ := strconv.ParseInt(q.Get("offset"), 10, 64)
	chunkIndex, _ := strconv.Atoi(q.Get("chunkIndex"))
	length, _ := strconv.Atoi(q.Get("length"))
	crc, _ := strconv.ParseUint(q.Get("crc32"), 10, 32)
	eof, _ := strconv.ParseBool(q.Get("eof"))
	size, _ := strconv.ParseInt(q.Get("size"), 10, 64)

	meta := ChunkMeta{
		RelPath:    q.Get("relPath"),
		ChunkIndex: chunkIndex,
		Offset:     offset,
		Length:     length,
		Crc32:      uint32(crc),
		Eof:        eof,
		Sha256:     q.Get("sha256"),
		Size:       size,
	}

	// Raw octet-stream body — do not parse as multipart.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ftWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := m.ReceiveChunk(id, meta, body); err != nil {
		ftWriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ftWriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ===================== helpers =====================

func taskIDFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
}

func ftWriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func ftWriteError(w http.ResponseWriter, statusCode int, message string) {
	ftWriteJSON(w, statusCode, map[string]string{"error": message})
}

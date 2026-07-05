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
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/casuallc/vigil/docker"
	"github.com/gorilla/mux"
)

// handleDockerListContainers GET /api/docker/containers?all=true
func (s *Server) handleDockerListContainers(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	all := r.URL.Query().Get("all") == "true"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	containers, err := s.dockerManager.ListContainers(ctx, all)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, docker.ToContainerSummaries(containers))
}

// handleDockerInspectContainer GET /api/docker/containers/{id}
func (s *Server) handleDockerInspectContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := s.dockerManager.InspectContainer(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// handleDockerCreateContainer POST /api/docker/containers
func (s *Server) handleDockerCreateContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	var req docker.CreateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	id, err := s.dockerManager.CreateContainer(ctx, req.Name, req.Image, req.Cmd, req.Env, req.Ports)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// handleDockerRemoveContainer DELETE /api/docker/containers/{id}?force=true
func (s *Server) handleDockerRemoveContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	force := r.URL.Query().Get("force") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.dockerManager.RemoveContainer(ctx, id, force); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// handleDockerStartContainer POST /api/docker/containers/{id}/start
func (s *Server) handleDockerStartContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.dockerManager.StartContainer(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// handleDockerStopContainer POST /api/docker/containers/{id}/stop
func (s *Server) handleDockerStopContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	var req struct {
		Timeout *int `json:"timeout,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := s.dockerManager.StopContainer(ctx, id, req.Timeout); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// handleDockerRestartContainer POST /api/docker/containers/{id}/restart
func (s *Server) handleDockerRestartContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	var req struct {
		Timeout *int `json:"timeout,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := s.dockerManager.RestartContainer(ctx, id, req.Timeout); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

// handleDockerPauseContainer POST /api/docker/containers/{id}/pause
func (s *Server) handleDockerPauseContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.dockerManager.PauseContainer(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// handleDockerUnpauseContainer POST /api/docker/containers/{id}/unpause
func (s *Server) handleDockerUnpauseContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.dockerManager.UnpauseContainer(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unpaused"})
}

// handleDockerExecContainer POST /api/docker/containers/{id}/exec
func (s *Server) handleDockerExecContainer(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	var req docker.ExecContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	output, err := s.dockerManager.ExecCommand(ctx, id, []string{"sh", "-c", req.Command}, req.Tty)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// handleDockerStreamLogs GET /api/docker/containers/{id}/logs?follow=true&tail=100&since=...
func (s *Server) handleDockerStreamLogs(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	follow := r.URL.Query().Get("follow") == "true"
	tail := r.URL.Query().Get("tail")
	since := r.URL.Query().Get("since")

	ctx := r.Context()
	if !follow {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	if err := s.dockerManager.StreamLogs(ctx, id, follow, tail, since, w); err != nil {
		log.Printf("Docker log stream error: %v", err)
	}
}

// handleDockerStreamStats GET /api/docker/containers/{id}/stats?stream=true
func (s *Server) handleDockerStreamStats(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	id := mux.Vars(r)["id"]
	stream := r.URL.Query().Get("stream") != "false" // default true

	ctx := r.Context()
	if !stream {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	if err := s.dockerManager.StreamStats(ctx, id, stream, w); err != nil {
		log.Printf("Docker stats stream error: %v", err)
	}
}

// handleDockerComposeDeploy POST /api/docker/compose
func (s *Server) handleDockerComposeDeploy(w http.ResponseWriter, r *http.Request) {
	if s.composeManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker compose manager not initialized")
		return
	}

	var req docker.ComposeDeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	status, err := s.composeManager.DeployProject(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, status)
}

// handleDockerComposeGet GET /api/docker/compose/{project}
func (s *Server) handleDockerComposeGet(w http.ResponseWriter, r *http.Request) {
	if s.composeManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker compose manager not initialized")
		return
	}

	project := mux.Vars(r)["project"]
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status, err := s.composeManager.GetProject(ctx, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// handleDockerComposeRemove DELETE /api/docker/compose/{project}?force=true
func (s *Server) handleDockerComposeRemove(w http.ResponseWriter, r *http.Request) {
	if s.composeManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker compose manager not initialized")
		return
	}

	project := mux.Vars(r)["project"]
	force := r.URL.Query().Get("force") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := s.composeManager.RemoveProject(ctx, project, force); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// handleDockerPing GET /api/docker/ping
func (s *Server) handleDockerPing(w http.ResponseWriter, r *http.Request) {
	if s.dockerManager == nil {
		writeError(w, http.StatusServiceUnavailable, "docker manager not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ping, err := s.dockerManager.Ping(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ping)
}

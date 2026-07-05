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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casuallc/vigil/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/gorilla/mux"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// composeFakeClient implements docker.Client for compose handler tests.
type composeFakeClient struct {
	containers         []types.Container
	createdID          string
	timesStarted       int
	timesStopped       int
	timesRemoved       int
	containerCreateErr error
	containerStartErr  error
	containerListErr   error
	imagesPulled       []string
	networksCreated    []string
	volumesCreated     []string
}

func (f *composeFakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.containers, f.containerListErr
}

func (f *composeFakeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

func (f *composeFakeClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	f.timesStarted++
	return f.containerStartErr
}

func (f *composeFakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	f.timesStopped++
	return nil
}

func (f *composeFakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}

func (f *composeFakeClient) ContainerPause(ctx context.Context, containerID string) error {
	return nil
}

func (f *composeFakeClient) ContainerUnpause(ctx context.Context, containerID string) error {
	return nil
}

func (f *composeFakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	f.timesRemoved++
	return nil
}

func (f *composeFakeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{ID: f.createdID}, f.containerCreateErr
}

func (f *composeFakeClient) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (types.IDResponse, error) {
	return types.IDResponse{}, nil
}

func (f *composeFakeClient) ContainerExecAttach(ctx context.Context, execID string, options container.ExecAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (f *composeFakeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}

func (f *composeFakeClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return nil
}

func (f *composeFakeClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func (f *composeFakeClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(nil)}, nil
}

func (f *composeFakeClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

func (f *composeFakeClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	f.imagesPulled = append(f.imagesPulled, ref)
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *composeFakeClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	f.networksCreated = append(f.networksCreated, name)
	return network.CreateResponse{ID: name + "-id"}, nil
}

func (f *composeFakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	return nil, nil
}

func (f *composeFakeClient) NetworkRemove(ctx context.Context, networkID string) error {
	return nil
}

func (f *composeFakeClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	f.volumesCreated = append(f.volumesCreated, options.Name)
	return volume.Volume{Name: options.Name}, nil
}

func (f *composeFakeClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	return volume.ListResponse{}, nil
}

func (f *composeFakeClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return nil
}

func (f *composeFakeClient) Close() error {
	return nil
}

func newComposeTestServer(fc *composeFakeClient) *Server {
	mgr := docker.NewManagerWithClient(fc)
	return &Server{
		dockerManager:  mgr,
		composeManager: docker.NewComposeManager(mgr),
	}
}

func TestHandleDockerComposeDeploy(t *testing.T) {
	fc := &composeFakeClient{createdID: "cid1"}
	server := newComposeTestServer(fc)

	body, _ := json.Marshal(docker.ComposeDeployRequest{
		Name:    "demo",
		Content: "services:\n  web:\n    image: nginx:alpine\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerComposeDeploy(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var status docker.ComposeProjectStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if status.Name != "demo" {
		t.Fatalf("unexpected project name: %s", status.Name)
	}
}

func TestHandleDockerComposeDeploy_InvalidYAML(t *testing.T) {
	fc := &composeFakeClient{createdID: "cid1"}
	server := newComposeTestServer(fc)

	body, _ := json.Marshal(docker.ComposeDeployRequest{
		Name:    "demo",
		Content: "not: valid: yaml: [",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerComposeDeploy(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleDockerComposeDeploy_Conflict(t *testing.T) {
	fc := &composeFakeClient{
		containers: []types.Container{
			{ID: "old", Labels: map[string]string{docker.ComposeProjectLabel: "demo", docker.ComposeServiceLabel: "web"}},
		},
	}
	server := newComposeTestServer(fc)

	body, _ := json.Marshal(docker.ComposeDeployRequest{
		Name:    "demo",
		Content: "services:\n  web:\n    image: nginx:alpine\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerComposeDeploy(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestHandleDockerComposeGet(t *testing.T) {
	fc := &composeFakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", Labels: map[string]string{docker.ComposeProjectLabel: "demo", docker.ComposeServiceLabel: "web"}},
		},
	}
	server := newComposeTestServer(fc)

	req := httptest.NewRequest(http.MethodGet, "/api/docker/compose/demo", nil)
	req = mux.SetURLVars(req, map[string]string{"project": "demo"})
	rr := httptest.NewRecorder()
	server.handleDockerComposeGet(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var status docker.ComposeProjectStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(status.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(status.Services))
	}
}

func TestHandleDockerComposeRemove(t *testing.T) {
	fc := &composeFakeClient{
		containers: []types.Container{
			{ID: "c1", Labels: map[string]string{docker.ComposeProjectLabel: "demo", docker.ComposeServiceLabel: "web"}},
		},
	}
	server := newComposeTestServer(fc)

	req := httptest.NewRequest(http.MethodDelete, "/api/docker/compose/demo?force=true", nil)
	req = mux.SetURLVars(req, map[string]string{"project": "demo"})
	rr := httptest.NewRecorder()
	server.handleDockerComposeRemove(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if fc.timesRemoved != 1 {
		t.Fatalf("expected remove once, got %d", fc.timesRemoved)
	}
}

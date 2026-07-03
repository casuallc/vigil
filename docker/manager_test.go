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

package docker

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// fakeClient implements the docker.Client interface for tests.
type fakeClient struct {
	containers         []types.Container
	inspect            types.ContainerJSON
	createdID          string
	execID             string
	hijacked           types.HijackedResponse
	timesStopped       int
	timesStarted       int
	timesRestarted     int
	timesPaused        int
	timesUnpaused      int
	timesRemoved       int
	timesResized       int
	containerListErr   error
	containerStopErr   error
	containerStartErr  error
	containerCreateErr error
	pingErr            error
}

func (f *fakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.containers, f.containerListErr
}

func (f *fakeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return f.inspect, nil
}

func (f *fakeClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	f.timesStarted++
	return f.containerStartErr
}

func (f *fakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	f.timesStopped++
	return f.containerStopErr
}

func (f *fakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	f.timesRestarted++
	return nil
}

func (f *fakeClient) ContainerPause(ctx context.Context, containerID string) error {
	f.timesPaused++
	return nil
}

func (f *fakeClient) ContainerUnpause(ctx context.Context, containerID string) error {
	f.timesUnpaused++
	return nil
}

func (f *fakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	f.timesRemoved++
	return nil
}

func (f *fakeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{ID: f.createdID}, f.containerCreateErr
}

func (f *fakeClient) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (types.IDResponse, error) {
	return types.IDResponse{ID: f.execID}, nil
}

func (f *fakeClient) ContainerExecAttach(ctx context.Context, execID string, options container.ExecAttachOptions) (types.HijackedResponse, error) {
	return f.hijacked, nil
}

func (f *fakeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{Running: false}, nil
}

func (f *fakeClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	f.timesResized++
	return nil
}

func (f *fakeClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func (f *fakeClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(nil)}, nil
}

func (f *fakeClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, f.pingErr
}

func (f *fakeClient) Close() error {
	return nil
}

func TestManager_ListContainers(t *testing.T) {
	fc := &fakeClient{
		containers: []types.Container{
			{ID: "abc123", Names: []string{"/test"}, Image: "alpine"},
		},
	}
	m := NewManagerWithClient(fc)

	got, err := m.ListContainers(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "abc123" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestManager_StopContainer(t *testing.T) {
	fc := &fakeClient{}
	m := NewManagerWithClient(fc)

	timeout := 10
	if err := m.StopContainer(context.Background(), "abc", &timeout); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.timesStopped != 1 {
		t.Fatalf("expected ContainerStop called once, got %d", fc.timesStopped)
	}
}

func TestManager_StartContainer(t *testing.T) {
	fc := &fakeClient{}
	m := NewManagerWithClient(fc)

	if err := m.StartContainer(context.Background(), "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.timesStarted != 1 {
		t.Fatalf("expected ContainerStart called once, got %d", fc.timesStarted)
	}
}

func TestManager_CreateContainer(t *testing.T) {
	fc := &fakeClient{createdID: "newid"}
	m := NewManagerWithClient(fc)

	id, err := m.CreateContainer(context.Background(), "mycontainer", "alpine", []string{"sleep", "1"}, []string{"FOO=bar"}, map[string]string{"80": "8080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "newid" {
		t.Fatalf("expected id newid, got %s", id)
	}
}

func TestToContainerSummaries(t *testing.T) {
	containers := []types.Container{
		{
			ID:      "id1",
			Names:   []string{"/c1"},
			Image:   "alpine",
			Command: "sleep 1",
			Status:  "running",
			State:   "running",
			Ports: []types.Port{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
			},
		},
	}
	sums := ToContainerSummaries(containers)
	if len(sums) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(sums))
	}
	if sums[0].ID != "id1" || len(sums[0].Ports) != 1 {
		t.Fatalf("unexpected summary: %+v", sums[0])
	}
}

func TestParseTimeout(t *testing.T) {
	if v := ParseTimeout("10"); v == nil || *v != 10 {
		t.Fatalf("expected 10, got %v", v)
	}
	if v := ParseTimeout(""); v != nil {
		t.Fatalf("expected nil, got %v", v)
	}
	if v := ParseTimeout("abc"); v != nil {
		t.Fatalf("expected nil for invalid, got %v", v)
	}
}

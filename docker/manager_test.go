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
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// FakeClient implements the docker.Client interface for tests.
type FakeClient struct {
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
	// Compose-related fake state
	imagesPulled     []string
	networksCreated  []string
	volumesCreated   []string
	networkList      []network.Summary
	volumeList       volume.ListResponse
	pullErr          error
	networkCreateErr error
	networkListErr   error
	networkRemoveErr error
	volumeCreateErr  error
	volumeListErr    error
	volumeRemoveErr  error
}

func (f *FakeClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	f.imagesPulled = append(f.imagesPulled, ref)
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *FakeClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	if f.networkCreateErr != nil {
		return network.CreateResponse{}, f.networkCreateErr
	}
	f.networksCreated = append(f.networksCreated, name)
	return network.CreateResponse{ID: name + "-id"}, nil
}

func (f *FakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	if f.networkListErr != nil {
		return nil, f.networkListErr
	}
	return f.networkList, nil
}

func (f *FakeClient) NetworkRemove(ctx context.Context, networkID string) error {
	return f.networkRemoveErr
}

func (f *FakeClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	if f.volumeCreateErr != nil {
		return volume.Volume{}, f.volumeCreateErr
	}
	f.volumesCreated = append(f.volumesCreated, options.Name)
	return volume.Volume{Name: options.Name}, nil
}

func (f *FakeClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	if f.volumeListErr != nil {
		return volume.ListResponse{}, f.volumeListErr
	}
	return f.volumeList, nil
}

func (f *FakeClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return f.volumeRemoveErr
}

func (f *FakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.containers, f.containerListErr
}

func (f *FakeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return f.inspect, nil
}

func (f *FakeClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	f.timesStarted++
	return f.containerStartErr
}

func (f *FakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	f.timesStopped++
	return f.containerStopErr
}

func (f *FakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	f.timesRestarted++
	return nil
}

func (f *FakeClient) ContainerPause(ctx context.Context, containerID string) error {
	f.timesPaused++
	return nil
}

func (f *FakeClient) ContainerUnpause(ctx context.Context, containerID string) error {
	f.timesUnpaused++
	return nil
}

func (f *FakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	f.timesRemoved++
	return nil
}

func (f *FakeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{ID: f.createdID}, f.containerCreateErr
}

func (f *FakeClient) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (types.IDResponse, error) {
	return types.IDResponse{ID: f.execID}, nil
}

func (f *FakeClient) ContainerExecAttach(ctx context.Context, execID string, options container.ExecAttachOptions) (types.HijackedResponse, error) {
	return f.hijacked, nil
}

func (f *FakeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{Running: false}, nil
}

func (f *FakeClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	f.timesResized++
	return nil
}

func (f *FakeClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func (f *FakeClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(nil)}, nil
}

func (f *FakeClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, f.pingErr
}

func (f *FakeClient) Close() error {
	return nil
}

func TestManager_ListContainers(t *testing.T) {
	fc := &FakeClient{
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
	fc := &FakeClient{}
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
	fc := &FakeClient{}
	m := NewManagerWithClient(fc)

	if err := m.StartContainer(context.Background(), "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.timesStarted != 1 {
		t.Fatalf("expected ContainerStart called once, got %d", fc.timesStarted)
	}
}

func TestManager_CreateContainer(t *testing.T) {
	fc := &FakeClient{createdID: "newid"}
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

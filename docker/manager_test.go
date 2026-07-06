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
	"net/http"
	"net/http/httptest"
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
	// Image load/tag fake state
	loadRespBody  io.ReadCloser
	loadErr       error
	taggedImages  []string // list of "source->target"
	tagErr        error
	// Image management fake state
	imagesListed    []image.Summary
	imageInspect    types.ImageInspect
	imageHistory    []image.HistoryResponseItem
	imageRemoved    []image.DeleteResponse
	imageListErr    error
	imageInspectErr error
	imageRemoveErr  error
	imageHistoryErr error
}

func (f *FakeClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	f.imagesPulled = append(f.imagesPulled, ref)
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *FakeClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (image.LoadResponse, error) {
	if f.loadErr != nil {
		return image.LoadResponse{}, f.loadErr
	}
	body := f.loadRespBody
	if body == nil {
		body = io.NopCloser(strings.NewReader(""))
	}
	return image.LoadResponse{Body: body}, nil
}

func (f *FakeClient) ImageTag(ctx context.Context, source, target string) error {
	if f.tagErr != nil {
		return f.tagErr
	}
	f.taggedImages = append(f.taggedImages, source+"->"+target)
	return nil
}

func (f *FakeClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	if f.imageListErr != nil {
		return nil, f.imageListErr
	}
	return f.imagesListed, nil
}

func (f *FakeClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return f.imageInspect, nil, f.imageInspectErr
}

func (f *FakeClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	if f.imageRemoveErr != nil {
		return nil, f.imageRemoveErr
	}
	return f.imageRemoved, nil
}

func (f *FakeClient) ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	if f.imageHistoryErr != nil {
		return nil, f.imageHistoryErr
	}
	return f.imageHistory, nil
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

func TestManager_LoadImage(t *testing.T) {
	body := `{"stream":"Loaded image: myrepo/myimage:v1.0\n"}` + "\n" +
		`{"stream":"Loaded image ID: sha256:abc123\n"}` + "\n"
	fc := &FakeClient{
		loadRespBody: io.NopCloser(strings.NewReader(body)),
	}
	m := NewManagerWithClient(fc)

	images, err := m.LoadImage(context.Background(), strings.NewReader("tar content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d: %v", len(images), images)
	}
	if images[0] != "myrepo/myimage:v1.0" {
		t.Fatalf("unexpected image[0]: %s", images[0])
	}
	if images[1] != "sha256:abc123" {
		t.Fatalf("unexpected image[1]: %s", images[1])
	}
}

func TestManager_LoadImage_Error(t *testing.T) {
	body := `{"errorDetail":{"message":"archive contains invalid tar"}}` + "\n"
	fc := &FakeClient{
		loadRespBody: io.NopCloser(strings.NewReader(body)),
	}
	m := NewManagerWithClient(fc)

	_, err := m.LoadImage(context.Background(), strings.NewReader("tar content"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "archive contains invalid tar") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_LoadImageFromURL_TagOverride(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake tar body"))
	}))
	defer ts.Close()

	body := `{"stream":"Loaded image: origin:latest\n"}` + "\n"
	fc := &FakeClient{
		loadRespBody: io.NopCloser(strings.NewReader(body)),
	}
	m := NewManagerWithClient(fc)

	images, err := m.LoadImageFromURL(context.Background(), ts.URL, ImageMetadata{
		Name: "myrepo",
		Tag:  "v1.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 2 || images[1] != "myrepo:v1.0" {
		t.Fatalf("expected tag override image, got %v", images)
	}
	if len(fc.taggedImages) != 1 || fc.taggedImages[0] != "origin:latest->myrepo:v1.0" {
		t.Fatalf("expected tag call, got %v", fc.taggedImages)
	}
}

func TestManager_LoadImageFromURL_SizeMismatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("12345"))
	}))
	defer ts.Close()

	fc := &FakeClient{}
	m := NewManagerWithClient(fc)

	_, err := m.LoadImageFromURL(context.Background(), ts.URL, ImageMetadata{Size: 100})
	if err == nil {
		t.Fatalf("expected size mismatch error")
	}
	if !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManager_ListImages(t *testing.T) {
	fc := &FakeClient{
		imagesListed: []image.Summary{
			{ID: "sha256:abc", RepoTags: []string{"alpine:latest"}, Size: 1024},
		},
	}
	m := NewManagerWithClient(fc)

	got, err := m.ListImages(context.Background(), image.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sha256:abc" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestManager_InspectImage(t *testing.T) {
	fc := &FakeClient{
		imageInspect: types.ImageInspect{ID: "sha256:abc", Size: 2048},
	}
	m := NewManagerWithClient(fc)

	info, err := m.InspectImage(context.Background(), "alpine:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != "sha256:abc" {
		t.Fatalf("unexpected image id: %s", info.ID)
	}
}

func TestManager_RemoveImage(t *testing.T) {
	fc := &FakeClient{
		imageRemoved: []image.DeleteResponse{{Untagged: "alpine:latest"}, {Deleted: "sha256:abc"}},
	}
	m := NewManagerWithClient(fc)

	got, err := m.RemoveImage(context.Background(), "alpine:latest", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 delete responses, got %d", len(got))
	}
}

func TestManager_TagImage(t *testing.T) {
	fc := &FakeClient{}
	m := NewManagerWithClient(fc)

	if err := m.TagImage(context.Background(), "alpine:latest", "myrepo:v1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.taggedImages) != 1 || fc.taggedImages[0] != "alpine:latest->myrepo:v1" {
		t.Fatalf("unexpected tag call: %v", fc.taggedImages)
	}
}

func TestManager_ImageHistory(t *testing.T) {
	fc := &FakeClient{
		imageHistory: []image.HistoryResponseItem{{ID: "sha256:abc", CreatedBy: "/bin/sh"}},
	}
	m := NewManagerWithClient(fc)

	history, err := m.ImageHistory(context.Background(), "alpine:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].ID != "sha256:abc" {
		t.Fatalf("unexpected history: %+v", history)
	}
}

func TestToImageSummaries(t *testing.T) {
	images := []image.Summary{
		{ID: "sha256:abc", RepoTags: []string{"alpine:latest"}, Size: 1024},
	}
	sums := ToImageSummaries(images)
	if len(sums) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(sums))
	}
	if sums[0].ID != "sha256:abc" || sums[0].Size != 1024 {
		t.Fatalf("unexpected summary: %+v", sums[0])
	}
}

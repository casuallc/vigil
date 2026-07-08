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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/casuallc/vigil/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/gorilla/mux"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// loadImageFakeClient implements docker.Client for load image handler tests.
type loadImageFakeClient struct {
	loadRespBody io.ReadCloser
}

func (f *loadImageFakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return nil, nil
}
func (f *loadImageFakeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}
func (f *loadImageFakeClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return nil
}
func (f *loadImageFakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}
func (f *loadImageFakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}
func (f *loadImageFakeClient) ContainerPause(ctx context.Context, containerID string) error { return nil }
func (f *loadImageFakeClient) ContainerUnpause(ctx context.Context, containerID string) error {
	return nil
}
func (f *loadImageFakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return nil
}
func (f *loadImageFakeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{}, nil
}
func (f *loadImageFakeClient) ContainerExecCreate(ctx context.Context, container string, options container.ExecOptions) (types.IDResponse, error) {
	return types.IDResponse{}, nil
}
func (f *loadImageFakeClient) ContainerExecAttach(ctx context.Context, execID string, options container.ExecAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}
func (f *loadImageFakeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}
func (f *loadImageFakeClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return nil
}
func (f *loadImageFakeClient) ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *loadImageFakeClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(strings.NewReader(""))}, nil
}
func (f *loadImageFakeClient) Ping(ctx context.Context) (types.Ping, error) { return types.Ping{}, nil }
func (f *loadImageFakeClient) ServerVersion(ctx context.Context) (types.Version, error) { return types.Version{}, nil }
func (f *loadImageFakeClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *loadImageFakeClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (image.LoadResponse, error) {
	body := f.loadRespBody
	if body == nil {
		body = io.NopCloser(strings.NewReader(""))
	}
	return image.LoadResponse{Body: body}, nil
}
func (f *loadImageFakeClient) ImageTag(ctx context.Context, source, target string) error { return nil }
func (f *loadImageFakeClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return nil, nil
}
func (f *loadImageFakeClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return types.ImageInspect{}, nil, nil
}
func (f *loadImageFakeClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	return nil, nil
}
func (f *loadImageFakeClient) ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	return nil, nil
}
func (f *loadImageFakeClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return network.CreateResponse{}, nil
}
func (f *loadImageFakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	return nil, nil
}
func (f *loadImageFakeClient) NetworkRemove(ctx context.Context, networkID string) error { return nil }
func (f *loadImageFakeClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return volume.Volume{}, nil
}
func (f *loadImageFakeClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	return volume.ListResponse{}, nil
}
func (f *loadImageFakeClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error { return nil }
func (f *loadImageFakeClient) Close() error { return nil }

func newLoadImageTestServer(t *testing.T, fc *loadImageFakeClient) *Server {
	store, err := newLoadImageTaskStore(filepath.Join(t.TempDir(), "docker_load_tasks.json"))
	if err != nil {
		t.Fatalf("failed to create load image task store: %v", err)
	}
	return &Server{
		dockerManager:  docker.NewManagerWithClient(fc),
		loadImageTasks: store,
	}
}

func TestHandleDockerLoadImage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake tar"))
	}))
	defer ts.Close()

	body := `{"stream":"Loaded image: myapp:v1.0\n"}` + "\n"
	fc := &loadImageFakeClient{loadRespBody: io.NopCloser(strings.NewReader(body))}
	server := newLoadImageTestServer(t, fc)

	reqBody, _ := json.Marshal(docker.LoadImageRequest{
		URL: ts.URL,
		Metadata: docker.ImageMetadata{
			Name: "renamed",
			Tag:  "latest",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/load", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerLoadImage(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp docker.LoadImageResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TaskID == "" {
		t.Fatalf("expected task_id")
	}

	// Poll for completion (with timeout).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/docker/images/load/"+resp.TaskID+"/status", nil)
		statusReq = muxSetURLVars(statusReq, map[string]string{"id": resp.TaskID})
		statusRR := httptest.NewRecorder()
		server.handleDockerLoadImageStatus(statusRR, statusReq)

		if statusRR.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", statusRR.Code, statusRR.Body.String())
		}

		var task docker.LoadImageTask
		if err := json.Unmarshal(statusRR.Body.Bytes(), &task); err != nil {
			t.Fatalf("failed to decode task: %v", err)
		}
		if task.State == taskStateSuccess {
			if len(task.Images) == 0 {
				t.Fatalf("expected images in success state")
			}
			return
		}
		if task.State == taskStateFailed {
			t.Fatalf("task failed: %s", task.ErrorMsg)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("task did not complete in time")
}

func muxSetURLVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

func TestHandleDockerLoadImage_MissingURL(t *testing.T) {
	fc := &loadImageFakeClient{}
	server := newLoadImageTestServer(t, fc)

	reqBody, _ := json.Marshal(docker.LoadImageRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/load", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerLoadImage(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerLoadImageList(t *testing.T) {
	fc := &loadImageFakeClient{}
	server := newLoadImageTestServer(t, fc)

	now := time.Now()
	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-1", URL: "http://a.tar", State: taskStateSuccess, CreatedAt: now.Add(-2 * time.Hour)})
	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-2", URL: "http://b.tar", State: taskStateFailed, CreatedAt: now.Add(-1 * time.Hour)})
	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-3", URL: "http://c.tar", State: taskStateSuccess, CreatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images/load", nil)
	rr := httptest.NewRecorder()
	server.handleDockerLoadImageList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var tasks []docker.LoadImageTask
	if err := json.Unmarshal(rr.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to decode tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "task-3" {
		t.Fatalf("expected newest task first, got %s", tasks[0].ID)
	}
}

func TestHandleDockerLoadImageList_FilterByState(t *testing.T) {
	fc := &loadImageFakeClient{}
	server := newLoadImageTestServer(t, fc)

	now := time.Now()
	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-1", URL: "http://a.tar", State: taskStateSuccess, CreatedAt: now.Add(-time.Hour)})
	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-2", URL: "http://b.tar", State: taskStateFailed, CreatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images/load?state=failed", nil)
	rr := httptest.NewRecorder()
	server.handleDockerLoadImageList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var tasks []docker.LoadImageTask
	if err := json.Unmarshal(rr.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to decode tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "task-2" {
		t.Fatalf("unexpected filtered tasks: %v", tasks)
	}
}

func TestHandleDockerLoadImageList_DockerManagerNil(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images/load", nil)
	rr := httptest.NewRecorder()
	server.handleDockerLoadImageList(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerLoadImageDelete(t *testing.T) {
	fc := &loadImageFakeClient{}
	server := newLoadImageTestServer(t, fc)

	server.loadImageTasks.create(&docker.LoadImageTask{ID: "task-1", URL: "http://a.tar", State: taskStateSuccess})

	req := httptest.NewRequest(http.MethodDelete, "/api/docker/images/load/task-1", nil)
	req = muxSetURLVars(req, map[string]string{"id": "task-1"})
	rr := httptest.NewRecorder()
	server.handleDockerLoadImageDelete(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if _, ok := server.loadImageTasks.get("task-1"); ok {
		t.Fatal("expected task to be deleted")
	}
}

func TestHandleDockerLoadImageDelete_NotFound(t *testing.T) {
	fc := &loadImageFakeClient{}
	server := newLoadImageTestServer(t, fc)

	req := httptest.NewRequest(http.MethodDelete, "/api/docker/images/load/missing", nil)
	req = muxSetURLVars(req, map[string]string{"id": "missing"})
	rr := httptest.NewRecorder()
	server.handleDockerLoadImageDelete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// imageHandlerFakeClient implements docker.Client for image handler tests.
type imageHandlerFakeClient struct {
	loadRespBody io.ReadCloser
	images       []image.Summary
	inspect      types.ImageInspect
	removed      []image.DeleteResponse
	history      []image.HistoryResponseItem
	pullErr      error
	removeErr    error
	tagErr       error
	inspectErr   error
	historyErr   error
}

func (f *imageHandlerFakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return nil, nil
}
func (f *imageHandlerFakeClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}
func (f *imageHandlerFakeClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerPause(ctx context.Context, containerID string) error { return nil }
func (f *imageHandlerFakeClient) ContainerUnpause(ctx context.Context, containerID string) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return container.CreateResponse{}, nil
}
func (f *imageHandlerFakeClient) ContainerExecCreate(ctx context.Context, container string, options container.ExecOptions) (types.IDResponse, error) {
	return types.IDResponse{}, nil
}
func (f *imageHandlerFakeClient) ContainerExecAttach(ctx context.Context, execID string, options container.ExecAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}
func (f *imageHandlerFakeClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}
func (f *imageHandlerFakeClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return nil
}
func (f *imageHandlerFakeClient) ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *imageHandlerFakeClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{Body: io.NopCloser(strings.NewReader(""))}, nil
}
func (f *imageHandlerFakeClient) Ping(ctx context.Context) (types.Ping, error) { return types.Ping{}, nil }
func (f *imageHandlerFakeClient) ServerVersion(ctx context.Context) (types.Version, error) { return types.Version{}, nil }
func (f *imageHandlerFakeClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *imageHandlerFakeClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (image.LoadResponse, error) {
	body := f.loadRespBody
	if body == nil {
		body = io.NopCloser(strings.NewReader(""))
	}
	return image.LoadResponse{Body: body}, nil
}
func (f *imageHandlerFakeClient) ImageTag(ctx context.Context, source, target string) error { return f.tagErr }
func (f *imageHandlerFakeClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return f.images, nil
}
func (f *imageHandlerFakeClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return f.inspect, nil, f.inspectErr
}
func (f *imageHandlerFakeClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	if f.removeErr != nil {
		return nil, f.removeErr
	}
	return f.removed, nil
}
func (f *imageHandlerFakeClient) ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	return f.history, f.historyErr
}
func (f *imageHandlerFakeClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return network.CreateResponse{}, nil
}
func (f *imageHandlerFakeClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	return nil, nil
}
func (f *imageHandlerFakeClient) NetworkRemove(ctx context.Context, networkID string) error { return nil }
func (f *imageHandlerFakeClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return volume.Volume{}, nil
}
func (f *imageHandlerFakeClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	return volume.ListResponse{}, nil
}
func (f *imageHandlerFakeClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error { return nil }
func (f *imageHandlerFakeClient) Close() error { return nil }

func newImageHandlerTestServer(fc *imageHandlerFakeClient) *Server {
	return &Server{dockerManager: docker.NewManagerWithClient(fc)}
}

func TestHandleDockerListImages(t *testing.T) {
	fc := &imageHandlerFakeClient{
		images: []image.Summary{{ID: "sha256:abc", RepoTags: []string{"alpine:latest"}, Size: 1024}},
	}
	server := newImageHandlerTestServer(fc)

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images", nil)
	rr := httptest.NewRecorder()

	server.handleDockerListImages(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var out []docker.ImageSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(out) != 1 || out[0].ID != "sha256:abc" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestHandleDockerInspectImage(t *testing.T) {
	fc := &imageHandlerFakeClient{inspect: types.ImageInspect{ID: "sha256:abc", Size: 2048}}
	server := newImageHandlerTestServer(fc)

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images/alpine:latest", nil)
	req = muxSetURLVars(req, map[string]string{"id": "alpine:latest"})
	rr := httptest.NewRecorder()

	server.handleDockerInspectImage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out types.ImageInspect
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if out.ID != "sha256:abc" {
		t.Fatalf("unexpected id: %s", out.ID)
	}
}

func TestHandleDockerPullImage(t *testing.T) {
	fc := &imageHandlerFakeClient{}
	server := newImageHandlerTestServer(fc)

	reqBody, _ := json.Marshal(docker.PullImageRequest{Image: "alpine:latest"})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/pull", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerPullImage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerPullImage_MissingImage(t *testing.T) {
	fc := &imageHandlerFakeClient{}
	server := newImageHandlerTestServer(fc)

	reqBody, _ := json.Marshal(docker.PullImageRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/pull", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerPullImage(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerRemoveImage(t *testing.T) {
	fc := &imageHandlerFakeClient{
		removed: []image.DeleteResponse{{Untagged: "alpine:latest"}, {Deleted: "sha256:abc"}},
	}
	server := newImageHandlerTestServer(fc)

	req := httptest.NewRequest(http.MethodDelete, "/api/docker/images/alpine:latest?force=true", nil)
	req = muxSetURLVars(req, map[string]string{"id": "alpine:latest"})
	rr := httptest.NewRecorder()

	server.handleDockerRemoveImage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out docker.ImageRemoveResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(out.Deleted) != 2 {
		t.Fatalf("expected 2 deleted items, got %d", len(out.Deleted))
	}
}

func TestHandleDockerTagImage(t *testing.T) {
	fc := &imageHandlerFakeClient{}
	server := newImageHandlerTestServer(fc)

	reqBody, _ := json.Marshal(docker.TagImageRequest{Source: "alpine:latest", Target: "myrepo:v1"})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/tag", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerTagImage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerTagImage_MissingSource(t *testing.T) {
	fc := &imageHandlerFakeClient{}
	server := newImageHandlerTestServer(fc)

	reqBody, _ := json.Marshal(docker.TagImageRequest{Target: "myrepo:v1"})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/tag", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerTagImage(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDockerImageHistory(t *testing.T) {
	fc := &imageHandlerFakeClient{
		history: []image.HistoryResponseItem{{ID: "sha256:abc", CreatedBy: "/bin/sh"}},
	}
	server := newImageHandlerTestServer(fc)

	req := httptest.NewRequest(http.MethodGet, "/api/docker/images/alpine:latest/history", nil)
	req = muxSetURLVars(req, map[string]string{"id": "alpine:latest"})
	rr := httptest.NewRecorder()

	server.handleDockerImageHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

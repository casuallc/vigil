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

func newLoadImageTestServer(fc *loadImageFakeClient) *Server {
	return &Server{
		dockerManager:  docker.NewManagerWithClient(fc),
		loadImageTasks: newLoadImageTaskStore(),
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
	server := newLoadImageTestServer(fc)

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
	server := newLoadImageTestServer(fc)

	reqBody, _ := json.Marshal(docker.LoadImageRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/docker/images/load", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleDockerLoadImage(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

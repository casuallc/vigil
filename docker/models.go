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

import "time"

// CreateContainerRequest is the body for POST /api/docker/containers.
type CreateContainerRequest struct {
	Name  string            `json:"name"`
	Image string            `json:"image"`
	Cmd   []string          `json:"cmd,omitempty"`
	Env   []string          `json:"env,omitempty"`
	Ports map[string]string `json:"ports,omitempty"`
}

// ExecContainerRequest is the body for POST /api/docker/containers/{id}/exec.
type ExecContainerRequest struct {
	Command string `json:"command"`
	Tty     bool   `json:"tty,omitempty"`
}

// WSExecRequest is the first message sent by the client on an exec WebSocket.
type WSExecRequest struct {
	Command string `json:"command"`
	Tty     bool   `json:"tty"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// WSResizeMessage is sent by the client to resize an interactive exec session.
type WSResizeMessage struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// ContainerSummary is a simplified container representation for the API.
type ContainerSummary struct {
	ID      string            `json:"id"`
	Names   []string          `json:"names"`
	Image   string            `json:"image"`
	Command string            `json:"command"`
	Created int64             `json:"created"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Ports   []PortMapping     `json:"ports"`
	Labels  map[string]string `json:"labels"`
}

// PortMapping describes a published container port.
type PortMapping struct {
	IP          string `json:"ip"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port"`
	Type        string `json:"type"`
}

// ComposeDeployRequest is the body for POST /api/docker/compose.
type ComposeDeployRequest struct {
	Name    string            `json:"name"`
	Content string            `json:"content"`
	Start   *bool             `json:"start,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ComposeDeployFromDirRequest is the body for POST /api/docker/compose/dir.
type ComposeDeployFromDirRequest struct {
	Name string            `json:"name"`
	Dir  string            `json:"dir"`
	Start *bool            `json:"start,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
}

// ComposeServiceStatus represents one service in a compose project response.
type ComposeServiceStatus struct {
	Name       string             `json:"name"`
	Image      string             `json:"image"`
	Command    string             `json:"command,omitempty"`
	Replicas   int                `json:"replicas"`
	Restart    string             `json:"restart,omitempty"`
	Containers []ContainerSummary `json:"containers"`
}

// ComposeProjectStatus is the response for compose project endpoints.
type ComposeProjectStatus struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`
	Services []ComposeServiceStatus `json:"services"`
}

// ComposeVersionResponse is the response for GET /api/docker/compose/version.
type ComposeVersionResponse struct {
	Version   string `json:"version"`
	RawOutput string `json:"raw_output,omitempty"`
}

// ImageMetadata is the metadata supplied by the client for the tar being loaded.
type ImageMetadata struct {
	Name     string            `json:"name,omitempty"`     // optional target repository
	Tag      string            `json:"tag,omitempty"`      // optional target tag
	Platform string            `json:"platform,omitempty"` // optional platform, e.g. linux/amd64
	Size     int64             `json:"size,omitempty"`     // optional expected size in bytes
	SHA256   string            `json:"sha256,omitempty"`   // optional sha256 checksum
	Labels   map[string]string `json:"labels,omitempty"`   // optional labels
}

// LoadImageRequest is the body for POST /api/docker/images/load.
type LoadImageRequest struct {
	URL      string        `json:"url"`
	Metadata ImageMetadata `json:"metadata"`
}

// LoadImageResponse is returned on a successful load request.
type LoadImageResponse struct {
	TaskID string `json:"task_id"`
}

// LoadImageTask represents the state of an asynchronous image load operation.
type LoadImageTask struct {
	ID        string        `json:"id"`
	URL       string        `json:"url"`
	Metadata  ImageMetadata `json:"metadata"`
	State     string        `json:"state"`               // pending/downloading/loading/success/failed
	Images    []string      `json:"images,omitempty"`    // loaded/renamed image refs
	ErrorMsg  string        `json:"error_msg,omitempty"` // set when state is failed
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ImageSummary is a simplified image representation for the API.
type ImageSummary struct {
	ID          string            `json:"id"`
	Containers  int64             `json:"containers"`
	Created     int64             `json:"created"`
	Labels      map[string]string `json:"labels"`
	ParentID    string            `json:"parent_id"`
	RepoDigests []string          `json:"repo_digests"`
	RepoTags    []string          `json:"repo_tags"`
	SharedSize  int64             `json:"shared_size"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size,omitempty"`
}

// PullImageRequest is the body for POST /api/docker/images/pull.
type PullImageRequest struct {
	Image string `json:"image"`
}

// TagImageRequest is the body for POST /api/docker/images/tag.
type TagImageRequest struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ImageDeleteResponseItem mirrors image.DeleteResponse for API responses.
type ImageDeleteResponseItem struct {
	Deleted  string `json:"deleted,omitempty"`
	Untagged string `json:"untagged,omitempty"`
}

// ImageRemoveResponse is returned by DELETE /api/docker/images/{id}.
type ImageRemoveResponse struct {
	Deleted []ImageDeleteResponseItem `json:"deleted"`
}

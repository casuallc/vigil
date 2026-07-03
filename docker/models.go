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

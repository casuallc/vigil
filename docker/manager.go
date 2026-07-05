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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// Manager wraps the Docker Engine API client to manage local containers.
type Manager struct {
	cli Client
}

// NewManager creates a new Docker manager using environment-based configuration
// (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH). An optional host override
// can be provided; if empty, the environment is used.
func NewManager(host string) (*Manager, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &Manager{cli: cli}, nil
}

// NewManagerWithClient creates a manager from an existing client (useful for tests).
func NewManagerWithClient(cli Client) *Manager {
	return &Manager{cli: cli}
}

// Close closes the underlying Docker client connection.
func (m *Manager) Close() error {
	if m.cli != nil {
		return m.cli.Close()
	}
	return nil
}

// Ping checks connectivity to the Docker daemon.
func (m *Manager) Ping(ctx context.Context) (types.Ping, error) {
	return m.cli.Ping(ctx)
}

// ListContainers returns containers from the Docker daemon.
func (m *Manager) ListContainers(ctx context.Context, all bool) ([]types.Container, error) {
	return m.cli.ContainerList(ctx, container.ListOptions{All: all})
}

// InspectContainer returns detailed information about a container.
func (m *Manager) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return m.cli.ContainerInspect(ctx, id)
}

// StartContainer starts a container.
func (m *Manager) StartContainer(ctx context.Context, id string) error {
	return m.cli.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container with an optional timeout (seconds).
func (m *Manager) StopContainer(ctx context.Context, id string, timeout *int) error {
	opts := container.StopOptions{}
	if timeout != nil {
		t := *timeout
		opts.Timeout = &t
	}
	return m.cli.ContainerStop(ctx, id, opts)
}

// RestartContainer restarts a container with an optional timeout (seconds).
func (m *Manager) RestartContainer(ctx context.Context, id string, timeout *int) error {
	opts := container.StopOptions{}
	if timeout != nil {
		t := *timeout
		opts.Timeout = &t
	}
	return m.cli.ContainerRestart(ctx, id, opts)
}

// PauseContainer pauses a container.
func (m *Manager) PauseContainer(ctx context.Context, id string) error {
	return m.cli.ContainerPause(ctx, id)
}

// UnpauseContainer unpauses a container.
func (m *Manager) UnpauseContainer(ctx context.Context, id string) error {
	return m.cli.ContainerUnpause(ctx, id)
}

// RemoveContainer removes a container, optionally forcing removal.
func (m *Manager) RemoveContainer(ctx context.Context, id string, force bool) error {
	return m.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

// CreateContainer creates a new container from an image.
func (m *Manager) CreateContainer(ctx context.Context, name, image string, cmd, env []string, ports map[string]string) (string, error) {
	cfg := &container.Config{
		Image: image,
		Cmd:   cmd,
		Env:   env,
	}
	hostCfg := &container.HostConfig{}

	if len(ports) > 0 {
		exposedPorts := make(nat.PortSet)
		portBindings := make(nat.PortMap)
		for containerPort, hostValue := range ports {
			var cp nat.Port
			if strings.Contains(containerPort, "/") {
				cp = nat.Port(containerPort)
			} else {
				var err error
				cp, err = nat.NewPort("tcp", containerPort)
				if err != nil {
					return "", err
				}
			}
			exposedPorts[cp] = struct{}{}

			hostIP := ""
			hostPort := hostValue
			if idx := strings.LastIndex(hostValue, ":"); idx >= 0 {
				hostIP = hostValue[:idx]
				hostPort = hostValue[idx+1:]
			}
			portBindings[cp] = []nat.PortBinding{
				{HostIP: hostIP, HostPort: hostPort},
			}
		}
		cfg.ExposedPorts = exposedPorts
		hostCfg.PortBindings = portBindings
	}

	resp, err := m.cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, name)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ExecCommand runs a one-shot command inside a container and returns the output.
func (m *Manager) ExecCommand(ctx context.Context, id string, cmd []string, tty bool) (string, error) {
	execResp, err := m.cli.ContainerExecCreate(ctx, id, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          tty,
	})
	if err != nil {
		return "", err
	}

	attach, err := m.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: tty})
	if err != nil {
		return "", err
	}
	defer attach.Close()

	var output []byte
	if tty {
		output, err = io.ReadAll(attach.Reader)
	} else {
		var buf strings.Builder
		_, err = stdcopy.StdCopy(&buf, &buf, attach.Reader)
		output = []byte(buf.String())
	}
	if err != nil {
		return "", err
	}

	// Wait for the exec to finish.
	for {
		inspect, err := m.cli.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return "", err
		}
		if !inspect.Running {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return string(output), nil
}

// StreamLogs streams container logs to the provided writer.
func (m *Manager) StreamLogs(ctx context.Context, id string, follow bool, tail, since string, w io.Writer) error {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
	}
	if tail != "" {
		opts.Tail = tail
	}
	if since != "" {
		opts.Since = since
	}

	rc, err := m.cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = stdcopy.StdCopy(w, w, rc)
	return err
}

// StreamStats streams container stats JSON to the provided writer.
func (m *Manager) StreamStats(ctx context.Context, id string, stream bool, w io.Writer) error {
	stats, err := m.cli.ContainerStats(ctx, id, stream)
	if err != nil {
		return err
	}
	defer stats.Body.Close()

	_, err = io.Copy(w, stats.Body)
	return err
}

// ExecInteractive creates an interactive exec session and returns the hijacked
// connection along with the exec ID. The caller is responsible for closing the
// returned HijackedResponse.
func (m *Manager) ExecInteractive(ctx context.Context, id string, cmd []string, tty bool, w, h int) (types.HijackedResponse, string, error) {
	execResp, err := m.cli.ContainerExecCreate(ctx, id, container.ExecOptions{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          tty,
	})
	if err != nil {
		return types.HijackedResponse{}, "", err
	}

	consoleSize := &[2]uint{uint(h), uint(w)}
	attachOpts := container.ExecAttachOptions{Tty: tty}
	if w > 0 && h > 0 {
		attachOpts.ConsoleSize = consoleSize
	}

	attach, err := m.cli.ContainerExecAttach(ctx, execResp.ID, attachOpts)
	if err != nil {
		return types.HijackedResponse{}, "", err
	}

	if w > 0 && h > 0 {
		_ = m.cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
			Width:  uint(w),
			Height: uint(h),
		})
	}

	return attach, execResp.ID, nil
}

// ExecResize resizes an active exec session.
func (m *Manager) ExecResize(ctx context.Context, execID string, w, h int) error {
	return m.cli.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Width:  uint(w),
		Height: uint(h),
	})
}

// ToContainerSummaries converts docker types.Container slices to our summary type.
func ToContainerSummaries(containers []types.Container) []ContainerSummary {
	out := make([]ContainerSummary, 0, len(containers))
	for _, c := range containers {
		ports := make([]PortMapping, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, PortMapping{
				IP:          p.IP,
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
			})
		}
		out = append(out, ContainerSummary{
			ID:      c.ID,
			Names:   c.Names,
			Image:   c.Image,
			Command: c.Command,
			Created: c.Created,
			Status:  c.Status,
			State:   c.State,
			Ports:   ports,
			Labels:  c.Labels,
		})
	}
	return out
}

// ParseTimeout parses an integer timeout string; returns nil if empty/invalid.
func ParseTimeout(s string) *int {
	if s == "" {
		return nil
	}
	if v, err := strconv.Atoi(s); err == nil {
		return &v
	}
	return nil
}

// PullImage pulls an image and waits for the operation to complete.
func (m *Manager) PullImage(ctx context.Context, imageRef string) error {
	rc, err := m.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	// Drain the pull progress stream so the daemon finishes the pull.
	dec := json.NewDecoder(rc)
	for {
		var msg map[string]interface{}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

// LoadImage streams a tar archive into the Docker daemon and returns the image
// references reported by the daemon.
func (m *Manager) LoadImage(ctx context.Context, r io.Reader) ([]string, error) {
	resp, err := m.cli.ImageLoad(ctx, r, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var images []string
	dec := json.NewDecoder(resp.Body)
	for {
		var msg map[string]interface{}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return images, err
		}

		if stream, ok := msg["stream"].(string); ok {
			// Typical success lines:
			// "Loaded image: foo:bar\n"
			// "Loaded image ID: sha256:...\n"
			const loadedImagePrefix = "Loaded image:"
			const loadedIDPrefix = "Loaded image ID:"
			for _, line := range strings.Split(stream, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, loadedImagePrefix) {
					images = append(images, strings.TrimSpace(strings.TrimPrefix(line, loadedImagePrefix)))
				} else if strings.HasPrefix(line, loadedIDPrefix) {
					images = append(images, strings.TrimSpace(strings.TrimPrefix(line, loadedIDPrefix)))
				}
			}
		}

		if errMsg := msg["error"]; errMsg != nil {
			return images, fmt.Errorf("docker load error: %v", errMsg)
		}
		if detail, ok := msg["errorDetail"].(map[string]interface{}); ok {
			message, _ := detail["message"].(string)
			return images, fmt.Errorf("docker load error: %s", message)
		}
	}

	return images, nil
}

// LoadImageFromURL downloads a docker tar archive from url and loads it into
// the local docker daemon. Optional metadata may be used for validation and
// tagging.
func (m *Manager) LoadImageFromURL(ctx context.Context, downloadURL string, meta ImageMetadata) ([]string, error) {
	if downloadURL == "" {
		return nil, fmt.Errorf("download URL is required")
	}
	u, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}

	tmpFile, err := os.CreateTemp("", "docker-load-*.tar")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to download tar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	written, copyErr := io.Copy(tmpFile, resp.Body)
	if closeErr := tmpFile.Close(); closeErr != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	if copyErr != nil {
		return nil, fmt.Errorf("failed to download tar: %w", copyErr)
	}

	if meta.Size > 0 && written != meta.Size {
		return nil, fmt.Errorf("size mismatch: expected %d, got %d", meta.Size, written)
	}

	if meta.SHA256 != "" {
		if err := verifyFileSHA256(tmpPath, meta.SHA256); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()

	images, err := m.LoadImage(ctx, f)
	if err != nil {
		return images, err
	}

	// Apply optional tag override.
	if meta.Name != "" && meta.Tag != "" && len(images) > 0 {
		target := meta.Name + ":" + meta.Tag
		if err := m.cli.ImageTag(ctx, images[0], target); err != nil {
			return images, fmt.Errorf("failed to tag image %q as %q: %w", images[0], target, err)
		}
		images = append(images, target)
	}

	return images, nil
}

func verifyFileSHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if strings.HasPrefix(expected, "sha256:") {
		expected = strings.TrimPrefix(expected, "sha256:")
	}
	if actual != strings.ToLower(expected) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// ListProjectContainers returns containers belonging to a compose project.
func (m *Manager) ListProjectContainers(ctx context.Context, project string) ([]types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("%s=%s", ComposeProjectLabel, project))
	return m.cli.ContainerList(ctx, container.ListOptions{Filters: f, All: true})
}

// CreateNetwork creates a project-scoped network.
func (m *Manager) CreateNetwork(ctx context.Context, name, project string, cfg ComposeNetwork) (string, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "bridge"
	}
	labels := map[string]string{ComposeProjectLabel: project}
	for k, v := range cfg.Labels {
		labels[k] = v
	}
	resp, err := m.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: driver,
		Labels: labels,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// CreateVolume creates a project-scoped volume.
func (m *Manager) CreateVolume(ctx context.Context, name, project string, cfg ComposeVolume) (string, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "local"
	}
	labels := map[string]string{ComposeProjectLabel: project}
	for k, v := range cfg.Labels {
		labels[k] = v
	}
	vol, err := m.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:       name,
		Driver:     driver,
		DriverOpts: map[string]string{},
		Labels:     labels,
	})
	if err != nil {
		return "", err
	}
	return vol.Name, nil
}

// ListProjectNetworks returns networks labeled with the compose project.
func (m *Manager) ListProjectNetworks(ctx context.Context, project string) ([]network.Summary, error) {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("%s=%s", ComposeProjectLabel, project))
	return m.cli.NetworkList(ctx, network.ListOptions{Filters: f})
}

// ListProjectVolumes returns volumes labeled with the compose project.
func (m *Manager) ListProjectVolumes(ctx context.Context, project string) ([]*volume.Volume, error) {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("%s=%s", ComposeProjectLabel, project))
	resp, err := m.cli.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	return resp.Volumes, nil
}

// RemoveNetwork removes a network by ID or name.
func (m *Manager) RemoveNetwork(ctx context.Context, id string) error {
	return m.cli.NetworkRemove(ctx, id)
}

// RemoveVolume removes a volume by name.
func (m *Manager) RemoveVolume(ctx context.Context, name string, force bool) error {
	return m.cli.VolumeRemove(ctx, name, force)
}

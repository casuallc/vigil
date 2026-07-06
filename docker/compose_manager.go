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
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
)

// ComposeManager deploys and removes docker-compose projects using container labels.
type ComposeManager struct {
	mgr *Manager
}

// NewComposeManager creates a compose manager from a Docker manager.
func NewComposeManager(mgr *Manager) *ComposeManager {
	return &ComposeManager{mgr: mgr}
}

// Version returns the installed Docker Compose version by invoking the CLI.
// It tries the modern plugin form `docker compose version --short` first,
// then falls back to the legacy `docker-compose version --short`.
func (cm *ComposeManager) Version(ctx context.Context) (*ComposeVersionResponse, error) {
	out, err := cm.runComposeVersion(ctx, "docker", "compose")
	if err != nil {
		out, err = cm.runComposeVersion(ctx, "docker-compose")
		if err != nil {
			return nil, fmt.Errorf("failed to get docker-compose version: %w", err)
		}
	}
	return &ComposeVersionResponse{
		Version:   strings.TrimSpace(out),
		RawOutput: strings.TrimSpace(out),
	}, nil
}

func (cm *ComposeManager) runComposeVersion(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], append(args[1:], "version", "--short")...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// DeployProject deploys a compose project from YAML.
func (cm *ComposeManager) DeployProject(ctx context.Context, req ComposeDeployRequest) (*ComposeProjectStatus, error) {
	project, err := NormalizeProjectName(req.Name)
	if err != nil {
		return nil, err
	}
	if req.Content == "" {
		return nil, fmt.Errorf("compose content is required")
	}

	cf, err := ParseCompose([]byte(req.Content))
	if err != nil {
		return nil, err
	}

	// Stateless conflict check: reject if the project already has containers.
	existing, err := cm.mgr.ListProjectContainers(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing project containers: %w", err)
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("project %q already exists", project)
	}

	// Pull images once per image reference.
	images := map[string]struct{}{}
	for _, svc := range cf.Services {
		images[svc.Image] = struct{}{}
	}
	for img := range images {
		if err := cm.mgr.PullImage(ctx, img); err != nil {
			return nil, fmt.Errorf("failed to pull image %q: %w", img, err)
		}
	}

	// Collect referenced networks and volumes.
	referencedNetworks := map[string]struct{}{}
	if len(cf.Services) > 0 {
		referencedNetworks["default"] = struct{}{}
	}
	referencedVolumes := map[string]struct{}{}
	for _, svc := range cf.Services {
		for _, netName := range svc.Networks {
			referencedNetworks[netName] = struct{}{}
		}
		for _, vm := range svc.Volumes {
			if _, ok := cf.Volumes[vm.Source]; ok {
				referencedVolumes[vm.Source] = struct{}{}
			}
		}
	}

	// Create networks.
	createdNetworks := map[string]string{}
	for netName := range referencedNetworks {
		if netName == "default" {
			dockerName := ProjectNetworkName(project, "default")
			id, err := cm.mgr.CreateNetwork(ctx, dockerName, project, ComposeNetwork{})
			if err != nil {
				return nil, fmt.Errorf("failed to create default network: %w", err)
			}
			createdNetworks[dockerName] = id
			continue
		}
		cfg, ok := cf.Networks[netName]
		if ok && cfg.External {
			// Verify external network exists (best-effort).
			list, err := cm.mgr.cli.NetworkList(ctx, network.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to list networks: %w", err)
			}
			found := false
			for _, n := range list {
				if n.Name == netName {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("external network %q not found", netName)
			}
			continue
		}
		dockerName := ProjectNetworkName(project, netName)
		id, err := cm.mgr.CreateNetwork(ctx, dockerName, project, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create network %q: %w", netName, err)
		}
		createdNetworks[dockerName] = id
	}

	// Create volumes.
	createdVolumes := map[string]string{}
	for volName := range referencedVolumes {
		cfg, ok := cf.Volumes[volName]
		if ok && cfg.External {
			// Verify external volume exists (best-effort).
			list, err := cm.mgr.cli.VolumeList(ctx, volume.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to list volumes: %w", err)
			}
			found := false
			for _, v := range list.Volumes {
				if v.Name == volName {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("external volume %q not found", volName)
			}
			continue
		}
		dockerName := ProjectVolumeName(project, volName)
		id, err := cm.mgr.CreateVolume(ctx, dockerName, project, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create volume %q: %w", volName, err)
		}
		createdVolumes[dockerName] = id
	}

	// Create containers.
	createdContainers := []types.Container{}
	for svcName, svc := range cf.Services {
		replicas := ServiceReplicas(svc)
		for i := 1; i <= replicas; i++ {
			cfg, hostCfg, netCfg, err := ToDockerConfigs(project, svcName, svc, i, cf.Volumes, cf.Networks)
			if err != nil {
				return nil, fmt.Errorf("failed to build config for service %q: %w", svcName, err)
			}
			name := ServiceContainerName(project, svcName, i)
			resp, err := cm.mgr.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, name)
			if err != nil {
				return nil, fmt.Errorf("failed to create container %q: %w", name, err)
			}
			createdContainers = append(createdContainers, types.Container{ID: resp.ID, Names: []string{"/" + name}, Image: svc.Image, Labels: cfg.Labels})
		}
	}

	// Start containers unless explicitly disabled.
	start := true
	if req.Start != nil {
		start = *req.Start
	}
	if start {
		for _, c := range createdContainers {
			if err := cm.mgr.StartContainer(ctx, c.ID); err != nil {
				return nil, fmt.Errorf("failed to start container %s: %w", c.ID, err)
			}
		}
	}

	status := cm.buildProjectStatus(project, cf.Services, createdContainers)
	return status, nil
}

// GetProject returns the current status of a compose project from container labels.
func (cm *ComposeManager) GetProject(ctx context.Context, project string) (*ComposeProjectStatus, error) {
	project, err := NormalizeProjectName(project)
	if err != nil {
		return nil, err
	}

	containers, err := cm.mgr.ListProjectContainers(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to list project containers: %w", err)
	}

	// Build a minimal service map from labels; the original YAML is not stored.
	services := map[string]ComposeService{}
	for _, c := range containers {
		svcName := c.Labels[ComposeServiceLabel]
		services[svcName] = ComposeService{Image: c.Image}
	}

	status := cm.buildProjectStatus(project, services, containers)
	return status, nil
}

// RemoveProject stops and removes all containers, networks, and volumes for a project.
func (cm *ComposeManager) RemoveProject(ctx context.Context, project string, force bool) error {
	project, err := NormalizeProjectName(project)
	if err != nil {
		return err
	}

	containers, err := cm.mgr.ListProjectContainers(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to list project containers: %w", err)
	}

	for _, c := range containers {
		if err := cm.mgr.RemoveContainer(ctx, c.ID, force); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", c.ID, err)
		}
	}

	networks, err := cm.mgr.ListProjectNetworks(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to list project networks: %w", err)
	}
	for _, n := range networks {
		if err := cm.mgr.RemoveNetwork(ctx, n.ID); err != nil {
			return fmt.Errorf("failed to remove network %s: %w", n.Name, err)
		}
	}

	volumes, err := cm.mgr.ListProjectVolumes(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to list project volumes: %w", err)
	}
	for _, v := range volumes {
		if err := cm.mgr.RemoveVolume(ctx, v.Name, force); err != nil {
			return fmt.Errorf("failed to remove volume %s: %w", v.Name, err)
		}
	}

	return nil
}

func (cm *ComposeManager) buildProjectStatus(project string, services map[string]ComposeService, containers []types.Container) *ComposeProjectStatus {
	status := &ComposeProjectStatus{Name: project}

	groups := map[string][]types.Container{}
	for _, c := range containers {
		svcName := c.Labels[ComposeServiceLabel]
		groups[svcName] = append(groups[svcName], c)
	}

	svcNames := make([]string, 0, len(groups))
	for name := range groups {
		svcNames = append(svcNames, name)
	}
	sort.Strings(svcNames)

	for _, svcName := range svcNames {
		svc := services[svcName]
		svcStatus := ComposeServiceStatus{
			Name:     svcName,
			Image:    svc.Image,
			Replicas: len(groups[svcName]),
		}
		if len(svc.Command) > 0 {
			svcStatus.Command = fmt.Sprintf("%v", svc.Command)
		}
		for _, c := range groups[svcName] {
			svcStatus.Containers = append(svcStatus.Containers, ToContainerSummaries([]types.Container{c})[0])
		}
		status.Services = append(status.Services, svcStatus)
	}

	return status
}

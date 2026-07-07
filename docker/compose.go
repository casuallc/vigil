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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

const (
	ComposeProjectLabel = "com.docker.compose.project"
	ComposeServiceLabel = "com.docker.compose.service"
	ComposeOneoffLabel  = "com.docker.compose.oneoff"
)

// ComposeNetwork is a minimal adapter for network creation parameters.
type ComposeNetwork struct {
	Driver string
	Labels map[string]string
}

// ComposeVolume is a minimal adapter for volume creation parameters.
type ComposeVolume struct {
	Driver string
	Labels map[string]string
}

func toComposeNetwork(cfg types.NetworkConfig) ComposeNetwork {
	return ComposeNetwork{
		Driver: cfg.Driver,
		Labels: cfg.Labels,
	}
}

func toComposeVolume(cfg types.VolumeConfig) ComposeVolume {
	return ComposeVolume{
		Driver: cfg.Driver,
		Labels: cfg.Labels,
	}
}

// LoadCompose loads a compose project from in-memory YAML content.
// Environment variable interpolation uses the provided env map.
func LoadCompose(_ context.Context, projectName string, content []byte, env map[string]string) (*types.Project, error) {
	details := types.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []types.ConfigFile{
			{Filename: "docker-compose.yml", Content: content},
		},
		Environment: env,
	}
	return loader.Load(details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
		options.SkipResolveEnvironment = true
	})
}

// LoadComposeFromDir loads a compose project from a server-side directory,
// reading docker-compose.yml and honoring the provided env map.
func LoadComposeFromDir(_ context.Context, projectName, dir string, env map[string]string) (*types.Project, error) {
	details := types.ConfigDetails{
		WorkingDir: dir,
		ConfigFiles: []types.ConfigFile{
			{Filename: filepath.Join(dir, "docker-compose.yml")},
		},
		Environment: env,
	}
	return loader.Load(details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
		options.SkipResolveEnvironment = true
	})
}

// NormalizeProjectName validates and normalizes a compose project name.
func NormalizeProjectName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("project name is required")
	}
	lower := strings.ToLower(name)
	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(lower) {
		return "", fmt.Errorf("project name must contain only lowercase letters, digits, underscores and hyphens")
	}
	return lower, nil
}

// ServiceContainerName returns the canonical compose container name.
func ServiceContainerName(project, service string, index int) string {
	return fmt.Sprintf("%s_%s_%d", project, service, index)
}

// ServiceReplicas returns the effective replica count for a compose service.
func ServiceReplicas(svc types.ServiceConfig) int {
	if svc.Deploy != nil && svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
		return *svc.Deploy.Replicas
	}
	if svc.Scale != nil && *svc.Scale > 0 {
		return *svc.Scale
	}
	return 1
}

// serviceToDockerConfigs converts a compose service to Docker SDK configs for a single replica.
func serviceToDockerConfigs(project, service string, svc types.ServiceConfig, index int, networks types.Networks) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	labels := map[string]string{
		ComposeProjectLabel: project,
		ComposeServiceLabel: service,
		ComposeOneoffLabel:  "False",
	}
	for k, v := range svc.Labels {
		labels[k] = v
	}

	cfg := &container.Config{
		Image:      svc.Image,
		Cmd:        []string(svc.Command),
		Labels:     labels,
		Hostname:   service,
		Domainname: project,
	}
	if len(svc.Command) == 0 {
		cfg.Cmd = nil
	}

	for k, v := range svc.Environment {
		if v == nil {
			cfg.Env = append(cfg.Env, fmt.Sprintf("%s=", k))
		} else {
			cfg.Env = append(cfg.Env, fmt.Sprintf("%s=%s", k, *v))
		}
	}

	hostCfg := &container.HostConfig{
		RestartPolicy: toRestartPolicy(svc.Restart),
	}

	if len(svc.Ports) > 0 {
		exposed, bindings, err := toPortBindings(svc.Ports)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid port spec for service %q: %w", service, err)
		}
		cfg.ExposedPorts = exposed
		hostCfg.PortBindings = bindings
	}

	for _, vm := range svc.Volumes {
		if err := applyVolumeMount(hostCfg, vm, project); err != nil {
			return nil, nil, nil, fmt.Errorf("invalid volume mount for service %q: %w", service, err)
		}
	}

	netCfg := toNetworkConfig(project, service, svc.Networks, networks)

	// Multi-replica hostname uniqueness.
	if ServiceReplicas(svc) > 1 {
		cfg.Hostname = fmt.Sprintf("%s-%d", service, index)
	}

	return cfg, hostCfg, netCfg, nil
}

func toPortBindings(ports []types.ServicePortConfig) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}

	for _, p := range ports {
		protocol := strings.ToLower(p.Protocol)
		if protocol == "" {
			protocol = "tcp"
		}
		port := nat.Port(fmt.Sprintf("%d/%s", p.Target, protocol))
		exposed[port] = struct{}{}

		var pb []nat.PortBinding
		if p.Published != "" {
			hostIP := p.HostIP
			if hostIP == "" {
				hostIP = "0.0.0.0"
			}
			pb = append(pb, nat.PortBinding{HostIP: hostIP, HostPort: p.Published})
		}
		bindings[port] = pb
	}

	return exposed, bindings, nil
}

func applyVolumeMount(hostCfg *container.HostConfig, vm types.ServiceVolumeConfig, project string) error {
	if vm.Target == "" {
		return fmt.Errorf("volume mount target is required")
	}

	switch vm.Type {
	case "volume", "":
		// Named volume or anonymous volume.
		source := vm.Source
		if source != "" {
			source = ProjectVolumeName(project, source)
		}
		hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
			Type:     mount.TypeVolume,
			Source:   source,
			Target:   vm.Target,
			ReadOnly: vm.ReadOnly,
		})
	case "bind":
		bind := fmt.Sprintf("%s:%s", vm.Source, vm.Target)
		if vm.ReadOnly {
			bind = fmt.Sprintf("%s:ro", bind)
		} else if vm.Bind != nil && vm.Bind.SELinux != "" {
			bind = fmt.Sprintf("%s:%s", bind, vm.Bind.SELinux)
		}
		hostCfg.Binds = append(hostCfg.Binds, bind)
	case "tmpfs":
		hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
			Type:     mount.TypeTmpfs,
			Target:   vm.Target,
			ReadOnly: vm.ReadOnly,
		})
	default:
		return fmt.Errorf("unsupported volume type %q", vm.Type)
	}

	return nil
}

func toNetworkConfig(project, service string, svcNetworks map[string]*types.ServiceNetworkConfig, networks types.Networks) *network.NetworkingConfig {
	cfg := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}

	networkNames := make([]string, 0, len(svcNetworks))
	for name := range svcNetworks {
		networkNames = append(networkNames, name)
	}
	if len(networkNames) == 0 {
		networkNames = []string{"default"}
	}

	for _, netName := range networkNames {
		dockerNetName := ProjectNetworkName(project, netName)
		if netName != "default" {
			if n, ok := networks[netName]; ok && bool(n.External) {
				dockerNetName = netName
			}
		}
		cfg.EndpointsConfig[dockerNetName] = &network.EndpointSettings{
			Aliases: []string{service},
		}
	}
	return cfg
}

func toRestartPolicy(restart string) container.RestartPolicy {
	if restart == "" {
		return container.RestartPolicy{}
	}
	parts := strings.SplitN(restart, ":", 2)
	mode := parts[0]
	switch mode {
	case "no":
		return container.RestartPolicy{Name: container.RestartPolicyDisabled}
	case "always":
		return container.RestartPolicy{Name: container.RestartPolicyAlways}
	case "unless-stopped":
		return container.RestartPolicy{Name: container.RestartPolicyUnlessStopped}
	case "on-failure":
		rp := container.RestartPolicy{Name: container.RestartPolicyOnFailure}
		if len(parts) == 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				rp.MaximumRetryCount = n
			}
		}
		return rp
	default:
		return container.RestartPolicy{Name: container.RestartPolicyMode(mode)}
	}
}

// ProjectResourceName returns the Docker resource name for a compose project resource.
func ProjectResourceName(project, name string) string {
	if name == "default" {
		return fmt.Sprintf("%s_default", project)
	}
	return fmt.Sprintf("%s_%s", project, name)
}

// ProjectNetworkName returns the Docker network name for a project network.
func ProjectNetworkName(project, name string) string {
	if name == "" || name == "default" {
		return fmt.Sprintf("%s_default", project)
	}
	return fmt.Sprintf("%s_%s", project, name)
}

// ProjectVolumeName returns the Docker volume name for a project volume.
func ProjectVolumeName(project, name string) string {
	return fmt.Sprintf("%s_%s", project, name)
}

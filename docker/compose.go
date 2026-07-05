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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/google/shlex"
	"gopkg.in/yaml.v3"
)

const (
	ComposeProjectLabel = "com.docker.compose.project"
	ComposeServiceLabel = "com.docker.compose.service"
	ComposeOneoffLabel  = "com.docker.compose.oneoff"
)

// ComposeFile is a minimal in-memory representation of a docker-compose.yml file.
type ComposeFile struct {
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume  `yaml:"volumes,omitempty"`
}

// ComposeService describes a single service in a compose file.
type ComposeService struct {
	Image       string                 `yaml:"image"`
	Command     composeCommand         `yaml:"command,omitempty"`
	Environment composeEnvironment     `yaml:"environment,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Volumes     []composeVolumeMount   `yaml:"volumes,omitempty"`
	Networks    composeServiceNetworks `yaml:"networks,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	Labels      composeLabels          `yaml:"labels,omitempty"`
	Deploy      ComposeDeploy          `yaml:"deploy,omitempty"`
	Replicas    *int                   `yaml:"replicas,omitempty"`
}

// ComposeDeploy holds the deploy block of a service.
type ComposeDeploy struct {
	Replicas *int `yaml:"replicas,omitempty"`
}

// ComposeNetwork describes a top-level network.
type ComposeNetwork struct {
	Driver   string        `yaml:"driver,omitempty"`
	External bool          `yaml:"external,omitempty"`
	Labels   composeLabels `yaml:"labels,omitempty"`
}

// ComposeVolume describes a top-level volume.
type ComposeVolume struct {
	Driver   string        `yaml:"driver,omitempty"`
	External bool          `yaml:"external,omitempty"`
	Labels   composeLabels `yaml:"labels,omitempty"`
}

// ParseCompose parses a docker-compose.yml content into ComposeFile.
func ParseCompose(content []byte) (*ComposeFile, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(content, &cf); err != nil {
		return nil, fmt.Errorf("failed to parse compose yaml: %w", err)
	}
	if len(cf.Services) == 0 {
		return nil, fmt.Errorf("compose file must define at least one service")
	}
	for name, svc := range cf.Services {
		if svc.Image == "" {
			return nil, fmt.Errorf("service %q is missing image", name)
		}
	}
	return &cf, nil
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

// ServiceReplicas returns the effective replica count for a service.
func ServiceReplicas(svc ComposeService) int {
	if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
		return *svc.Deploy.Replicas
	}
	if svc.Replicas != nil && *svc.Replicas > 0 {
		return *svc.Replicas
	}
	return 1
}

// ToDockerConfigs converts a compose service to Docker SDK configs for a single replica.
func ToDockerConfigs(project, service string, svc ComposeService, index int, volumes map[string]ComposeVolume, networks map[string]ComposeNetwork) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
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
		Env:        svc.Environment.ToSlice(),
		Labels:     labels,
		Hostname:   service,
		Domainname: project,
	}
	if len(svc.Command) == 0 {
		cfg.Cmd = nil
	}

	hostCfg := &container.HostConfig{
		RestartPolicy: toRestartPolicy(svc.Restart),
	}

	if len(svc.Ports) > 0 {
		exposed, bindings, err := nat.ParsePortSpecs(svc.Ports)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid port spec for service %q: %w", service, err)
		}
		cfg.ExposedPorts = exposed
		hostCfg.PortBindings = bindings
	}

	for _, vm := range svc.Volumes {
		if err := applyVolumeMount(hostCfg, vm, volumes, project); err != nil {
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

// composeServiceNetworks handles both list and map forms of service networks.
type composeServiceNetworks []string

func (n *composeServiceNetworks) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*n = list
		return nil
	case yaml.MappingNode:
		list := make([]string, 0, len(value.Content)/2)
		for i := 0; i < len(value.Content); i += 2 {
			list = append(list, value.Content[i].Value)
		}
		*n = list
		return nil
	case yaml.ScalarNode:
		*n = []string{value.Value}
		return nil
	default:
		return fmt.Errorf("networks must be a string, list or map")
	}
}

// composeCommand handles command as either a string or a list.
type composeCommand []string

func (c *composeCommand) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		parts, err := shlex.Split(value.Value)
		if err != nil {
			return fmt.Errorf("invalid command shell string: %w", err)
		}
		*c = parts
		return nil
	}
	var parts []string
	if err := value.Decode(&parts); err != nil {
		return err
	}
	*c = parts
	return nil
}

// composeEnvironment handles environment as either a map or a list of KEY=VALUE.
type composeEnvironment map[string]string

func (e *composeEnvironment) UnmarshalYAML(value *yaml.Node) error {
	m := make(map[string]string)
	switch value.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(value.Content); i += 2 {
			key := value.Content[i].Value
			valNode := value.Content[i+1]
			if valNode.Tag == "!!null" {
				m[key] = ""
			} else {
				m[key] = valNode.Value
			}
		}
	case yaml.SequenceNode:
		for _, item := range value.Content {
			s := item.Value
			if idx := strings.Index(s, "="); idx >= 0 {
				m[s[:idx]] = s[idx+1:]
			} else {
				m[s] = ""
			}
		}
	default:
		return fmt.Errorf("environment must be a map or a list")
	}
	*e = m
	return nil
}

func (e composeEnvironment) ToSlice() []string {
	out := make([]string, 0, len(e))
	for k, v := range e {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

// composeLabels handles labels as either a map or a list of key=value.
type composeLabels map[string]string

func (l *composeLabels) UnmarshalYAML(value *yaml.Node) error {
	m := make(map[string]string)
	switch value.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(value.Content); i += 2 {
			m[value.Content[i].Value] = value.Content[i+1].Value
		}
	case yaml.SequenceNode:
		for _, item := range value.Content {
			s := item.Value
			if idx := strings.Index(s, "="); idx >= 0 {
				m[s[:idx]] = s[idx+1:]
			} else {
				m[s] = ""
			}
		}
	default:
		return fmt.Errorf("labels must be a map or a list")
	}
	*l = m
	return nil
}

// composeVolumeMount handles volume mounts in short string or object form.
type composeVolumeMount struct {
	Type     string `yaml:"type,omitempty"`
	Source   string `yaml:"source,omitempty"`
	Target   string `yaml:"target,omitempty"`
	ReadOnly bool   `yaml:"read_only,omitempty"`
	Mode     string `yaml:"-"`
}

func (v *composeVolumeMount) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		parts := strings.Split(value.Value, ":")
		v.Source = parts[0]
		if len(parts) > 1 {
			v.Target = parts[1]
		}
		if len(parts) > 2 {
			v.Mode = parts[2]
			v.ReadOnly = v.Mode == "ro" || v.Mode == "readonly"
		}
		return nil
	}
	type raw composeVolumeMount
	var r raw
	if err := value.Decode(&r); err != nil {
		return err
	}
	*v = composeVolumeMount(r)
	return nil
}

func applyVolumeMount(hostCfg *container.HostConfig, vm composeVolumeMount, volumes map[string]ComposeVolume, project string) error {
	if vm.Target == "" {
		return fmt.Errorf("volume mount target is required")
	}
	_, isNamedVolume := volumes[vm.Source]
	isPath := vm.Source == "" || strings.HasPrefix(vm.Source, "/") || strings.HasPrefix(vm.Source, "./") || strings.HasPrefix(vm.Source, "~/")

	if isNamedVolume && !isPath {
		// Named volume.
		if hostCfg.Mounts == nil {
			hostCfg.Mounts = []mount.Mount{}
		}
		hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
			Type:     mount.TypeVolume,
			Source:   ProjectResourceName(project, vm.Source),
			Target:   vm.Target,
			ReadOnly: vm.ReadOnly,
		})
		return nil
	}

	// Bind mount.
	bind := fmt.Sprintf("%s:%s", vm.Source, vm.Target)
	if vm.Mode != "" {
		bind = fmt.Sprintf("%s:%s", bind, vm.Mode)
	}
	hostCfg.Binds = append(hostCfg.Binds, bind)
	return nil
}

func toNetworkConfig(project, service string, svcNetworks composeServiceNetworks, networks map[string]ComposeNetwork) *network.NetworkingConfig {
	cfg := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}

	networkNames := svcNetworks
	if len(networkNames) == 0 {
		networkNames = []string{"default"}
	}

	for _, netName := range networkNames {
		dockerNetName := ProjectNetworkName(project, netName)
		if netName != "default" {
			if n, ok := networks[netName]; ok && n.External {
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

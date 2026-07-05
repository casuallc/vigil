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
	"testing"

	"github.com/docker/docker/api/types/mount"
)

func TestParseCompose(t *testing.T) {
	content := `
services:
  web:
    image: nginx:alpine
    command: nginx -g 'daemon off;'
    environment:
      FOO: bar
      EMPTY:
    ports:
      - "8080:80"
    volumes:
      - ./data:/data:ro
    networks:
      - frontend
    restart: unless-stopped
    labels:
      - app=web
  api:
    image: myapp:latest
    command: ["node", "server.js"]
    environment:
      - KEY=val
    deploy:
      replicas: 2
networks:
  frontend:
    driver: bridge
volumes:
  data:
    driver: local
`
	cf, err := ParseCompose([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cf.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(cf.Services))
	}
	web := cf.Services["web"]
	if web.Image != "nginx:alpine" {
		t.Fatalf("unexpected image: %s", web.Image)
	}
	if len(web.Command) != 3 || web.Command[0] != "nginx" {
		t.Fatalf("unexpected command: %v", web.Command)
	}
	if web.Environment["FOO"] != "bar" || web.Environment["EMPTY"] != "" {
		t.Fatalf("unexpected environment: %v", web.Environment)
	}
	if len(web.Ports) != 1 || web.Ports[0] != "8080:80" {
		t.Fatalf("unexpected ports: %v", web.Ports)
	}
	if len(web.Volumes) != 1 || web.Volumes[0].Source != "./data" || web.Volumes[0].Target != "/data" || !web.Volumes[0].ReadOnly {
		t.Fatalf("unexpected volumes: %+v", web.Volumes)
	}
	if len(web.Networks) != 1 || web.Networks[0] != "frontend" {
		t.Fatalf("unexpected networks: %v", web.Networks)
	}
	if web.Restart != "unless-stopped" {
		t.Fatalf("unexpected restart: %s", web.Restart)
	}
	if web.Labels["app"] != "web" {
		t.Fatalf("unexpected labels: %v", web.Labels)
	}
	api := cf.Services["api"]
	if len(api.Command) != 2 || api.Command[0] != "node" {
		t.Fatalf("unexpected api command: %v", api.Command)
	}
	if ServiceReplicas(api) != 2 {
		t.Fatalf("expected 2 replicas, got %d", ServiceReplicas(api))
	}
	if cf.Networks["frontend"].Driver != "bridge" {
		t.Fatalf("unexpected network driver: %s", cf.Networks["frontend"].Driver)
	}
	if cf.Volumes["data"].Driver != "local" {
		t.Fatalf("unexpected volume driver: %s", cf.Volumes["data"].Driver)
	}
}

func TestNormalizeProjectName(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		valid bool
	}{
		{"MyProject", "myproject", true},
		{"my-project", "my-project", true},
		{"my_project", "my_project", true},
		{"", "", false},
		{"my/project", "", false},
		{"my@project", "", false},
	}
	for _, tc := range cases {
		got, err := NormalizeProjectName(tc.in)
		if tc.valid {
			if err != nil || got != tc.want {
				t.Fatalf("NormalizeProjectName(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
			}
		} else {
			if err == nil {
				t.Fatalf("NormalizeProjectName(%q) expected error", tc.in)
			}
		}
	}
}

func TestServiceContainerName(t *testing.T) {
	if got := ServiceContainerName("demo", "web", 1); got != "demo_web_1" {
		t.Fatalf("unexpected name: %s", got)
	}
}

func TestToDockerConfigs(t *testing.T) {
	svc := ComposeService{
		Image:   "nginx:alpine",
		Command: composeCommand{"nginx", "-g", "daemon off;"},
		Environment: composeEnvironment{
			"FOO": "bar",
		},
		Ports: []string{"8080:80"},
		Volumes: []composeVolumeMount{
			{Source: "./data", Target: "/data", ReadOnly: true, Mode: "ro"},
		},
		Networks: composeServiceNetworks{"frontend"},
		Restart:  "on-failure:3",
		Labels:   composeLabels{"app": "web"},
	}
	volumes := map[string]ComposeVolume{
		"data": {Driver: "local"},
	}
	networks := map[string]ComposeNetwork{
		"frontend": {Driver: "bridge"},
	}

	cfg, hostCfg, netCfg, err := ToDockerConfigs("demo", "web", svc, 1, volumes, networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Image != "nginx:alpine" {
		t.Fatalf("unexpected image: %s", cfg.Image)
	}
	if len(cfg.Cmd) != 3 {
		t.Fatalf("unexpected cmd: %v", cfg.Cmd)
	}
	if cfg.Labels[ComposeProjectLabel] != "demo" || cfg.Labels[ComposeServiceLabel] != "web" {
		t.Fatalf("unexpected labels: %v", cfg.Labels)
	}
	if cfg.Hostname != "web" {
		t.Fatalf("unexpected hostname: %s", cfg.Hostname)
	}
	if len(cfg.ExposedPorts) != 1 {
		t.Fatalf("unexpected exposed ports: %v", cfg.ExposedPorts)
	}
	if len(hostCfg.Binds) != 1 || hostCfg.Binds[0] != "./data:/data:ro" {
		t.Fatalf("unexpected binds: %v", hostCfg.Binds)
	}
	if hostCfg.RestartPolicy.Name != "on-failure" || hostCfg.RestartPolicy.MaximumRetryCount != 3 {
		t.Fatalf("unexpected restart policy: %+v", hostCfg.RestartPolicy)
	}
	if netCfg == nil || len(netCfg.EndpointsConfig) != 1 {
		t.Fatalf("unexpected network config: %+v", netCfg)
	}
	if _, ok := netCfg.EndpointsConfig["demo_frontend"]; !ok {
		t.Fatalf("expected demo_frontend network, got %v", netCfg.EndpointsConfig)
	}
}

func TestToDockerConfigs_NamedVolume(t *testing.T) {
	svc := ComposeService{
		Image: "app:latest",
		Volumes: []composeVolumeMount{
			{Source: "data", Target: "/data", ReadOnly: true},
		},
	}
	volumes := map[string]ComposeVolume{
		"data": {Driver: "local"},
	}

	_, hostCfg, _, err := ToDockerConfigs("demo", "app", svc, 1, volumes, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hostCfg.Mounts) != 1 {
		t.Fatalf("expected 1 mount, got %v", hostCfg.Mounts)
	}
	if hostCfg.Mounts[0].Type != mount.TypeVolume || hostCfg.Mounts[0].Source != "demo_data" {
		t.Fatalf("unexpected mount: %+v", hostCfg.Mounts[0])
	}
}

func TestToDockerConfigs_ExternalNetwork(t *testing.T) {
	svc := ComposeService{
		Image:    "app:latest",
		Networks: composeServiceNetworks{"public"},
	}
	networks := map[string]ComposeNetwork{
		"public": {External: true},
	}
	_, _, netCfg, err := ToDockerConfigs("demo", "app", svc, 1, nil, networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := netCfg.EndpointsConfig["public"]; !ok {
		t.Fatalf("expected external network name 'public', got %v", netCfg.EndpointsConfig)
	}
}

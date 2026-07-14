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
	"os"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func strPtr(s string) *string { return &s }

func serviceByName(project *types.Project, name string) (types.ServiceConfig, bool) {
	for _, svc := range project.Services {
		if svc.Name == name {
			return svc, true
		}
	}
	return types.ServiceConfig{}, false
}

func TestLoadCompose(t *testing.T) {
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
      - /host/data:/data:ro
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
	proj, err := LoadCompose(context.Background(), "demo", []byte(content), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proj.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(proj.Services))
	}

	web, ok := serviceByName(proj, "web")
	if !ok {
		t.Fatalf("service web not found")
	}
	if web.Image != "nginx:alpine" {
		t.Fatalf("unexpected image: %s", web.Image)
	}
	if len(web.Command) != 3 || web.Command[0] != "nginx" {
		t.Fatalf("unexpected command: %v", web.Command)
	}
	if web.Environment["FOO"] == nil || *web.Environment["FOO"] != "bar" {
		t.Fatalf("unexpected environment FOO: %v", web.Environment["FOO"])
	}
	if web.Environment["EMPTY"] != nil && *web.Environment["EMPTY"] != "" {
		t.Fatalf("unexpected environment EMPTY: %v", web.Environment["EMPTY"])
	}
	if len(web.Ports) != 1 || web.Ports[0].Target != 80 {
		t.Fatalf("unexpected ports: %v", web.Ports)
	}
	if len(web.Volumes) != 1 || web.Volumes[0].Source != "/host/data" || web.Volumes[0].Target != "/data" || !web.Volumes[0].ReadOnly {
		t.Fatalf("unexpected volumes: %+v", web.Volumes)
	}
	if len(web.Networks) != 1 {
		t.Fatalf("unexpected networks: %v", web.Networks)
	}
	if _, ok := web.Networks["frontend"]; !ok {
		t.Fatalf("expected frontend network")
	}
	if web.Restart != "unless-stopped" {
		t.Fatalf("unexpected restart: %s", web.Restart)
	}
	if web.Labels["app"] != "web" {
		t.Fatalf("unexpected labels: %v", web.Labels)
	}

	api, ok := serviceByName(proj, "api")
	if !ok {
		t.Fatalf("service api not found")
	}
	if len(api.Command) != 2 || api.Command[0] != "node" {
		t.Fatalf("unexpected api command: %v", api.Command)
	}
	if ServiceReplicas(api) != 2 {
		t.Fatalf("expected 2 replicas, got %d", ServiceReplicas(api))
	}

	if proj.Networks["frontend"].Driver != "bridge" {
		t.Fatalf("unexpected network driver: %s", proj.Networks["frontend"].Driver)
	}
	if proj.Volumes["data"].Driver != "local" {
		t.Fatalf("unexpected volume driver: %s", proj.Volumes["data"].Driver)
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

func TestServiceToDockerConfigs(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "web",
		Image:   "nginx:alpine",
		Command: types.ShellCommand{"nginx", "-g", "daemon off;"},
		Environment: types.MappingWithEquals{
			"FOO": strPtr("bar"),
		},
		Ports: []types.ServicePortConfig{
			{Target: 80, Published: "8080", Protocol: "tcp"},
		},
		Volumes: []types.ServiceVolumeConfig{
			{Type: "bind", Source: "./data", Target: "/data", ReadOnly: true},
		},
		Networks: map[string]*types.ServiceNetworkConfig{
			"frontend": {},
		},
		Restart: "on-failure:3",
		Labels:  types.Labels{"app": "web"},
	}
	networks := types.Networks{
		"frontend": {Driver: "bridge"},
	}

	cfg, hostCfg, netCfg, err := serviceToDockerConfigs("demo", "web", svc, 1, networks)
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
	if cfg.Labels[ComposeRestartLabel] != "on-failure:3" {
		t.Fatalf("unexpected restart label: %s", cfg.Labels[ComposeRestartLabel])
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

func TestServiceToDockerConfigs_OneOffRestart(t *testing.T) {
	svc := types.ServiceConfig{
		Name:    "init",
		Image:   "app:latest",
		Restart: "no",
	}

	cfg, hostCfg, _, err := serviceToDockerConfigs("demo", "init", svc, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Labels[ComposeRestartLabel] != "no" {
		t.Fatalf("expected restart label 'no', got %q", cfg.Labels[ComposeRestartLabel])
	}
	if hostCfg.RestartPolicy.Name != container.RestartPolicyDisabled {
		t.Fatalf("expected disabled restart policy, got %+v", hostCfg.RestartPolicy)
	}
}

func TestServiceToDockerConfigs_NamedVolume(t *testing.T) {
	svc := types.ServiceConfig{
		Name:  "app",
		Image: "app:latest",
		Volumes: []types.ServiceVolumeConfig{
			{Type: "volume", Source: "data", Target: "/data", ReadOnly: true},
		},
	}

	_, hostCfg, _, err := serviceToDockerConfigs("demo", "app", svc, 1, nil)
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

func TestServiceToDockerConfigs_ExternalNetwork(t *testing.T) {
	svc := types.ServiceConfig{
		Name:  "app",
		Image: "app:latest",
		Networks: map[string]*types.ServiceNetworkConfig{
			"public": {},
		},
	}
	networks := types.Networks{
		"public": {External: true},
	}

	_, _, netCfg, err := serviceToDockerConfigs("demo", "app", svc, 1, networks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := netCfg.EndpointsConfig["public"]; !ok {
		t.Fatalf("expected external network name 'public', got %v", netCfg.EndpointsConfig)
	}
}

func TestLoadCompose_EnvInterpolation(t *testing.T) {
	content := `
services:
  web:
    image: ${IMAGE}
    environment:
      PORT: ${PORT:-8080}
`
	env := map[string]string{
		"IMAGE": "myapp:v1",
	}
	proj, err := LoadCompose(context.Background(), "demo", []byte(content), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	web, ok := serviceByName(proj, "web")
	if !ok {
		t.Fatalf("service web not found")
	}
	if web.Image != "myapp:v1" {
		t.Fatalf("unexpected image: %s", web.Image)
	}
	if web.Environment["PORT"] == nil || *web.Environment["PORT"] != "8080" {
		t.Fatalf("unexpected PORT: %v", web.Environment["PORT"])
	}
}

func TestLoadComposeFromDir(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  web:
    image: nginx:alpine
    environment:
      APP: ${APP}
`
	if err := os.WriteFile(dir+"/docker-compose.yml", []byte(content), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	if err := os.WriteFile(dir+"/.env", []byte("APP=fromenv\n"), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	env := map[string]string{"APP": "fromreq"}
	proj, err := LoadComposeFromDir(context.Background(), "demo", dir, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	web, ok := serviceByName(proj, "web")
	if !ok {
		t.Fatalf("service web not found")
	}
	if web.Environment["APP"] == nil || *web.Environment["APP"] != "fromreq" {
		t.Fatalf("request env should override .env, got: %v", web.Environment["APP"])
	}
}

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	content := `# comment
IMAGE=myimage:v2

QUOTED="double"
SINGLE='single'
NO_VALUE=
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["IMAGE"] != "myimage:v2" {
		t.Fatalf("unexpected IMAGE: %q", env["IMAGE"])
	}
	if env["QUOTED"] != "double" {
		t.Fatalf("unexpected QUOTED: %q", env["QUOTED"])
	}
	if env["SINGLE"] != "single" {
		t.Fatalf("unexpected SINGLE: %q", env["SINGLE"])
	}
	if v, ok := env["NO_VALUE"]; !ok || v != "" {
		t.Fatalf("unexpected NO_VALUE: %q", v)
	}
	if _, ok := env["# comment"]; ok {
		t.Fatalf("comment should not be parsed")
	}
}

func TestLoadEnvFile_Missing(t *testing.T) {
	env, err := LoadEnvFile("/nonexistent/path/.env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 0 {
		t.Fatalf("expected empty env, got %v", env)
	}
}

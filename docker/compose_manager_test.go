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

	"github.com/docker/docker/api/types"
)

func TestComposeManager_DeployProject_ImageExists(t *testing.T) {
	fc := &FakeClient{
		createdID:      "cid1",
		existingImages: map[string]struct{}{"nginx:alpine": {}},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	content := `
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	status, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Name != "demo" {
		t.Fatalf("unexpected project name: %s", status.Name)
	}
	if len(fc.imagesPulled) != 0 {
		t.Fatalf("expected no pulls when image exists locally, got %v", fc.imagesPulled)
	}
	if fc.timesStarted != 1 {
		t.Fatalf("expected container started once, got %d", fc.timesStarted)
	}
}

func TestComposeManager_DeployProject(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	content := `
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	status, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Name != "demo" {
		t.Fatalf("unexpected project name: %s", status.Name)
	}
	if len(status.Services) != 1 || status.Services[0].Name != "web" {
		t.Fatalf("unexpected services: %+v", status.Services)
	}
	if len(fc.imagesPulled) != 1 || fc.imagesPulled[0] != "nginx:alpine" {
		t.Fatalf("unexpected pulled images: %v", fc.imagesPulled)
	}
	if len(fc.networksCreated) != 1 || fc.networksCreated[0] != "demo_default" {
		t.Fatalf("unexpected networks: %v", fc.networksCreated)
	}
	if fc.timesStarted != 1 {
		t.Fatalf("expected container started once, got %d", fc.timesStarted)
	}
}

func TestComposeManager_DeployProject_ExistingConflict(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{ID: "old", Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"}},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	_, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: "services:\n  web:\n    image: nginx:alpine\n",
	})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}

func TestComposeManager_GetProject(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"}},
			{ID: "c2", Names: []string{"/demo_api_1"}, Image: "app:latest", Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "api"}},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(status.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(status.Services))
	}
}

func TestComposeManager_GetProject_OneOffExited(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{
				ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", State: "running",
				Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"},
			},
			{
				ID: "c2", Names: []string{"/demo_init_1"}, Image: "app:latest", State: "exited",
				Labels: map[string]string{
					ComposeProjectLabel: "demo",
					ComposeServiceLabel: "init",
					ComposeRestartLabel: "no",
				},
			},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "running" {
		t.Fatalf("expected status running, got %s", status.Status)
	}
	if len(status.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(status.Services))
	}
	initSvc := serviceStatusByName(status.Services, "init")
	if initSvc == nil {
		t.Fatalf("init service not found in %+v", status.Services)
	}
	if initSvc.Restart != "no" {
		t.Fatalf("expected restart no, got %q", initSvc.Restart)
	}
}

func serviceStatusByName(services []ComposeServiceStatus, name string) *ComposeServiceStatus {
	for i := range services {
		if services[i].Name == name {
			return &services[i]
		}
	}
	return nil
}

func TestComposeManager_GetProject_Running(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{
				ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", State: "running",
				Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"},
			},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "running" {
		t.Fatalf("expected status running, got %s", status.Status)
	}
}

func TestComposeManager_GetProject_Partial(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{
				ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", State: "running",
				Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"},
			},
			{
				ID: "c2", Names: []string{"/demo_api_1"}, Image: "app:latest", State: "exited",
				Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "api"},
			},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "partial" {
		t.Fatalf("expected status partial, got %s", status.Status)
	}
}

func TestComposeManager_GetProject_Stopped(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{
				ID: "c1", Names: []string{"/demo_web_1"}, Image: "nginx:alpine", State: "exited",
				Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"},
			},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "stopped" {
		t.Fatalf("expected status stopped, got %s", status.Status)
	}
}

func TestComposeManager_GetProject_CompletedOnlyOneOffs(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{
				ID: "c1", Names: []string{"/demo_init_1"}, Image: "app:latest", State: "exited",
				Labels: map[string]string{
					ComposeProjectLabel: "demo",
					ComposeServiceLabel: "init",
					ComposeRestartLabel: "no",
				},
			},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	status, err := cm.GetProject(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "completed" {
		t.Fatalf("expected status completed, got %s", status.Status)
	}
}

func TestComposeManager_RemoveProject(t *testing.T) {
	fc := &FakeClient{
		containers: []types.Container{
			{ID: "c1", Labels: map[string]string{ComposeProjectLabel: "demo", ComposeServiceLabel: "web"}},
		},
	}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	if err := cm.RemoveProject(context.Background(), "demo", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.timesRemoved != 1 {
		t.Fatalf("expected container removed once, got %d", fc.timesRemoved)
	}
}

func TestComposeManager_DeployProject_Volume(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	content := `
services:
  app:
    image: app:latest
    volumes:
      - data:/data
volumes:
  data:
    driver: local
`
	_, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.volumesCreated) != 1 || fc.volumesCreated[0] != "demo_data" {
		t.Fatalf("unexpected volumes: %v", fc.volumesCreated)
	}
}

func TestComposeManager_DeployProject_ExternalNetworkNotFound(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	content := `
services:
  app:
    image: app:latest
    networks:
      - public
networks:
  public:
    external: true
`
	_, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: content,
	})
	if err == nil {
		t.Fatalf("expected external network not found error")
	}
}

func TestComposeManager_DeployProjectFromDir(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	dir := t.TempDir()
	composePath := dir + "/docker-compose.yml"
	content := "services:\n  web:\n    image: nginx:alpine\n"
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	status, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Name: "demo",
		Dir:  dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Name != "demo" {
		t.Fatalf("unexpected project name: %s", status.Name)
	}
	if len(status.Services) != 1 || status.Services[0].Name != "web" {
		t.Fatalf("unexpected services: %+v", status.Services)
	}
}

func TestComposeManager_DeployProjectFromDir_NameFromDir(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	parent := t.TempDir()
	dir := parent + "/myproject"
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	if err := os.WriteFile(dir+"/docker-compose.yml", []byte("services:\n  web:\n    image: nginx:alpine\n"), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	status, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Name != "myproject" {
		t.Fatalf("expected project name myproject, got %s", status.Name)
	}
}

func TestComposeManager_DeployProjectFromDir_MissingDir(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	_, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Name: "demo",
		Dir:  "/nonexistent/path",
	})
	if err == nil {
		t.Fatalf("expected error for missing dir")
	}
}

func TestComposeManager_DeployProjectFromDir_NotADirectory(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	f, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()

	_, err = cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Name: "demo",
		Dir:  f.Name(),
	})
	if err == nil {
		t.Fatalf("expected error for file path")
	}
}

func TestComposeManager_DeployProjectFromDir_MissingComposeFile(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	dir := t.TempDir()
	_, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Name: "demo",
		Dir:  dir,
	})
	if err == nil {
		t.Fatalf("expected error for missing compose file")
	}
}

func TestComposeManager_DeployProjectFromDir_InvalidProjectName(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	parent := t.TempDir()
	dir := parent + "/my project!"
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	if err := os.WriteFile(dir+"/docker-compose.yml", []byte("services:\n  web:\n    image: nginx:alpine\n"), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	_, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Dir: dir,
	})
	if err == nil {
		t.Fatalf("expected error for invalid project name")
	}
}

func TestComposeManager_DeployProject_EnvInterpolation(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	content := `
services:
  web:
    image: ${TEST_COMPOSE_IMAGE:-nginx:alpine}
`
	_, err := cm.DeployProject(context.Background(), ComposeDeployRequest{
		Name:    "demo",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.imagesPulled) != 1 || fc.imagesPulled[0] != "nginx:alpine" {
		t.Fatalf("unexpected pulled images: %v", fc.imagesPulled)
	}
}

func TestComposeManager_DeployProjectFromDir_EnvFile(t *testing.T) {
	fc := &FakeClient{createdID: "cid1"}
	mgr := NewManagerWithClient(fc)
	cm := NewComposeManager(mgr)

	dir := t.TempDir()
	composeContent := "services:\n  web:\n    image: ${APP_IMAGE}\n"
	envContent := "APP_IMAGE=fromenv:v1\n"
	if err := os.WriteFile(dir+"/docker-compose.yml", []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	if err := os.WriteFile(dir+"/.env", []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	_, err := cm.DeployProjectFromDir(context.Background(), ComposeDeployFromDirRequest{
		Name: "demo",
		Dir:  dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.imagesPulled) != 1 || fc.imagesPulled[0] != "fromenv:v1" {
		t.Fatalf("unexpected pulled images: %v", fc.imagesPulled)
	}
}

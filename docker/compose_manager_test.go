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
	"testing"

	"github.com/docker/docker/api/types"
)

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

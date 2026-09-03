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

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/casuallc/vigil/proxy"
	"github.com/gorilla/mux"
)

func newProxyTestServer(t *testing.T, cfg *proxy.ProxyConfig) *Server {
	t.Helper()
	pm, err := proxy.NewManager(cfg, filepath.Join(t.TempDir(), "vigil.db"), nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { pm.Shutdown(nil) })
	return &Server{proxyManager: pm}
}

func proxyRequest(t *testing.T, method, path string, body interface{}, vars map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}
	return httptest.NewRecorder(), req
}

func TestProxyHandlersDisabled(t *testing.T) {
	server := &Server{}
	rr, req := proxyRequest(t, http.MethodGet, "/api/proxy/instances", nil, nil)
	server.handleProxyList(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestProxyHandlersCRUD(t *testing.T) {
	server := newProxyTestServer(t, &proxy.ProxyConfig{Enabled: true})

	// Create (stopped).
	create := ProxyCreateRequest{Config: proxy.InstanceConfig{
		Name:         "web",
		Listen:       "127.0.0.1:0",
		Target:       "http://127.0.0.1:9000",
		AllowPrivate: true,
	}}
	rr, req := proxyRequest(t, http.MethodPost, "/api/proxy/instances", create, nil)
	server.handleProxyCreate(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Duplicate create.
	rr, req = proxyRequest(t, http.MethodPost, "/api/proxy/instances", create, nil)
	server.handleProxyCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("duplicate create: expected 400, got %d", rr.Code)
	}

	// List.
	rr, req = proxyRequest(t, http.MethodGet, "/api/proxy/instances", nil, nil)
	server.handleProxyList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}
	var list []proxy.InstanceStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list: got %s, err=%v", rr.Body.String(), err)
	}
	if list[0].State != proxy.StateStopped {
		t.Fatalf("new instance state = %q, want stopped", list[0].State)
	}

	// Start then stop.
	rr, req = proxyRequest(t, http.MethodPost, "/api/proxy/instances/web/start", nil, map[string]string{"name": "web"})
	server.handleProxyStart(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("start: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var started proxy.InstanceStatus
	_ = json.Unmarshal(rr.Body.Bytes(), &started)
	if started.State != proxy.StateRunning {
		t.Fatalf("state after start = %q, want running", started.State)
	}

	rr, req = proxyRequest(t, http.MethodPost, "/api/proxy/instances/web/stop", nil, map[string]string{"name": "web"})
	server.handleProxyStop(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stop: expected 200, got %d", rr.Code)
	}

	// Update.
	updatedCfg := create.Config
	updatedCfg.MaxBodyMB = 10
	rr, req = proxyRequest(t, http.MethodPut, "/api/proxy/instances/web", updatedCfg, map[string]string{"name": "web"})
	server.handleProxyUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Delete.
	rr, req = proxyRequest(t, http.MethodDelete, "/api/proxy/instances/web", nil, map[string]string{"name": "web"})
	server.handleProxyDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", rr.Code)
	}

	// Gone now.
	rr, req = proxyRequest(t, http.MethodGet, "/api/proxy/instances/web", nil, map[string]string{"name": "web"})
	server.handleProxyGet(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", rr.Code)
	}
}

func TestProxyDeleteConfigOriginConflict(t *testing.T) {
	cfg := &proxy.ProxyConfig{
		Enabled: true,
		Instances: []proxy.InstanceConfig{{
			Name:   "static",
			Listen: "127.0.0.1:0",
			Target: "http://127.0.0.1:9000",
		}},
	}
	pm, err := proxy.NewManager(cfg, filepath.Join(t.TempDir(), "vigil.db"), nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	// Register the config instance without binding its listener.
	if err := pm.Recover(nil); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	defer pm.Shutdown(nil)
	server := &Server{proxyManager: pm}

	rr, req := proxyRequest(t, http.MethodDelete, "/api/proxy/instances/static", nil, map[string]string{"name": "static"})
	server.handleProxyDelete(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("delete config instance: expected 409, got %d", rr.Code)
	}
}

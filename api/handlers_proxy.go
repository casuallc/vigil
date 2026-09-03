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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/casuallc/vigil/proxy"
	"github.com/gorilla/mux"
)

// proxyManagerOr503 returns the proxy manager or writes a 503 error.
func (s *Server) proxyManagerOr503(w http.ResponseWriter) *proxy.Manager {
	if s.proxyManager == nil {
		writeError(w, http.StatusServiceUnavailable, "proxy feature is disabled; set proxy.enabled: true in config.yaml")
		return nil
	}
	return s.proxyManager
}

// proxyError maps proxy package sentinel errors to HTTP status codes.
func proxyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, proxy.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, proxy.ErrConfigOrigin):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

// handleProxyList handles GET /api/proxy/instances
func (s *Server) handleProxyList(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	writeJSON(w, http.StatusOK, m.List())
}

// handleProxyCreate handles POST /api/proxy/instances
func (s *Server) handleProxyCreate(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	var req ProxyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Config.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := m.Create(req.Config, req.Autostart); err != nil {
		proxyError(w, err)
		return
	}
	status, _ := m.Get(req.Config.Name)
	writeJSON(w, http.StatusCreated, status)
}

// handleProxyGet handles GET /api/proxy/instances/{name}
func (s *Server) handleProxyGet(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	status, err := m.Get(mux.Vars(r)["name"])
	if err != nil {
		proxyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// handleProxyUpdate handles PUT /api/proxy/instances/{name}
func (s *Server) handleProxyUpdate(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	name := mux.Vars(r)["name"]
	var cfg proxy.InstanceConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := m.Update(name, cfg); err != nil {
		proxyError(w, err)
		return
	}
	status, _ := m.Get(name)
	writeJSON(w, http.StatusOK, status)
}

// handleProxyDelete handles DELETE /api/proxy/instances/{name}
func (s *Server) handleProxyDelete(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	if err := m.Delete(mux.Vars(r)["name"]); err != nil {
		proxyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// handleProxyStart handles POST /api/proxy/instances/{name}/start
func (s *Server) handleProxyStart(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	name := mux.Vars(r)["name"]
	if err := m.Start(name); err != nil {
		proxyError(w, err)
		return
	}
	status, _ := m.Get(name)
	writeJSON(w, http.StatusOK, status)
}

// handleProxyStop handles POST /api/proxy/instances/{name}/stop
func (s *Server) handleProxyStop(w http.ResponseWriter, r *http.Request) {
	m := s.proxyManagerOr503(w)
	if m == nil {
		return
	}
	name := mux.Vars(r)["name"]
	if err := m.Stop(name); err != nil {
		proxyError(w, err)
		return
	}
	status, _ := m.Get(name)
	writeJSON(w, http.StatusOK, status)
}

// handleProxyStatus handles GET /api/proxy/instances/{name}/status
func (s *Server) handleProxyStatus(w http.ResponseWriter, r *http.Request) {
	s.handleProxyGet(w, r)
}

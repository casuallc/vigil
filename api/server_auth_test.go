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
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casuallc/vigil/config"
)

func newAuthTestServer() *Server {
	return &Server{
		config: &config.Config{
			BasicAuth: config.BasicAuth{
				Enabled:  true,
				Username: "bbx",
				Password: "newpass",
			},
		},
	}
}

func basicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

// TestBasicAuthMiddlewareWebSocketQueryCredentials covers the WebSocket upgrade
// path: browsers cannot set an Authorization header on a handshake, so the web
// console carries the credentials in the URL instead.
func TestBasicAuthMiddlewareWebSocketQueryCredentials(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		authHeader string
		upgrade    bool
		wantStatus int
	}{
		{
			name:       "query credentials accepted on upgrade",
			url:        "/api/vms/ssh/ws?vm_name=web-1&username=bbx&password=newpass",
			upgrade:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "stale cached credentials fall back to query credentials",
			url:        "/api/vms/ssh/ws?vm_name=web-1&username=bbx&password=newpass",
			authHeader: basicAuthHeader("bbx", "oldpass"),
			upgrade:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid header credentials win",
			url:        "/api/vms/ssh/ws?vm_name=web-1&username=bbx&password=stale",
			authHeader: basicAuthHeader("bbx", "newpass"),
			upgrade:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong query credentials rejected on upgrade",
			url:        "/api/vms/ssh/ws?vm_name=web-1&username=bbx&password=oldpass",
			upgrade:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "query credentials rejected on plain request",
			url:        "/api/vms/servers?username=bbx&password=newpass",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "upgrade without credentials rejected",
			url:        "/api/vms/ssh/ws?vm_name=web-1",
			upgrade:    true,
			wantStatus: http.StatusUnauthorized,
		},
	}

	server := newAuthTestServer()
	reached := false
	handler := server.BasicAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reached = false

			req := httptest.NewRequest(http.MethodGet, c.url, nil)
			if c.authHeader != "" {
				req.Header.Set("Authorization", c.authHeader)
			}
			if c.upgrade {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if got := reached; got != (c.wantStatus == http.StatusOK) {
				t.Fatalf("handler reached = %v, want %v", got, c.wantStatus == http.StatusOK)
			}
			if c.wantStatus == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("expected a challenge header on rejection")
			}
		})
	}
}

func TestRedactQueryCredentials(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"vm_name=web-1", "vm_name=web-1"},
		{"vm_name=web-1&username=bbx&password=secret", "vm_name=web-1&username=bbx&password=***"},
		{"password=secret&password=other", "password=***&password=***"},
		{"PWD=secret", "PWD=***"},
		{"vm_name=web-1&passwd=secret", "vm_name=web-1&passwd=***"},
	}

	for _, c := range cases {
		if got := redactQueryCredentials(c.in); got != c.want {
			t.Errorf("redactQueryCredentials(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

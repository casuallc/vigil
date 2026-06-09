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
  "net"
  "net/http"
  "strconv"
  "time"
)

// handleNetworkProbe handles POST /api/network/probe endpoint
func (s *Server) handleNetworkProbe(w http.ResponseWriter, r *http.Request) {
  var req NetworkProbeRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
  }

  // Validate required fields
  if req.TargetIP == "" {
    writeError(w, http.StatusBadRequest, "targetIp is required")
    return
  }
  if req.Port <= 0 || req.Port > 65535 {
    writeError(w, http.StatusBadRequest, "port must be between 1 and 65535")
    return
  }

  // Default protocol to tcp
  protocol := req.Protocol
  if protocol == "" {
    protocol = "tcp"
  }

  // Default timeout to 5000ms
  timeout := 5 * time.Second
  if req.TimeoutMs > 0 {
    timeout = time.Duration(req.TimeoutMs) * time.Millisecond
  }

  addr := net.JoinHostPort(req.TargetIP, strconv.Itoa(req.Port))
  start := time.Now()
  conn, err := net.DialTimeout(protocol, addr, timeout)
  latency := time.Since(start)

  if err != nil {
    writeJSON(w, http.StatusOK, NetworkProbeResponse{
      Reachable: false,
      LatencyMs: -1,
      Error:     err.Error(),
    })
    return
  }
  defer conn.Close()

  writeJSON(w, http.StatusOK, NetworkProbeResponse{
    Reachable: true,
    LatencyMs: latency.Milliseconds(),
    Error:     "",
  })
}

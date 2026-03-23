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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/casuallc/vigil/inspection"
)

// handleCosmicInspect handles cosmic inspection requests
func (s *Server) handleCosmicInspect(w http.ResponseWriter, r *http.Request) {
	var req inspection.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate request parameters
	if len(req.Checks) == 0 {
		writeError(w, http.StatusBadRequest, "No checks specified in request")
		return
	}

	// Execute checks
	var results []inspection.CheckResult
	totalChecks := len(req.Checks)
	passedChecks := 0
	warningChecks := 0
	errorChecks := 0

	for _, check := range req.Checks {
		result := inspection.ExecuteCheck(check, req.Env)
		results = append(results, result)

		// Count check results
		switch strings.ToLower(result.Status) {
		case inspection.StatusOk:
			passedChecks++
		case inspection.StatusError:
			errorChecks++
		}
	}

	// Determine overall status
	overallStatus := inspection.StatusOk
	if errorChecks > 0 {
		overallStatus = inspection.StatusError
	}

	response := inspection.Result{
		ID: req.Meta.JobName,
		Meta: inspection.ResultMeta{
			System:  req.Meta.System,
			Host:    req.Meta.Host,
			JobName: req.Meta.JobName,
			Time:    time.Now(),
			Status:  overallStatus,
		},
		Results: results,
		Summary: inspection.SummaryInfo{
			TotalChecks:   totalChecks,
			OK:            passedChecks,
			Warn:          warningChecks,
			Critical:      errorChecks,
			OverallStatus: overallStatus,
			StartedAt:     req.Meta.Time,
			FinishedAt:    time.Now(),
		},
	}

	writeJSON(w, http.StatusOK, response)
}

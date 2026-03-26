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
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
)

// ------------------------- Schedule Handlers -------------------------

// handleListSchedules handles GET /api/schedules
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	// Get query parameters
	enabled := r.URL.Query().Get("enabled")
	vmName := r.URL.Query().Get("vm_name")

	query := "SELECT id, name, description, command, vm_names, cron, enabled, timeout, created_by, created_at, updated_at, last_run_at, last_run_status FROM schedules WHERE 1=1"
	var args []interface{}

	if enabled != "" {
		query += " AND enabled = ?"
		args = append(args, map[string]bool{"true": true, "1": true}[enabled])
	}

	if vmName != "" {
		query += " AND vm_names LIKE ?"
		args = append(args, "%"+vmName+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.scheduleDB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query schedules: "+err.Error())
		return
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sch Schedule
		var vmNamesJSON string
		var enabledInt int
		var lastRunAt sql.NullTime
		err := rows.Scan(
			&sch.ID,
			&sch.Name,
			&sch.Description,
			&sch.Command,
			&vmNamesJSON,
			&sch.Cron,
			&enabledInt,
			&sch.Timeout,
			&sch.CreatedBy,
			&sch.CreatedAt,
			&sch.UpdatedAt,
			&lastRunAt,
			&sch.LastRunStatus,
		)
		if err != nil {
			continue
		}
		sch.Enabled = enabledInt == 1
		json.Unmarshal([]byte(vmNamesJSON), &sch.VMNames)
		if lastRunAt.Valid {
			sch.LastRunAt = &lastRunAt.Time
		}
		// Calculate next run time
		sch.NextRunAt = calculateNextRun(sch.Cron, sch.LastRunAt)
		schedules = append(schedules, sch)
	}

	writeJSON(w, http.StatusOK, schedules)
}

// handleCreateSchedule handles POST /api/schedules
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" || req.Command == "" || req.Cron == "" || len(req.VMNames) == 0 {
		writeError(w, http.StatusBadRequest, "name, command, cron and vm_names are required")
		return
	}

	// Set defaults
	if req.Timeout == 0 {
		req.Timeout = 300
	}

	schedule := Schedule{
		ID:          fmt.Sprintf("schedule_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Description: req.Description,
		Command:     req.Command,
		VMNames:     req.VMNames,
		Cron:        req.Cron,
		Enabled:     req.Enabled,
		Timeout:     req.Timeout,
		CreatedBy:   s.getCurrentUser(r),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	vmNamesJSON, _ := json.Marshal(schedule.VMNames)

	_, err := s.scheduleDB.Exec(
		"INSERT INTO schedules (id, name, description, command, vm_names, cron, enabled, timeout, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		schedule.ID,
		schedule.Name,
		schedule.Description,
		schedule.Command,
		string(vmNamesJSON),
		schedule.Cron,
		boolToInt(schedule.Enabled),
		schedule.Timeout,
		schedule.CreatedBy,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create schedule: "+err.Error())
		return
	}

	schedule.NextRunAt = calculateNextRun(schedule.Cron, nil)
	writeJSON(w, http.StatusCreated, schedule)
}

// handleGetSchedule handles GET /api/schedules/{id}
func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var sch Schedule
	var vmNamesJSON string
	var enabledInt int
	var lastRunAt sql.NullTime

	err := s.scheduleDB.QueryRow(
		"SELECT id, name, description, command, vm_names, cron, enabled, timeout, created_by, created_at, updated_at, last_run_at, last_run_status FROM schedules WHERE id = ?",
		id,
	).Scan(
		&sch.ID,
		&sch.Name,
		&sch.Description,
		&sch.Command,
		&vmNamesJSON,
		&sch.Cron,
		&enabledInt,
		&sch.Timeout,
		&sch.CreatedBy,
		&sch.CreatedAt,
		&sch.UpdatedAt,
		&lastRunAt,
		&sch.LastRunStatus,
	)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get schedule: "+err.Error())
		return
	}

	sch.Enabled = enabledInt == 1
	json.Unmarshal([]byte(vmNamesJSON), &sch.VMNames)
	if lastRunAt.Valid {
		sch.LastRunAt = &lastRunAt.Time
	}
	sch.NextRunAt = calculateNextRun(sch.Cron, sch.LastRunAt)

	writeJSON(w, http.StatusOK, sch)
}

// handleUpdateSchedule handles PUT /api/schedules/{id}
func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" || req.Command == "" || req.Cron == "" || len(req.VMNames) == 0 {
		writeError(w, http.StatusBadRequest, "name, command, cron and vm_names are required")
		return
	}

	// Set defaults
	if req.Timeout == 0 {
		req.Timeout = 300
	}

	vmNamesJSON, _ := json.Marshal(req.VMNames)

	_, err := s.scheduleDB.Exec(
		"UPDATE schedules SET name = ?, description = ?, command = ?, vm_names = ?, cron = ?, enabled = ?, timeout = ?, updated_at = ? WHERE id = ?",
		req.Name,
		req.Description,
		req.Command,
		string(vmNamesJSON),
		req.Cron,
		boolToInt(req.Enabled),
		req.Timeout,
		time.Now(),
		id,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update schedule: "+err.Error())
		return
	}

	// Get updated schedule
	s.handleGetSchedule(w, r)
}

// handleDeleteSchedule handles DELETE /api/schedules/{id}
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Check if schedule exists
	var exists bool
	err := s.scheduleDB.QueryRow("SELECT EXISTS(SELECT 1 FROM schedules WHERE id = ?)", id).Scan(&exists)
	if err != nil || !exists {
		writeError(w, http.StatusNotFound, "Schedule not found")
		return
	}

	_, err = s.scheduleDB.Exec("DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete schedule: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Schedule deleted successfully"})
}

// handleToggleSchedule handles POST /api/schedules/{id}/toggle
func (s *Server) handleToggleSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Get current enabled status
	var enabled int
	err := s.scheduleDB.QueryRow("SELECT enabled FROM schedules WHERE id = ?", id).Scan(&enabled)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get schedule: "+err.Error())
		return
	}

	// Toggle status
	newEnabled := enabled == 0
	_, err = s.scheduleDB.Exec("UPDATE schedules SET enabled = ?, updated_at = ? WHERE id = ?", boolToInt(newEnabled), time.Now(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to toggle schedule: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ToggleScheduleResponse{
		ID:      id,
		Enabled: newEnabled,
	})
}

// handleRunSchedule handles POST /api/schedules/{id}/run
func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Get schedule details
	var sch Schedule
	var vmNamesJSON string
	err := s.scheduleDB.QueryRow(
		"SELECT id, name, command, vm_names, timeout FROM schedules WHERE id = ?",
		id,
	).Scan(&sch.ID, &sch.Name, &sch.Command, &vmNamesJSON, &sch.Timeout)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get schedule: "+err.Error())
		return
	}

	json.Unmarshal([]byte(vmNamesJSON), &sch.VMNames)

	// Create execution record
	executionID := fmt.Sprintf("exec_%d", time.Now().UnixNano())
	_, err = s.scheduleExecutionDB.Exec(
		"INSERT INTO schedule_executions (id, schedule_id, triggered_at, status) VALUES (?, ?, ?, ?)",
		executionID,
		id,
		time.Now(),
		"running",
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create execution record: "+err.Error())
		return
	}

	// Trigger async execution (in production, this would be handled by a scheduler)
	go s.executeSchedule(sch, executionID)

	writeJSON(w, http.StatusOK, RunScheduleResponse{
		Message:     "Task triggered successfully",
		ExecutionID: executionID,
	})
}

// executeSchedule executes a schedule on target VMs
func (s *Server) executeSchedule(sch Schedule, executionID string) {
	results := make([]ScheduleExecutionResult, 0, len(sch.VMNames))
	overallStatus := "success"

	for _, vmName := range sch.VMNames {
		result := ScheduleExecutionResult{
			VMName: vmName,
			Status: "failed",
		}

		// Get VM info
		vmInfo, err := s.vmManager.GetVM(vmName)
		if err != nil {
			result.Error = "VM not found: " + err.Error()
			overallStatus = "failed"
			results = append(results, result)
			continue
		}

		// Create SSH client
		sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
			Host:     vmInfo.IP,
			Port:     vmInfo.Port,
			Username: vmInfo.Username,
			Password: vmInfo.Password,
			KeyPath:  vmInfo.KeyPath,
		})
		if err != nil {
			result.Error = "Failed to create SSH client: " + err.Error()
			overallStatus = "failed"
			results = append(results, result)
			continue
		}

		// Connect to SSH server
		if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
			result.Error = "Failed to connect: " + err.Error()
			overallStatus = "failed"
			results = append(results, result)
			continue
		}
		defer sshClient.Close()

		// Execute command
		start := time.Now()
		output, err := sshClient.ExecuteCommand(sch.Command)
		duration := time.Since(start).Milliseconds()

		result.Duration = duration
		if err != nil {
			result.Error = err.Error()
			overallStatus = "partial"
		} else {
			result.Status = "success"
			result.Output = output
		}
		results = append(results, result)
	}

	// Update execution record
	resultsJSON, _ := json.Marshal(results)
	_, _ = s.scheduleExecutionDB.Exec(
		"UPDATE schedule_executions SET completed_at = ?, status = ?, results = ? WHERE id = ?",
		time.Now(),
		overallStatus,
		string(resultsJSON),
		executionID,
	)

	// Update schedule last run info
	_, _ = s.scheduleDB.Exec(
		"UPDATE schedules SET last_run_at = ?, last_run_status = ? WHERE id = ?",
		time.Now(),
		overallStatus,
		sch.ID,
	)
}

// handleGetScheduleHistory handles GET /api/schedules/{id}/history
func (s *Server) handleGetScheduleHistory(w http.ResponseWriter, r *http.Request) {
	if s.scheduleExecutionDB == nil {
		writeError(w, http.StatusInternalServerError, "Schedule execution database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Get pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Get total count
	var total int
	err := s.scheduleExecutionDB.QueryRow("SELECT COUNT(*) FROM schedule_executions WHERE schedule_id = ?", id).Scan(&total)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count executions: "+err.Error())
		return
	}

	// Get executions
	rows, err := s.scheduleExecutionDB.Query(
		"SELECT id, schedule_id, triggered_at, completed_at, status, results FROM schedule_executions WHERE schedule_id = ? ORDER BY triggered_at DESC LIMIT ? OFFSET ?",
		id, pageSize, offset,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query executions: "+err.Error())
		return
	}
	defer rows.Close()

	var executions []ScheduleExecution
	for rows.Next() {
		var exec ScheduleExecution
		var resultsJSON string
		var completedAt sql.NullTime
		err := rows.Scan(&exec.ID, &exec.ScheduleID, &exec.TriggeredAt, &completedAt, &exec.Status, &resultsJSON)
		if err != nil {
			continue
		}
		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
		}
		json.Unmarshal([]byte(resultsJSON), &exec.Results)
		executions = append(executions, exec)
	}

	writeJSON(w, http.StatusOK, ScheduleExecutionListResponse{
		Total: total,
		Items: executions,
	})
}

// calculateNextRun calculates the next run time based on cron expression
func calculateNextRun(cronExpr string, lastRun *time.Time) *time.Time {
	if cronExpr == "" {
		return nil
	}

	now := time.Now()
	// Start from the next minute
	t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())

	// Parse cron expression (supports standard 5-field cron: minute hour day month weekday)
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 {
		return nil
	}

	// Try up to 366 days (leap year) worth of minutes
	for i := 0; i < 366*24*60; i++ {
		if matchesCron(t, fields) {
			return &t
		}
		t = t.Add(time.Minute)
	}

	return nil
}

// matchesCron checks if a given time matches the cron expression
func matchesCron(t time.Time, fields []string) bool {
	minute, hour, day, month, weekday := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Check each field
	if !matchesField(minute, t.Minute(), 0, 59) {
		return false
	}
	if !matchesField(hour, t.Hour(), 0, 23) {
		return false
	}
	if !matchesField(day, t.Day(), 1, 31) {
		return false
	}
	if !matchesField(month, int(t.Month()), 1, 12) {
		return false
	}
	if !matchesField(weekday, int(t.Weekday()), 0, 6) {
		return false
	}

	return true
}

// matchesField checks if a value matches a cron field pattern
func matchesField(field string, value int, min, max int) bool {
	// Handle wildcard
	if field == "*" {
		return true
	}

	// Handle list (comma-separated)
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			if matchesField(part, value, min, max) {
				return true
			}
		}
		return false
	}

	// Handle range (dash)
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) == 2 {
			start := parseIntOrWildcard(parts[0], min)
			end := parseIntOrWildcard(parts[1], max)
			return value >= start && value <= end
		}
	}

	// Handle step (slash)
	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) == 2 {
			step := parseIntOrWildcard(parts[1], 1)
			if step == 0 {
				return false
			}
			// Handle */N (every N) or X/N (starting from X)
			if parts[0] == "*" {
				return value%step == 0
			}
			start := parseIntOrWildcard(parts[0], min)
			return value >= start && (value-start)%step == 0
		}
	}

	// Handle specific value or wildcard start
	val := parseIntOrWildcard(field, min)
	return value == val
}

// parseIntOrWildcard parses a number, treating * as the min value
func parseIntOrWildcard(s string, defaultVal int) int {
	if s == "*" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

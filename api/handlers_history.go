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
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// handleListCommandHistory lists command history
func (s *Server) handleListCommandHistory(w http.ResponseWriter, r *http.Request) {
	if s.commandHistoryDB == nil {
		writeError(w, http.StatusInternalServerError, "Command history database not available")
		return
	}

	// Get query parameters
	vmName := r.URL.Query().Get("vm_name")
	search := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	query := "SELECT id, vm_name, command, executed_by, executed_at, status, duration_ms FROM command_history WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM command_history WHERE 1=1"
	var args []interface{}
	var countArgs []interface{}

	if vmName != "" {
		query += " AND vm_name = ?"
		countQuery += " AND vm_name = ?"
		args = append(args, vmName)
		countArgs = append(countArgs, vmName)
	}

	if search != "" {
		query += " AND command LIKE ?"
		countQuery += " AND command LIKE ?"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		countArgs = append(countArgs, searchPattern)
	}

	// Get total count
	var total int
	err := s.commandHistoryDB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = 0
	}

	query += " ORDER BY executed_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.commandHistoryDB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query history: "+err.Error())
		return
	}
	defer rows.Close()

	var items []CommandHistory
	for rows.Next() {
		var h CommandHistory
		err := rows.Scan(
			&h.ID,
			&h.VMName,
			&h.Command,
			&h.ExecutedBy,
			&h.ExecutedAt,
			&h.Status,
			&h.DurationMs,
		)
		if err != nil {
			continue
		}
		items = append(items, h)
	}

	response := struct {
		Total    int              `json:"total"`
		Page     int              `json:"page"`
		PageSize int              `json:"page_size"`
		Items    []CommandHistory `json:"items"`
	}{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleRecordCommandHistory records command execution history
func (s *Server) handleRecordCommandHistory(w http.ResponseWriter, r *http.Request) {
	if s.commandHistoryDB == nil {
		writeError(w, http.StatusInternalServerError, "Command history database not available")
		return
	}

	var req struct {
		VMName     string `json:"vm_name"`
		Command    string `json:"command"`
		Status     string `json:"status"`
		DurationMs int64  `json:"duration_ms"`
		Output     string `json:"output"`
		Error      string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.VMName == "" || req.Command == "" {
		writeError(w, http.StatusBadRequest, "vm_name and command are required")
		return
	}

	history := CommandHistory{
		ID:         fmt.Sprintf("hist_%d", time.Now().UnixNano()),
		VMName:     req.VMName,
		Command:    req.Command,
		ExecutedBy: s.getCurrentUser(r),
		ExecutedAt: time.Now(),
		Status:     req.Status,
		DurationMs: req.DurationMs,
		Output:     req.Output,
		Error:      req.Error,
	}

	_, err := s.commandHistoryDB.Exec(
		"INSERT INTO command_history (id, vm_name, command, executed_by, executed_at, status, duration_ms, output, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		history.ID,
		history.VMName,
		history.Command,
		history.ExecutedBy,
		history.ExecutedAt,
		history.Status,
		history.DurationMs,
		history.Output,
		history.Error,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to record history: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, history)
}

// handleDeleteCommandHistory deletes a command history entry
func (s *Server) handleDeleteCommandHistory(w http.ResponseWriter, r *http.Request) {
	if s.commandHistoryDB == nil {
		writeError(w, http.StatusInternalServerError, "Command history database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	_, err := s.commandHistoryDB.Exec("DELETE FROM command_history WHERE id = ?", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete history: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "History deleted successfully"})
}

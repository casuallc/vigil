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
	"time"

	"github.com/gorilla/mux"
)

// handleListCommandTemplates lists command templates
func (s *Server) handleListCommandTemplates(w http.ResponseWriter, r *http.Request) {
	if s.commandTemplateDB == nil {
		writeError(w, http.StatusInternalServerError, "Command template database not available")
		return
	}

	username := s.getCurrentUser(r)
	isAdmin := s.isAdmin(r)

	// Get query parameters
	category := r.URL.Query().Get("category")
	sharedOnly := r.URL.Query().Get("shared") == "true"
	search := r.URL.Query().Get("search")

	query := "SELECT id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at FROM command_templates WHERE 1=1"
	var args []interface{}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	if sharedOnly {
		query += " AND is_shared = 1"
	} else {
		// Non-admin can only see shared or own templates
		if !isAdmin {
			query += " AND (is_shared = 1 OR created_by = ?)"
			args = append(args, username)
		}
	}

	if search != "" {
		query += " AND (name LIKE ? OR description LIKE ? OR command LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := s.commandTemplateDB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query templates: "+err.Error())
		return
	}
	defer rows.Close()

	var templates []CommandTemplate
	for rows.Next() {
		var tpl CommandTemplate
		var variablesJSON string
		var isShared int
		err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tpl.Description,
			&tpl.Command,
			&variablesJSON,
			&tpl.Category,
			&isShared,
			&tpl.CreatedBy,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		)
		if err != nil {
			continue
		}
		tpl.IsShared = isShared == 1
		json.Unmarshal([]byte(variablesJSON), &tpl.Variables)
		templates = append(templates, tpl)
	}

	writeJSON(w, http.StatusOK, templates)
}

// handleCreateCommandTemplate creates a command template
func (s *Server) handleCreateCommandTemplate(w http.ResponseWriter, r *http.Request) {
	if s.commandTemplateDB == nil {
		writeError(w, http.StatusInternalServerError, "Command template database not available")
		return
	}

	var req CommandTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" || req.Command == "" {
		writeError(w, http.StatusBadRequest, "name and command are required")
		return
	}

	req.ID = fmt.Sprintf("tpl_%d", time.Now().UnixNano())
	req.CreatedBy = s.getCurrentUser(r)
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	variablesJSON, _ := json.Marshal(req.Variables)

	_, err := s.commandTemplateDB.Exec(
		"INSERT INTO command_templates (id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		req.ID,
		req.Name,
		req.Description,
		req.Command,
		string(variablesJSON),
		req.Category,
		boolToInt(req.IsShared),
		req.CreatedBy,
		req.CreatedAt,
		req.UpdatedAt,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create template: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, req)
}

// handleGetCommandTemplate gets a command template
func (s *Server) handleGetCommandTemplate(w http.ResponseWriter, r *http.Request) {
	if s.commandTemplateDB == nil {
		writeError(w, http.StatusInternalServerError, "Command template database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var tpl CommandTemplate
	var variablesJSON string
	var isShared int

	err := s.commandTemplateDB.QueryRow(
		"SELECT id, name, description, command, variables, category, is_shared, created_by, created_at, updated_at FROM command_templates WHERE id = ?",
		id,
	).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Description,
		&tpl.Command,
		&variablesJSON,
		&tpl.Category,
		&isShared,
		&tpl.CreatedBy,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get template: "+err.Error())
		return
	}

	tpl.IsShared = isShared == 1
	json.Unmarshal([]byte(variablesJSON), &tpl.Variables)

	writeJSON(w, http.StatusOK, tpl)
}

// handleUpdateCommandTemplate updates a command template
func (s *Server) handleUpdateCommandTemplate(w http.ResponseWriter, r *http.Request) {
	if s.commandTemplateDB == nil {
		writeError(w, http.StatusInternalServerError, "Command template database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var req CommandTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Check permission
	var createdBy string
	err := s.commandTemplateDB.QueryRow("SELECT created_by FROM command_templates WHERE id = ?", id).Scan(&createdBy)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}

	username := s.getCurrentUser(r)
	if createdBy != username && !s.isAdmin(r) {
		writeError(w, http.StatusForbidden, "Permission denied")
		return
	}

	variablesJSON, _ := json.Marshal(req.Variables)

	_, err = s.commandTemplateDB.Exec(
		"UPDATE command_templates SET name = ?, description = ?, command = ?, variables = ?, category = ?, is_shared = ?, updated_at = ? WHERE id = ?",
		req.Name,
		req.Description,
		req.Command,
		string(variablesJSON),
		req.Category,
		boolToInt(req.IsShared),
		time.Now(),
		id,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update template: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Template updated successfully"})
}

// handleDeleteCommandTemplate deletes a command template
func (s *Server) handleDeleteCommandTemplate(w http.ResponseWriter, r *http.Request) {
	if s.commandTemplateDB == nil {
		writeError(w, http.StatusInternalServerError, "Command template database not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Check permission
	var createdBy string
	err := s.commandTemplateDB.QueryRow("SELECT created_by FROM command_templates WHERE id = ?", id).Scan(&createdBy)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}

	username := s.getCurrentUser(r)
	if createdBy != username && !s.isAdmin(r) {
		writeError(w, http.StatusForbidden, "Permission denied")
		return
	}

	_, err = s.commandTemplateDB.Exec("DELETE FROM command_templates WHERE id = ?", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete template: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Template deleted successfully"})
}

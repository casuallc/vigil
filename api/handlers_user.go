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
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/casuallc/vigil/models"
	"github.com/gorilla/mux"
)

// handleRegisterUser handles user registration
func (s *Server) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	// Only allow registration if user database exists
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	// Parse request body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if user.Username == "" || user.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Check if user already exists
	if _, exists := s.userDatabase.GetUser(user.Username); exists {
		writeError(w, http.StatusConflict, "User already exists")
		return
	}

	// Set default role
	if user.Role == "" {
		user.Role = "user"
	}

	// Generate unique ID
	user.ID = fmt.Sprintf("usr_%d", time.Now().Unix())

	// Create user
	if err := s.userDatabase.CreateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	// Return success response (without password)
	responseUser := user
	responseUser.Password = ""
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user":    responseUser,
	})
}

// handleUserLogin handles user login
func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	// Check if user database is available
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	// Parse request body
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if loginReq.Username == "" || loginReq.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Get client IP
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}

	// Get User-Agent and device info
	userAgent := r.Header.Get("User-Agent")
	deviceInfo := ""
	if strings.Contains(userAgent, "Mobile") {
		deviceInfo = "mobile"
	} else if strings.Contains(userAgent, "Tablet") {
		deviceInfo = "tablet"
	} else {
		deviceInfo = "desktop"
	}

	// Validate password
	isValid, err := s.userDatabase.ValidatePassword(loginReq.Username, loginReq.Password)
	if err != nil {
		log.Printf("Error validating password for user %s: %v", loginReq.Username, err)
		writeError(w, http.StatusInternalServerError, "Failed to validate credentials")
		return
	}

	if !isValid {
		// Log failed login attempt
		if s.loginLogDatabase != nil {
			user, exists := s.userDatabase.GetUser(loginReq.Username)
			userID := ""
			if exists {
				userID = user.ID
			}
			s.loginLogDatabase.LogLogin(loginReq.Username, userID, clientIP, userAgent, deviceInfo, "failed")
		}
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Get user info
	user, exists := s.userDatabase.GetUser(loginReq.Username)
	if !exists {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Update login status
	if s.userDatabase != nil {
		if err := s.userDatabase.UpdateLoginStatus(loginReq.Username, clientIP); err != nil {
			log.Printf("Error updating login status for user %s: %v", loginReq.Username, err)
		}
	}

	// Log successful login
	if s.loginLogDatabase != nil {
		if err := s.loginLogDatabase.LogLogin(loginReq.Username, user.ID, clientIP, userAgent, deviceInfo, "success"); err != nil {
			log.Printf("Error logging login for user %s: %v", loginReq.Username, err)
		}
	}

	// Return success response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"avatar":     user.Avatar,
			"nickname":   user.Nickname,
			"region":     user.Region,
			"configs":    user.Configs,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// handleListUsers handles listing all users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		username, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		// Check if super admin or registered admin
		isSuperAdmin := username == s.config.BasicAuth.Username
		isAdmin := false

		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(username); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		// Only super admin or admin can list all users
		if !isSuperAdmin && !isAdmin {
			writeError(w, http.StatusForbidden, "Access denied: Admin role required to list users")
			return
		}
	}

	// Get all users
	users := s.userDatabase.GetAllUsers()

	// Create response without passwords
	responseUsers := make([]map[string]interface{}, len(users))
	for i, user := range users {
		responseUsers[i] = map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"avatar":     user.Avatar,
			"nickname":   user.Nickname,
			"region":     user.Region,
			"configs":    user.Configs,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, responseUsers)
}

// handleGetUser handles getting user details
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	vars := mux.Vars(r)
	targetUsername := vars["username"]

	if targetUsername == "" {
		writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		requestingUsername, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
		isAdmin := false
		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		// Allow access if super admin, admin, or user requesting own info
		if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
			writeError(w, http.StatusForbidden, "Access denied: Cannot access other user's information")
			return
		}
	}

	// Get user
	user, exists := s.userDatabase.GetUser(targetUsername)
	if !exists {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Create response without password
	responseUser := map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"avatar":     user.Avatar,
		"nickname":   user.Nickname,
		"region":     user.Region,
		"configs":    user.Configs,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	writeJSON(w, http.StatusOK, responseUser)
}

// handleUpdateUser handles updating user information
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	vars := mux.Vars(r)
	targetUsername := vars["username"]

	if targetUsername == "" {
		writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Parse request body
	var updateData struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		Password string `json:"password"`
		Avatar   string `json:"avatar"`
		Nickname string `json:"nickname"`
		Region   string `json:"region"`
		Configs  string `json:"configs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		requestingUsername, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
		isAdmin := false
		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		canUpdate := isSuperAdmin || isAdmin || requestingUsername == targetUsername
		if !canUpdate {
			writeError(w, http.StatusForbidden, "Access denied: Cannot update other user's information")
			return
		}

		// Prevent non-admin users from changing roles
		if !isSuperAdmin && !isAdmin && updateData.Role != "" {
			writeError(w, http.StatusForbidden, "Access denied: Regular users cannot change roles")
			return
		}
	}

	// Prepare updated user data
	updatedUser := &models.User{
		Username: targetUsername,
		Email:    updateData.Email,
		Role:     updateData.Role,
		Password: updateData.Password,
		Avatar:   updateData.Avatar,
		Nickname: updateData.Nickname,
		Region:   updateData.Region,
		Configs:  updateData.Configs,
	}

	// Update user
	if err := s.userDatabase.UpdateUser(targetUsername, updatedUser); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

// handleDeleteUser handles deleting a user
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	vars := mux.Vars(r)
	targetUsername := vars["username"]

	if targetUsername == "" {
		writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		requestingUsername, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
		isAdmin := false
		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		// Only super admin or admin can delete users
		if !isSuperAdmin && !isAdmin {
			writeError(w, http.StatusForbidden, "Access denied: Admin role required to delete users")
			return
		}

		// Prevent deletion of super admin
		if targetUsername == s.config.BasicAuth.Username {
			writeError(w, http.StatusForbidden, "Access denied: Cannot delete super admin user")
			return
		}
	}

	// Delete user
	if err := s.userDatabase.DeleteUser(targetUsername); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// handleGetUserConfigs handles getting user configs
func (s *Server) handleGetUserConfigs(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	vars := mux.Vars(r)
	targetUsername := vars["username"]

	if targetUsername == "" {
		writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		requestingUsername, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
		isAdmin := false
		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
			writeError(w, http.StatusForbidden, "Access denied: Cannot access other user's configuration")
			return
		}
	}

	// Get user
	user, exists := s.userDatabase.GetUser(targetUsername)
	if !exists {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"configs": user.Configs,
	})
}

// handleUpdateUserConfigs handles updating user configs
func (s *Server) handleUpdateUserConfigs(w http.ResponseWriter, r *http.Request) {
	if s.userDatabase == nil {
		writeError(w, http.StatusInternalServerError, "User database not available")
		return
	}

	vars := mux.Vars(r)
	targetUsername := vars["username"]

	if targetUsername == "" {
		writeError(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Parse request body
	var req struct {
		Configs string `json:"configs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Check authentication
	if s.config.BasicAuth.Enabled {
		requestingUsername, _, ok := r.BasicAuth()
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		isSuperAdmin := requestingUsername == s.config.BasicAuth.Username
		isAdmin := false
		if !isSuperAdmin && s.userDatabase != nil {
			if user, exists := s.userDatabase.GetUser(requestingUsername); exists && user.Role == "admin" {
				isAdmin = true
			}
		}

		if !isSuperAdmin && !isAdmin && requestingUsername != targetUsername {
			writeError(w, http.StatusForbidden, "Access denied: Cannot update other user's configuration")
			return
		}
	}

	// Check user existence
	_, exists := s.userDatabase.GetUser(targetUsername)
	if !exists {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Update configs
	updatedUser := &models.User{
		Username: targetUsername,
		Configs:  req.Configs,
	}

	if err := s.userDatabase.UpdateUser(targetUsername, updatedUser); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update user configs: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User configs updated successfully"})
}

package api

import (
	"encoding/json"
	"fmt"
	"github.com/casuallc/vigil/config"
	"github.com/casuallc/vigil/monitor"
	"github.com/casuallc/vigil/process"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Server represents the HTTP API server

type Server struct {
	config         *config.Config
	processManager *process.Manager
	monitor        *monitor.Monitor
}

// NewServer creates a new API server
func NewServer(config *config.Config) *Server {
	processManager := process.NewManager()
	monitor := monitor.NewMonitor(processManager)

	return &Server{
		config:         config,
		processManager: processManager,
		monitor:        monitor,
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	r := mux.NewRouter()

	// Process management endpoints
	r.HandleFunc("/api/processes/scan", s.handleScanProcesses).Methods("GET")
	r.HandleFunc("/api/processes/manage", s.handleManageProcess).Methods("POST")
	r.HandleFunc("/api/processes/{name}/start", s.handleStartProcess).Methods("POST")
	r.HandleFunc("/api/processes/{name}/stop", s.handleStopProcess).Methods("POST")
	r.HandleFunc("/api/processes/{name}/restart", s.handleRestartProcess).Methods("POST")
	r.HandleFunc("/api/processes/{name}", s.handleGetProcess).Methods("GET")
	r.HandleFunc("/api/processes", s.handleListProcesses).Methods("GET")
	
	// Resource monitoring endpoints
	r.HandleFunc("/api/resources/system", s.handleGetSystemResources).Methods("GET")
	r.HandleFunc("/api/resources/process/{pid}", s.handleGetProcessResources).Methods("GET")

	// Configuration endpoints
	r.HandleFunc("/api/config", s.handleGetConfig).Methods("GET")
	r.HandleFunc("/api/config", s.handleUpdateConfig).Methods("PUT")

	// Health check
	r.HandleFunc("/health", s.handleHealthCheck).Methods("GET")

	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, r)
}

// API response helpers
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

// Handlers
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleScanProcesses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	processes, err := s.processManager.ScanProcesses(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, processes)
}

func (s *Server) handleManageProcess(w http.ResponseWriter, r *http.Request) {
	var process process.ManagedProcess
	if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.processManager.ManageProcess(process); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Process managed successfully"})
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := s.processManager.StartProcess(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process started successfully"})
}

func (s *Server) handleStopProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := s.processManager.StopProcess(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process stopped successfully"})
}

func (s *Server) handleRestartProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := s.processManager.RestartProcess(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Process restarted successfully"})
}

func (s *Server) handleGetProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	process, err := s.processManager.GetProcessStatus(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, process)
}

func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	processes, err := s.processManager.ListManagedProcesses()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, processes)
}

func (s *Server) handleGetSystemResources(w http.ResponseWriter, r *http.Request) {
	resources, err := monitor.GetSystemResourceUsage()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetProcessResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid PID")
		return
	}

	resources, err := monitor.GetProcessResourceUsage(pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save the new configuration
	if err := config.SaveConfig("config.yaml", &newConfig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update the in-memory configuration
	s.config = &newConfig

	writeJSON(w, http.StatusOK, map[string]string{"message": "Config updated successfully"})
}
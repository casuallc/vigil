package api

import (
  "github.com/gorilla/mux"
)

// Router 定义API路由注册函数
func (s *Server) Router() *mux.Router {
  r := mux.NewRouter()

  // Process management endpoints
  r.HandleFunc("/api/namespaces/{namespace}/processes/scan", s.handleScanProcesses).Methods("GET")
  r.HandleFunc("/api/namespaces/{namespace}/processes/manage", s.handleManageProcess).Methods("POST")
  r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/start", s.handleStartProcess).Methods("POST")
  r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/stop", s.handleStopProcess).Methods("POST")
  r.HandleFunc("/api/namespaces/{namespace}/processes/{name}/restart", s.handleRestartProcess).Methods("POST")
  r.HandleFunc("/api/namespaces/{namespace}/processes/{name}", s.handleGetProcess).Methods("GET")
  r.HandleFunc("/api/namespaces/{namespace}/processes/{name}", s.handleDeleteProcess).Methods("DELETE")
  r.HandleFunc("/api/namespaces/{namespace}/processes", s.handleListProcesses).Methods("GET")
  
  // 保留原有的无namespace路由作为兼容
  r.HandleFunc("/api/processes/scan", s.handleScanProcesses).Methods("GET")
  r.HandleFunc("/api/processes/manage", s.handleManageProcess).Methods("POST")
  r.HandleFunc("/api/processes/{name}/start", s.handleStartProcess).Methods("POST")
  r.HandleFunc("/api/processes/{name}/stop", s.handleStopProcess).Methods("POST")
  r.HandleFunc("/api/processes/{name}/restart", s.handleRestartProcess).Methods("POST")
  r.HandleFunc("/api/processes/{name}", s.handleGetProcess).Methods("GET")
  r.HandleFunc("/api/processes/{name}", s.handleDeleteProcess).Methods("DELETE")
  r.HandleFunc("/api/processes", s.handleListProcesses).Methods("GET")

  // Resource monitoring endpoints
  r.HandleFunc("/api/resources/system", s.handleGetSystemResources).Methods("GET")
  r.HandleFunc("/api/resources/process/{pid}", s.handleGetProcessResources).Methods("GET")

  // Configuration endpoints
  r.HandleFunc("/api/config", s.handleGetConfig).Methods("GET")
  r.HandleFunc("/api/config", s.handleUpdateConfig).Methods("PUT")

  // Health check
  r.HandleFunc("/health", s.handleHealthCheck).Methods("GET")

  return r
}

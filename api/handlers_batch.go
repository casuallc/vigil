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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/casuallc/vigil/vm"
	"github.com/gorilla/mux"
)

// handleBatchExec handles batch command execution
func (s *Server) handleBatchExec(w http.ResponseWriter, r *http.Request) {
	var req BatchExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if len(req.VMNames) == 0 {
		writeError(w, http.StatusBadRequest, "vm_names is required")
		return
	}
	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	// Generate task ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	// Get current user
	username := s.getCurrentUser(r)

	// Result channel
	resultChan := make(chan BatchExecResult, len(req.VMNames))

	// Execute function
	executeOnVM := func(vmName string) {
		start := time.Now()
		result := BatchExecResult{
			VMName: vmName,
			Status: "failed",
		}

		// Get VM info
		vmInfo, err := s.vmManager.GetVM(vmName)
		if err != nil {
			result.Error = "VM not found: " + err.Error()
			result.DurationMs = time.Since(start).Milliseconds()
			resultChan <- result
			return
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
			result.DurationMs = time.Since(start).Milliseconds()
			resultChan <- result
			return
		}

		// Connect to SSH server
		if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
			result.Error = "Failed to connect: " + err.Error()
			result.DurationMs = time.Since(start).Milliseconds()
			resultChan <- result
			return
		}
		defer sshClient.Close()

		// Execute with timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
		defer cancel()

		done := make(chan struct {
			output string
			err    error
		}, 1)

		go func() {
			output, err := sshClient.ExecuteCommand(req.Command)
			done <- struct {
				output string
				err    error
			}{output, err}
		}()

		select {
		case res := <-done:
			result.DurationMs = time.Since(start).Milliseconds()
			if res.err != nil {
				result.Error = res.err.Error()
			} else {
				result.Status = "success"
				result.Output = res.output
			}
		case <-ctx.Done():
			result.DurationMs = time.Since(start).Milliseconds()
			result.Error = "Command execution timeout"
		}

		resultChan <- result

		// Record command history
		if s.commandHistoryDB != nil {
			status := "success"
			if result.Status != "success" {
				status = "failed"
			}
			s.recordCommandHistory(vmName, req.Command, username, status, result.DurationMs, result.Output, result.Error)
		}
	}

	// Execute commands
	if req.Parallel {
		// Parallel execution with concurrency limit
		semaphore := make(chan struct{}, 10)
		for _, vmName := range req.VMNames {
			go func(name string) {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				executeOnVM(name)
			}(vmName)
		}
	} else {
		// Sequential execution
		go func() {
			for _, vmName := range req.VMNames {
				executeOnVM(vmName)
			}
		}()
	}

	// Collect results
	var results []BatchExecResult
	successCount := 0
	failedCount := 0

	for i := 0; i < len(req.VMNames); i++ {
		result := <-resultChan
		results = append(results, result)
		if result.Status == "success" {
			successCount++
		} else {
			failedCount++
		}
	}
	close(resultChan)

	response := BatchExecResponse{
		TaskID:  taskID,
		Total:   len(req.VMNames),
		Success: successCount,
		Failed:  failedCount,
		Results: results,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetVMResources gets single VM resource monitoring
func (s *Server) handleGetVMResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["name"]
	if vmName == "" {
		writeError(w, http.StatusBadRequest, "VM name is required")
		return
	}

	// Get VM info
	vmInfo, err := s.vmManager.GetVM(vmName)
	if err != nil {
		writeError(w, http.StatusNotFound, "VM not found: "+err.Error())
		return
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
		writeError(w, http.StatusInternalServerError, "Failed to create SSH client: "+err.Error())
		return
	}

	// Connect to SSH server
	if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to VM: "+err.Error())
		return
	}
	defer sshClient.Close()

	// Collect resource info
	resourceInfo := &VMResourceInfo{
		VMName:      vmName,
		CollectedAt: time.Now(),
	}

	// Get CPU usage
	cpuOutput, err := sshClient.ExecuteCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
	if err == nil && cpuOutput != "" {
		cpuOutput = strings.TrimSpace(cpuOutput)
		if cpu, err := strconv.ParseFloat(cpuOutput, 64); err == nil {
			resourceInfo.CPUUsage = cpu
		}
	}

	// Get memory info
	memOutput, err := sshClient.ExecuteCommand("free -m | awk 'NR==2{printf \"%.2f %.2f %.2f\", $2/1024,$3/1024,$3/$2*100}'")
	if err == nil {
		parts := strings.Fields(memOutput)
		if len(parts) >= 3 {
			if total, err := strconv.ParseFloat(parts[0], 64); err == nil {
				resourceInfo.MemoryTotalGB = total
			}
			if used, err := strconv.ParseFloat(parts[1], 64); err == nil {
				resourceInfo.MemoryUsedGB = used
			}
			if usage, err := strconv.ParseFloat(parts[2], 64); err == nil {
				resourceInfo.MemoryUsage = usage
			}
		}
	}

	// Get disk info
	diskOutput, err := sshClient.ExecuteCommand("df -h / | awk 'NR==2{print $2,$3,$5}' | sed 's/G//g' | sed 's/%//g'")
	if err == nil {
		parts := strings.Fields(diskOutput)
		if len(parts) >= 3 {
			if total, err := strconv.ParseFloat(parts[0], 64); err == nil {
				resourceInfo.DiskTotalGB = total
			}
			if used, err := strconv.ParseFloat(parts[1], 64); err == nil {
				resourceInfo.DiskUsedGB = used
			}
			if usage, err := strconv.ParseFloat(parts[2], 64); err == nil {
				resourceInfo.DiskUsage = usage
			}
		}
	}

	// Get load average
	loadOutput, err := sshClient.ExecuteCommand("uptime | awk -F'load average:' '{print $2}' | tr -d ','")
	if err == nil {
		parts := strings.Fields(loadOutput)
		for _, part := range parts {
			if load, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
				resourceInfo.LoadAverage = append(resourceInfo.LoadAverage, load)
			}
		}
	}

	// Get uptime
	uptimeOutput, err := sshClient.ExecuteCommand("uptime -p")
	if err == nil {
		resourceInfo.Uptime = strings.TrimSpace(uptimeOutput)
	}

	// Get network info (requires two samples)
	netOutput1, _ := sshClient.ExecuteCommand("cat /proc/net/dev | grep eth0 | awk '{print $2,$10}'")
	time.Sleep(1 * time.Second)
	netOutput2, _ := sshClient.ExecuteCommand("cat /proc/net/dev | grep eth0 | awk '{print $2,$10}'")

	if netOutput1 != "" && netOutput2 != "" {
		parts1 := strings.Fields(netOutput1)
		parts2 := strings.Fields(netOutput2)
		if len(parts1) >= 2 && len(parts2) >= 2 {
			if rx1, err := strconv.ParseInt(parts1[0], 10, 64); err == nil {
				if rx2, err := strconv.ParseInt(parts2[0], 10, 64); err == nil {
					resourceInfo.Network.RXBytesPerSec = rx2 - rx1
				}
			}
			if tx1, err := strconv.ParseInt(parts1[1], 10, 64); err == nil {
				if tx2, err := strconv.ParseInt(parts2[1], 10, 64); err == nil {
					resourceInfo.Network.TXBytesPerSec = tx2 - tx1
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, resourceInfo)
}

// handleBatchGetVMResources gets resources for multiple VMs
func (s *Server) handleBatchGetVMResources(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VMNames []string `json:"vm_names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(req.VMNames) == 0 {
		writeError(w, http.StatusBadRequest, "vm_names is required")
		return
	}

	// Result channel
	resultChan := make(chan VMResourceInfo, len(req.VMNames))

	// Concurrency limit
	semaphore := make(chan struct{}, 10)

	for _, vmName := range req.VMNames {
		go func(name string) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			info := VMResourceInfo{
				VMName: name,
				Status: "error",
			}

			vmInfo, err := s.vmManager.GetVM(name)
			if err != nil {
				info.Error = "VM not found"
				resultChan <- info
				return
			}

			sshClient, err := vm.NewSSHClient(&vm.SSHConfig{
				Host:     vmInfo.IP,
				Port:     vmInfo.Port,
				Username: vmInfo.Username,
				Password: vmInfo.Password,
				KeyPath:  vmInfo.KeyPath,
			})
			if err != nil {
				info.Error = "Failed to create SSH client"
				resultChan <- info
				return
			}

			if err := sshClient.Connect(vmInfo.IP, vmInfo.Port); err != nil {
				info.Error = "Failed to connect"
				resultChan <- info
				return
			}
			defer sshClient.Close()

			info.CollectedAt = time.Now()

			// Get CPU usage
			cpuOutput, _ := sshClient.ExecuteCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
			if cpuOutput != "" {
				if cpu, err := strconv.ParseFloat(strings.TrimSpace(cpuOutput), 64); err == nil {
					info.CPUUsage = cpu
				}
			}

			// Get memory usage
			memOutput, _ := sshClient.ExecuteCommand("free | awk 'NR==2{printf \"%.0f\", $3/$2*100}'")
			if memOutput != "" {
				if mem, err := strconv.ParseFloat(strings.TrimSpace(memOutput), 64); err == nil {
					info.MemoryUsage = mem
				}
			}

			// Get disk usage
			diskOutput, _ := sshClient.ExecuteCommand("df -h / | awk 'NR==2{print $5}' | sed 's/%//g'")
			if diskOutput != "" {
				if disk, err := strconv.ParseFloat(strings.TrimSpace(diskOutput), 64); err == nil {
					info.DiskUsage = disk
				}
			}

			// Determine status
			if info.CPUUsage > 80 || info.MemoryUsage > 80 || info.DiskUsage > 85 {
				info.Status = "warning"
			} else {
				info.Status = "ok"
			}

			resultChan <- info
		}(vmName)
	}

	// Collect results
	var results []VMResourceInfo
	for i := 0; i < len(req.VMNames); i++ {
		results = append(results, <-resultChan)
	}
	close(resultChan)

	writeJSON(w, http.StatusOK, results)
}

// handleVMFileTransfer handles cross-VM file transfer
func (s *Server) handleVMFileTransfer(w http.ResponseWriter, r *http.Request) {
	var req FileTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.SourceVM == "" || req.SourcePath == "" || req.TargetVM == "" || req.TargetPath == "" {
		writeError(w, http.StatusBadRequest, "source_vm, source_path, target_vm, and target_path are required")
		return
	}

	start := time.Now()

	// Get source VM info
	sourceVM, err := s.vmManager.GetVM(req.SourceVM)
	if err != nil {
		writeError(w, http.StatusNotFound, "Source VM not found: "+err.Error())
		return
	}

	// Get target VM info
	targetVM, err := s.vmManager.GetVM(req.TargetVM)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target VM not found: "+err.Error())
		return
	}

	// Create source SSH client
	sourceSSH, err := vm.NewSSHClient(&vm.SSHConfig{
		Host:     sourceVM.IP,
		Port:     sourceVM.Port,
		Username: sourceVM.Username,
		Password: sourceVM.Password,
		KeyPath:  sourceVM.KeyPath,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create source SSH client: "+err.Error())
		return
	}

	if err := sourceSSH.Connect(sourceVM.IP, sourceVM.Port); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to source VM: "+err.Error())
		return
	}
	defer sourceSSH.Close()

	// Create target SSH client
	targetSSH, err := vm.NewSSHClient(&vm.SSHConfig{
		Host:     targetVM.IP,
		Port:     targetVM.Port,
		Username: targetVM.Username,
		Password: targetVM.Password,
		KeyPath:  targetVM.KeyPath,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create target SSH client: "+err.Error())
		return
	}

	if err := targetSSH.Connect(targetVM.IP, targetVM.Port); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to connect to target VM: "+err.Error())
		return
	}
	defer targetSSH.Close()

	// Check if source file exists
	checkOutput, err := sourceSSH.ExecuteCommand(fmt.Sprintf("test -f %s && echo 'exists' || echo 'not found'", req.SourcePath))
	if err != nil || strings.TrimSpace(checkOutput) != "exists" {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error":       "Source file not found",
			"source_vm":   req.SourceVM,
			"source_path": req.SourcePath,
		})
		return
	}

	// Get file size
	sizeOutput, _ := sourceSSH.ExecuteCommand(fmt.Sprintf("stat -c %%s %s", req.SourcePath))
	fileSize := int64(0)
	if sizeOutput != "" {
		fileSize, _ = strconv.ParseInt(strings.TrimSpace(sizeOutput), 10, 64)
	}

	// Download file content
	fileContent, err := sourceSSH.DownloadFile(req.SourcePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to download file from source: "+err.Error())
		return
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(req.TargetPath)
	_, _ = targetSSH.ExecuteCommand(fmt.Sprintf("mkdir -p %s", targetDir))

	// Upload file to target
	err = targetSSH.UploadFileContent(fileContent, req.TargetPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to upload file to target: "+err.Error())
		return
	}

	duration := time.Since(start)

	response := FileTransferResponse{
		Message:          "File transferred successfully",
		BytesTransferred: int64(len(fileContent)),
		DurationMs:       duration.Milliseconds(),
	}

	if fileSize > 0 {
		response.BytesTransferred = fileSize
	}

	writeJSON(w, http.StatusOK, response)
}

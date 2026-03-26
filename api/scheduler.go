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
	"log"
	"sync"
	"time"

	"github.com/casuallc/vigil/vm"
)

// Scheduler handles automatic execution of scheduled tasks
type Scheduler struct {
	scheduleDB          *sql.DB
	scheduleExecutionDB *sql.DB
	vmManager           *vm.Manager
	stopChan            chan struct{}
	wg                  sync.WaitGroup
	interval            time.Duration
	mu                  sync.RWMutex
	running             bool
}

// NewScheduler creates a new scheduler
func NewScheduler(scheduleDB, scheduleExecutionDB *sql.DB, vmManager *vm.Manager) *Scheduler {
	return &Scheduler{
		scheduleDB:          scheduleDB,
		scheduleExecutionDB: scheduleExecutionDB,
		vmManager:           vmManager,
		interval:            time.Minute, // Check every minute
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()

	log.Printf("Scheduler started, checking tasks every %v", s.interval)
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
	log.Printf("Scheduler stopped")
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndExecuteTasks()
		case <-s.stopChan:
			return
		}
	}
}

// checkAndExecuteTasks checks for due tasks and executes them
func (s *Scheduler) checkAndExecuteTasks() {
	tasks, err := s.getDueTasks()
	if err != nil {
		log.Printf("Scheduler error: failed to get due tasks: %v", err)
		return
	}

	for _, task := range tasks {
		go s.executeScheduledTask(task)
	}
}

// ScheduleTask represents a task to be executed
type ScheduleTask struct {
	ID        string
	Name      string
	Command   string
	VMNames   []string
	Timeout   int
	Cron      string
	Enabled   bool
	LastRunAt *time.Time
}

// getDueTasks returns all enabled tasks that are due for execution
func (s *Scheduler) getDueTasks() ([]ScheduleTask, error) {
	query := `SELECT id, name, command, vm_names, cron, timeout, last_run_at
			  FROM schedules
			  WHERE enabled = 1`

	rows, err := s.scheduleDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ScheduleTask
	now := time.Now()

	for rows.Next() {
		var task ScheduleTask
		var vmNamesJSON string
		var lastRunAt sql.NullTime

		err := rows.Scan(&task.ID, &task.Name, &task.Command, &vmNamesJSON, &task.Cron, &task.Timeout, &lastRunAt)
		if err != nil {
			log.Printf("Scheduler: failed to scan task: %v", err)
			continue
		}

		// Parse VM names
		if err := json.Unmarshal([]byte(vmNamesJSON), &task.VMNames); err != nil {
			log.Printf("Scheduler: failed to parse VM names for task %s: %v", task.ID, err)
			continue
		}

		// Check if task is due
		if s.isTaskDue(task.Cron, lastRunAt, now) {
			tasks = append(tasks, task)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// isTaskDue checks if a task should run based on its cron expression and last run time
func (s *Scheduler) isTaskDue(cronExpr string, lastRun sql.NullTime, now time.Time) bool {
	// Calculate when the task should have run
	nextRun := calculateNextRun(cronExpr, nil)
	if nextRun == nil {
		return false
	}

	// If there's a last run time, use it to calculate next run
	if lastRun.Valid {
		nextRun = calculateNextRun(cronExpr, &lastRun.Time)
		if nextRun == nil {
			return false
		}
	}

	// Task is due if the next run time has passed
	return now.After(*nextRun) || now.Equal(*nextRun)
}

// executeScheduledTask executes a single scheduled task
func (s *Scheduler) executeScheduledTask(task ScheduleTask) {
	log.Printf("Scheduler: executing task %s (%s)", task.Name, task.ID)

	// Create execution record
	executionID := generateExecutionID()
	_, err := s.scheduleExecutionDB.Exec(
		"INSERT INTO schedule_executions (id, schedule_id, triggered_at, status, trigger_type) VALUES (?, ?, ?, ?, ?)",
		executionID,
		task.ID,
		time.Now(),
		"running",
		"auto", // auto = scheduled, manual = triggered by user
	)
	if err != nil {
		log.Printf("Scheduler: failed to create execution record for task %s: %v", task.ID, err)
		return
	}

	// Execute the task (similar to handleRunSchedule)
	s.executeScheduleWithID(task, executionID)
}

// executeScheduleWithID executes a schedule with a given execution ID
func (s *Scheduler) executeScheduleWithID(task ScheduleTask, executionID string) {
	results := make([]ScheduleExecutionResult, 0, len(task.VMNames))
	overallStatus := "success"

	for _, vmName := range task.VMNames {
		result := ScheduleExecutionResult{
			VMName: vmName,
			Status: "failed",
		}

		// Get VM info
		vmInfo, err := s.getVMInfo(vmName)
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

		// Execute command with timeout
		start := time.Now()
		output, err := sshClient.ExecuteCommand(task.Command)
		duration := time.Since(start).Milliseconds()

		sshClient.Close()

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
		task.ID,
	)

	log.Printf("Scheduler: task %s (%s) completed with status: %s", task.Name, task.ID, overallStatus)
}

// getVMInfo retrieves VM information
func (s *Scheduler) getVMInfo(name string) (*vm.VM, error) {
	if s.vmManager == nil {
		return nil, sql.ErrNoRows
	}

	return s.vmManager.GetVM(name)
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return "exec_" + time.Now().Format("20060102150405.000")
}

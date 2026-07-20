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

package filetransfer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Options configures a Manager. It is intentionally decoupled from the global
// config package so the file-transfer feature stays self-contained.
type Options struct {
	DataDir          string
	EncryptionKey    string
	DefaultChunkSize int
	Roots            []string
}

// Manager is the core orchestrator: task CRUD, lifecycle, execution, status
// aggregation and startup recovery.
type Manager struct {
	store            *Store
	fs               *FS
	registry         *transportRegistry
	defaultChunkSize int

	mu       sync.RWMutex
	runtimes map[int64]*taskRuntime
	wg       sync.WaitGroup // tracks executeTask goroutines; Shutdown waits on it
}

// taskRuntime holds the live state of a single task.
type taskRuntime struct {
	taskID int64

	mu       sync.Mutex
	config   TaskConfig
	state    TaskState
	errMsg   string
	cancel   context.CancelFunc
	progress map[string]*FileProgress

	paused   atomic.Bool
	canceled atomic.Bool
}

// NewManager builds a Manager and registers the DIRECT and KAFKA transports.
func NewManager(opts Options) *Manager {
	chunkSize := opts.DefaultChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSizeBytes
	}
	reg := newTransportRegistry()
	reg.register(newDirectTransport())
	reg.register(newKafkaTransport())
	return &Manager{
		store:            newStore(opts.DataDir, opts.EncryptionKey),
		fs:               newFS(opts.Roots),
		registry:         reg,
		defaultChunkSize: chunkSize,
		runtimes:         make(map[int64]*taskRuntime),
	}
}

// FS returns the path-jailed filesystem browser.
func (m *Manager) FS() *FS { return m.fs }

func (m *Manager) getRuntime(taskID int64) (*taskRuntime, error) {
	m.mu.RLock()
	rt, ok := m.runtimes[taskID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("task not found: %d", taskID)
	}
	return rt, nil
}

// ===================== CRUD =====================

// CreateTask registers a new task and persists it in IDLE state.
func (m *Manager) CreateTask(config TaskConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.runtimes[config.TaskID]; exists {
		return fmt.Errorf("task already exists: %d", config.TaskID)
	}
	if err := m.store.saveConfig(config.TaskID, config); err != nil {
		return err
	}
	if err := m.store.saveState(config.TaskID, StateIdle); err != nil {
		return err
	}
	m.runtimes[config.TaskID] = &taskRuntime{
		taskID:   config.TaskID,
		config:   config,
		state:    StateIdle,
		progress: make(map[string]*FileProgress),
	}
	return nil
}

// UpdateTask replaces a task's config; only allowed in IDLE or PAUSED.
func (m *Manager) UpdateTask(taskID int64, config TaskConfig) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state != StateIdle && rt.state != StatePaused {
		return fmt.Errorf("cannot update task in state: %s", rt.state)
	}
	config.TaskID = taskID
	if err := m.store.saveConfig(taskID, config); err != nil {
		return err
	}
	rt.config = config
	return nil
}

// DeleteTask cancels and removes a task and its persisted data.
func (m *Manager) DeleteTask(taskID int64) error {
	m.mu.Lock()
	rt, ok := m.runtimes[taskID]
	delete(m.runtimes, taskID)
	m.mu.Unlock()
	if ok {
		rt.requestCancel()
	}
	return m.store.deleteTask(taskID)
}

// ListTasks returns all task configs.
func (m *Manager) ListTasks() []TaskConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]TaskConfig, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		result = append(result, rt.config)
		rt.mu.Unlock()
	}
	return result
}

// GetConfig returns one task's config.
func (m *Manager) GetConfig(taskID int64) (TaskConfig, error) {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return TaskConfig{}, err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.config, nil
}

// ===================== lifecycle =====================

// Start launches execution for an IDLE or PAUSED task.
func (m *Manager) Start(taskID int64) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state != StateIdle && rt.state != StatePaused {
		return fmt.Errorf("cannot start task in state: %s", rt.state)
	}
	rt.errMsg = ""
	rt.paused.Store(false)
	rt.canceled.Store(false)
	rt.state = StateRunning
	_ = m.store.saveState(taskID, StateRunning)
	m.launch(rt)
	return nil
}

// Pause requests a pause of a RUNNING task.
func (m *Manager) Pause(taskID int64) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state != StateRunning {
		return fmt.Errorf("cannot pause task in state: %s", rt.state)
	}
	rt.paused.Store(true)
	rt.state = StatePaused
	_ = m.store.saveState(taskID, StatePaused)
	if rt.cancel != nil {
		rt.cancel()
	}
	return nil
}

// Resume restarts a PAUSED task, reloading persisted progress first.
func (m *Manager) Resume(taskID int64) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state != StatePaused {
		return fmt.Errorf("cannot resume task in state: %s", rt.state)
	}
	rt.paused.Store(false)
	rt.canceled.Store(false)
	rt.state = StateRunning
	_ = m.store.saveState(taskID, StateRunning)
	m.loadProgressLocked(rt)
	m.launch(rt)
	return nil
}

// Cancel stops a task.
func (m *Manager) Cancel(taskID int64) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.canceled.Store(true)
	rt.paused.Store(false)
	rt.state = StateCancelled
	_ = m.store.saveState(taskID, StateCancelled)
	if rt.cancel != nil {
		rt.cancel()
	}
	return nil
}

func (rt *taskRuntime) requestCancel() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.canceled.Store(true)
	if rt.cancel != nil {
		rt.cancel()
	}
}

// launch starts the execution goroutine. Caller holds rt.mu.
func (m *Manager) launch(rt *taskRuntime) {
	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.executeTask(ctx, rt)
	}()
}

func (m *Manager) loadProgressLocked(rt *taskRuntime) {
	persisted, err := m.store.loadProgress(rt.taskID)
	if err != nil {
		return
	}
	for i := range persisted {
		fp := persisted[i]
		rt.progress[fp.RelPath] = &fp
	}
}

// Shutdown cancels all running goroutines without changing persisted state, so
// tasks marked RUNNING resume on next startup. It blocks until every task
// goroutine has returned.
func (m *Manager) Shutdown() {
	m.mu.RLock()
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		if rt.cancel != nil {
			rt.cancel()
		}
		rt.mu.Unlock()
	}
	m.mu.RUnlock()
	m.wg.Wait()
}

// Recover loads persisted tasks at startup and resumes those left RUNNING.
func (m *Manager) Recover() error {
	ids, err := m.store.listTaskIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		config, err := m.store.loadConfig(id)
		if err != nil || config == nil {
			continue
		}
		state, _ := m.store.loadState(id)
		if state == "" {
			state = StateIdle
		}
		rt := &taskRuntime{
			taskID:   id,
			config:   *config,
			state:    state,
			progress: make(map[string]*FileProgress),
		}
		m.mu.Lock()
		m.runtimes[id] = rt
		m.mu.Unlock()

		if state == StateRunning {
			rt.mu.Lock()
			m.loadProgressLocked(rt)
			m.launch(rt)
			rt.mu.Unlock()
			log.Printf("filetransfer: resumed task %d from persisted RUNNING state", id)
		}
	}
	return nil
}

// ===================== status =====================

// GetStatus returns the aggregated status and per-file progress.
func (m *Manager) GetStatus(taskID int64) (TaskStatus, error) {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return TaskStatus{}, err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()

	status := TaskStatus{TaskID: taskID, State: rt.state, ErrorMsg: rt.errMsg}
	files := make([]FileProgress, 0, len(rt.progress))
	var totalBytes, transferredBytes int64
	var completedFiles int
	for _, fp := range rt.progress {
		files = append(files, *fp)
		if fp.TotalBytes > 0 {
			totalBytes += fp.TotalBytes
		}
		transferredBytes += fp.ReceivedBytes
		if fp.Completed {
			completedFiles++
		}
	}
	status.Files = files
	status.TotalBytes = totalBytes
	status.TransferredBytes = transferredBytes
	status.TotalFiles = len(files)
	status.CompletedFiles = completedFiles
	if totalBytes > 0 {
		status.Progress = int(transferredBytes * 100 / totalBytes)
	}
	return status, nil
}

// GetProgress returns just the per-file progress (for resume queries).
func (m *Manager) GetProgress(taskID int64) ([]FileProgress, error) {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return nil, err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	files := make([]FileProgress, 0, len(rt.progress))
	for _, fp := range rt.progress {
		files = append(files, *fp)
	}
	return files, nil
}

// ===================== RECV: receive a chunk =====================

// ReceiveChunk writes one chunk to {targetDir}/{relPath}.part, and on EOF
// verifies the whole-file SHA-256 and applies the overwrite policy.
func (m *Manager) ReceiveChunk(taskID int64, meta ChunkMeta, body []byte) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()

	config := rt.config
	if config.Role != RoleRecv {
		return fmt.Errorf("task %d is not RECV role", taskID)
	}
	if config.TargetDir == "" {
		return fmt.Errorf("targetDir not set for task %d", taskID)
	}
	if strings.Contains(meta.RelPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", meta.RelPath)
	}

	targetDir, err := filepath.Abs(filepath.Clean(config.TargetDir))
	if err != nil {
		return err
	}
	partFile := filepath.Clean(filepath.Join(targetDir, meta.RelPath+".part"))
	if !within(targetDir, partFile) {
		return fmt.Errorf("path escapes target directory: %s", meta.RelPath)
	}

	if err := os.MkdirAll(filepath.Dir(partFile), 0o755); err != nil {
		return err
	}
	if err := writeChunkAt(partFile, meta.Offset, body); err != nil {
		return err
	}

	fp := rt.progress[meta.RelPath]
	if fp == nil {
		fp = &FileProgress{RelPath: meta.RelPath, TotalBytes: -1}
		rt.progress[meta.RelPath] = fp
	}
	fp.ReceivedBytes = meta.Offset + int64(len(body))

	if meta.Eof {
		fp.Completed = true
		fp.TotalBytes = meta.Offset + int64(len(body))
		if meta.Sha256 != "" {
			actual, err := computeSHA256(partFile)
			if err != nil {
				return err
			}
			if !strings.EqualFold(actual, meta.Sha256) {
				return fmt.Errorf("sha256 mismatch for %s", meta.RelPath)
			}
		}
		finalFile := filepath.Clean(filepath.Join(targetDir, meta.RelPath))
		if err := applyOverwritePolicy(finalFile, partFile, config.OverwritePolicy); err != nil {
			return err
		}
	}

	_ = m.store.saveProgress(taskID, progressValues(rt.progress))
	return nil
}

// ===================== execution =====================

func (m *Manager) executeTask(ctx context.Context, rt *taskRuntime) {
	rt.mu.Lock()
	role := rt.config.Role
	rt.mu.Unlock()

	var err error
	if role == RoleSend {
		err = m.executeSend(ctx, rt)
	} else {
		err = m.executeRecv(ctx, rt)
	}
	if err != nil && ctx.Err() == nil {
		rt.mu.Lock()
		rt.errMsg = err.Error()
		rt.state = StateFailed
		rt.mu.Unlock()
		_ = m.store.saveState(rt.taskID, StateFailed)
		log.Printf("filetransfer: task %d failed: %v", rt.taskID, err)
	}
}

func (m *Manager) executeSend(ctx context.Context, rt *taskRuntime) error {
	rt.mu.Lock()
	config := rt.config
	rt.mu.Unlock()

	manifest := config.Manifest
	if len(manifest) == 0 {
		manifest = buildManifest(config)
	}
	if len(manifest) == 0 {
		m.setState(rt, StateSuccess)
		return nil
	}

	transport, ok := m.registry.get(config.RelayType)
	if !ok {
		return fmt.Errorf("no transport registered for relay type %s", config.RelayType)
	}
	if config.RelayType == RelayDirect && len(config.Targets) == 0 {
		return fmt.Errorf("no targets configured for SEND task")
	}

	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = m.defaultChunkSize
	}
	config.ChunkSize = chunkSize

	// Initialise progress for every file.
	rt.mu.Lock()
	for _, entry := range manifest {
		if _, exists := rt.progress[entry.RelPath]; !exists {
			rt.progress[entry.RelPath] = &FileProgress{RelPath: entry.RelPath, TotalBytes: entry.Size}
		}
	}
	rt.mu.Unlock()

	// KAFKA relay sends to the topic once (no per-target fan-out); DIRECT
	// fans out to each target.
	targets := config.Targets
	if config.RelayType == RelayKafka {
		targets = []TargetConfig{{}}
	}

	allSuccess := true
	for _, file := range manifest {
		if rt.canceled.Load() {
			m.setState(rt, StateCancelled)
			return nil
		}
		if rt.paused.Load() {
			m.setState(rt, StatePaused)
			return nil
		}
		if fp := m.progressFor(rt, file.RelPath); fp != nil && fp.Completed {
			continue
		}

		fileSuccess := false
		for _, target := range targets {
			if err := m.sendFileToTarget(ctx, rt, config, target, file, transport); err != nil {
				if ctx.Err() != nil {
					return nil // paused/cancelled mid-flight
				}
				log.Printf("filetransfer: task %d failed to send %s: %v", rt.taskID, file.RelPath, err)
				allSuccess = false
			} else {
				fileSuccess = true
			}
		}
		if fileSuccess {
			rt.mu.Lock()
			if fp := rt.progress[file.RelPath]; fp != nil {
				fp.Completed = true
				fp.ReceivedBytes = file.Size
			}
			_ = m.store.saveProgress(rt.taskID, progressValues(rt.progress))
			rt.mu.Unlock()
		}
	}

	if allSuccess {
		m.setState(rt, StateSuccess)
	} else {
		m.setState(rt, StatePartialFailed)
	}
	return nil
}

func (m *Manager) sendFileToTarget(ctx context.Context, rt *taskRuntime, config TaskConfig, target TargetConfig, file FileEntry, transport RelayTransport) error {
	source := findSourcePath(config, file.RelPath)
	if source == "" {
		return fmt.Errorf("source file not found: %s", file.RelPath)
	}
	reader := func(offset int64, length int) ([]byte, error) {
		f, err := os.Open(source)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buf := make([]byte, length)
		n, err := f.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		return buf[:n], nil
	}
	sink := func(n int) {
		rt.mu.Lock()
		if fp := rt.progress[file.RelPath]; fp != nil {
			fp.ReceivedBytes += int64(n)
		}
		rt.mu.Unlock()
	}
	return transport.SendFile(ctx, config, target, file, reader, sink)
}

func (m *Manager) executeRecv(ctx context.Context, rt *taskRuntime) error {
	rt.mu.Lock()
	config := rt.config
	rt.mu.Unlock()

	if config.RelayType == RelayKafka {
		err := consumeKafka(ctx, config.Kafka, func(meta ChunkMeta, data []byte) error {
			return m.ReceiveChunk(rt.taskID, meta, data)
		})
		// On ctx cancel, reflect the requested pause/cancel state.
		if rt.paused.Load() {
			m.setState(rt, StatePaused)
		} else if rt.canceled.Load() {
			m.setState(rt, StateCancelled)
		}
		return err
	}
	// DIRECT RECV: chunks arrive via the HTTP handler; nothing to do here but
	// remain RUNNING until paused/cancelled.
	m.setState(rt, StateRunning)
	return nil
}

// ===================== helpers =====================

func (m *Manager) setState(rt *taskRuntime, state TaskState) {
	rt.mu.Lock()
	rt.state = state
	rt.mu.Unlock()
	_ = m.store.saveState(rt.taskID, state)
}

func (m *Manager) progressFor(rt *taskRuntime, relPath string) *FileProgress {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.progress[relPath]
}

func progressValues(progress map[string]*FileProgress) []FileProgress {
	out := make([]FileProgress, 0, len(progress))
	for _, fp := range progress {
		out = append(out, *fp)
	}
	return out
}

// writeChunkAt writes body at offset in path, creating/extending the file.
func writeChunkAt(path string, offset int64, body []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteAt(body, offset); err != nil {
		return err
	}
	return nil
}

// computeSHA256 streams the file and returns the lowercase hex digest.
func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// buildManifest expands SourcePaths into file entries with size and SHA-256.
// Directory entries use forward-slash relative paths; a single file uses its
// base name.
func buildManifest(config TaskConfig) []FileEntry {
	var manifest []FileEntry
	for _, sp := range config.SourcePaths {
		abs, err := filepath.Abs(filepath.Clean(sp))
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if info.IsDir() {
			_ = filepath.Walk(abs, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(abs, path)
				if err != nil {
					return nil
				}
				sha, err := computeSHA256(path)
				if err != nil {
					return nil
				}
				manifest = append(manifest, FileEntry{
					RelPath: filepath.ToSlash(rel),
					Size:    fi.Size(),
					Sha256:  sha,
				})
				return nil
			})
		} else {
			sha, err := computeSHA256(abs)
			if err != nil {
				continue
			}
			manifest = append(manifest, FileEntry{
				RelPath: filepath.Base(abs),
				Size:    info.Size(),
				Sha256:  sha,
			})
		}
	}
	return manifest
}

// findSourcePath resolves a manifest relPath back to an on-disk source file.
func findSourcePath(config TaskConfig, relPath string) string {
	for _, sp := range config.SourcePaths {
		abs, err := filepath.Abs(filepath.Clean(sp))
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if info.IsDir() {
			candidate := filepath.Join(abs, filepath.FromSlash(relPath))
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		} else if filepath.Base(abs) == relPath {
			return abs
		}
	}
	return ""
}

// applyOverwritePolicy moves partPath to its final destination according to
// the policy (OVERWRITE/SKIP/RENAME), defaulting to OVERWRITE.
func applyOverwritePolicy(finalPath, partPath string, policy OverwritePolicy) error {
	switch policy {
	case Skip:
		if _, err := os.Stat(finalPath); err == nil {
			return os.Remove(partPath)
		}
		return os.Rename(partPath, finalPath)
	case Rename:
		dest := finalPath
		counter := 1
		for {
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				break
			}
			name := filepath.Base(finalPath)
			ext := filepath.Ext(name)
			base := strings.TrimSuffix(name, ext)
			dest = filepath.Join(filepath.Dir(finalPath), fmt.Sprintf("%s_%d%s", base, counter, ext))
			counter++
		}
		return os.Rename(partPath, dest)
	default: // Overwrite (and empty policy)
		// os.Rename does not replace an existing file on Windows, so remove first.
		_ = os.Remove(finalPath)
		return os.Rename(partPath, finalPath)
	}
}

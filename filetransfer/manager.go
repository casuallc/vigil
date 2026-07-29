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
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

	// recvFiles tracks per-file received byte ranges (RECV side) so chunks
	// arriving out of order — from parallel KAFKA sends — are reassembled by
	// offset and finalised only once the file is fully covered. In-memory
	// only: the KAFKA relay never resumes mid-file and DIRECT seeds the
	// ranges from persisted progress on resume.
	recvFiles map[string]*recvFileState

	// Timing (guarded by rt.mu): startedAt/finishedAt mark the first run and
	// the terminal state; activeMs accumulates run time excluding pauses;
	// runStart is the current run's start (zero while not RUNNING).
	startedAt  time.Time
	finishedAt time.Time
	activeMs   int64
	runStart   time.Time
	rate       rateWindow

	// lastPersist is the last time progress/recvstate were flushed to disk
	// (guarded by rt.mu); RECV-side persistence is throttled to this cadence.
	lastPersist time.Time

	paused   atomic.Bool
	canceled atomic.Bool
}

// recvFileState is the RECV-side reassembly state for one file.
type recvFileState struct {
	ranges    intervalSet
	eofSeen   bool
	total     int64
	sha256    string
	finalized bool

	// Incremental SHA-256 of the received bytes, folded in offset order as
	// chunks land so finalise does not re-read the whole file from disk (a
	// multi-minute stall for very large files). Out-of-order chunks are
	// buffered up to maxHashBufferBytes; past the cap, or after a restart
	// restore, incremental hashing is abandoned (hashBroken) and finalise
	// falls back to hashing the finished .part file.
	// Guarded by hashMu (NOT rt.mu): hashing a chunk costs about as much as
	// writing it, and serialising it across all files/partitions would cap
	// receiver throughput.
	hashMu     sync.Mutex
	hasher     hash.Hash
	hashNext   int64
	hashBuf    map[int64][]byte
	hashBufLen int64
	hashBroken bool
}

// recvStateFor returns the reassembly state for relPath, creating the map
// and entry lazily. Caller must hold rt.mu.
func (rt *taskRuntime) recvStateFor(relPath string) *recvFileState {
	if rt.recvFiles == nil {
		rt.recvFiles = make(map[string]*recvFileState)
	}
	st := rt.recvFiles[relPath]
	if st == nil {
		st = &recvFileState{}
		rt.recvFiles[relPath] = st
	}
	return st
}

// beginRunLocked records the start of a RUNNING period. Idempotent while
// already running. A SUCCESS task flipping back to RUNNING (new files
// arriving) clears finishedAt. Caller must hold rt.mu.
func (rt *taskRuntime) beginRunLocked(now time.Time) {
	if rt.startedAt.IsZero() {
		rt.startedAt = now
	}
	if rt.runStart.IsZero() {
		rt.runStart = now
		rt.finishedAt = time.Time{}
	}
}

// endRunLocked closes the current RUNNING period, folding its duration into
// activeMs, and stamps finishedAt on terminal states. Idempotent while not
// running. Caller must hold rt.mu.
func (rt *taskRuntime) endRunLocked(now time.Time, terminal bool) {
	if !rt.runStart.IsZero() {
		rt.activeMs += now.Sub(rt.runStart).Milliseconds()
		rt.runStart = time.Time{}
	}
	if terminal {
		rt.finishedAt = now
	}
}

// timingLocked snapshots the persisted timing form. Caller must hold rt.mu.
func (rt *taskRuntime) timingLocked() taskTiming {
	var t taskTiming
	if !rt.startedAt.IsZero() {
		t.StartedAt = rt.startedAt.UnixMilli()
	}
	if !rt.finishedAt.IsZero() {
		t.FinishedAt = rt.finishedAt.UnixMilli()
	}
	t.ActiveMs = rt.activeMs
	return t
}

// applyTiming restores persisted timing (used at Recover, before the runtime
// is shared, so no lock is needed).
func (rt *taskRuntime) applyTiming(t taskTiming) {
	if t.StartedAt > 0 {
		rt.startedAt = time.UnixMilli(t.StartedAt)
	}
	if t.FinishedAt > 0 {
		rt.finishedAt = time.UnixMilli(t.FinishedAt)
	}
	rt.activeMs = t.ActiveMs
}

// isTerminalState reports whether s ends the task's timing.
func isTerminalState(s TaskState) bool {
	switch s {
	case StateSuccess, StateFailed, StatePartialFailed, StateCancelled:
		return true
	}
	return false
}

// rateWindowBuckets is the number of per-second buckets kept for the
// trailing transfer-rate estimate.
const rateWindowBuckets = 10

// rateWindowReportSecs is the trailing window length reported as the current
// transfer rate.
const rateWindowReportSecs = 5

// rateWindow tracks transferred bytes in per-second buckets to estimate the
// current send/receive rate. Live-traffic estimate only; not persisted.
type rateWindow struct {
	secs  [rateWindowBuckets]int64
	bytes [rateWindowBuckets]int64
}

// add records n transferred bytes at now. Caller must hold rt.mu.
func (w *rateWindow) add(now time.Time, n int64) {
	if n <= 0 {
		return
	}
	sec := now.Unix()
	i := sec % rateWindowBuckets
	if w.secs[i] != sec {
		w.secs[i] = sec
		w.bytes[i] = 0
	}
	w.bytes[i] += n
}

// perSecond returns the trailing average rate over the last
// rateWindowReportSecs seconds.
func (w *rateWindow) perSecond(now time.Time) int64 {
	cutoff := now.Unix() - rateWindowReportSecs
	var sum int64
	for i := 0; i < rateWindowBuckets; i++ {
		if w.secs[i] > cutoff {
			sum += w.bytes[i]
		}
	}
	return sum / rateWindowReportSecs
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
	rt.beginRunLocked(time.Now())
	_ = m.store.saveState(taskID, StateRunning)
	_ = m.store.saveTiming(taskID, rt.timingLocked())
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
	rt.endRunLocked(time.Now(), false)
	m.persistProgressLocked(rt, taskID)
	_ = m.store.saveState(taskID, StatePaused)
	_ = m.store.saveTiming(taskID, rt.timingLocked())
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
	if rt.state != StatePaused {
		rt.mu.Unlock()
		return fmt.Errorf("cannot resume task in state: %s", rt.state)
	}
	rt.paused.Store(false)
	rt.canceled.Store(false)
	rt.state = StateRunning
	rt.beginRunLocked(time.Now())
	_ = m.store.saveState(taskID, StateRunning)
	_ = m.store.saveTiming(taskID, rt.timingLocked())
	pending := m.loadProgressLocked(rt)
	m.launch(rt)
	rt.mu.Unlock()
	m.finalizePending(rt, pending)
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
	rt.endRunLocked(time.Now(), true)
	m.persistProgressLocked(rt, taskID)
	_ = m.store.saveState(taskID, StateCancelled)
	_ = m.store.saveTiming(taskID, rt.timingLocked())
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

// loadProgressLocked restores persisted per-file progress and reassembly
// state, and returns finalise operations for files that were fully received
// but never finalised. Caller holds rt.mu; the returned ops must be run
// AFTER releasing it (see finalizePending).
func (m *Manager) loadProgressLocked(rt *taskRuntime) []*finalizeOp {
	persisted, err := m.store.loadProgress(rt.taskID)
	if err != nil {
		return nil
	}
	for i := range persisted {
		fp := persisted[i]
		rt.progress[fp.RelPath] = &fp
	}
	// Restore the exact reassembly state when available. The ReceivedBytes
	// scalar cannot represent holes from out-of-order delivery, and the
	// chunks filling them may be past the committed Kafka offset, so
	// restoring anything less precise can wedge a file that is actually
	// complete — or finalise one that is not.
	states, err := m.store.loadRecvState(rt.taskID)
	if err == nil && states != nil {
		for rel, sp := range states {
			st := rt.recvStateFor(rel)
			st.ranges = intervalSet{intervals: append([]interval(nil), sp.Ranges...)}
			st.eofSeen = sp.EofSeen
			st.total = sp.Total
			st.sha256 = sp.Sha256
			st.finalized = sp.Finalized
		}
	} else {
		// Legacy fallback (tasks written before recvstate.json existed):
		// treat persisted bytes as a contiguous prefix. Exact for sequential
		// delivery (DIRECT, or KAFKA with parallelism 1).
		for i := range persisted {
			fp := persisted[i]
			if !fp.Completed && fp.ReceivedBytes > 0 {
				rt.recvStateFor(fp.RelPath).ranges.insert(0, fp.ReceivedBytes)
			}
		}
	}

	// Files restored from disk keep their bytes but not their incremental
	// hash state: finalise falls back to hashing the .part file from disk.
	for _, st := range rt.recvFiles {
		st.hashMu.Lock()
		st.hashBroken = true
		st.hashMu.Unlock()
	}

	// A file can be fully received yet never finalised: recvstate.json is
	// persisted as soon as the last chunk lands, so a crash before or during
	// the rename leaves Finalized set with no future chunk to trigger the
	// rename again (finalised files skip chunk writes). Re-finalise such
	// files on load instead of leaving them wedged.
	if rt.config.TargetDir == "" {
		return nil
	}
	targetDir, err := filepath.Abs(filepath.Clean(rt.config.TargetDir))
	if err != nil {
		return nil
	}
	var pending []*finalizeOp
	dirty := false
	for rel, st := range rt.recvFiles {
		fp := rt.progress[rel]
		if !st.eofSeen || st.ranges.covered() < st.total || (fp != nil && fp.Completed) {
			continue
		}
		partFile := filepath.Clean(filepath.Join(targetDir, rel+".part"))
		finalFile := filepath.Clean(filepath.Join(targetDir, rel))
		if _, err := os.Stat(partFile); os.IsNotExist(err) {
			if _, ferr := os.Stat(finalFile); ferr == nil {
				// The rename happened but the completion was never persisted.
				if fp == nil {
					fp = &FileProgress{RelPath: rel, TotalBytes: st.total}
					rt.progress[rel] = fp
				}
				fp.Completed = true
				fp.ReceivedBytes = st.total
				dirty = true
			}
			continue
		}
		if fp == nil {
			fp = &FileProgress{RelPath: rel, TotalBytes: st.total, ReceivedBytes: st.total}
			rt.progress[rel] = fp
			dirty = true
		}
		st.finalized = true
		pending = append(pending, &finalizeOp{
			relPath:   rel,
			partFile:  partFile,
			finalFile: finalFile,
			sha256:    st.sha256,
			policy:    rt.config.OverwritePolicy,
		})
	}
	if dirty {
		m.persistProgressLocked(rt, rt.taskID)
	}
	return pending
}

// finalizePending runs the finalise operations recovered by
// loadProgressLocked (outside rt.mu) and completes the task when nothing
// remains. Failures roll back the finalized flag so chunk redelivery can
// retry, and are logged rather than returned: the task itself is running.
func (m *Manager) finalizePending(rt *taskRuntime, pending []*finalizeOp) {
	for _, op := range pending {
		if err := m.finalizeFile(rt, rt.taskID, op); err != nil {
			log.Printf("filetransfer: task %d cannot re-finalise %s: %v", rt.taskID, op.relPath, err)
		}
	}
	if len(pending) == 0 {
		return
	}
	rt.mu.Lock()
	allDone := len(rt.progress) > 0 && allFilesCompletedLocked(rt)
	state := rt.state
	rt.mu.Unlock()
	if allDone && state == StateRunning {
		m.setState(rt, StateSuccess)
		m.stopConsumerIfComplete(rt)
	}
}

// saveRecvStateLocked persists the RECV reassembly state of every in-flight
// file. Caller must hold rt.mu.
func (m *Manager) saveRecvStateLocked(rt *taskRuntime) {
	states := make(map[string]recvFileStatePersist, len(rt.recvFiles))
	for rel, st := range rt.recvFiles {
		states[rel] = recvFileStatePersist{
			Ranges:    append([]interval(nil), st.ranges.intervals...),
			EofSeen:   st.eofSeen,
			Total:     st.total,
			Sha256:    st.sha256,
			Finalized: st.finalized,
		}
	}
	_ = m.store.saveRecvState(rt.taskID, states)
}

// Shutdown cancels all running goroutines without changing persisted state, so
// tasks marked RUNNING resume on next startup. It blocks until every task
// goroutine has returned, then closes transports holding reusable resources
// (the KAFKA transport caches its producer).
func (m *Manager) Shutdown() {
	m.mu.RLock()
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		if rt.cancel != nil {
			rt.cancel()
		}
		// Flush throttled RECV progress so a resume after restart does not
		// re-receive up to persistInterval of chunks.
		m.persistProgressLocked(rt, rt.taskID)
		rt.mu.Unlock()
	}
	m.mu.RUnlock()
	m.wg.Wait()
	for _, t := range m.registry.m {
		if c, ok := t.(io.Closer); ok {
			_ = c.Close()
		}
	}
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
		if timing, err := m.store.loadTiming(id); err == nil {
			rt.applyTiming(timing)
		}
		m.mu.Lock()
		m.runtimes[id] = rt
		m.mu.Unlock()

		if state == StateRunning {
			rt.mu.Lock()
			rt.beginRunLocked(time.Now())
			pending := m.loadProgressLocked(rt)
			m.launch(rt)
			rt.mu.Unlock()
			m.finalizePending(rt, pending)
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
		disp := *fp
		if st := rt.recvFiles[fp.RelPath]; st != nil {
			// Display true coverage (out-of-order arrivals included); the
			// plain ReceivedBytes is the contiguous prefix reserved for
			// DIRECT resume via the /progress endpoint.
			if c := st.ranges.covered(); c > disp.ReceivedBytes {
				disp.ReceivedBytes = c
			}
		}
		files = append(files, disp)
		if disp.TotalBytes > 0 {
			totalBytes += disp.TotalBytes
		}
		transferredBytes += disp.ReceivedBytes
		if disp.Completed {
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
	now := time.Now()
	if !rt.startedAt.IsZero() {
		status.StartedAt = rt.startedAt.UnixMilli()
	}
	if !rt.finishedAt.IsZero() {
		status.FinishedAt = rt.finishedAt.UnixMilli()
	}
	elapsed := rt.activeMs
	if !rt.runStart.IsZero() {
		elapsed += now.Sub(rt.runStart).Milliseconds()
	}
	status.ElapsedMs = elapsed
	if elapsed > 0 {
		status.BytesPerSecond = transferredBytes * 1000 / elapsed
	}
	status.CurrentBytesPerSecond = rt.rate.perSecond(now)
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

// persistInterval bounds how often a RECEIVING task flushes progress and
// reassembly state to disk. Per-chunk JSON writes used to dominate chunk
// latency; throttling loses at most this much progress on a crash (resent
// chunks are deduplicated by the interval set), which is cheap insurance.
const persistInterval = time.Second

// finalizeOp captures everything needed to verify and move a completed .part
// file, so the work can run outside rt.mu (see finalizeFile).
type finalizeOp struct {
	relPath   string
	partFile  string
	finalFile string
	sha256    string
	policy    OverwritePolicy
	// receivedSHA is the incremental digest of the received bytes when
	// available; empty means finalise must hash the .part file from disk.
	receivedSHA string
}

// ReceiveChunk writes one chunk to {targetDir}/{relPath}.part. Chunks may
// arrive out of order (parallel sends): once the EOF chunk has been seen AND
// the received ranges cover the whole file, the SHA-256 is verified and the
// overwrite policy applied — outside rt.mu, so hashing a very large file does
// not stall other in-flight chunks.
//
// When every file the task knows about is completed the task transitions to
// SUCCESS; if chunks for a NEW file arrive later (another transfer over the
// same topic), the task flips back to RUNNING. For the KAFKA relay this only
// applies while the task is still RUNNING: on SUCCESS the consume loop exits
// and closes the consumer connection (see stopConsumerIfComplete).
func (m *Manager) ReceiveChunk(taskID int64, meta ChunkMeta, body []byte) error {
	rt, err := m.getRuntime(taskID)
	if err != nil {
		return err
	}

	// Phase 1 (locked): validate and resolve paths; redeliveries of an
	// already-finalised file are skipped without touching the disk.
	rt.mu.Lock()
	config := rt.config
	if config.Role != RoleRecv {
		rt.mu.Unlock()
		return fmt.Errorf("task %d is not RECV role", taskID)
	}
	if config.TargetDir == "" {
		rt.mu.Unlock()
		return fmt.Errorf("targetDir not set for task %d", taskID)
	}
	if strings.Contains(meta.RelPath, "..") {
		rt.mu.Unlock()
		return fmt.Errorf("path traversal not allowed: %s", meta.RelPath)
	}
	targetDir, err := filepath.Abs(filepath.Clean(config.TargetDir))
	if err != nil {
		rt.mu.Unlock()
		return err
	}
	partFile := filepath.Clean(filepath.Join(targetDir, meta.RelPath+".part"))
	if !within(targetDir, partFile) {
		rt.mu.Unlock()
		return fmt.Errorf("path escapes target directory: %s", meta.RelPath)
	}
	st := rt.recvStateFor(meta.RelPath)
	skipWrite := st.finalized
	rt.mu.Unlock()

	// Phase 2 (unlocked): the disk write itself. Concurrent chunks for the
	// same file land at disjoint offsets, so writers do not conflict — this
	// is what lets pipelined sends actually overlap on the receiver. The
	// incremental hash is fed here too (before this chunk's coverage is
	// recorded in phase 3), so a completing chunk always finds its own bytes
	// already folded in.
	wrote := false
	if !skipWrite {
		if err := os.MkdirAll(filepath.Dir(partFile), 0o755); err != nil {
			return err
		}
		if err := writeChunkAt(partFile, meta.Offset, body); err != nil {
			return err
		}
		st.feedHasher(meta.Offset, body)
		wrote = true
	}

	// Phase 3 (locked): record coverage and decide on finalisation.
	rt.mu.Lock()
	var orphanPart string
	var newFile bool
	var pending *finalizeOp
	if wrote && !st.finalized {
		fp := rt.progress[meta.RelPath]
		if fp == nil {
			fp = &FileProgress{RelPath: meta.RelPath, TotalBytes: -1}
			rt.progress[meta.RelPath] = fp
			newFile = true
		}
		// The sender reports the total size on every chunk, so progress is
		// computable from the first chunk instead of only once the EOF
		// chunk arrives (with pipelined sends, that is near the very end).
		if meta.Size > 0 && fp.TotalBytes <= 0 {
			fp.TotalBytes = meta.Size
		}
		newly := st.ranges.insert(meta.Offset, meta.Offset+int64(len(body)))
		// Report the contiguous prefix rather than total coverage: the
		// /progress endpoint doubles as the DIRECT resume offset, and
		// resuming from a coverage figure that includes out-of-order chunks
		// past a hole would skip the hole and corrupt the file.
		fp.ReceivedBytes = st.ranges.prefixEnd()
		rt.rate.add(time.Now(), newly)

		if meta.Eof {
			st.eofSeen = true
			st.total = meta.Offset + int64(len(body))
			st.sha256 = meta.Sha256
			fp.TotalBytes = st.total
		}

		if st.eofSeen && st.ranges.covered() >= st.total {
			// Mark finalized under the lock so concurrent chunks for this
			// file are skipped as redeliveries while finalizeFile runs
			// outside it.
			st.finalized = true
			pending = &finalizeOp{
				relPath:   meta.RelPath,
				partFile:  partFile,
				finalFile: filepath.Clean(filepath.Join(targetDir, meta.RelPath)),
				sha256:    st.sha256,
				policy:    config.OverwritePolicy,
			}
			// Full coverage means every chunk has been folded in order, so
			// the incremental digest is final and finalise needs no disk
			// re-read at all. A chunk still feeding its bytes (between the
			// phase-2 write and here) leaves hashNext short of total, which
			// safely falls back to hashing from disk.
			st.hashMu.Lock()
			if st.hasher != nil && !st.hashBroken && st.hashNext == st.total {
				pending.receivedSHA = hex.EncodeToString(st.hasher.Sum(nil))
			}
			st.hashMu.Unlock()
		}

		// Throttled persistence: always flush on the first chunk of a file
		// and when a file completes; otherwise at most once per
		// persistInterval. Pause/Cancel and terminal states force a flush
		// via setState.
		now := time.Now()
		if newFile || pending != nil || now.Sub(rt.lastPersist) >= persistInterval {
			m.persistProgressLocked(rt, taskID)
		}
	} else if wrote {
		// The file was finalised while this chunk was being written. Bytes
		// written before the rename are identical (same source) and harmless;
		// if the rename already completed, the .part this write recreated is
		// an orphan and must not be left behind.
		if fp := rt.progress[meta.RelPath]; fp != nil && fp.Completed {
			orphanPart = partFile
		}
	}
	allDone := len(rt.progress) > 0 && allFilesCompletedLocked(rt)
	state := rt.state
	rt.mu.Unlock()
	if orphanPart != "" {
		_ = os.Remove(orphanPart)
	}

	if pending != nil {
		if err := m.finalizeFile(rt, taskID, pending); err != nil {
			return err
		}
		rt.mu.Lock()
		allDone = len(rt.progress) > 0 && allFilesCompletedLocked(rt)
		state = rt.state
		rt.mu.Unlock()
	}
	if newFile && state == StateSuccess {
		// Another transfer started over the same channel.
		m.setState(rt, StateRunning)
		state = StateRunning
	}
	if allDone && state == StateRunning {
		m.setState(rt, StateSuccess)
	}
	return nil
}

// maxHashBufferBytes caps out-of-order chunks retained for incremental
// hashing; past the cap the receiver abandons it and finalise falls back to
// hashing the finished file from disk.
const maxHashBufferBytes = 64 << 20

// feedHasher folds a newly written chunk into the file's incremental
// SHA-256 in offset order, buffering out-of-order arrivals. It runs outside
// rt.mu, guarded by st.hashMu, so hashing never serialises concurrent chunk
// handlers across files or partitions. Chunks redelivered past the drain
// point are ignored; a redelivery of a buffered chunk simply replaces it
// (same source, same bytes).
func (st *recvFileState) feedHasher(offset int64, body []byte) {
	st.hashMu.Lock()
	defer st.hashMu.Unlock()
	if st.hashBroken || offset < st.hashNext {
		return
	}
	if st.hasher == nil {
		st.hasher = sha256.New()
	}
	if st.hashBuf == nil {
		st.hashBuf = make(map[int64][]byte)
	}
	if _, exists := st.hashBuf[offset]; !exists {
		st.hashBufLen += int64(len(body))
	}
	st.hashBuf[offset] = body
	if st.hashBufLen > maxHashBufferBytes {
		// An early chunk is holding back too much memory: give up on
		// incremental hashing, finalise will read the file from disk.
		st.hashBroken = true
		st.hasher = nil
		st.hashBuf = nil
		st.hashBufLen = 0
		return
	}
	for {
		data, ok := st.hashBuf[st.hashNext]
		if !ok {
			break
		}
		delete(st.hashBuf, st.hashNext)
		st.hashBufLen -= int64(len(data))
		_, _ = st.hasher.Write(data)
		st.hashNext += int64(len(data))
	}
}

// persistProgressLocked forces a progress/recvstate flush. Caller holds rt.mu.
func (m *Manager) persistProgressLocked(rt *taskRuntime, taskID int64) {
	_ = m.store.saveProgress(taskID, progressValues(rt.progress))
	m.saveRecvStateLocked(rt)
	rt.lastPersist = time.Now()
}

// finalizeFile verifies the completed .part file and moves it to its final
// destination. It runs outside rt.mu. The digest normally comes from the
// incremental hasher (computed while chunks landed); without it (restart or
// buffer overflow) the file is hashed from disk — slow for very large files
// but correct. On failure the finalized flag is rolled back so a redelivered
// chunk retries finalisation (matching the previous at-least-once semantics).
func (m *Manager) finalizeFile(rt *taskRuntime, taskID int64, op *finalizeOp) error {
	if op.sha256 != "" {
		actual := op.receivedSHA
		if actual == "" {
			sum, err := computeSHA256(op.partFile)
			if err != nil {
				m.rollbackFinalize(rt, op.relPath)
				return err
			}
			actual = sum
		}
		if !strings.EqualFold(actual, op.sha256) {
			m.rollbackFinalize(rt, op.relPath)
			return fmt.Errorf("sha256 mismatch for %s", op.relPath)
		}
	}
	if err := applyOverwritePolicy(op.finalFile, op.partFile, op.policy); err != nil {
		m.rollbackFinalize(rt, op.relPath)
		return err
	}
	rt.mu.Lock()
	if fp := rt.progress[op.relPath]; fp != nil {
		fp.Completed = true
	}
	m.persistProgressLocked(rt, taskID)
	rt.mu.Unlock()
	return nil
}

// rollbackFinalize clears the finalized flag after a failed finalizeFile so
// the file can be finalised again on chunk redelivery.
func (m *Manager) rollbackFinalize(rt *taskRuntime, relPath string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if st, ok := rt.recvFiles[relPath]; ok {
		st.finalized = false
	}
}

// allFilesCompletedLocked reports whether every known file is completed.
// Caller must hold rt.mu.
func allFilesCompletedLocked(rt *taskRuntime) bool {
	for _, fp := range rt.progress {
		if !fp.Completed {
			return false
		}
	}
	return true
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
		m.persistProgressLocked(rt, rt.taskID)
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

	// Lazily hash files the manifest does not carry a digest for: the pool
	// overlaps hashing with the transfer and the EOF chunk blocks on the
	// result (see resolveSHA256). Files without a resolvable local source
	// keep the legacy behaviour (no digest; receiver skips verification).
	hashCtx, stopHash := context.WithCancel(ctx)
	defer stopHash()
	futures := make(map[string]*hashFuture)
	var pool *hashPool
	for _, e := range manifest {
		if e.Sha256 != "" {
			continue
		}
		src := findSourcePath(config, e.RelPath)
		if src == "" {
			continue
		}
		fut := &hashFuture{done: make(chan struct{})}
		futures[e.RelPath] = fut
		if pool == nil {
			pool = newHashPool(hashCtx, hashWorkers, len(manifest))
		}
		pool.submit(src, fut)
	}

	// KAFKA relay sends to the topic once (no per-target fan-out); DIRECT
	// fans out to each target.
	targets := config.Targets
	if config.RelayType == RelayKafka {
		targets = []TargetConfig{{}}
	}

	if effectiveParallelism(config.Parallelism) > 1 {
		return m.sendFilesParallel(ctx, rt, config, targets, manifest, transport, futures)
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
		if !m.sendOneFile(ctx, rt, config, targets, file, transport, futures) {
			if ctx.Err() != nil {
				return nil // paused/cancelled mid-flight
			}
			allSuccess = false
		}
	}

	if allSuccess {
		m.setState(rt, StateSuccess)
	} else {
		m.setState(rt, StatePartialFailed)
	}
	return nil
}

// sendFilesParallel delivers the manifest with a worker pool of
// config.Parallelism goroutines, each sending whole files independently.
func (m *Manager) sendFilesParallel(ctx context.Context, rt *taskRuntime, config TaskConfig, targets []TargetConfig, manifest []FileEntry, transport RelayTransport, futures map[string]*hashFuture) error {
	workers := effectiveParallelism(config.Parallelism)
	var allSuccess atomic.Bool
	allSuccess.Store(true)

	jobs := make(chan FileEntry)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				if rt.canceled.Load() || rt.paused.Load() {
					continue
				}
				if fp := m.progressFor(rt, file.RelPath); fp != nil && fp.Completed {
					continue
				}
				if !m.sendOneFile(ctx, rt, config, targets, file, transport, futures) {
					if ctx.Err() != nil {
						continue // paused/cancelled mid-flight
					}
					allSuccess.Store(false)
				}
			}
		}()
	}
feed:
	for _, file := range manifest {
		select {
		case jobs <- file:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	if rt.canceled.Load() {
		m.setState(rt, StateCancelled)
		return nil
	}
	if rt.paused.Load() {
		m.setState(rt, StatePaused)
		return nil
	}
	if allSuccess.Load() {
		m.setState(rt, StateSuccess)
	} else {
		m.setState(rt, StatePartialFailed)
	}
	return nil
}

// sendOneFile delivers one manifest entry to every target and, on success,
// marks the file completed. It reports whether the file reached at least one
// target; a false return with ctx cancelled means paused/cancelled mid-flight.
func (m *Manager) sendOneFile(ctx context.Context, rt *taskRuntime, config TaskConfig, targets []TargetConfig, file FileEntry, transport RelayTransport, futures map[string]*hashFuture) bool {
	fileSuccess := false
	for _, target := range targets {
		if err := m.sendFileToTarget(ctx, rt, config, target, file, transport, futures); err != nil {
			if ctx.Err() != nil {
				return false
			}
			log.Printf("filetransfer: task %d failed to send %s: %v", rt.taskID, file.RelPath, err)
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
	return fileSuccess
}

func (m *Manager) sendFileToTarget(ctx context.Context, rt *taskRuntime, config TaskConfig, target TargetConfig, file FileEntry, transport RelayTransport, futures map[string]*hashFuture) error {
	source := findSourcePath(config, file.RelPath)
	if source == "" {
		return fmt.Errorf("source file not found: %s", file.RelPath)
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	// One handle for the whole file: ReadAt is safe for concurrent use, so
	// the chunk-level worker pool can share it.
	reader := func(offset int64, length int) ([]byte, error) {
		buf := make([]byte, length)
		n, err := src.ReadAt(buf, offset)
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
		rt.rate.add(time.Now(), int64(n))
		rt.mu.Unlock()
	}
	var sha HashFunc
	if fut := futures[file.RelPath]; fut != nil {
		sha = fut.get
	}
	return transport.SendFile(ctx, config, target, file, reader, sink, sha)
}

func (m *Manager) executeRecv(ctx context.Context, rt *taskRuntime) error {
	rt.mu.Lock()
	config := rt.config
	rt.mu.Unlock()

	if config.RelayType == RelayKafka {
		err := consumeKafka(ctx, config.Kafka, func(meta ChunkMeta, data []byte) error {
			if err := m.ReceiveChunk(rt.taskID, meta, data); err != nil {
				return err
			}
			m.stopConsumerIfComplete(rt)
			return nil
		})
		// On ctx cancel, reflect the requested pause/cancel state.
		if rt.paused.Load() {
			m.setState(rt, StatePaused)
		} else if rt.canceled.Load() {
			m.setState(rt, StateCancelled)
		}
		return err
	}
	// DIRECT RECV: chunks arrive via the HTTP handler; nothing to do here.
	// The task is already RUNNING (set by Start/Resume/Recover) — do NOT
	// re-assert the state here: this goroutine races with ReceiveChunk,
	// which may already have moved the task to SUCCESS.
	return nil
}

// stopConsumerIfComplete cancels the task context once a RECV task has
// reached SUCCESS, so the KAFKA consume loop exits and the consumer-group
// connection closes instead of polling the topic idly. Chunks for a later
// transfer over the same topic then require restarting the task.
func (m *Manager) stopConsumerIfComplete(rt *taskRuntime) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state == StateSuccess && rt.cancel != nil {
		rt.cancel()
	}
}

// ===================== helpers =====================

func (m *Manager) setState(rt *taskRuntime, state TaskState) {
	rt.mu.Lock()
	now := time.Now()
	if state == StateRunning {
		rt.beginRunLocked(now)
	} else {
		rt.endRunLocked(now, isTerminalState(state))
	}
	rt.state = state
	timing := rt.timingLocked()
	if state == StatePaused || isTerminalState(state) {
		// Force a progress flush: RECV-side persistence is otherwise
		// throttled, and resume correctness depends on these files.
		m.persistProgressLocked(rt, rt.taskID)
	}
	rt.mu.Unlock()
	_ = m.store.saveState(rt.taskID, state)
	_ = m.store.saveTiming(rt.taskID, timing)
	if isTerminalState(state) {
		m.closeKafkaProducerIfIdle()
	}
}

// closeKafkaProducerIfIdle closes the cached KAFKA producer once no RUNNING
// SEND task can still use it, so a finished transfer does not hold broker
// connections open. RECV tasks consume via a consumer group whose connection
// closes when its consume loop exits, so they do not keep the producer alive.
func (m *Manager) closeKafkaProducerIfIdle() {
	t, ok := m.registry.get(RelayKafka)
	if !ok {
		return
	}
	closer, ok := t.(io.Closer)
	if !ok {
		return
	}
	m.mu.RLock()
	for _, other := range m.runtimes {
		other.mu.Lock()
		busy := other.state == StateRunning &&
			other.config.RelayType == RelayKafka && other.config.Role == RoleSend
		other.mu.Unlock()
		if busy {
			m.mu.RUnlock()
			return
		}
	}
	m.mu.RUnlock()
	_ = closer.Close()
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

// buildManifest expands SourcePaths into file entries with sizes. It only
// stats files — deliberately no content reads: hashing a large tree up front
// delayed the first chunk by minutes (a 100GB file = one full disk pass).
// The SHA-256 is computed lazily by a hashPool while the transfer runs and
// attached to the EOF chunk (see resolveSHA256). Directory entries use
// forward-slash relative paths; a single file uses its base name.
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
				manifest = append(manifest, FileEntry{
					RelPath: filepath.ToSlash(rel),
					Size:    fi.Size(),
				})
				return nil
			})
		} else {
			manifest = append(manifest, FileEntry{
				RelPath: filepath.Base(abs),
				Size:    info.Size(),
			})
		}
	}
	return manifest
}

// hashFuture is the pending result of a lazily computed SHA-256: computed
// once by a hashPool, waitable by any number of consumers (multiple targets,
// or a retried EOF chunk).
type hashFuture struct {
	done chan struct{}
	sum  string
	err  error
}

// get blocks until the hash is ready or ctx is done.
func (f *hashFuture) get(ctx context.Context) (string, error) {
	select {
	case <-f.done:
		return f.sum, f.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// hashWorkers bounds concurrent digest computations: hashing already reads
// the whole file once, and more readers mostly thrash the disk the transfer
// is also reading from.
const hashWorkers = 2

// hashPool computes file SHA-256 digests with bounded concurrency so hashing
// overlaps the transfer instead of blocking task start. Workers stop picking
// up new jobs when ctx is done (an in-progress hash runs to completion —
// harmless read-only work); jobs never picked leave their future pending, so
// get must only be called with a cancellable ctx.
type hashPool struct {
	ctx  context.Context
	jobs chan hashJob
}

type hashJob struct {
	path string
	fut  *hashFuture
}

// newHashPool starts the workers. capacity must cover every submit so
// submission never blocks task startup.
func newHashPool(ctx context.Context, workers, capacity int) *hashPool {
	p := &hashPool{ctx: ctx, jobs: make(chan hashJob, capacity)}
	for i := 0; i < workers; i++ {
		go p.work()
	}
	return p
}

func (p *hashPool) work() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case j, ok := <-p.jobs:
			if !ok {
				return
			}
			j.fut.sum, j.fut.err = computeSHA256(j.path)
			close(j.fut.done)
		}
	}
}

func (p *hashPool) submit(path string, fut *hashFuture) {
	p.jobs <- hashJob{path: path, fut: fut}
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

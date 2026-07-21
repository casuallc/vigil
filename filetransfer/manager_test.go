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
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func newTestManager(t *testing.T, roots ...string) *Manager {
	t.Helper()
	m := NewManager(Options{
		DataDir:       t.TempDir(),
		EncryptionKey: testKey,
		Roots:         roots,
	})
	// Stop task goroutines before t.TempDir cleanup removes the data dir;
	// cleanups run LIFO, so this must be registered after t.TempDir().
	t.Cleanup(m.Shutdown)
	return m
}

func TestComputeSHA256(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := computeSHA256(f)
	if err != nil {
		t.Fatalf("computeSHA256: %v", err)
	}
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("sha256 mismatch: got %s want %s", got, want)
	}
}

func TestBuildManifestWalksDirectoryWithForwardSlashes(t *testing.T) {
	src := t.TempDir()
	_ = os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0o644)
	_ = os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("bbbbb"), 0o644)

	manifest := buildManifest(TaskConfig{SourcePaths: []string{src}})

	if len(manifest) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(manifest), manifest)
	}
	byRel := map[string]FileEntry{}
	for _, e := range manifest {
		byRel[e.RelPath] = e
	}
	a, ok := byRel["a.txt"]
	if !ok || a.Size != 3 || a.Sha256 != sha256Hex([]byte("aaa")) {
		t.Fatalf("bad a.txt entry: %+v", a)
	}
	b, ok := byRel["sub/b.txt"] // forward slash regardless of OS
	if !ok || b.Size != 5 {
		t.Fatalf("bad sub/b.txt entry: %+v (keys=%v)", b, keysOf(byRel))
	}
}

func TestBuildManifestSingleFileUsesBaseName(t *testing.T) {
	src := t.TempDir()
	f := filepath.Join(src, "only.txt")
	_ = os.WriteFile(f, []byte("data"), 0o644)

	manifest := buildManifest(TaskConfig{SourcePaths: []string{f}})
	if len(manifest) != 1 || manifest[0].RelPath != "only.txt" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func keysOf(m map[string]FileEntry) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func TestApplyOverwritePolicyOverwrite(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "f.txt")
	part := filepath.Join(dir, "f.txt.part")
	_ = os.WriteFile(final, []byte("old"), 0o644)
	_ = os.WriteFile(part, []byte("new"), 0o644)

	if err := applyOverwritePolicy(final, part, Overwrite); err != nil {
		t.Fatalf("applyOverwritePolicy: %v", err)
	}
	got, _ := os.ReadFile(final)
	if string(got) != "new" {
		t.Fatalf("expected overwrite, got %q", got)
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatal("part file should be gone")
	}
}

func TestApplyOverwritePolicySkipKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "f.txt")
	part := filepath.Join(dir, "f.txt.part")
	_ = os.WriteFile(final, []byte("old"), 0o644)
	_ = os.WriteFile(part, []byte("new"), 0o644)

	if err := applyOverwritePolicy(final, part, Skip); err != nil {
		t.Fatalf("applyOverwritePolicy: %v", err)
	}
	got, _ := os.ReadFile(final)
	if string(got) != "old" {
		t.Fatalf("expected skip to keep old, got %q", got)
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatal("part file should be removed on skip")
	}
}

func TestApplyOverwritePolicyRenameCreatesSuffixed(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "f.txt")
	part := filepath.Join(dir, "f.txt.part")
	_ = os.WriteFile(final, []byte("old"), 0o644)
	_ = os.WriteFile(part, []byte("new"), 0o644)

	if err := applyOverwritePolicy(final, part, Rename); err != nil {
		t.Fatalf("applyOverwritePolicy: %v", err)
	}
	// Original kept; renamed copy created as f_1.txt.
	if got, _ := os.ReadFile(final); string(got) != "old" {
		t.Fatalf("expected original kept, got %q", got)
	}
	renamed := filepath.Join(dir, "f_1.txt")
	if got, _ := os.ReadFile(renamed); string(got) != "new" {
		t.Fatalf("expected f_1.txt with new content, got %q", got)
	}
}

func TestReceiveChunkWritesAndFinalizes(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)

	if err := m.CreateTask(TaskConfig{
		TaskID:          1,
		Role:            RoleRecv,
		RelayType:       RelayDirect,
		TargetDir:       targetDir,
		OverwritePolicy: Overwrite,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	content := []byte("hello world")
	meta := ChunkMeta{
		RelPath: "docs/a.txt",
		Offset:  0,
		Length:  len(content),
		Eof:     true,
		Sha256:  sha256Hex(content),
	}
	if err := m.ReceiveChunk(1, meta, content); err != nil {
		t.Fatalf("ReceiveChunk: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(targetDir, "docs", "a.txt"))
	if err != nil {
		t.Fatalf("read finalized file: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("content mismatch: %q", got)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "docs", "a.txt.part")); !os.IsNotExist(err) {
		t.Fatal(".part file should be gone after eof")
	}

	status, err := m.GetStatus(1)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.CompletedFiles != 1 || status.Progress != 100 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestReceiveChunkResumeAcrossTwoChunks(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 2, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite})

	full := []byte("abcdefghij")
	// First chunk: bytes 0..5, not eof.
	if err := m.ReceiveChunk(2, ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 5, Eof: false}, full[:5]); err != nil {
		t.Fatalf("chunk1: %v", err)
	}
	// Second chunk: bytes 5..10, eof with full-file sha.
	if err := m.ReceiveChunk(2, ChunkMeta{RelPath: "f.bin", Offset: 5, Length: 5, Eof: true, Sha256: sha256Hex(full)}, full[5:]); err != nil {
		t.Fatalf("chunk2: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(targetDir, "f.bin"))
	if string(got) != "abcdefghij" {
		t.Fatalf("resume content mismatch: %q", got)
	}
}

func TestReceiveChunkRejectsTraversal(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 3, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite})

	err := m.ReceiveChunk(3, ChunkMeta{RelPath: "../escape.txt", Offset: 0, Length: 1, Eof: true}, []byte("x"))
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestReceiveChunkSha256MismatchFails(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 4, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite})

	err := m.ReceiveChunk(4, ChunkMeta{RelPath: "a.txt", Offset: 0, Length: 3, Eof: true, Sha256: "deadbeef"}, []byte("abc"))
	if err == nil {
		t.Fatal("expected sha256 mismatch error")
	}
}

// TestReceiveChunkOutOfOrderFinalizes delivers the EOF chunk first (as can
// happen with parallel KAFKA sends) and asserts the file is finalised only
// once the byte ranges cover the whole file.
func TestReceiveChunkOutOfOrderFinalizes(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 5, Role: RoleRecv, RelayType: RelayKafka, TargetDir: targetDir, OverwritePolicy: Overwrite})

	full := []byte("0123456789")
	finalPath := filepath.Join(targetDir, "f.bin")

	// EOF chunk arrives first (bytes 6..10).
	if err := m.ReceiveChunk(5, ChunkMeta{RelPath: "f.bin", Offset: 6, Length: 4, Eof: true, Sha256: sha256Hex(full)}, full[6:]); err != nil {
		t.Fatalf("eof chunk: %v", err)
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("file must not be finalised before all chunks arrive")
	}
	status, err := m.GetStatus(5)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.CompletedFiles != 0 {
		t.Fatalf("expected 0 completed files, got %+v", status)
	}

	// Middle chunk: still incomplete.
	if err := m.ReceiveChunk(5, ChunkMeta{RelPath: "f.bin", Offset: 3, Length: 3}, full[3:6]); err != nil {
		t.Fatalf("mid chunk: %v", err)
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatal("file must not be finalised with a gap")
	}

	// First chunk closes the gap: finalisation happens now.
	if err := m.ReceiveChunk(5, ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 3}, full[:3]); err != nil {
		t.Fatalf("first chunk: %v", err)
	}
	got, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("read finalised file: %v", err)
	}
	if string(got) != string(full) {
		t.Fatalf("content mismatch: %q", got)
	}
	status, err = m.GetStatus(5)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.CompletedFiles != 1 || status.Progress != 100 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

// TestReceiveChunkDuplicateDeliveryCountsOnce ensures redelivered chunks
// (at-least-once Kafka delivery) do not inflate progress.
func TestReceiveChunkDuplicateDeliveryCountsOnce(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 6, Role: RoleRecv, RelayType: RelayKafka, TargetDir: targetDir, OverwritePolicy: Overwrite})

	meta := ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 5}
	for i := 0; i < 3; i++ {
		if err := m.ReceiveChunk(6, meta, []byte("hello")); err != nil {
			t.Fatalf("chunk %d: %v", i, err)
		}
	}
	status, err := m.GetStatus(6)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.TransferredBytes != 5 {
		t.Fatalf("TransferredBytes=%d, want 5", status.TransferredBytes)
	}
}

// countingTransport records the maximum number of concurrent SendFile calls.
type countingTransport struct {
	cur   atomic.Int32
	max   atomic.Int32
	delay time.Duration
}

func (c *countingTransport) Type() RelayType { return RelayDirect }
func (c *countingTransport) SendFile(_ context.Context, _ TaskConfig, _ TargetConfig, _ FileEntry, _ ChunkReader, _ ProgressSink) error {
	n := c.cur.Add(1)
	for {
		old := c.max.Load()
		if n <= old || c.max.CompareAndSwap(old, n) {
			break
		}
	}
	time.Sleep(c.delay)
	c.cur.Add(-1)
	return nil
}

// TestExecuteSendParallelFiles verifies the file-level worker pool actually
// overlaps sends and completes all files.
func TestExecuteSendParallelFiles(t *testing.T) {
	src := t.TempDir()
	for _, name := range []string{"a.bin", "b.bin", "c.bin", "d.bin"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("data-"+name), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
	m := newTestManager(t, src)
	ct := &countingTransport{delay: 50 * time.Millisecond}
	m.registry.register(ct) // replace the real DIRECT transport

	if err := m.CreateTask(TaskConfig{
		TaskID:      7,
		Role:        RoleSend,
		RelayType:   RelayDirect,
		SourcePaths: []string{src},
		Parallelism: 4,
		Targets:     []TargetConfig{{Host: "unused", Port: 1}},
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := m.Start(7); err != nil {
		t.Fatalf("Start: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		status, err := m.GetStatus(7)
		if err != nil {
			t.Fatalf("GetStatus: %v", err)
		}
		if status.State == StateSuccess || status.State == StateFailed || status.State == StatePartialFailed {
			if status.State != StateSuccess {
				t.Fatalf("unexpected end state: %+v", status)
			}
			if status.CompletedFiles != 4 {
				t.Fatalf("CompletedFiles=%d, want 4", status.CompletedFiles)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not finish in time: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := ct.max.Load(); got < 2 {
		t.Fatalf("sends were not concurrent: max in-flight = %d", got)
	}
}

// TestTaskTimingLifecycle verifies started/finished timestamps, the
// pause-frozen elapsed clock, and timing persistence across a restart.
func TestTaskTimingLifecycle(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(Options{DataDir: dir, EncryptionKey: testKey})

	cfg := TaskConfig{TaskID: 8, Role: RoleRecv, RelayType: RelayDirect, TargetDir: t.TempDir()}
	if err := m.CreateTask(cfg); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := m.Start(8); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(30 * time.Millisecond)

	st, err := m.GetStatus(8)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.StartedAt == 0 {
		t.Fatal("startedAt not set while running")
	}
	if st.FinishedAt != 0 {
		t.Fatal("finishedAt set while running")
	}
	if st.ElapsedMs <= 0 {
		t.Fatalf("elapsedMs=%d, want > 0", st.ElapsedMs)
	}

	// Paused: the elapsed clock must freeze.
	if err := m.Pause(8); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	st, _ = m.GetStatus(8)
	frozen := st.ElapsedMs
	time.Sleep(30 * time.Millisecond)
	st, _ = m.GetStatus(8)
	if st.ElapsedMs != frozen {
		t.Fatalf("elapsed moved while paused: %d -> %d", frozen, st.ElapsedMs)
	}

	if err := m.Cancel(8); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	st, _ = m.GetStatus(8)
	if st.FinishedAt == 0 {
		t.Fatal("finishedAt not set after cancel")
	}

	// Timing survives a manager restart.
	m.Shutdown()
	m2 := NewManager(Options{DataDir: dir, EncryptionKey: testKey})
	t.Cleanup(m2.Shutdown)
	if err := m2.Recover(); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	st, err = m2.GetStatus(8)
	if err != nil {
		t.Fatalf("GetStatus after recover: %v", err)
	}
	if st.StartedAt == 0 || st.FinishedAt == 0 {
		t.Fatalf("timing not persisted: %+v", st)
	}
	if st.ElapsedMs != frozen {
		t.Fatalf("elapsedMs after recover = %d, want %d", st.ElapsedMs, frozen)
	}
}

// TestReceiveRateReported verifies the trailing receive rate shows up in
// the task status after chunks land.
func TestReceiveRateReported(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 9, Role: RoleRecv, RelayType: RelayKafka, TargetDir: targetDir, OverwritePolicy: Overwrite})

	if err := m.ReceiveChunk(9, ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 10}, []byte("0123456789")); err != nil {
		t.Fatalf("ReceiveChunk: %v", err)
	}
	st, err := m.GetStatus(9)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.CurrentBytesPerSecond <= 0 {
		t.Fatalf("currentBytesPerSecond=%d, want > 0", st.CurrentBytesPerSecond)
	}
}

// TestReceiveChunkRestartRestoresExactRecvState simulates a service restart
// mid-transfer with an out-of-order hole: the EOF chunk was consumed (and
// its offset committed) before the restart, so it will never be
// redelivered. Recovery must restore the exact ranges + EOF state from
// recvstate.json rather than guessing a contiguous prefix, or the file can
// never finalise.
func TestReceiveChunkRestartRestoresExactRecvState(t *testing.T) {
	dir := t.TempDir()
	targetDir := t.TempDir()
	m := NewManager(Options{DataDir: dir, EncryptionKey: testKey})

	// DIRECT RECV so the task can enter RUNNING without a Kafka broker; the
	// reassembly path (ReceiveChunk) is shared by both relay types.
	if err := m.CreateTask(TaskConfig{TaskID: 21, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := m.Start(21); err != nil {
		t.Fatalf("Start: %v", err)
	}
	full := []byte("0123456789abcde") // 15 bytes, three 5-byte chunks
	// Chunks [0,5) and the EOF chunk [10,15) land; the hole [5,10) does not
	// arrive before the "restart".
	if err := m.ReceiveChunk(21, ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 5}, full[:5]); err != nil {
		t.Fatalf("chunk1: %v", err)
	}
	if err := m.ReceiveChunk(21, ChunkMeta{RelPath: "f.bin", Offset: 10, Length: 5, Eof: true, Sha256: sha256Hex(full)}, full[10:]); err != nil {
		t.Fatalf("eof chunk: %v", err)
	}
	m.Shutdown()

	m2 := NewManager(Options{DataDir: dir, EncryptionKey: testKey})
	t.Cleanup(m2.Shutdown)
	if err := m2.Recover(); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	// A duplicate of [0,5) (harmless), then the hole [5,10) — the EOF chunk
	// itself is NOT redelivered (committed before the restart).
	if err := m2.ReceiveChunk(21, ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 5}, full[:5]); err != nil {
		t.Fatalf("duplicate chunk: %v", err)
	}
	if err := m2.ReceiveChunk(21, ChunkMeta{RelPath: "f.bin", Offset: 5, Length: 5}, full[5:10]); err != nil {
		t.Fatalf("hole chunk: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(targetDir, "f.bin"))
	if err != nil {
		t.Fatalf("read finalised file: %v", err)
	}
	if string(got) != string(full) {
		t.Fatalf("content mismatch: %q", got)
	}
	st, err := m2.GetStatus(21)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.CompletedFiles != 1 || st.Progress != 100 {
		t.Fatalf("unexpected status after restart recovery: %+v", st)
	}
}

// TestReceiveChunkRedeliveryAfterFinalizeSkipped ensures chunks of an
// already-finalised file are dropped without recreating an orphan .part.
func TestReceiveChunkRedeliveryAfterFinalizeSkipped(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 22, Role: RoleRecv, RelayType: RelayKafka, TargetDir: targetDir, OverwritePolicy: Overwrite})

	content := []byte("done")
	meta := ChunkMeta{RelPath: "f.txt", Offset: 0, Length: 4, Eof: true, Sha256: sha256Hex(content)}
	if err := m.ReceiveChunk(22, meta, content); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "f.txt")); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	// Redelivered after finalisation: must not recreate the .part file.
	if err := m.ReceiveChunk(22, meta, content); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "f.txt.part")); !os.IsNotExist(err) {
		t.Fatal("orphan .part file recreated after finalise")
	}
}

// TestReceiveChunkZeroByteFileFinalizes verifies a zero-byte file (single
// empty EOF chunk) is created and marked completed.
func TestReceiveChunkZeroByteFileFinalizes(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 23, Role: RoleRecv, RelayType: RelayKafka, TargetDir: targetDir, OverwritePolicy: Overwrite})

	emptySha := sha256Hex(nil)
	if err := m.ReceiveChunk(23, ChunkMeta{RelPath: "empty.bin", Offset: 0, Length: 0, Eof: true, Sha256: emptySha}, nil); err != nil {
		t.Fatalf("ReceiveChunk: %v", err)
	}
	info, err := os.Stat(filepath.Join(targetDir, "empty.bin"))
	if err != nil {
		t.Fatalf("zero-byte file not created: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("zero-byte file size = %d", info.Size())
	}
	st, err := m.GetStatus(23)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.CompletedFiles != 1 {
		t.Fatalf("unexpected status: %+v", st)
	}
}

// TestReceiveChunkTaskSuccessAndResumeOnNewFile verifies a RECV task
// transitions to SUCCESS once every known file is completed, and flips back
// to RUNNING when chunks of a NEW file arrive (another transfer over the
// same channel).
func TestReceiveChunkTaskSuccessAndResumeOnNewFile(t *testing.T) {
	targetDir := t.TempDir()
	m := newTestManager(t, targetDir)
	_ = m.CreateTask(TaskConfig{TaskID: 24, Role: RoleRecv, RelayType: RelayDirect, TargetDir: targetDir, OverwritePolicy: Overwrite})
	if err := m.Start(24); err != nil {
		t.Fatalf("Start: %v", err)
	}

	c1 := []byte("first")
	if err := m.ReceiveChunk(24, ChunkMeta{RelPath: "a.txt", Offset: 0, Length: 5, Eof: true, Sha256: sha256Hex(c1)}, c1); err != nil {
		t.Fatalf("file a: %v", err)
	}
	st, err := m.GetStatus(24)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.State != StateSuccess {
		t.Fatalf("state=%s, want SUCCESS after all files completed", st.State)
	}
	if st.FinishedAt == 0 {
		t.Fatal("finishedAt not set on SUCCESS")
	}

	// A new file arrives (second transfer): task must resume RUNNING.
	c2 := []byte("secondfile!")
	if err := m.ReceiveChunk(24, ChunkMeta{RelPath: "b.txt", Offset: 0, Length: 6}, c2[:6]); err != nil {
		t.Fatalf("file b chunk1: %v", err)
	}
	st, _ = m.GetStatus(24)
	if st.State != StateRunning {
		t.Fatalf("state=%s, want RUNNING after new file arrived", st.State)
	}
	if st.FinishedAt != 0 {
		t.Fatal("finishedAt must be cleared while running again")
	}

	if err := m.ReceiveChunk(24, ChunkMeta{RelPath: "b.txt", Offset: 6, Length: 6, Eof: true, Sha256: sha256Hex(c2)}, c2[6:]); err != nil {
		t.Fatalf("file b chunk2: %v", err)
	}
	st, _ = m.GetStatus(24)
	if st.State != StateSuccess || st.CompletedFiles != 2 {
		t.Fatalf("unexpected final status: %+v", st)
	}
}

func TestCreateTaskRejectsDuplicate(t *testing.T) {
	m := newTestManager(t)
	cfg := TaskConfig{TaskID: 10, Role: RoleSend, RelayType: RelayDirect}
	if err := m.CreateTask(cfg); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := m.CreateTask(cfg); err == nil {
		t.Fatal("expected duplicate task creation to fail")
	}
}

func TestUpdateTaskRejectedWhileRunning(t *testing.T) {
	m := newTestManager(t)
	cfg := TaskConfig{TaskID: 10, Role: RoleRecv, RelayType: RelayDirect, TargetDir: t.TempDir()}
	_ = m.CreateTask(cfg)
	// Starting a DIRECT RECV task moves it to RUNNING (it then waits for
	// chunks via HTTP), which must block config updates.
	if err := m.Start(10); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.UpdateTask(10, cfg); err == nil {
		t.Fatal("expected update to be rejected while RUNNING")
	}
}

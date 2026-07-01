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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func newTestManager(t *testing.T, roots ...string) *Manager {
	t.Helper()
	return NewManager(Options{
		DataDir:       t.TempDir(),
		EncryptionKey: testKey,
		Roots:         roots,
	})
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

func TestCreateTaskRejectsDuplicate(t *testing.T) {
	m := newTestManager(t)
	cfg := TaskConfig{TaskID: 9, Role: RoleSend, RelayType: RelayDirect}
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

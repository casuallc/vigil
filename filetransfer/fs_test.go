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
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSafeRejectsEmptyPath(t *testing.T) {
	fs := newFS([]string{t.TempDir()})
	if _, err := fs.resolveSafe(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestResolveSafeRejectsDotDot(t *testing.T) {
	root := t.TempDir()
	fs := newFS([]string{root})
	if _, err := fs.resolveSafe(filepath.Join(root, "..", "escape")); err == nil {
		t.Fatal("expected error for path containing ..")
	}
}

func TestResolveSafeAllowsPathInsideRoot(t *testing.T) {
	root := t.TempDir()
	fs := newFS([]string{root})
	child := filepath.Join(root, "sub", "file.txt")
	if _, err := fs.resolveSafe(child); err != nil {
		t.Fatalf("expected child of root to be allowed, got %v", err)
	}
}

func TestResolveSafeRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir() // a different directory tree
	fs := newFS([]string{root})
	if _, err := fs.resolveSafe(filepath.Join(other, "file.txt")); err == nil {
		t.Fatal("expected error for path outside root")
	}
}

func TestResolveSafeRejectsPrefixSiblingNotChild(t *testing.T) {
	root := t.TempDir()
	// A sibling whose path shares a string prefix with root must not pass
	// (e.g. root "/x/data" should not admit "/x/database").
	sibling := root + "_extra"
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}
	fs := newFS([]string{root})
	if _, err := fs.resolveSafe(filepath.Join(sibling, "f.txt")); err == nil {
		t.Fatal("expected prefix-sibling to be rejected")
	}
}

func TestResolveSafeEmptyRootsDefaultsToHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	fs := newFS(nil)

	if _, err := fs.resolveSafe(filepath.Join(home, "somefile")); err != nil {
		t.Fatalf("expected path under home to be allowed, got %v", err)
	}
	// The volume/filesystem root is an ancestor of home, hence outside it.
	outside := filepath.VolumeName(home) + string(os.PathSeparator)
	if _, err := fs.resolveSafe(outside); err == nil {
		t.Fatal("expected path outside home to be rejected")
	}
}

func TestListReturnsEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fs := newFS([]string{root})

	items, err := fs.list(root)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	byName := map[string]FsItem{}
	for _, it := range items {
		byName[it.Name] = it
	}
	if f, ok := byName["a.txt"]; !ok || f.IsDir || f.Size != 5 {
		t.Fatalf("unexpected file entry: %+v", f)
	}
	if d, ok := byName["sub"]; !ok || !d.IsDir || d.Size != 0 {
		t.Fatalf("unexpected dir entry: %+v", d)
	}
}

func TestListOnFileFails(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.txt")
	_ = os.WriteFile(file, []byte("x"), 0o644)
	fs := newFS([]string{root})
	if _, err := fs.list(file); err == nil {
		t.Fatal("expected error listing a non-directory")
	}
}

func TestStatFileAndDir(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "a.txt"), []byte("12345"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("678"), 0o644)
	fs := newFS([]string{root})

	fileStat, err := fs.stat(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if fileStat["isDir"].(bool) || fileStat["size"].(int64) != 5 {
		t.Fatalf("unexpected file stat: %+v", fileStat)
	}

	dirStat, err := fs.stat(root)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !dirStat["isDir"].(bool) {
		t.Fatalf("expected dir")
	}
	if dirStat["fileCount"].(int64) != 2 || dirStat["totalSize"].(int64) != 8 {
		t.Fatalf("unexpected dir stat: %+v", dirStat)
	}
}

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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// FS provides directory browsing with a path jail. Access is confined to the
// configured roots, or the user's home directory when roots is empty.
type FS struct {
	roots []string
}

func newFS(roots []string) *FS {
	return &FS{roots: roots}
}

// realize returns an absolute, cleaned, symlink-resolved path. If the path
// does not exist (so symlinks cannot be resolved) it falls back to the
// absolute cleaned form.
func realize(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

// within reports whether target is root itself or a descendant of root, using
// a separator-aware comparison so "/x/data" does not admit "/x/database".
func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// allowedRoots returns the effective jail roots: the configured roots, or the
// user's home directory when none are configured.
func (fs *FS) allowedRoots() ([]string, error) {
	if len(fs.roots) > 0 {
		return fs.roots, nil
	}
	home, err := defaultHome()
	if err != nil {
		return nil, fmt.Errorf("user home not available: %w", err)
	}
	return []string{home}, nil
}

// defaultHome resolves the current user's home directory. It prefers
// os.UserHomeDir() (the $HOME env var), but falls back to the OS user database
// when $HOME is unset — which is common when the server runs as a daemon (e.g.
// under systemd or `su`) with no login environment.
func defaultHome() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home, nil
	}
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return u.HomeDir, nil
	}
	return "", fmt.Errorf("$HOME is not defined")
}

// resolveSafe validates path against the jail and returns the resolved
// absolute path: reject blank, reject any ".." segment, then require
// containment in a root.
func (fs *FS) resolveSafe(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	resolved := realize(path)

	roots, err := fs.allowedRoots()
	if err != nil {
		return "", err
	}
	for _, root := range roots {
		if within(realize(root), resolved) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("path escapes allowed roots: %s", path)
}

// list returns the entries of a directory.
func (fs *FS) list(path string) ([]FsItem, error) {
	resolved, err := fs.resolveSafe(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", path)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}
	items := make([]FsItem, 0, len(entries))
	for _, e := range entries {
		item := FsItem{Name: e.Name(), IsDir: e.IsDir()}
		if fi, err := e.Info(); err == nil {
			if !item.IsDir {
				item.Size = fi.Size()
			}
			item.Mtime = fi.ModTime().UnixMilli()
		}
		items = append(items, item)
	}
	return items, nil
}

// stat returns file/directory statistics: isDir, size, fileCount, totalSize.
// For directories size is 0 and fileCount/totalSize aggregate recursively; for
// files fileCount/totalSize are 0.
func (fs *FS) stat(path string) (map[string]interface{}, error) {
	resolved, err := fs.resolveSafe(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	if info.IsDir() {
		var fileCount, totalSize int64
		walkErr := filepath.Walk(resolved, func(_ string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fi.IsDir() {
				fileCount++
				totalSize += fi.Size()
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
		result["isDir"] = true
		result["size"] = int64(0)
		result["fileCount"] = fileCount
		result["totalSize"] = totalSize
	} else {
		result["isDir"] = false
		result["size"] = info.Size()
		result["fileCount"] = int64(0)
		result["totalSize"] = int64(0)
	}
	return result, nil
}

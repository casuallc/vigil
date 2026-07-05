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

package dockerregistry

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Manager is a filesystem-backed Docker Registry V2 storage manager.
type Manager struct {
	root string
	mu   sync.RWMutex
}

// NewManager creates a new registry manager rooted at root.
func NewManager(root string) (*Manager, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("invalid registry root: %w", err)
	}
	m := &Manager{root: abs}
	for _, dir := range []string{m.repositoriesDir(), m.blobsDir(), m.uploadsDir()} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("create registry directory %s: %w", dir, err)
		}
	}
	return m, nil
}

// Close cleans up any in-progress resources. For the filesystem backend this is a no-op.
func (m *Manager) Close() error {
	return nil
}

func (m *Manager) repositoriesDir() string { return filepath.Join(m.root, "repositories") }
func (m *Manager) blobsDir() string        { return filepath.Join(m.root, "blobs") }
func (m *Manager) uploadsDir() string      { return filepath.Join(m.root, "uploads") }

// validateName ensures a repository name is safe and well-formed.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("repository name cannot start or end with '/'")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("repository name cannot contain '..'")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("repository name cannot be an absolute path")
	}
	segments := strings.Split(name, "/")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("repository name contains empty segment")
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("repository name contains invalid segment %q", seg)
		}
		if strings.Contains(seg, ":") {
			return fmt.Errorf("repository name segment cannot contain ':'")
		}
	}
	return nil
}

// repoPath returns the resolved directory for a repository.
func (m *Manager) repoPath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	base := m.repositoriesDir()
	full := filepath.Join(base, filepath.FromSlash(name))
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	// Ensure containment within repositories dir.
	if !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("repository path escapes root: %s", name)
	}
	return abs, nil
}

// blobPath returns the resolved directory for a blob.
func (m *Manager) blobPath(digest string) (string, error) {
	algo, hexStr, err := ParseDigest(digest)
	if err != nil {
		return "", err
	}
	base := m.blobsDir()
	full := filepath.Join(base, algo, hexStr)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("blob path escapes root: %s", digest)
	}
	return abs, nil
}

// uploadPath returns the resolved directory for an upload session.
func (m *Manager) uploadPath(uuid string) (string, error) {
	if uuid == "" || strings.Contains(uuid, "..") || strings.Contains(uuid, string(filepath.Separator)) || strings.Contains(uuid, "/") {
		return "", fmt.Errorf("invalid upload uuid: %s", uuid)
	}
	base := m.uploadsDir()
	full := filepath.Join(base, uuid)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("upload path escapes root: %s", uuid)
	}
	return abs, nil
}

// Repositories lists all repository names.
func (m *Manager) Repositories() ([]string, error) {
	base := m.repositoriesDir()
	var repos []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		manifestsDir := filepath.Join(path, "_manifests")
		if _, err := os.Stat(manifestsDir); err == nil {
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			repos = append(repos, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(repos)
	return repos, nil
}

// Tags lists tags for a repository.
func (m *Manager) Tags(name string) ([]string, error) {
	rp, err := m.repoPath(name)
	if err != nil {
		return nil, err
	}
	tagsFile := filepath.Join(rp, "_tags.json")
	data, err := os.ReadFile(tagsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	var tr TagsResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, err
	}
	return tr.Tags, nil
}

// GetManifest returns a manifest by tag or digest.
func (m *Manager) GetManifest(name, reference string) (mediaType string, body []byte, digest string, err error) {
	rp, err := m.repoPath(name)
	if err != nil {
		return "", nil, "", err
	}

	manifestsDir := filepath.Join(rp, "_manifests")

	var tagFile string
	if isDigest(reference) {
		// Find tag whose digest matches reference.
		idx, err := readIndex(rp)
		if err != nil {
			return "", nil, "", err
		}
		found := false
		for tag, d := range idx {
			if d == reference {
				tagFile = filepath.Join(manifestsDir, tag)
				digest = d
				found = true
				break
			}
		}
		if !found {
			return "", nil, "", ErrManifestUnknown
		}
	} else {
		tagFile = filepath.Join(manifestsDir, reference)
	}

	body, err = os.ReadFile(tagFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil, "", ErrManifestUnknown
		}
		return "", nil, "", err
	}
	mt, err := os.ReadFile(tagFile + ".mediaType")
	if err != nil {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	} else {
		mediaType = string(mt)
	}
	if digest == "" {
		d, err := DigestFromReader(bytes.NewReader(body))
		if err != nil {
			return "", nil, "", err
		}
		digest = d
	}
	return mediaType, body, digest, nil
}

// PutManifest stores a manifest for a tag.
func (m *Manager) PutManifest(name, reference, mediaType string, body []byte) (string, error) {
	if isDigest(reference) {
		return "", fmt.Errorf("cannot store manifest by digest reference: %s", reference)
	}
	rp, err := m.repoPath(name)
	if err != nil {
		return "", err
	}

	digest, err := DigestFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	manifestsDir := filepath.Join(rp, "_manifests")
	if err := os.MkdirAll(manifestsDir, 0750); err != nil {
		return "", err
	}

	// Write manifest atomically.
	manifestPath := filepath.Join(manifestsDir, reference)
	if err := writeFileAtomic(manifestPath, body); err != nil {
		return "", err
	}
	if err := writeFileAtomic(manifestPath+".mediaType", []byte(mediaType)); err != nil {
		return "", err
	}

	// Update index and tags list.
	m.mu.Lock()
	defer m.mu.Unlock()
	idx, err := readIndex(rp)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	idx[reference] = digest
	if err := writeIndex(rp, idx); err != nil {
		return "", err
	}

	tags, err := m.Tags(name)
	if err != nil {
		return "", err
	}
	if !contains(tags, reference) {
		tags = append(tags, reference)
		sort.Strings(tags)
	}
	if err := writeTags(rp, name, tags); err != nil {
		return "", err
	}

	return digest, nil
}

// DeleteManifest removes a manifest tag.
func (m *Manager) DeleteManifest(name, reference string) error {
	rp, err := m.repoPath(name)
	if err != nil {
		return err
	}
	manifestsDir := filepath.Join(rp, "_manifests")
	manifestPath := filepath.Join(manifestsDir, reference)
	if _, err := os.Stat(manifestPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrManifestUnknown
		}
		return err
	}
	if err := os.Remove(manifestPath); err != nil {
		return err
	}
	_ = os.Remove(manifestPath + ".mediaType")

	m.mu.Lock()
	defer m.mu.Unlock()
	idx, err := readIndex(rp)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(idx, reference)
	if err := writeIndex(rp, idx); err != nil {
		return err
	}

	tags, err := m.Tags(name)
	if err != nil {
		return err
	}
	tags = remove(tags, reference)
	if err := writeTags(rp, name, tags); err != nil {
		return err
	}
	return nil
}

// StatBlob returns the size of a blob if it exists.
func (m *Manager) StatBlob(name, digest string) (int64, error) {
	// Name is validated on push/pull; existence does not depend on it for blobs.
	bp, err := m.blobPath(digest)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(filepath.Join(bp, "data"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, ErrBlobUnknown
		}
		return 0, err
	}
	return info.Size(), nil
}

// OpenBlob opens a blob for reading and returns its size.
func (m *Manager) OpenBlob(name, digest string) (io.ReadCloser, int64, error) {
	bp, err := m.blobPath(digest)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(filepath.Join(bp, "data"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, ErrBlobUnknown
		}
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// CreateUpload initializes a new blob upload session.
func (m *Manager) CreateUpload(name string) (string, error) {
	if _, err := m.repoPath(name); err != nil {
		return "", err
	}
	uuid := newUUID()
	up, err := m.uploadPath(uuid)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(up, 0750); err != nil {
		return "", err
	}
	state := UploadState{Offset: 0, Started: time.Now()}
	if err := writeUploadState(up, state); err != nil {
		return "", err
	}
	// Create empty data file.
	f, err := os.Create(filepath.Join(up, "data"))
	if err != nil {
		return "", err
	}
	_ = f.Close()
	return uuid, nil
}

// ReadUploadState returns the current state of an upload.
func (m *Manager) ReadUploadState(uuid string) (UploadState, error) {
	up, err := m.uploadPath(uuid)
	if err != nil {
		return UploadState{}, err
	}
	return readUploadState(up)
}

// WriteUploadChunk appends data to an upload at the expected offset.
func (m *Manager) WriteUploadChunk(uuid string, r io.Reader, offset int64) (int64, error) {
	up, err := m.uploadPath(uuid)
	if err != nil {
		return 0, err
	}
	state, err := readUploadState(up)
	if err != nil {
		return 0, ErrBlobUploadUnknown
	}
	if state.Offset != offset {
		return 0, ErrBlobUploadInvalid
	}
	dataPath := filepath.Join(up, "data")
	f, err := os.OpenFile(dataPath, os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, digest, err := HashCopy(f, r)
	if err != nil {
		return n, err
	}
	_ = digest
	state.Offset += n
	if err := writeUploadState(up, state); err != nil {
		return n, err
	}
	return n, nil
}

// CompleteUpload finalizes a blob upload, verifies the digest, and moves it to blob storage.
func (m *Manager) CompleteUpload(uuid, digest string) (string, error) {
	up, err := m.uploadPath(uuid)
	if err != nil {
		return "", err
	}
	if _, err := readUploadState(up); err != nil {
		return "", ErrBlobUploadUnknown
	}
	dataPath := filepath.Join(up, "data")

	computed, err := DigestFromFile(dataPath)
	if err != nil {
		return "", err
	}
	if computed != digest {
		return "", ErrDigestInvalid
	}

	bp, err := m.blobPath(digest)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(bp, 0750); err != nil {
		return "", err
	}
	dest := filepath.Join(bp, "data")
	if err := os.Rename(dataPath, dest); err != nil {
		// If blob already exists, just remove the upload data.
		if _, statErr := os.Stat(dest); statErr == nil {
			_ = os.RemoveAll(up)
			return digest, nil
		}
		return "", err
	}
	if err := writeFileAtomic(filepath.Join(bp, "digest"), []byte(digest)); err != nil {
		return "", err
	}
	_ = os.RemoveAll(up)
	return digest, nil
}

// DeleteUpload cancels an in-progress upload.
func (m *Manager) DeleteUpload(uuid string) error {
	up, err := m.uploadPath(uuid)
	if err != nil {
		return err
	}
	return os.RemoveAll(up)
}

// DeleteBlob removes a blob from storage.
func (m *Manager) DeleteBlob(name, digest string) error {
	bp, err := m.blobPath(digest)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(bp, "data")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrBlobUnknown
		}
		return err
	}
	return os.RemoveAll(bp)
}

// Sentinel errors matching Docker Registry error codes.
var (
	ErrNameUnknown       = errors.New(ErrCodeNameUnknown)
	ErrManifestUnknown   = errors.New(ErrCodeManifestUnknown)
	ErrBlobUnknown       = errors.New(ErrCodeBlobUnknown)
	ErrDigestInvalid     = errors.New(ErrCodeDigestInvalid)
	ErrBlobUploadInvalid = errors.New(ErrCodeBlobUploadInvalid)
	ErrBlobUploadUnknown = errors.New(ErrCodeBlobUploadUnknown)
)

func isDigest(s string) bool {
	return strings.Contains(s, ":")
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readIndex(rp string) (map[string]string, error) {
	path := filepath.Join(rp, "_index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	idx := map[string]string{}
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func writeIndex(rp string, idx map[string]string) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(rp, "_index.json"), data)
}

func writeTags(rp, name string, tags []string) error {
	tr := TagsResponse{Name: name, Tags: tags}
	data, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(rp, "_tags.json"), data)
}

func readUploadState(up string) (UploadState, error) {
	data, err := os.ReadFile(filepath.Join(up, "state.json"))
	if err != nil {
		return UploadState{}, err
	}
	var s UploadState
	if err := json.Unmarshal(data, &s); err != nil {
		return UploadState{}, err
	}
	return s, nil
}

func writeUploadState(up string, state UploadState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(up, "state.json"), data)
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func remove(ss []string, s string) []string {
	out := ss[:0]
	for _, v := range ss {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

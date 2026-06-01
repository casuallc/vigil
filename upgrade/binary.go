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

package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	newBinaryName = "bbx-server.new"
)

// BinaryManager handles binary download, verification, and replacement
type BinaryManager struct {
	dataDir string
}

// NewBinaryManager creates a new binary manager
func NewBinaryManager(dataDir string) *BinaryManager {
	return &BinaryManager{dataDir: dataDir}
}

// NewBinaryPath returns the path for the new binary
func (bm *BinaryManager) NewBinaryPath() string {
	return filepath.Join(bm.dataDir, newBinaryName)
}

// SaveFromReader saves binary data from a reader to the temp location
func (bm *BinaryManager) SaveFromReader(r io.Reader) (string, error) {
	path := bm.NewBinaryPath()
	if err := os.MkdirAll(bm.dataDir, 0755); err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(path)
		return "", err
	}

	// Make executable
	if err := os.Chmod(path, 0755); err != nil {
		os.Remove(path)
		return "", err
	}

	// Return absolute path to ensure fork/exec works regardless of working directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // fallback to relative
	}
	return absPath, nil
}

// Download downloads a binary from a URL
func (bm *BinaryManager) Download(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return bm.SaveFromReader(resp.Body)
}

// VerifyChecksum verifies the SHA256 checksum of the binary
func (bm *BinaryManager) VerifyChecksum(path, checksum string) error {
	if checksum == "" {
		return nil // skip verification if no checksum provided
	}

	// Parse checksum format: "sha256:abc123..." or just "abc123..."
	expected := checksum
	if strings.HasPrefix(checksum, "sha256:") {
		expected = strings.TrimPrefix(checksum, "sha256:")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// VerifyExecutable checks if the file is a valid executable
func (bm *BinaryManager) VerifyExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("not a file")
	}

	// Check if executable
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("file is not executable")
		}
	}

	// Verify it's a valid binary by checking magic bytes
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := f.Read(magic); err != nil {
		return err
	}

	// ELF: 0x7f 'E' 'L' 'F'
	// PE:  'M' 'Z'
	// Mach-O: 0xFE 0xED 0xFA 0xCE or 0xCF 0xFA 0xED 0xFE
	isELF := magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
	isPE := magic[0] == 'M' && magic[1] == 'Z'
	isMachO := (magic[0] == 0xFE && magic[1] == 0xED && magic[2] == 0xFA && magic[3] == 0xCE) ||
		(magic[0] == 0xCF && magic[1] == 0xFA && magic[2] == 0xED && magic[3] == 0xFE)

	if !isELF && !isPE && !isMachO {
		return fmt.Errorf("not a valid executable (unrecognized format)")
	}

	return nil
}

// Cleanup removes the temporary binary
func (bm *BinaryManager) Cleanup() error {
	path := bm.NewBinaryPath()
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}

// Replace moves the new binary to the standard path
func (bm *BinaryManager) Replace(standardPath string) error {
	return replaceBinary(bm.NewBinaryPath(), standardPath)
}

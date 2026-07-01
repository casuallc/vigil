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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const gcmIVLength = 12

const (
	defaultDataDirName = ".admq-file-transfer-agent"
	tasksDirName       = "tasks"
	configFileName     = "config.json"
	stateFileName      = "state.json"
	progressFileName   = "progress.json"
)

// deriveAESKey returns the first 16 bytes of the configured key as the AES-128
// key, matching the Java agent (key.substring(0,16) UTF-8 bytes).
func deriveAESKey(key string) ([]byte, error) {
	b := []byte(key)
	if len(b) < 16 {
		return nil, fmt.Errorf("encryption_key must be at least 16 bytes, got %d", len(b))
	}
	return b[:16], nil
}

// encryptField encrypts plaintext with AES-128-GCM and returns
// base64(iv || ciphertext+tag). Blank input passes through unchanged, and any
// failure degrades to returning the plaintext (matching Java behaviour).
func encryptField(plaintext string, key []byte) string {
	if plaintext == "" {
		return plaintext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return plaintext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}
	iv := make([]byte, gcmIVLength)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return plaintext
	}
	// Seal appends the ciphertext+tag to iv, giving iv || ciphertext+tag.
	sealed := gcm.Seal(iv, iv, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed)
}

// decryptField reverses encryptField. Blank input passes through, and any
// value that is not our ciphertext is returned unchanged (plaintext fallback).
func decryptField(ciphertext string, key []byte) string {
	if ciphertext == "" {
		return ciphertext
	}
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil || len(decoded) <= gcmIVLength {
		return ciphertext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ciphertext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertext
	}
	iv, enc := decoded[:gcmIVLength], decoded[gcmIVLength:]
	plain, err := gcm.Open(nil, iv, enc, nil)
	if err != nil {
		return ciphertext
	}
	return string(plain)
}

// Store persists task config/state/progress under {dataDir}/tasks/{taskId}/,
// mirroring the Java agent's on-disk layout. Sensitive config fields are
// AES-GCM encrypted at rest.
type Store struct {
	tasksDir string
	encKey   string
}

// newStore resolves the tasks directory: an empty dataDir defaults to
// ~/.admq-file-transfer-agent (matching Java), otherwise {dataDir}/tasks.
func newStore(dataDir, encKey string) *Store {
	var tasksDir string
	if dataDir == "" {
		home, err := defaultHome()
		if err != nil {
			home = "."
		}
		tasksDir = filepath.Join(home, defaultDataDirName, tasksDirName)
	} else {
		tasksDir = filepath.Join(dataDir, tasksDirName)
	}
	return &Store{tasksDir: tasksDir, encKey: encKey}
}

func (s *Store) taskDir(taskID int64) string {
	return filepath.Join(s.tasksDir, strconv.FormatInt(taskID, 10))
}

func (s *Store) writeJSONFile(taskID int64, name string, v interface{}) error {
	dir := s.taskDir(taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

// saveConfig writes config.json with sensitive fields encrypted. The caller's
// config is never mutated (sensitive fields are encrypted on a deep copy).
func (s *Store) saveConfig(taskID int64, cfg TaskConfig) error {
	key, keyErr := deriveAESKey(s.encKey)
	toSave := cfg
	// Deep-copy the slices/pointers we mutate so the in-memory config keeps
	// its plaintext secrets.
	if cfg.Targets != nil {
		toSave.Targets = make([]TargetConfig, len(cfg.Targets))
		copy(toSave.Targets, cfg.Targets)
		if keyErr == nil {
			for i := range toSave.Targets {
				toSave.Targets[i].AuthPass = encryptField(toSave.Targets[i].AuthPass, key)
			}
		}
	}
	if cfg.Kafka != nil {
		k := *cfg.Kafka
		if keyErr == nil {
			k.Password = encryptField(k.Password, key)
		}
		toSave.Kafka = &k
	}
	return s.writeJSONFile(taskID, configFileName, toSave)
}

// loadConfig reads and decrypts config.json. Returns (nil, nil) when absent.
func (s *Store) loadConfig(taskID int64) (*TaskConfig, error) {
	data, err := os.ReadFile(filepath.Join(s.taskDir(taskID), configFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg TaskConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if key, keyErr := deriveAESKey(s.encKey); keyErr == nil {
		for i := range cfg.Targets {
			cfg.Targets[i].AuthPass = decryptField(cfg.Targets[i].AuthPass, key)
		}
		if cfg.Kafka != nil {
			cfg.Kafka.Password = decryptField(cfg.Kafka.Password, key)
		}
	}
	return &cfg, nil
}

// saveState writes state.json as a JSON string (e.g. "RUNNING").
func (s *Store) saveState(taskID int64, state TaskState) error {
	return s.writeJSONFile(taskID, stateFileName, string(state))
}

// loadState reads state.json. Returns ("", nil) when absent.
func (s *Store) loadState(taskID int64) (TaskState, error) {
	data, err := os.ReadFile(filepath.Join(s.taskDir(taskID), stateFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var state string
	if err := json.Unmarshal(data, &state); err != nil {
		return "", err
	}
	return TaskState(state), nil
}

// saveProgress writes progress.json as a JSON array of FileProgress.
func (s *Store) saveProgress(taskID int64, progress []FileProgress) error {
	if progress == nil {
		progress = []FileProgress{}
	}
	return s.writeJSONFile(taskID, progressFileName, progress)
}

// loadProgress reads progress.json. Returns an empty slice when absent.
func (s *Store) loadProgress(taskID int64) ([]FileProgress, error) {
	data, err := os.ReadFile(filepath.Join(s.taskDir(taskID), progressFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return []FileProgress{}, nil
		}
		return nil, err
	}
	var progress []FileProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// deleteTask removes the entire {taskId}/ directory.
func (s *Store) deleteTask(taskID int64) error {
	return os.RemoveAll(s.taskDir(taskID))
}

// listTaskIDs returns the numeric task directory names, skipping any others.
func (s *Store) listTaskIDs() ([]int64, error) {
	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, err := strconv.ParseInt(e.Name(), 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

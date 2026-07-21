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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const gcmIVLength = 12

const (
	defaultDataDirName = ".vigil-file-transfer"
	legacyDataDirName  = ".admq-file-transfer-agent"
	tasksDirName       = "tasks"
	configFileName     = "config.json"
	stateFileName      = "state.json"
	progressFileName   = "progress.json"
	timingFileName     = "timing.json"
	recvStateFileName  = "recvstate.json"
)

// deriveAESKey hashes the configured key with SHA-256, yielding the 32-byte
// AES-256 key. An empty key means encryption is disabled (fields are stored
// as plaintext).
func deriveAESKey(key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption_key is not set")
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:], nil
}

// encryptField encrypts plaintext with AES-256-GCM and returns
// base64(iv || ciphertext+tag). Blank input passes through unchanged.
func encryptField(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv := make([]byte, gcmIVLength)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	// Seal appends the ciphertext+tag to iv, giving iv || ciphertext+tag.
	sealed := gcm.Seal(iv, iv, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// decryptField reverses encryptField. Blank input passes through, and a value
// that is not our ciphertext (legacy plaintext) is returned unchanged. A value
// that has the ciphertext shape but fails to decrypt — e.g. it was encrypted
// with a different key or an incompatible format — is an error.
func decryptField(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" {
		return ciphertext, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil || len(decoded) <= gcmIVLength {
		return ciphertext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv, enc := decoded[:gcmIVLength], decoded[gcmIVLength:]
	plain, err := gcm.Open(nil, iv, enc, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt field: %w", err)
	}
	return string(plain), nil
}

// Store persists task config/state/progress under {dataDir}/tasks/{taskId}/.
// Sensitive config fields are AES-GCM encrypted at rest.
type Store struct {
	tasksDir string
	encKey   string
}

// newStore resolves the tasks directory: an empty dataDir defaults to
// ~/.vigil-file-transfer, otherwise {dataDir}/tasks.
func newStore(dataDir, encKey string) *Store {
	var tasksDir string
	if dataDir == "" {
		home, err := defaultHome()
		if err != nil {
			home = "."
		}
		tasksDir = filepath.Join(migrateLegacyDataDir(home), tasksDirName)
	} else {
		tasksDir = filepath.Join(dataDir, tasksDirName)
	}
	return &Store{tasksDir: tasksDir, encKey: encKey}
}

// migrateLegacyDataDir renames a legacy default data directory
// (~/.admq-file-transfer-agent) left by older builds to the current default
// when the new directory does not exist yet, and returns the effective base
// directory.
func migrateLegacyDataDir(home string) string {
	newDir := filepath.Join(home, defaultDataDirName)
	if _, err := os.Stat(newDir); err == nil {
		return newDir
	}
	legacyDir := filepath.Join(home, legacyDataDirName)
	if _, err := os.Stat(legacyDir); err == nil {
		if err := os.Rename(legacyDir, newDir); err != nil {
			log.Printf("filetransfer: cannot migrate legacy data dir %s -> %s: %v", legacyDir, newDir, err)
		} else {
			log.Printf("filetransfer: migrated legacy data dir %s -> %s", legacyDir, newDir)
		}
	}
	return newDir
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
				enc, err := encryptField(toSave.Targets[i].AuthPass, key)
				if err != nil {
					return fmt.Errorf("encrypt target authPass: %w", err)
				}
				toSave.Targets[i].AuthPass = enc
			}
		}
	}
	if cfg.Kafka != nil {
		k := *cfg.Kafka
		if keyErr == nil {
			enc, err := encryptField(k.Password, key)
			if err != nil {
				return fmt.Errorf("encrypt kafka password: %w", err)
			}
			k.Password = enc
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
			dec, err := decryptField(cfg.Targets[i].AuthPass, key)
			if err != nil {
				return nil, fmt.Errorf("task %d: decrypt target authPass (wrong encryption_key or incompatible stored format): %w", taskID, err)
			}
			cfg.Targets[i].AuthPass = dec
		}
		if cfg.Kafka != nil {
			dec, err := decryptField(cfg.Kafka.Password, key)
			if err != nil {
				return nil, fmt.Errorf("task %d: decrypt kafka password (wrong encryption_key or incompatible stored format): %w", taskID, err)
			}
			cfg.Kafka.Password = dec
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

// taskTiming is the persisted form of a task's timing information (epoch
// milliseconds; 0 means unset).
type taskTiming struct {
	StartedAt  int64 `json:"startedAt,omitempty"`
	FinishedAt int64 `json:"finishedAt,omitempty"`
	ActiveMs   int64 `json:"activeMs"`
}

// saveTiming writes timing.json.
func (s *Store) saveTiming(taskID int64, t taskTiming) error {
	return s.writeJSONFile(taskID, timingFileName, t)
}

// loadTiming reads timing.json. Returns the zero value when absent.
func (s *Store) loadTiming(taskID int64) (taskTiming, error) {
	var t taskTiming
	data, err := os.ReadFile(filepath.Join(s.taskDir(taskID), timingFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return t, err
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return taskTiming{}, err
	}
	return t, nil
}

// recvFileStatePersist is the persisted form of one file's RECV reassembly
// state. Persisting the exact ranges (rather than the ReceivedBytes scalar)
// is what makes restart recovery correct for out-of-order parallel
// transfers: a scalar cannot express holes, and the chunks filling them may
// already be past the committed Kafka offset.
type recvFileStatePersist struct {
	Ranges    []interval `json:"ranges,omitempty"`
	EofSeen   bool       `json:"eofSeen,omitempty"`
	Total     int64      `json:"total,omitempty"`
	Sha256    string     `json:"sha256,omitempty"`
	Finalized bool       `json:"finalized,omitempty"`
}

// saveRecvState writes recvstate.json (RECV-side reassembly state per file).
func (s *Store) saveRecvState(taskID int64, states map[string]recvFileStatePersist) error {
	if states == nil {
		states = map[string]recvFileStatePersist{}
	}
	return s.writeJSONFile(taskID, recvStateFileName, states)
}

// loadRecvState reads recvstate.json. Returns (nil, nil) when absent (e.g.
// tasks written by older versions).
func (s *Store) loadRecvState(taskID int64) (map[string]recvFileStatePersist, error) {
	data, err := os.ReadFile(filepath.Join(s.taskDir(taskID), recvStateFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var states map[string]recvFileStatePersist
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, err
	}
	return states, nil
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

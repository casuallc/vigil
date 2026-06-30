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
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return newStore(t.TempDir(), testKey)
}

func sampleConfig() TaskConfig {
	return TaskConfig{
		TaskID:          7,
		Role:            RoleSend,
		RelayType:       RelayDirect,
		OverwritePolicy: Overwrite,
		Targets: []TargetConfig{
			{Host: "10.0.0.1", Port: 8080, AuthUser: "admq", AuthPass: "p@ss"},
		},
		Kafka: &KafkaConfig{BootstrapServers: "b:9092", Topic: "t", Password: "kpw"},
	}
}

func TestSaveLoadConfigRoundtripDecrypts(t *testing.T) {
	s := newTestStore(t)
	if err := s.saveConfig(7, sampleConfig()); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	loaded, err := s.loadConfig(7)
	if err != nil || loaded == nil {
		t.Fatalf("loadConfig: %v (loaded=%v)", err, loaded)
	}
	if loaded.Targets[0].AuthPass != "p@ss" {
		t.Fatalf("AuthPass not decrypted: %q", loaded.Targets[0].AuthPass)
	}
	if loaded.Kafka.Password != "kpw" {
		t.Fatalf("Kafka password not decrypted: %q", loaded.Kafka.Password)
	}
}

func TestSaveConfigEncryptsSensitiveFieldsOnDisk(t *testing.T) {
	dir := t.TempDir()
	s := newStore(dir, testKey)
	if err := s.saveConfig(7, sampleConfig()); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "tasks", "7", "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	if strings.Contains(string(raw), "p@ss") {
		t.Fatal("AuthPass stored in plaintext on disk")
	}
	if strings.Contains(string(raw), "kpw") {
		t.Fatal("Kafka password stored in plaintext on disk")
	}
}

func TestSaveConfigDoesNotMutateCaller(t *testing.T) {
	s := newTestStore(t)
	cfg := sampleConfig()
	if err := s.saveConfig(7, cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
	if cfg.Targets[0].AuthPass != "p@ss" {
		t.Fatalf("caller config mutated: AuthPass=%q", cfg.Targets[0].AuthPass)
	}
	if cfg.Kafka.Password != "kpw" {
		t.Fatalf("caller config mutated: Kafka password=%q", cfg.Kafka.Password)
	}
}

func TestSaveLoadStateRoundtrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.saveState(7, StateRunning); err != nil {
		t.Fatalf("saveState: %v", err)
	}
	got, err := s.loadState(7)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got != StateRunning {
		t.Fatalf("state mismatch: got %q want %q", got, StateRunning)
	}
}

func TestSaveLoadProgressRoundtrip(t *testing.T) {
	s := newTestStore(t)
	progress := []FileProgress{{RelPath: "a/b.txt", ReceivedBytes: 10, TotalBytes: 20}}
	if err := s.saveProgress(7, progress); err != nil {
		t.Fatalf("saveProgress: %v", err)
	}
	got, err := s.loadProgress(7)
	if err != nil {
		t.Fatalf("loadProgress: %v", err)
	}
	if len(got) != 1 || got[0].RelPath != "a/b.txt" || got[0].ReceivedBytes != 10 {
		t.Fatalf("progress mismatch: %+v", got)
	}
}

func TestLoadProgressMissingReturnsEmpty(t *testing.T) {
	s := newTestStore(t)
	got, err := s.loadProgress(999)
	if err != nil {
		t.Fatalf("loadProgress: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty progress, got %+v", got)
	}
}

func TestListTaskIDsSkipsNonNumeric(t *testing.T) {
	dir := t.TempDir()
	s := newStore(dir, testKey)
	_ = s.saveState(3, StateIdle)
	_ = s.saveState(11, StateIdle)
	// A stray non-numeric directory must be ignored.
	if err := os.MkdirAll(filepath.Join(dir, "tasks", "junk"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ids, err := s.listTaskIDs()
	if err != nil {
		t.Fatalf("listTaskIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %v", ids)
	}
}

func TestDeleteTaskRemovesDir(t *testing.T) {
	s := newTestStore(t)
	_ = s.saveConfig(7, sampleConfig())
	if err := s.deleteTask(7); err != nil {
		t.Fatalf("deleteTask: %v", err)
	}
	loaded, err := s.loadConfig(7)
	if err != nil {
		t.Fatalf("loadConfig after delete: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil config after delete")
	}
}

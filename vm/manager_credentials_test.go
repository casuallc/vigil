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

package vm

import (
  "path/filepath"
  "testing"

  _ "modernc.org/sqlite"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// TestUpdateVMCredentialsPersists verifies that a credential update survives a
// reload of the database. Previously the API handler only mutated the in-memory
// VM and called the no-op SaveVMs, so a changed password was silently lost on
// the next restart - SSH then kept using the stale password.
func TestUpdateVMCredentialsPersists(t *testing.T) {
  dbPath := filepath.Join(t.TempDir(), "vms.db")

  manager := NewManagerWithConfig(dbPath, testEncryptionKey)
  if manager.db == nil {
    t.Fatal("failed to open test database")
  }
  if err := manager.AddVM(NewVM("web-1", "10.0.0.1", 22, "root", "oldpass", "")); err != nil {
    t.Fatalf("AddVM: %v", err)
  }

  if err := manager.UpdateVMCredentials("web-1", "newpass", ""); err != nil {
    t.Fatalf("UpdateVMCredentials: %v", err)
  }

  if got, _ := manager.GetVM("web-1"); got.Password != "newpass" {
    t.Fatalf("in-memory password = %q, want %q", got.Password, "newpass")
  }

  // The password must be stored encrypted, never in plaintext.
  var stored string
  if err := manager.db.QueryRow("SELECT password FROM vms WHERE name = ?", "web-1").Scan(&stored); err != nil {
    t.Fatalf("query stored password: %v", err)
  }
  if stored == "newpass" {
    t.Fatal("password stored in plaintext")
  }
  manager.Close()

  reloaded := NewManagerWithConfig(dbPath, testEncryptionKey)
  defer reloaded.Close()

  reloadedVM, err := reloaded.GetVM("web-1")
  if err != nil {
    t.Fatalf("GetVM after reload: %v", err)
  }
  if reloadedVM.Password != "newpass" {
    t.Fatalf("password after reload = %q, want %q", reloadedVM.Password, "newpass")
  }
  if reloadedVM.KeyPath != "" {
    t.Fatalf("key path after reload = %q, want empty", reloadedVM.KeyPath)
  }
}

// TestUpdateVMCredentialsPartialUpdate checks that a key_path-only update leaves
// the stored password untouched, and vice versa.
func TestUpdateVMCredentialsPartialUpdate(t *testing.T) {
  manager := NewManagerWithConfig(filepath.Join(t.TempDir(), "vms.db"), testEncryptionKey)
  defer manager.Close()

  if err := manager.AddVM(NewVM("web-1", "10.0.0.1", 22, "root", "oldpass", "/keys/old")); err != nil {
    t.Fatalf("AddVM: %v", err)
  }

  if err := manager.UpdateVMCredentials("web-1", "", "/keys/new"); err != nil {
    t.Fatalf("UpdateVMCredentials: %v", err)
  }

  got, err := manager.GetVM("web-1")
  if err != nil {
    t.Fatalf("GetVM: %v", err)
  }
  if got.Password != "oldpass" {
    t.Fatalf("password = %q, want %q", got.Password, "oldpass")
  }
  if got.KeyPath != "/keys/new" {
    t.Fatalf("key path = %q, want %q", got.KeyPath, "/keys/new")
  }

  var storedPassword, storedKeyPath string
  if err := manager.db.QueryRow("SELECT password, key_path FROM vms WHERE name = ?", "web-1").
    Scan(&storedPassword, &storedKeyPath); err != nil {
    t.Fatalf("query stored credentials: %v", err)
  }
  if storedPassword == "oldpass" || storedKeyPath == "/keys/new" {
    t.Fatal("credentials stored in plaintext")
  }
}

func TestUpdateVMCredentialsErrors(t *testing.T) {
  manager := NewManagerWithConfig(filepath.Join(t.TempDir(), "vms.db"), testEncryptionKey)
  defer manager.Close()

  if err := manager.AddVM(NewVM("web-1", "10.0.0.1", 22, "root", "oldpass", "")); err != nil {
    t.Fatalf("AddVM: %v", err)
  }

  if err := manager.UpdateVMCredentials("missing", "newpass", ""); err == nil {
    t.Fatal("expected an error for an unknown VM")
  }

  if err := manager.UpdateVMCredentials("web-1", "", ""); err == nil {
    t.Fatal("expected an error when no field is provided")
  }
}

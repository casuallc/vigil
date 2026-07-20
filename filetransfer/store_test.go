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
	"encoding/base64"
	"testing"
)

const testKey = "vigil-filetransfer-test-key"

func TestDeriveAESKeyHashesWithSHA256(t *testing.T) {
	key, err := deriveAESKey(testKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sha256.Sum256([]byte(testKey))
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
	if string(key) != string(want[:]) {
		t.Fatal("expected SHA-256 of the configured key")
	}
}

func TestDeriveAESKeyRejectsEmptyKey(t *testing.T) {
	if _, err := deriveAESKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	plain := "super-secret-password"

	enc, err := encryptField(plain, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == plain {
		t.Fatal("ciphertext must differ from plaintext")
	}

	dec, err := decryptField(enc, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != plain {
		t.Fatalf("roundtrip mismatch: got %q want %q", dec, plain)
	}
}

func TestEncryptProducesIVPlusCiphertext(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	enc, err := encryptField("x", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("ciphertext is not standard base64: %v", err)
	}
	// 12-byte IV + at least the 16-byte GCM tag.
	if len(raw) <= 12 {
		t.Fatalf("expected iv(12)+ciphertext+tag, got %d bytes", len(raw))
	}
}

func TestDecryptPlaintextPassthrough(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	// A value that is not our ciphertext should be returned unchanged.
	got, err := decryptField("not-encrypted-plaintext", key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "not-encrypted-plaintext" {
		t.Fatalf("expected plaintext passthrough, got %q", got)
	}
}

func TestDecryptWrongKeyReturnsError(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	enc, err := encryptField("secret", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	otherKey, _ := deriveAESKey("another-key")
	if _, err := decryptField(enc, otherKey); err == nil {
		t.Fatal("expected error when decrypting with the wrong key")
	}
}

func TestEncryptDecryptBlankPassthrough(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	got, err := encryptField("", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got != "" {
		t.Fatalf("expected blank encrypt passthrough, got %q", got)
	}
	got, err = decryptField("", key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "" {
		t.Fatalf("expected blank decrypt passthrough, got %q", got)
	}
}

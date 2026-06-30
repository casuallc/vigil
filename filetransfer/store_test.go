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
	"encoding/base64"
	"testing"
)

const testKey = "admq-file-transfer-agent-key-16"

func TestDeriveAESKeyTakesFirst16Bytes(t *testing.T) {
	key, err := deriveAESKey(testKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 16 {
		t.Fatalf("expected 16-byte key, got %d", len(key))
	}
	if string(key) != "admq-file-transf" {
		t.Fatalf("expected first 16 bytes, got %q", string(key))
	}
}

func TestDeriveAESKeyRejectsShortKey(t *testing.T) {
	if _, err := deriveAESKey("short"); err == nil {
		t.Fatal("expected error for key shorter than 16 bytes")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	plain := "super-secret-password"

	enc := encryptField(plain, key)
	if enc == plain {
		t.Fatal("ciphertext must differ from plaintext")
	}

	dec := decryptField(enc, key)
	if dec != plain {
		t.Fatalf("roundtrip mismatch: got %q want %q", dec, plain)
	}
}

func TestEncryptProducesIVPlusCiphertext(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	enc := encryptField("x", key)

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("ciphertext is not standard base64: %v", err)
	}
	// 12-byte IV + at least the 16-byte GCM tag.
	if len(raw) <= 12 {
		t.Fatalf("expected iv(12)+ciphertext+tag, got %d bytes", len(raw))
	}
}

func TestDecryptPlaintextFallback(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	// A value that is not our ciphertext should be returned unchanged.
	if got := decryptField("not-encrypted-plaintext", key); got != "not-encrypted-plaintext" {
		t.Fatalf("expected plaintext passthrough, got %q", got)
	}
}

func TestEncryptDecryptBlankPassthrough(t *testing.T) {
	key, _ := deriveAESKey(testKey)
	if got := encryptField("", key); got != "" {
		t.Fatalf("expected blank encrypt passthrough, got %q", got)
	}
	if got := decryptField("", key); got != "" {
		t.Fatalf("expected blank decrypt passthrough, got %q", got)
	}
}

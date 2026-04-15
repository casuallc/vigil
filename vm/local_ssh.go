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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// generateSSHKeyPair generates an Ed25519 SSH key pair at the given privateKeyPath.
// The public key is written to privateKeyPath + ".pub".
// If the key pair already exists, generation is skipped.
func generateSSHKeyPair(privateKeyPath string) (publicKeyPath string, err error) {
	publicKeyPath = privateKeyPath + ".pub"

	// Skip if both files already exist
	if _, err := os.Stat(privateKeyPath); err == nil {
		if _, err := os.Stat(publicKeyPath); err == nil {
			return publicKeyPath, nil
		}
	}

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(privateKeyPath), 0700); err != nil {
		return "", fmt.Errorf("failed to create key directory: %w", err)
	}

	// Generate Ed25519 key pair
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Marshal private key to PKCS#8 PEM
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Generate public key in OpenSSH authorized_keys format
	publicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", fmt.Errorf("failed to create ssh public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	if err := os.WriteFile(publicKeyPath, publicKeyBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write public key: %w", err)
	}

	return publicKeyPath, nil
}

// ensureAuthorizedKeys ensures the given public key is present in ~/.ssh/authorized_keys.
func ensureAuthorizedKeys(publicKeyPath string) error {
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}
	publicKey := strings.TrimSpace(string(publicKeyBytes))

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	authorizedKeysPath := filepath.Join(sshDir, "authorized_keys")
	var existingContent string
	if data, err := os.ReadFile(authorizedKeysPath); err == nil {
		existingContent = string(data)
	}

	// Check if the key is already present
	if strings.Contains(existingContent, publicKey) {
		return nil
	}

	// Append the public key
	f, err := os.OpenFile(authorizedKeysPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open authorized_keys: %w", err)
	}
	defer f.Close()

	// Add a newline before the key if the file doesn't end with one
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to authorized_keys: %w", err)
		}
	}

	if _, err := f.WriteString(publicKey + "\n"); err != nil {
		return fmt.Errorf("failed to write public key to authorized_keys: %w", err)
	}

	return nil
}

// getLocalSSHInfo returns the current OS username and default SSH port.
func getLocalSSHInfo() (username string, port int, err error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get current user: %w", err)
	}
	return currentUser.Username, 22, nil
}

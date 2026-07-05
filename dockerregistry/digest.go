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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// ParseDigest parses a digest string like "sha256:abcdef..." into algorithm and hex.
func ParseDigest(d string) (algo, hexStr string, err error) {
	parts := strings.SplitN(d, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid digest format: %q", d)
	}
	if parts[0] != "sha256" {
		return "", "", fmt.Errorf("unsupported digest algorithm: %q", parts[0])
	}
	return parts[0], parts[1], nil
}

// DigestFromReader computes a sha256 digest from a stream.
func DigestFromReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// DigestFromFile computes a sha256 digest from a file.
func DigestFromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return DigestFromReader(f)
}

// newHash returns a new hash.Hash for the given algorithm.
func newHash(algo string) (hash.Hash, error) {
	switch algo {
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported digest algorithm: %q", algo)
	}
}

// HashCopy copies from r to w while computing a sha256 digest.
func HashCopy(w io.Writer, r io.Reader) (n int64, digest string, err error) {
	h := sha256.New()
	mw := io.MultiWriter(w, h)
	n, err = io.Copy(mw, r)
	if err != nil {
		return n, "", err
	}
	return n, "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

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

import "context"

// defaultChunkSizeBytes is the fallback chunk size (1MB) when a task does not
// specify one.
const defaultChunkSizeBytes = 1048576

// maxParallelism caps TaskConfig.Parallelism to bound goroutines and
// in-flight chunk buffers.
const maxParallelism = 16

// effectiveParallelism normalises a configured parallelism: values <= 1 mean
// sequential; anything above maxParallelism is clamped.
func effectiveParallelism(n int) int {
	if n < 1 {
		return 1
	}
	if n > maxParallelism {
		return maxParallelism
	}
	return n
}

// ChunkReader reads length bytes starting at offset from the source file.
type ChunkReader func(offset int64, length int) ([]byte, error)

// ProgressSink is invoked after each chunk with the number of bytes sent.
type ProgressSink func(n int)

// RelayTransport sends one file's chunks to a single target. Implementations
// are selected per task by RelayType (DIRECT, KAFKA, ...).
type RelayTransport interface {
	Type() RelayType
	SendFile(ctx context.Context, cfg TaskConfig, target TargetConfig, file FileEntry, read ChunkReader, sink ProgressSink) error
}

// transportRegistry maps a RelayType to its transport implementation, so new
// transports can be added without scattering relay-type switches.
type transportRegistry struct {
	m map[RelayType]RelayTransport
}

func newTransportRegistry() *transportRegistry {
	return &transportRegistry{m: make(map[RelayType]RelayTransport)}
}

func (r *transportRegistry) register(t RelayTransport) {
	r.m[t.Type()] = t
}

func (r *transportRegistry) get(rt RelayType) (RelayTransport, bool) {
	t, ok := r.m[rt]
	return t, ok
}

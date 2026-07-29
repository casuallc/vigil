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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// directTransport sends chunks to a peer agent over its REST API (DIRECT
// relay), with resume support via the peer's /progress endpoint.
type directTransport struct {
	client *http.Client
}

func newDirectTransport() *directTransport {
	return &directTransport{client: &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
			// Keep enough idle connections for a full chunk window, otherwise
			// pipelined sends pay a TCP handshake per chunk.
			MaxIdleConnsPerHost: maxParallelism,
		},
	}}
}

func (d *directTransport) Type() RelayType { return RelayDirect }

// SendFile streams file in chunkSize pieces to target, resuming from the
// offset the peer reports. With cfg.Parallelism > 1 chunks are sent by a
// worker pool (an in-flight window) instead of stop-and-wait: the receiver
// reassembles by offset, so out-of-order arrival is safe. This is what makes
// a single large file fast — file-level parallelism alone cannot help it.
// directDefaultChunkSizeBytes is the DIRECT-relay fallback chunk size when a
// task does not specify one. Larger than the generic default: with the chunk
// pipeline the per-request overhead is amortised, and 4MB measured fastest
// (see BenchmarkDirectTransfer). The KAFKA relay keeps its own clamp just
// below the broker message limit.
const directDefaultChunkSizeBytes = 4 << 20

func (d *directTransport) SendFile(ctx context.Context, cfg TaskConfig, target TargetConfig, file FileEntry, read ChunkReader, sink ProgressSink, sha HashFunc) error {
	base := fmt.Sprintf("http://%s:%d", target.Host, target.Port)
	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = directDefaultChunkSizeBytes
	}

	resumeOffset := d.queryResumeOffset(ctx, base, target, file.RelPath)
	if file.Size == 0 {
		// Zero-byte file: deliver an empty EOF chunk so the receiver still
		// finalises (creates) the file.
		sum, err := resolveSHA256(ctx, file, sha)
		if err != nil {
			return fmt.Errorf("sha256 for %s: %w", file.RelPath, err)
		}
		meta := ChunkMeta{RelPath: file.RelPath, Eof: true, Sha256: sum}
		return d.postChunk(ctx, base, target, meta, nil)
	}
	totalChunks := int((file.Size - resumeOffset + int64(chunkSize) - 1) / int64(chunkSize))

	sendOne := func(idx int) error {
		offset := resumeOffset + int64(idx)*int64(chunkSize)
		length := chunkSize
		if remaining := file.Size - offset; remaining < int64(length) {
			length = int(remaining)
		}
		data, err := read(offset, length)
		if err != nil {
			return fmt.Errorf("read chunk at %d: %w", offset, err)
		}
		if len(data) == 0 {
			return nil
		}
		eof := offset+int64(len(data)) >= file.Size
		meta := ChunkMeta{
			RelPath:    file.RelPath,
			ChunkIndex: idx,
			Offset:     offset,
			Length:     len(data),
			Crc32:      crc32.ChecksumIEEE(data),
			Eof:        eof,
		}
		if eof {
			// The hash has had the whole transfer to compute; only the EOF
			// chunk blocks on it (and only if it is not ready yet).
			sum, err := resolveSHA256(ctx, file, sha)
			if err != nil {
				return fmt.Errorf("sha256 for %s: %w", file.RelPath, err)
			}
			meta.Sha256 = sum
		}
		if err := d.postChunk(ctx, base, target, meta, data); err != nil {
			return err
		}
		if sink != nil {
			sink(len(data))
		}
		return nil
	}

	if effectiveParallelism(cfg.Parallelism) <= 1 {
		for idx := 0; idx < totalChunks; idx++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := sendOne(idx); err != nil {
				return err
			}
		}
		return nil
	}

	workers := effectiveParallelism(cfg.Parallelism)
	g, gctx := errgroup.WithContext(ctx)
	var next atomic.Int64
	for w := 0; w < workers; w++ {
		g.Go(func() error {
			for {
				if err := gctx.Err(); err != nil {
					return err
				}
				idx := int(next.Add(1) - 1)
				if idx >= totalChunks {
					return nil
				}
				if err := sendOne(idx); err != nil {
					return err
				}
			}
		})
	}
	return g.Wait()
}

func (d *directTransport) queryResumeOffset(ctx context.Context, base string, target TargetConfig, relPath string) int64 {
	u := fmt.Sprintf("%s/api/transfer/tasks/%d/progress", base, target.AgentTaskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0
	}
	d.applyAuth(req, target)
	req.Header.Set("Accept", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	// The progress endpoint returns a bare []FileProgress.
	var progress []FileProgress
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		return 0
	}
	for _, fp := range progress {
		if fp.RelPath == relPath {
			return fp.ReceivedBytes
		}
	}
	return 0
}

func (d *directTransport) postChunk(ctx context.Context, base string, target TargetConfig, meta ChunkMeta, data []byte) error {
	q := url.Values{}
	q.Set("relPath", meta.RelPath)
	q.Set("chunkIndex", strconv.Itoa(meta.ChunkIndex))
	q.Set("offset", strconv.FormatInt(meta.Offset, 10))
	q.Set("length", strconv.Itoa(meta.Length))
	q.Set("crc32", strconv.FormatUint(uint64(meta.Crc32), 10))
	q.Set("eof", strconv.FormatBool(meta.Eof))
	if meta.Sha256 != "" {
		q.Set("sha256", meta.Sha256)
	}
	if target.RecvToken != "" {
		q.Set("recvToken", target.RecvToken)
	}
	u := fmt.Sprintf("%s/api/transfer/tasks/%d/chunks?%s", base, target.AgentTaskID, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	d.applyAuth(req, target)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("chunk POST failed: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (d *directTransport) applyAuth(req *http.Request, target TargetConfig) {
	if target.AuthUser != "" || target.AuthPass != "" {
		req.SetBasicAuth(target.AuthUser, target.AuthPass)
	}
}

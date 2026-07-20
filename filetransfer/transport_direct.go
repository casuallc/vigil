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
	"time"
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
		},
	}}
}

func (d *directTransport) Type() RelayType { return RelayDirect }

// SendFile streams file in chunkSize pieces to target, resuming from the
// offset the peer reports.
func (d *directTransport) SendFile(ctx context.Context, cfg TaskConfig, target TargetConfig, file FileEntry, read ChunkReader, sink ProgressSink) error {
	base := fmt.Sprintf("http://%s:%d", target.Host, target.Port)
	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSizeBytes
	}

	resumeOffset := d.queryResumeOffset(ctx, base, target, file.RelPath)
	offset := resumeOffset
	chunkIndex := 0
	for offset < file.Size {
		if err := ctx.Err(); err != nil {
			return err
		}
		length := chunkSize
		if remaining := file.Size - offset; remaining < int64(length) {
			length = int(remaining)
		}
		data, err := read(offset, length)
		if err != nil {
			return fmt.Errorf("read chunk at %d: %w", offset, err)
		}
		if len(data) == 0 {
			break
		}
		eof := offset+int64(len(data)) >= file.Size
		meta := ChunkMeta{
			RelPath:    file.RelPath,
			ChunkIndex: chunkIndex,
			Offset:     offset,
			Length:     len(data),
			Crc32:      crc32.ChecksumIEEE(data),
			Eof:        eof,
		}
		if eof {
			meta.Sha256 = file.Sha256
		}
		if err := d.postChunk(ctx, base, target, meta, data); err != nil {
			return err
		}
		offset += int64(len(data))
		chunkIndex++
		if sink != nil {
			sink(len(data))
		}
	}
	return nil
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

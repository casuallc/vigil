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
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// benchTransfer moves one file of size bytes from a sender Manager to a
// receiver Manager over loopback HTTP (DIRECT relay) and reports wall time.
func benchTransfer(b *testing.B, size int64, chunkSize, parallelism int) {
	recvDir := b.TempDir()
	sendDir := b.TempDir()

	// Receiver: a Manager with its routes on an httptest server.
	recvMgr := NewManager(Options{DataDir: filepath.Join(recvDir, "data")})
	defer recvMgr.Shutdown()
	const recvTaskID = 42
	if err := recvMgr.CreateTask(TaskConfig{
		TaskID:    recvTaskID,
		Role:      RoleRecv,
		RelayType: RelayDirect,
		TargetDir: filepath.Join(recvDir, "dst"),
	}); err != nil {
		b.Fatal(err)
	}
	router := mux.NewRouter()
	recvMgr.RegisterRoutes(router)
	srv := httptest.NewServer(router)
	defer srv.Close()
	if err := recvMgr.Start(recvTaskID); err != nil {
		b.Fatal(err)
	}
	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		b.Fatal(err)
	}

	// Source file.
	src := filepath.Join(sendDir, "big.bin")
	f, err := os.Create(src)
	if err != nil {
		b.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		b.Fatal(err)
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}

	// Sender.
	sendMgr := NewManager(Options{DataDir: filepath.Join(sendDir, "data")})
	defer sendMgr.Shutdown()
	const sendTaskID = 7
	cfg := TaskConfig{
		TaskID:      sendTaskID,
		Role:        RoleSend,
		RelayType:   RelayDirect,
		SourcePaths: []string{src},
		ChunkSize:   chunkSize,
		Parallelism: parallelism,
		Targets: []TargetConfig{{
			Host:        host,
			Port:        port,
			AgentTaskID: recvTaskID,
		}},
	}
	if err := sendMgr.CreateTask(cfg); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	start := time.Now()
	if err := sendMgr.Start(sendTaskID); err != nil {
		b.Fatal(err)
	}
	deadline := start.Add(10 * time.Minute)
	for {
		st, err := sendMgr.GetStatus(sendTaskID)
		if err != nil {
			b.Fatal(err)
		}
		if st.State == StateSuccess {
			break
		}
		if st.State != StateRunning && st.State != StateIdle {
			b.Fatalf("unexpected state %s: %s", st.State, st.ErrorMsg)
		}
		if time.Now().After(deadline) {
			b.Fatal("transfer timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
	elapsed := time.Since(start)
	b.StopTimer()

	// Verify content landed and matches size.
	got, err := os.Stat(filepath.Join(recvDir, "dst", "big.bin"))
	if err != nil {
		b.Fatal(err)
	}
	if got.Size() != size {
		b.Fatalf("size mismatch: got %d want %d", got.Size(), size)
	}
	mbs := float64(size) / 1024 / 1024 / elapsed.Seconds()
	b.ReportMetric(mbs, "MB/s")
	b.Logf("size=%dMB chunk=%dKB parallelism=%d elapsed=%s rate=%.1fMB/s",
		size/1024/1024, chunkSize/1024, parallelism, elapsed.Round(time.Millisecond), mbs)
}

func BenchmarkDirectTransfer(b *testing.B) {
	const size = 256 << 20 // 256MB
	b.Run("defaults", func(b *testing.B) {
		benchTransfer(b, size, 0, 0)
	})
	for _, p := range []int{1, 4, 8, 16} {
		b.Run(fmt.Sprintf("parallelism=%d", p), func(b *testing.B) {
			benchTransfer(b, size, defaultChunkSizeBytes, p)
		})
	}
	for _, cs := range []int{4 << 20, 8 << 20} {
		b.Run(fmt.Sprintf("chunk=%dMB/p=8", cs>>20), func(b *testing.B) {
			benchTransfer(b, size, cs, 8)
		})
	}
}

// benchReceive feeds one file of size bytes straight into a RECV task's
// ReceiveChunk — the hot path every relay lands in — with `concurrency`
// goroutines delivering chunks, and reports wall time.
func benchReceive(b *testing.B, size int64, concurrency int) {
	dir := b.TempDir()
	mgr := NewManager(Options{DataDir: filepath.Join(dir, "data")})
	defer mgr.Shutdown()
	const taskID = 1
	// DIRECT keeps executeTask inert (no broker needed); ReceiveChunk is the
	// same code path the KAFKA consumer invokes per chunk.
	if err := mgr.CreateTask(TaskConfig{
		TaskID:    taskID,
		Role:      RoleRecv,
		RelayType: RelayDirect,
		TargetDir: filepath.Join(dir, "dst"),
	}); err != nil {
		b.Fatal(err)
	}
	if err := mgr.Start(taskID); err != nil {
		b.Fatal(err)
	}

	chunk := make([]byte, defaultChunkSizeBytes)
	totalChunks := int((size + int64(len(chunk)) - 1) / int64(len(chunk)))
	deliver := func(idx int) error {
		off := int64(idx) * int64(len(chunk))
		n := int64(len(chunk))
		if off+n > size {
			n = size - off
		}
		meta := ChunkMeta{
			RelPath:    "big.bin",
			ChunkIndex: idx,
			Offset:     off,
			Length:     int(n),
			Eof:        off+n >= size,
			Size:       size,
		}
		return mgr.ReceiveChunk(taskID, meta, chunk[:n])
	}

	b.ResetTimer()
	start := time.Now()
	var next atomic.Int64
	var wg sync.WaitGroup
	errs := make(chan error, concurrency)
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				idx := int(next.Add(1) - 1)
				if idx >= totalChunks {
					return
				}
				if err := deliver(idx); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		b.Fatal(err)
	}
	elapsed := time.Since(start)
	b.StopTimer()

	got, err := os.Stat(filepath.Join(dir, "dst", "big.bin"))
	if err != nil {
		b.Fatal(err)
	}
	if got.Size() != size {
		b.Fatalf("size mismatch: got %d want %d", got.Size(), size)
	}
	mbs := float64(size) / 1024 / 1024 / elapsed.Seconds()
	b.ReportMetric(mbs, "MB/s")
	b.Logf("size=%dMB concurrency=%d elapsed=%s rate=%.1fMB/s",
		size/1024/1024, concurrency, elapsed.Round(time.Millisecond), mbs)
}

func BenchmarkReceiveChunks(b *testing.B) {
	const size = 256 << 20 // 256MB
	for _, c := range []int{1, 4, 8} {
		b.Run(fmt.Sprintf("concurrency=%d", c), func(b *testing.B) {
			benchReceive(b, size, c)
		})
	}
}

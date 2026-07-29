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
	"encoding/binary"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/IBM/sarama"
)

func TestEncodeKafkaMessageFormat(t *testing.T) {
	meta := ChunkMeta{RelPath: "dir/a.txt", ChunkIndex: 0, Offset: 0, Length: 3, Crc32: 12345, Eof: false}
	data := []byte("abc")

	msg, err := encodeKafkaMessage(meta, data)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Binary frame: [4-byte BE header length][JSON header][raw chunk bytes].
	if len(msg) < kafkaFrameLenBytes {
		t.Fatal("message shorter than frame prefix")
	}
	headerLen := int(binary.BigEndian.Uint32(msg[:kafkaFrameLenBytes]))
	if headerLen != len(msg)-kafkaFrameLenBytes-len(data) {
		t.Fatalf("header length %d inconsistent with message size %d", headerLen, len(msg))
	}
	header := string(msg[kafkaFrameLenBytes : kafkaFrameLenBytes+headerLen])
	body := msg[kafkaFrameLenBytes+headerLen:]
	if !strings.Contains(header, `"relPath"`) || !strings.Contains(header, `"crc32"`) {
		t.Fatalf("header missing expected fields: %s", header)
	}
	if !bytes.Equal(body, data) {
		t.Fatalf("body not the raw chunk: got %q want %q", body, data)
	}
}

func TestDecodeKafkaMessageRoundtrip(t *testing.T) {
	meta := ChunkMeta{RelPath: "dir/a.txt", ChunkIndex: 2, Offset: 6, Length: 3, Crc32: 999, Eof: true, Sha256: "deadbeef"}
	data := []byte("xyz")

	msg, err := encodeKafkaMessage(meta, data)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	gotMeta, gotData, err := decodeKafkaMessage(msg)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotMeta != meta {
		t.Fatalf("meta mismatch: got %+v want %+v", gotMeta, meta)
	}
	if !bytes.Equal(gotData, data) {
		t.Fatalf("data mismatch: got %q want %q", gotData, data)
	}
}

func TestDecodeKafkaMessageRejectsMalformed(t *testing.T) {
	if _, _, err := decodeKafkaMessage([]byte("ab")); err == nil {
		t.Fatal("expected error for message shorter than frame prefix")
	}
	// Frame prefix claims a header longer than the message.
	bogus := make([]byte, kafkaFrameLenBytes+2)
	binary.BigEndian.PutUint32(bogus, 100)
	if _, _, err := decodeKafkaMessage(bogus); err == nil {
		t.Fatal("expected error for header length exceeding message size")
	}
}

type stubTransport struct{ rt RelayType }

func (s stubTransport) Type() RelayType { return s.rt }
func (s stubTransport) SendFile(_ context.Context, _ TaskConfig, _ TargetConfig, _ FileEntry, _ ChunkReader, _ ProgressSink, _ HashFunc) error {
	return nil
}

// fakeKafkaProducer records produced messages and can be told to fail on the
// Nth call (1-based).
type fakeKafkaProducer struct {
	mu       sync.Mutex
	messages []*sarama.ProducerMessage
	calls    int
	failOn   int
	closed   bool
}

func (f *fakeKafkaProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.failOn > 0 && f.calls == f.failOn {
		return 0, 0, errors.New("broker unavailable")
	}
	f.messages = append(f.messages, msg)
	return 0, int64(f.calls), nil
}

func (f *fakeKafkaProducer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// sliceReader returns a ChunkReader over an in-memory byte slice.
func sliceReader(b []byte) ChunkReader {
	return func(offset int64, length int) ([]byte, error) {
		end := offset + int64(length)
		if end > int64(len(b)) {
			end = int64(len(b))
		}
		return b[offset:end], nil
	}
}

func TestSendKafkaChunksSequentialAndParallelEquivalent(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 200) // 3200 bytes
	file := FileEntry{RelPath: "dir/big.bin", Size: int64(len(payload)), Sha256: "abc123"}

	for _, parallelism := range []int{1, 4} {
		fake := &fakeKafkaProducer{}
		var sentBytes atomic.Int64
		sink := func(n int) { sentBytes.Add(int64(n)) }

		err := sendKafkaChunks(context.Background(), fake, "topic", file, sliceReader(payload), sink, nil, 256, parallelism)
		if err != nil {
			t.Fatalf("parallelism=%d: %v", parallelism, err)
		}
		wantChunks := (len(payload) + 255) / 256
		if len(fake.messages) != wantChunks {
			t.Fatalf("parallelism=%d: got %d messages, want %d", parallelism, len(fake.messages), wantChunks)
		}
		if int(sentBytes.Load()) != len(payload) {
			t.Fatalf("parallelism=%d: sink saw %d bytes, want %d", parallelism, sentBytes.Load(), len(payload))
		}
		// Reassemble by chunk index and compare against the source payload.
		reassembled := make([]byte, len(payload))
		eofSeen := 0
		for _, msg := range fake.messages {
			if parallelism <= 1 {
				// Sequential path: keyed by relPath (ordered, one partition).
				if key, _ := msg.Key.Encode(); string(key) != file.RelPath {
					t.Fatalf("parallelism=%d: message key = %q", parallelism, key)
				}
			} else if msg.Key != nil {
				// Parallel path: nil key lets the round-robin partitioner
				// spread chunks across all partitions.
				t.Fatalf("parallelism=%d: expected nil key, got %v", parallelism, msg.Key)
			}
			val, _ := msg.Value.Encode()
			meta, data, err := decodeKafkaMessage(val)
			if err != nil {
				t.Fatalf("parallelism=%d: decode: %v", parallelism, err)
			}
			copy(reassembled[meta.Offset:], data)
			if meta.Eof {
				eofSeen++
				if meta.Sha256 != file.Sha256 {
					t.Fatalf("parallelism=%d: EOF chunk missing sha256", parallelism)
				}
				if meta.Offset+int64(len(data)) != file.Size {
					t.Fatalf("parallelism=%d: EOF chunk does not end the file", parallelism)
				}
			}
		}
		if eofSeen != 1 {
			t.Fatalf("parallelism=%d: got %d EOF chunks, want 1", parallelism, eofSeen)
		}
		if !bytes.Equal(reassembled, payload) {
			t.Fatalf("parallelism=%d: reassembled payload mismatch", parallelism)
		}
	}
}

func TestSendKafkaChunksPropagatesSendError(t *testing.T) {
	payload := make([]byte, 1024)
	file := FileEntry{RelPath: "f.bin", Size: int64(len(payload))}
	fake := &fakeKafkaProducer{failOn: 2}

	err := sendKafkaChunks(context.Background(), fake, "topic", file, sliceReader(payload), nil, nil, 128, 4)
	if err == nil || !strings.Contains(err.Error(), "broker unavailable") {
		t.Fatalf("expected broker error, got %v", err)
	}
}

// TestSendKafkaChunksZeroByteFileSendsEmptyEOF ensures zero-byte files still
// reach the receiver (as a single empty EOF chunk) instead of vanishing.
func TestSendKafkaChunksZeroByteFileSendsEmptyEOF(t *testing.T) {
	fake := &fakeKafkaProducer{}
	file := FileEntry{RelPath: "empty.bin", Size: 0, Sha256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}

	if err := sendKafkaChunks(context.Background(), fake, "topic", file, sliceReader(nil), nil, nil, 256, 4); err != nil {
		t.Fatalf("sendKafkaChunks: %v", err)
	}
	if len(fake.messages) != 1 {
		t.Fatalf("got %d messages, want 1 empty EOF chunk", len(fake.messages))
	}
	val, _ := fake.messages[0].Value.Encode()
	meta, data, err := decodeKafkaMessage(val)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !meta.Eof || meta.Offset != 0 || meta.Length != 0 || len(data) != 0 {
		t.Fatalf("unexpected zero-file chunk: meta=%+v dataLen=%d", meta, len(data))
	}
	if meta.Sha256 != file.Sha256 {
		t.Fatalf("EOF chunk sha256 = %q, want %q", meta.Sha256, file.Sha256)
	}
}

// TestKafkaTransportReusesProducer verifies the cached producer is reused
// across files with the same effective config, recreated when the config
// changes, and released on Close.
func TestKafkaTransportReusesProducer(t *testing.T) {
	fake := &fakeKafkaProducer{}
	creates := 0
	tr := newKafkaTransport()
	tr.newSyncProducer = func(_ []string, _ *sarama.Config) (kafkaSyncProducer, error) {
		creates++
		return fake, nil
	}

	cfg := TaskConfig{
		RelayType: RelayKafka,
		Kafka:     &KafkaConfig{BootstrapServers: "broker:9092", Topic: "t"},
	}
	file := FileEntry{RelPath: "a.bin", Size: 10}
	read := sliceReader([]byte("0123456789"))
	for i := 0; i < 3; i++ {
		if err := tr.SendFile(context.Background(), cfg, TargetConfig{}, file, read, nil, nil); err != nil {
			t.Fatalf("SendFile %d: %v", i, err)
		}
	}
	if creates != 1 {
		t.Fatalf("producer created %d times across 3 sends, want 1 (reused)", creates)
	}

	// A config change must build a fresh producer.
	cfg.Kafka.Compression = "lz4"
	if err := tr.SendFile(context.Background(), cfg, TargetConfig{}, file, read, nil, nil); err != nil {
		t.Fatalf("SendFile with new config: %v", err)
	}
	if creates != 2 {
		t.Fatalf("producer created %d times after config change, want 2", creates)
	}

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !fake.closed {
		t.Fatal("cached producer was not closed")
	}
}

func TestKafkaCompressionCodec(t *testing.T) {
	cases := map[string]sarama.CompressionCodec{
		"":       sarama.CompressionSnappy,
		"snappy": sarama.CompressionSnappy,
		"none":   sarama.CompressionNone,
		"zstd":   sarama.CompressionZSTD,
		"lz4":    sarama.CompressionLZ4,
		"gzip":   sarama.CompressionGZIP,
	}
	for name, want := range cases {
		got, err := kafkaCompressionCodec(name)
		if err != nil || got != want {
			t.Fatalf("kafkaCompressionCodec(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := kafkaCompressionCodec("brotli"); err == nil {
		t.Fatal("expected error for unsupported codec")
	}
}

// fakeConsumerSession implements sarama.ConsumerGroupSession, recording
// marked messages.
type fakeConsumerSession struct {
	ctx    context.Context
	marked []*sarama.ConsumerMessage
}

func (f *fakeConsumerSession) Claims() map[string][]int32 { return nil }
func (f *fakeConsumerSession) MemberID() string           { return "" }
func (f *fakeConsumerSession) GenerationID() int32        { return 0 }
func (f *fakeConsumerSession) MarkOffset(string, int32, int64, string) {
}
func (f *fakeConsumerSession) Commit()                                  {}
func (f *fakeConsumerSession) ResetOffset(string, int32, int64, string) {}
func (f *fakeConsumerSession) MarkMessage(msg *sarama.ConsumerMessage, _ string) {
	f.marked = append(f.marked, msg)
}
func (f *fakeConsumerSession) Context() context.Context { return f.ctx }

// fakeConsumerClaim implements sarama.ConsumerGroupClaim over a channel.
type fakeConsumerClaim struct {
	sarama.ConsumerGroupClaim
	ch chan *sarama.ConsumerMessage
}

func (f fakeConsumerClaim) Messages() <-chan *sarama.ConsumerMessage { return f.ch }

func TestConsumeClaimSkipsUndecodableButMarks(t *testing.T) {
	session := &fakeConsumerSession{ctx: context.Background()}
	ch := make(chan *sarama.ConsumerMessage, 1)
	ch <- &sarama.ConsumerMessage{Topic: "t", Partition: 0, Offset: 7, Value: []byte("not-a-frame")}
	close(ch)

	h := &chunkConsumerHandler{handle: func(ChunkMeta, []byte) error { return nil }}
	if err := h.ConsumeClaim(session, fakeConsumerClaim{ch: ch}); err != nil {
		t.Fatalf("ConsumeClaim: %v", err)
	}
	if len(session.marked) != 1 {
		t.Fatalf("expected the bad message to be marked, got %d", len(session.marked))
	}
}

// TestConsumeClaimHandleErrorStopsSession ensures a failed chunk is NOT
// marked and the error propagates so the session restarts for redelivery.
func TestConsumeClaimHandleErrorStopsSession(t *testing.T) {
	session := &fakeConsumerSession{ctx: context.Background()}
	ch := make(chan *sarama.ConsumerMessage, 1)
	msg, err := encodeKafkaMessage(ChunkMeta{RelPath: "f.bin", Offset: 0, Length: 3}, []byte("abc"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	ch <- &sarama.ConsumerMessage{Topic: "t", Partition: 0, Offset: 9, Value: msg}
	close(ch)

	h := &chunkConsumerHandler{handle: func(ChunkMeta, []byte) error {
		return errors.New("disk full")
	}}
	if err := h.ConsumeClaim(session, fakeConsumerClaim{ch: ch}); err == nil {
		t.Fatal("expected ConsumeClaim to return the handle error")
	}
	if len(session.marked) != 0 {
		t.Fatalf("failed message must not be marked, got %d", len(session.marked))
	}
}

func TestTransportRegistryGet(t *testing.T) {
	reg := newTransportRegistry()
	reg.register(stubTransport{rt: RelayDirect})

	if _, ok := reg.get(RelayDirect); !ok {
		t.Fatal("expected DIRECT transport to be registered")
	}
	if _, ok := reg.get(RelayKafka); ok {
		t.Fatal("did not expect KAFKA transport")
	}
}

func TestKafkaRequiredAcks(t *testing.T) {
	cases := []struct {
		name    string
		want    sarama.RequiredAcks
		wantErr bool
	}{
		{"", sarama.WaitForAll, false},
		{"all", sarama.WaitForAll, false},
		{"ALL", sarama.WaitForAll, false},
		{"local", sarama.WaitForLocal, false},
		{"none", 0, true},
		{"bogus", 0, true},
	}
	for _, c := range cases {
		got, err := kafkaRequiredAcks(c.name)
		if c.wantErr {
			if err == nil {
				t.Fatalf("requiredAcks=%q: expected error, got %v", c.name, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Fatalf("requiredAcks=%q: got %v, %v; want %v", c.name, got, err, c.want)
		}
	}
}

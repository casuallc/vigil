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
	"strings"
	"testing"
)

func TestEncodeKafkaMessageFormat(t *testing.T) {
	meta := ChunkMeta{RelPath: "dir/a.txt", ChunkIndex: 0, Offset: 0, Length: 3, Crc32: 12345, Eof: false}
	data := []byte("abc")

	msg, err := encodeKafkaMessage(meta, data)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Exactly one newline separates the JSON header from the base64 body.
	idx := bytes.IndexByte(msg, '\n')
	if idx < 0 {
		t.Fatal("expected newline separator")
	}
	header := string(msg[:idx])
	body := string(msg[idx+1:])
	if !strings.Contains(header, `"relPath"`) || !strings.Contains(header, `"crc32"`) {
		t.Fatalf("header missing expected fields: %s", header)
	}
	if body != "YWJj" { // base64("abc")
		t.Fatalf("body not base64 of chunk: %q", body)
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
	if _, _, err := decodeKafkaMessage([]byte("no-newline-here")); err == nil {
		t.Fatal("expected error for message without newline")
	}
}

type stubTransport struct{ rt RelayType }

func (s stubTransport) Type() RelayType { return s.rt }
func (s stubTransport) SendFile(_ context.Context, _ TaskConfig, _ TargetConfig, _ FileEntry, _ ChunkReader, _ ProgressSink) error {
	return nil
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

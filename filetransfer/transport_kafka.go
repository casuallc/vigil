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
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"strings"
	"sync/atomic"

	"github.com/IBM/sarama"
	"golang.org/x/sync/errgroup"
)

// kafkaFrameLenBytes is the size of the big-endian uint32 header-length
// prefix that frames each Kafka message.
const kafkaFrameLenBytes = 4

// kafkaDefaultMaxMessageBytes applies when KafkaConfig.MaxMessageBytes is
// unset. It stays below the Kafka broker default message.max.bytes
// (1,000,012) so a stock broker accepts relayed chunks.
const kafkaDefaultMaxMessageBytes = 1_000_000

// kafkaHeaderMargin reserves room for the JSON ChunkMeta header when clamping
// the chunk size to the max message size.
const kafkaHeaderMargin = 4096

// kafkaTransport sends file chunks through a Kafka topic (KAFKA relay).
type kafkaTransport struct{}

func newKafkaTransport() *kafkaTransport { return &kafkaTransport{} }

func (k *kafkaTransport) Type() RelayType { return RelayKafka }

// kafkaSyncProducer abstracts sarama.SyncProducer so tests can inject a fake.
// sarama's SyncProducer is safe for concurrent use, so the parallel send path
// shares one producer across workers.
type kafkaSyncProducer interface {
	SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error)
	Close() error
}

// SendFile publishes each chunk as one Kafka record. Kafka relay does not
// resume; it always sends from offset 0.
//
// With cfg.Parallelism <= 1 records are keyed by relPath, keeping a file's
// chunks ordered on one partition (the historical behaviour). With
// parallelism > 1 a worker pool produces chunks concurrently with a nil key
// and a round-robin partitioner, so chunks spread across ALL topic
// partitions — Kafka's unit of parallelism — instead of pinning the file to
// a single partition/broker. Order is then no longer guaranteed; the
// receiver tolerates this by tracking received byte ranges and finalising
// only once the whole file is present.
func (k *kafkaTransport) SendFile(ctx context.Context, cfg TaskConfig, _ TargetConfig, file FileEntry, read ChunkReader, sink ProgressSink) error {
	if cfg.Kafka == nil {
		return fmt.Errorf("kafka config not set for KAFKA relay type")
	}
	parallelism := effectiveParallelism(cfg.Parallelism)
	saramaCfg, err := buildKafkaConfig(cfg.Kafka)
	if err != nil {
		return err
	}
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 3
	if parallelism > 1 {
		// Spread chunks over every partition (see SendFile comment).
		saramaCfg.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	}
	maxMsgBytes := cfg.Kafka.MaxMessageBytes
	if maxMsgBytes <= 0 {
		maxMsgBytes = kafkaDefaultMaxMessageBytes
	}
	saramaCfg.Producer.MaxMessageBytes = maxMsgBytes
	saramaCfg.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokerList(cfg.Kafka.BootstrapServers), saramaCfg)
	if err != nil {
		return fmt.Errorf("create kafka producer: %w", err)
	}
	defer producer.Close()

	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSizeBytes
	}
	// Clamp the chunk size so one framed message (4-byte length prefix + JSON
	// header + raw chunk) never exceeds MaxMessageBytes.
	if maxChunk := maxMsgBytes - kafkaFrameLenBytes - kafkaHeaderMargin; chunkSize > maxChunk {
		log.Printf("filetransfer: kafka chunkSize clamped from %d to %d (maxMessageBytes=%d)", chunkSize, maxChunk, maxMsgBytes)
		chunkSize = maxChunk
	}

	return sendKafkaChunks(ctx, producer, cfg.Kafka.Topic, file, read, sink, chunkSize, parallelism)
}

// sendKafkaChunks publishes all chunks of file. With parallelism <= 1 it
// sends strictly in order, keyed by relPath so the file stays ordered on one
// partition; otherwise a worker pool sends chunks concurrently with nil keys
// (round-robin across partitions — order no longer guaranteed, the receiver
// reassembles by offset).
func sendKafkaChunks(ctx context.Context, producer kafkaSyncProducer, topic string, file FileEntry, read ChunkReader, sink ProgressSink, chunkSize, parallelism int) error {
	totalChunks := int((file.Size + int64(chunkSize) - 1) / int64(chunkSize))
	if totalChunks <= 0 {
		return nil
	}
	// A nil key lets the round-robin partitioner fan chunks across all
	// partitions; the relPath key is only needed for the ordered sequential
	// path.
	var key sarama.Encoder
	if parallelism <= 1 {
		key = sarama.StringEncoder(file.RelPath)
	}

	sendOne := func(idx int) error {
		offset := int64(idx) * int64(chunkSize)
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
			meta.Sha256 = file.Sha256
		}
		msg, err := encodeKafkaMessage(meta, data)
		if err != nil {
			return err
		}
		if _, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: topic,
			Key:   key,
			Value: sarama.ByteEncoder(msg),
		}); err != nil {
			return fmt.Errorf("kafka send: %w", err)
		}
		if sink != nil {
			sink(len(data))
		}
		return nil
	}

	if parallelism <= 1 {
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

	g, gctx := errgroup.WithContext(ctx)
	var next atomic.Int64
	for w := 0; w < parallelism; w++ {
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

// consumeKafka runs a consumer-group loop, invoking handle for each chunk
// message until ctx is cancelled.
func consumeKafka(ctx context.Context, cfg *KafkaConfig, handle func(meta ChunkMeta, data []byte) error) error {
	if cfg == nil {
		return fmt.Errorf("kafka config not set for KAFKA relay type")
	}
	saramaCfg, err := buildKafkaConfig(cfg)
	if err != nil {
		return err
	}
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(brokerList(cfg.BootstrapServers), cfg.GroupID, saramaCfg)
	if err != nil {
		return fmt.Errorf("create kafka consumer group: %w", err)
	}
	defer group.Close()

	handler := &chunkConsumerHandler{handle: handle}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := group.Consume(ctx, []string{cfg.Topic}, handler); err != nil {
			if err == sarama.ErrClosedConsumerGroup {
				return nil
			}
			return err
		}
	}
}

// chunkConsumerHandler adapts the chunk handler to sarama's consumer-group API.
type chunkConsumerHandler struct {
	handle func(meta ChunkMeta, data []byte) error
}

func (h *chunkConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *chunkConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *chunkConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			meta, data, err := decodeKafkaMessage(msg.Value)
			if err == nil {
				_ = h.handle(meta, data)
			}
			session.MarkMessage(msg, "")
		}
	}
}

// buildKafkaConfig builds a base sarama.Config with SASL/TLS from KafkaConfig.
func buildKafkaConfig(cfg *KafkaConfig) (*sarama.Config, error) {
	c := sarama.NewConfig()
	c.Version = sarama.V2_0_0_0
	compression, err := kafkaCompressionCodec(cfg.Compression)
	if err != nil {
		return nil, err
	}
	c.Producer.Compression = compression
	if cfg.AuthEnabled {
		c.Net.SASL.Enable = true
		c.Net.SASL.User = cfg.Username
		c.Net.SASL.Password = cfg.Password
		c.Net.SASL.Handshake = true
		mechanism := cfg.SaslMechanism
		if mechanism == "" {
			mechanism = "PLAIN"
		}
		switch strings.ToUpper(mechanism) {
		case "PLAIN":
			c.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case "SCRAM-SHA-256":
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			c.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
		protocol := cfg.SecurityProtocol
		if protocol == "" {
			protocol = "SASL_PLAINTEXT"
		}
		if strings.Contains(strings.ToUpper(protocol), "SSL") {
			c.Net.TLS.Enable = true
		}
	}
	return c, nil
}

func brokerList(bootstrap string) []string {
	parts := strings.Split(bootstrap, ",")
	brokers := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			brokers = append(brokers, s)
		}
	}
	return brokers
}

// kafkaCompressionCodec maps the configured compression name to a sarama
// codec. Empty defaults to snappy.
func kafkaCompressionCodec(name string) (sarama.CompressionCodec, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "snappy":
		return sarama.CompressionSnappy, nil
	case "none":
		return sarama.CompressionNone, nil
	case "zstd":
		return sarama.CompressionZSTD, nil
	case "lz4":
		return sarama.CompressionLZ4, nil
	case "gzip":
		return sarama.CompressionGZIP, nil
	default:
		return sarama.CompressionNone, fmt.Errorf("unsupported kafka compression: %q (want none|snappy|zstd|lz4|gzip)", name)
	}
}

// encodeKafkaMessage builds the Kafka message body as a binary frame:
// [4-byte big-endian header length][JSON(ChunkMeta)][raw chunk bytes].
// The chunk payload is carried unencoded so chunkSize translates directly
// into on-the-wire message size.
func encodeKafkaMessage(meta ChunkMeta, chunkData []byte) ([]byte, error) {
	header, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, kafkaFrameLenBytes, kafkaFrameLenBytes+len(header)+len(chunkData))
	binary.BigEndian.PutUint32(msg, uint32(len(header)))
	msg = append(msg, header...)
	msg = append(msg, chunkData...)
	return msg, nil
}

// decodeKafkaMessage parses a message produced by encodeKafkaMessage back into
// its ChunkMeta header and raw chunk bytes.
func decodeKafkaMessage(msg []byte) (ChunkMeta, []byte, error) {
	if len(msg) < kafkaFrameLenBytes {
		return ChunkMeta{}, nil, fmt.Errorf("malformed kafka message: shorter than frame prefix")
	}
	headerLen := int(binary.BigEndian.Uint32(msg[:kafkaFrameLenBytes]))
	if headerLen > len(msg)-kafkaFrameLenBytes {
		return ChunkMeta{}, nil, fmt.Errorf("malformed kafka message: header length %d exceeds message size %d", headerLen, len(msg))
	}
	var meta ChunkMeta
	if err := json.Unmarshal(msg[kafkaFrameLenBytes:kafkaFrameLenBytes+headerLen], &meta); err != nil {
		return ChunkMeta{}, nil, fmt.Errorf("malformed kafka message header: %w", err)
	}
	return meta, msg[kafkaFrameLenBytes+headerLen:], nil
}

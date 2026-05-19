# eBPF IP-Level Traffic Collector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `ebpf_traffic` collector to the existing `exporter/` package that observes per-remote-IP byte and packet counts (ingress + egress) using cgroup_skb eBPF programs, exposed as Prometheus metrics through the existing `/metrics` and `/api/resources/system` endpoints.

**Architecture:** A pair of `cgroup_skb` programs (ingress + egress) attach to the root cgroup v2 hierarchy at server startup. They count IPv4 packets and bytes into a single `BPF_MAP_TYPE_LRU_HASH` keyed by `(remote_ip, direction)`. At Prometheus scrape time the collector iterates the map, sorts entries by bytes, and emits the top-N as `node_ebpf_traffic_bytes_total{remote_ip,direction}` and `node_ebpf_traffic_packets_total{...}`. Entries trimmed by the top-N cap are reported via a `node_ebpf_traffic_truncated_flows` gauge. The collector is Linux-only via build tags; on non-Linux the `registerLinuxCollector` no-op already prevents the `init()` from registering anything, so no stub is needed.

**Tech Stack:**

- `github.com/cilium/ebpf` (pure-Go BPF loader, no CGo)
- `github.com/cilium/ebpf/cmd/bpf2go` (dev-time tool to compile C → BPF object + Go bindings)
- BPF C source compiled by `clang -target bpf` (run inside WSL / Linux; outputs are committed)
- `libbpf` headers (`apt install libbpf-dev` on Debian/Ubuntu)
- Existing `prometheus/client_golang` for metric emission

**Scope (MVP):**

- IPv4 only
- Two metric families: `bytes_total`, `packets_total`, plus `truncated_flows` gauge
- Labels: `remote_ip`, `direction` (`ingress`/`egress`)
- Hardcoded defaults: cgroup path `/sys/fs/cgroup`, top-N = 1000, loopback excluded
- No PID, no local IP, no port, no protocol breakdown
- No explicit Close lifecycle — process exit closes all BPF fds, kernel reclaims programs and maps

**Out of scope (future work):**

- IPv6
- Configurable top-N / cgroup path via `conf/config.yaml`
- Per-protocol (TCP/UDP) breakdown
- Subnet aggregation
- Graceful detach on `bbx-server` SIGTERM

---

## File structure

| Path | Status | Responsibility |
| --- | --- | --- |
| `exporter/ebpf_traffic.go` | Create | Collector implementation, build tag `//go:build linux` |
| `exporter/ebpf_traffic_test.go` | Create | Unit tests for `ipToString` and `topNByBytes`, build tag `//go:build linux` |
| `exporter/bpf/doc.go` | Create | Empty package declaration so the package exists on all platforms |
| `exporter/bpf/traffic.bpf.c` | Create | BPF C source (cgroup_skb ingress + egress) |
| `exporter/bpf/gen.go` | Create | Holds the `//go:generate` directive for `bpf2go` |
| `exporter/bpf/traffic_bpfel.go` | Create (generated, committed) | `bpf2go` Go bindings for little-endian targets (amd64, arm64) |
| `exporter/bpf/traffic_bpfel.o` | Create (generated, committed) | Compiled BPF object, little-endian |
| `exporter/bpf/traffic_bpfeb.go` | Create (generated, committed) | `bpf2go` Go bindings for big-endian targets |
| `exporter/bpf/traffic_bpfeb.o` | Create (generated, committed) | Compiled BPF object, big-endian |
| `go.mod` / `go.sum` | Modify | Add `github.com/cilium/ebpf` |
| `docs/api/metrics.md` (if exists) or `docs/api/README.md` | Modify | Mention the new metrics and required capabilities |
| `docs/superpowers/specs/2026-05-16-node-exporter-design.md` | Modify | Append "Additional collectors" section listing `ebpf_traffic` |

---

## Task 1: Add cilium/ebpf dependency

**Files:**

- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run inside the repo root (WSL or any Go-capable shell):

```bash
go get github.com/cilium/ebpf@v0.16.0
```

Expected: `go.mod` gains a direct `require github.com/cilium/ebpf v0.16.0` line, `go.sum` gains entries.

- [ ] **Step 2: Verify tidy**

```bash
go mod tidy
go build ./...
```

Expected: build succeeds, no new lint complaints. (No code uses the new dep yet, so the linker may drop it — that is fine.)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat(exporter): add cilium/ebpf dependency for eBPF collector"
```

---

## Task 2: Scaffold the `bpf` package with a placeholder doc file

A Go package needs at least one file that compiles on every target platform. The generated `traffic_bpf*.go` files all have `//go:build linux` constraints, so on Windows builds the package would otherwise be empty and the import in `ebpf_traffic.go` would fail. `doc.go` (no build tag) makes the package always exist.

**Files:**

- Create: `exporter/bpf/doc.go`

- [ ] **Step 1: Write the placeholder file**

```go
// Package bpf holds the eBPF traffic collector's compiled BPF object and
// the bpf2go-generated Go loader for the exporter package. The actual
// loader code lives in build-tagged generated files (traffic_bpfel.go,
// traffic_bpfeb.go) so this file simply anchors the package on platforms
// where no generated file matches.
package bpf
```

- [ ] **Step 2: Verify build**

```bash
go build ./exporter/bpf/...
```

Expected: succeeds on every platform.

- [ ] **Step 3: Commit**

```bash
git add exporter/bpf/doc.go
git commit -m "feat(exporter): scaffold bpf subpackage"
```

---

## Task 3: Write failing tests for the Go helpers

The two pieces of pure Go logic that warrant unit tests:

1. `ipToString(raw [4]byte) string` — converts a 4-byte network-order IPv4 address to its dotted-quad string. (`[4]byte` is what `bpf2go` exposes for the `__u8 remote_ipv4[4]` C field, so there is no endianness ambiguity to test — but a regression here would silently mangle every label, so the test guards intent.)
2. `topNByBytes(samples []flowSample, n int) (kept []flowSample, truncated int)` — sorts by bytes descending and returns the first `n` along with the count of dropped entries.

**Files:**

- Create: `exporter/ebpf_traffic_test.go`

- [ ] **Step 1: Write the failing tests**

```go
//go:build linux

package exporter

import "testing"

func TestIPToString(t *testing.T) {
	cases := []struct {
		name string
		in   [4]byte
		want string
	}{
		{"loopback", [4]byte{127, 0, 0, 1}, "127.0.0.1"},
		{"public", [4]byte{8, 8, 8, 8}, "8.8.8.8"},
		{"private", [4]byte{10, 0, 0, 5}, "10.0.0.5"},
		{"zero", [4]byte{0, 0, 0, 0}, "0.0.0.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ipToString(tc.in)
			if got != tc.want {
				t.Fatalf("ipToString(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTopNByBytes(t *testing.T) {
	samples := []flowSample{
		{remoteIP: "1.1.1.1", direction: "ingress", bytes: 100, packets: 1},
		{remoteIP: "2.2.2.2", direction: "ingress", bytes: 500, packets: 5},
		{remoteIP: "3.3.3.3", direction: "egress", bytes: 50, packets: 1},
		{remoteIP: "4.4.4.4", direction: "egress", bytes: 200, packets: 2},
	}

	t.Run("under cap", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 10)
		if truncated != 0 {
			t.Fatalf("truncated = %d, want 0", truncated)
		}
		if len(kept) != 4 {
			t.Fatalf("len(kept) = %d, want 4", len(kept))
		}
		// Verify descending byte order.
		for i := 1; i < len(kept); i++ {
			if kept[i-1].bytes < kept[i].bytes {
				t.Fatalf("kept not sorted descending: %v", kept)
			}
		}
	})

	t.Run("over cap", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 2)
		if truncated != 2 {
			t.Fatalf("truncated = %d, want 2", truncated)
		}
		if len(kept) != 2 {
			t.Fatalf("len(kept) = %d, want 2", len(kept))
		}
		// Top two by bytes are 500 then 200.
		if kept[0].remoteIP != "2.2.2.2" || kept[1].remoteIP != "4.4.4.4" {
			t.Fatalf("unexpected top-N: %v", kept)
		}
	})

	t.Run("cap zero keeps nothing", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 0)
		if len(kept) != 0 {
			t.Fatalf("len(kept) = %d, want 0", len(kept))
		}
		if truncated != 4 {
			t.Fatalf("truncated = %d, want 4", truncated)
		}
	})
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./exporter/ -run "TestIPToString|TestTopNByBytes" -v
```

Expected: compile error — `undefined: ipToString`, `undefined: flowSample`, `undefined: topNByBytes`. That is the expected initial failure mode for TDD.

- [ ] **Step 3: Commit the failing tests**

```bash
git add exporter/ebpf_traffic_test.go
git commit -m "test(exporter): add failing tests for ebpf_traffic helpers"
```

---

## Task 4: Implement `flowSample`, `ipToString`, and `topNByBytes`

These three live in the main collector file. They are the only parts of the collector that do not require the kernel or root to test, so we build them first and confirm the unit tests go green before introducing any eBPF code.

**Files:**

- Create: `exporter/ebpf_traffic.go`

- [ ] **Step 1: Write the file with just the helpers**

```go
/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

//go:build linux

package exporter

import (
	"net"
	"sort"
)

// flowSample is the per-(remote_ip, direction) snapshot the collector pulls
// from the BPF map at scrape time.
type flowSample struct {
	remoteIP  string
	direction string
	bytes     float64
	packets   float64
}

// ipToString converts a 4-byte IPv4 address stored in network byte order
// (the layout used by the BPF map's flow_key.remote_ipv4 field) into its
// dotted-quad string. The byte array layout is endian-independent because
// each byte is one octet; element [0] is the first octet of the address.
func ipToString(raw [4]byte) string {
	return net.IPv4(raw[0], raw[1], raw[2], raw[3]).String()
}

// topNByBytes returns the first n samples sorted by bytes descending and
// the count of samples that were dropped. Passing n == 0 keeps nothing
// and reports every sample as truncated. The input slice is sorted in
// place — callers that need the original order must pass a copy.
func topNByBytes(samples []flowSample, n int) (kept []flowSample, truncated int) {
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].bytes > samples[j].bytes
	})
	if n < 0 {
		n = 0
	}
	if n >= len(samples) {
		return samples, 0
	}
	return samples[:n], len(samples) - n
}
```

- [ ] **Step 2: Run the tests to verify they pass**

```bash
go test ./exporter/ -run "TestIPToString|TestTopNByBytes" -v
```

Expected:

```
=== RUN   TestIPToString
=== RUN   TestIPToString/loopback
=== RUN   TestIPToString/public
=== RUN   TestIPToString/private
=== RUN   TestIPToString/zero
--- PASS: TestIPToString (...)
=== RUN   TestTopNByBytes
=== RUN   TestTopNByBytes/under_cap
=== RUN   TestTopNByBytes/over_cap
=== RUN   TestTopNByBytes/cap_zero_keeps_nothing
--- PASS: TestTopNByBytes (...)
PASS
ok      github.com/casuallc/vigil/exporter      ...
```

- [ ] **Step 3: Commit**

```bash
git add exporter/ebpf_traffic.go
git commit -m "feat(exporter): add ipToString and topNByBytes helpers"
```

---

## Task 5: Write the BPF C source

The C program holds the actual packet-counting logic. It compiles to a single `.o` containing two programs (`count_ingress`, `count_egress`) and one map (`flows`). The C code is portable across LE host architectures and uses only stable `struct __sk_buff` fields plus a fixed-layout `struct iphdr`.

**Files:**

- Create: `exporter/bpf/traffic.bpf.c`

- [ ] **Step 1: Write the BPF source**

```c
// SPDX-License-Identifier: Apache-2.0
//
// cgroup_skb programs that count IPv4 bytes and packets per remote IP and
// direction. Attach the egress program with BPF_CGROUP_INET_EGRESS and
// the ingress program with BPF_CGROUP_INET_INGRESS to the root cgroup v2
// hierarchy (typically /sys/fs/cgroup).
//
// Loopback traffic (127.0.0.0/8) is skipped to keep the LRU map focused on
// useful entries. IPv6 packets are skipped: this is an MVP and IPv6 support
// is tracked as future work.

#include <linux/bpf.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

#ifndef AF_INET
#define AF_INET 2
#endif

#define DIRECTION_INGRESS 0
#define DIRECTION_EGRESS  1

struct flow_key {
    __u8 remote_ipv4[4]; /* network byte order, first-octet-first */
    __u8 direction;
    __u8 _pad[3];
};

struct flow_stats {
    __u64 bytes;
    __u64 packets;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, struct flow_key);
    __type(value, struct flow_stats);
    __uint(max_entries, 8192);
} flows SEC(".maps");

static __always_inline int count(struct __sk_buff *skb, __u8 direction)
{
    if (skb->family != AF_INET) {
        return 1; /* pass non-IPv4 unchanged */
    }

    void *data     = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct iphdr *ip = data;
    if ((void *)(ip + 1) > data_end) {
        return 1; /* malformed / truncated, pass */
    }

    __be32 remote_be = (direction == DIRECTION_EGRESS) ? ip->daddr : ip->saddr;

    struct flow_key key = {};
    __builtin_memcpy(&key.remote_ipv4, &remote_be, sizeof(remote_be));
    key.direction = direction;

    /* Skip 127.0.0.0/8 — first octet equals 127. The remote_ipv4 array
     * preserves network byte order, so element [0] is always the leading
     * octet regardless of host endianness. */
    if (key.remote_ipv4[0] == 127) {
        return 1;
    }

    struct flow_stats *st = bpf_map_lookup_elem(&flows, &key);
    if (st) {
        __sync_fetch_and_add(&st->bytes, skb->len);
        __sync_fetch_and_add(&st->packets, 1);
    } else {
        struct flow_stats init = { .bytes = skb->len, .packets = 1 };
        bpf_map_update_elem(&flows, &key, &init, BPF_NOEXIST);
    }
    return 1; /* always allow the packet */
}

SEC("cgroup_skb/egress")
int count_egress(struct __sk_buff *skb)
{
    return count(skb, DIRECTION_EGRESS);
}

SEC("cgroup_skb/ingress")
int count_ingress(struct __sk_buff *skb)
{
    return count(skb, DIRECTION_INGRESS);
}

char LICENSE[] SEC("license") = "Apache-2.0";
```

- [ ] **Step 2: Commit (compilation happens in the next task)**

```bash
git add exporter/bpf/traffic.bpf.c
git commit -m "feat(exporter): add eBPF cgroup_skb traffic counter source"
```

---

## Task 6: Generate bpf2go bindings and the compiled BPF object

`bpf2go` runs `clang -target bpf` on the C source and emits a `.o` plus Go bindings. The outputs are committed so end users do not need clang to build.

**Files:**

- Create: `exporter/bpf/gen.go`
- Create (generated, committed): `exporter/bpf/traffic_bpfel.go`, `exporter/bpf/traffic_bpfel.o`, `exporter/bpf/traffic_bpfeb.go`, `exporter/bpf/traffic_bpfeb.o`

- [ ] **Step 1: Add the `go:generate` directive**

```go
// Package bpf — code generation driver.
//
// To regenerate the bindings and BPF object after modifying traffic.bpf.c,
// run inside WSL or a Linux shell with clang + libbpf headers installed:
//
//     sudo apt-get install -y clang libbpf-dev
//     go generate ./exporter/bpf/...
//
// The generated _bpfel and _bpfeb files plus the compiled .o objects are
// committed to the repository, so a plain `go build` does NOT require clang.

//go:build ignore

package bpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -I/usr/include/bpf" -type flow_key -type flow_stats traffic traffic.bpf.c
```

- [ ] **Step 2: Install the toolchain (WSL or Linux only — skip if already installed)**

```bash
sudo apt-get update
sudo apt-get install -y clang libbpf-dev
```

Expected: `clang --version` reports a version, `/usr/include/bpf/bpf_helpers.h` exists.

- [ ] **Step 3: Run the generator**

```bash
go generate ./exporter/bpf/...
```

Expected output (file names — sizes vary):

```
Compiled exporter/bpf/traffic_bpfel.o
Stripped exporter/bpf/traffic_bpfel.o
Wrote    exporter/bpf/traffic_bpfel.go
Compiled exporter/bpf/traffic_bpfeb.o
Stripped exporter/bpf/traffic_bpfeb.o
Wrote    exporter/bpf/traffic_bpfeb.go
```

Inspect the generated `_bpfel.go` file — it should define a `loadTraffic` function and types `trafficFlowKey`, `trafficFlowStats`, `trafficObjects`, `trafficPrograms`, `trafficMaps`. If the type names differ, write them down — the next task references them.

- [ ] **Step 4: Verify build still works on every platform**

```bash
GOOS=linux GOARCH=amd64 go build ./...
GOOS=linux GOARCH=arm64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```

Expected: all three succeed. The Windows build excludes both `_bpfel.go` (linux build tag) and `_bpfeb.go`, leaving only `doc.go` in the `bpf` package.

- [ ] **Step 5: Commit the generated artifacts**

```bash
git add exporter/bpf/gen.go exporter/bpf/traffic_bpfel.go exporter/bpf/traffic_bpfeb.go exporter/bpf/traffic_bpfel.o exporter/bpf/traffic_bpfeb.o
git commit -m "feat(exporter): generate eBPF object and Go bindings via bpf2go"
```

---

## Task 7: Implement the collector loader and attach logic

Now wire the generated bindings into the existing collector. This task adds the constructor, `Name()`, and `Update()` skeleton. `Update()` reads the BPF map, converts entries to `flowSample`, applies `topNByBytes`, and emits Prometheus metrics.

**Files:**

- Modify: `exporter/ebpf_traffic.go`

- [ ] **Step 1: Extend `ebpf_traffic.go`**

Append (the helpers from Task 4 stay at the top of the file; this block goes below them):

```go
import (
	"errors"
	"fmt"
	"os"

	"github.com/casuallc/vigil/exporter/bpf"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/prometheus/client_golang/prometheus"
)
```

Replace the existing import block in the file with the merged one (`net`, `sort` from Task 4 plus the new imports above). Then add:

```go
const (
	ebpfTrafficCollectorName = "ebpf_traffic"
	ebpfTrafficDefaultCgroup = "/sys/fs/cgroup"
	ebpfTrafficDefaultTopN   = 1000
)

// ebpfTrafficCollector observes per-remote-IPv4 byte and packet counts
// using two cgroup_skb BPF programs attached to the root cgroup v2.
// Construction loads and attaches the programs; the kernel auto-detaches
// when the bbx-server process exits and closes the link fds.
type ebpfTrafficCollector struct {
	objs    bpf.TrafficObjects
	ingress link.Link
	egress  link.Link
	topN    int
}

func newEBPFTrafficCollector() (Collector, error) {
	// BPF maps need locked memory above the default rlimit on older kernels.
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("ebpf_traffic: remove memlock rlimit: %w", err)
	}

	cgroupPath := ebpfTrafficDefaultCgroup
	if _, err := os.Stat(cgroupPath + "/cgroup.controllers"); err != nil {
		return nil, fmt.Errorf("ebpf_traffic: cgroup v2 not mounted at %s: %w", cgroupPath, err)
	}

	var objs bpf.TrafficObjects
	if err := bpf.LoadTrafficObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("ebpf_traffic: load BPF objects: %w", err)
	}

	egressLink, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetEgress,
		Program: objs.CountEgress,
	})
	if err != nil {
		_ = objs.Close()
		return nil, fmt.Errorf("ebpf_traffic: attach egress cgroup_skb: %w", err)
	}

	ingressLink, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetIngress,
		Program: objs.CountIngress,
	})
	if err != nil {
		_ = egressLink.Close()
		_ = objs.Close()
		return nil, fmt.Errorf("ebpf_traffic: attach ingress cgroup_skb: %w", err)
	}

	return &ebpfTrafficCollector{
		objs:    objs,
		ingress: ingressLink,
		egress:  egressLink,
		topN:    ebpfTrafficDefaultTopN,
	}, nil
}

func (c *ebpfTrafficCollector) Name() string { return ebpfTrafficCollectorName }

func (c *ebpfTrafficCollector) Update(ch chan<- prometheus.Metric) error {
	samples, err := c.snapshot()
	if err != nil {
		return err
	}

	kept, truncated := topNByBytes(samples, c.topN)

	bytesDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ebpf_traffic", "bytes_total"),
		"Bytes seen per remote IPv4 address and direction by cgroup_skb hooks since program load.",
		[]string{"remote_ip", "direction"}, nil,
	)
	packetsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ebpf_traffic", "packets_total"),
		"Packets seen per remote IPv4 address and direction by cgroup_skb hooks since program load.",
		[]string{"remote_ip", "direction"}, nil,
	)
	truncatedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ebpf_traffic", "truncated_flows"),
		"Number of map entries that were dropped from the top-N at this scrape.",
		nil, nil,
	)

	for _, s := range kept {
		ch <- prometheus.MustNewConstMetric(bytesDesc, prometheus.CounterValue, s.bytes, s.remoteIP, s.direction)
		ch <- prometheus.MustNewConstMetric(packetsDesc, prometheus.CounterValue, s.packets, s.remoteIP, s.direction)
	}
	ch <- prometheus.MustNewConstMetric(truncatedDesc, prometheus.GaugeValue, float64(truncated))
	return nil
}

// snapshot iterates the BPF map once and converts every entry into a flowSample.
// Iterating an LRU hash map under concurrent kernel writes is supported by
// cilium/ebpf; entries the kernel evicts mid-iteration are simply skipped.
func (c *ebpfTrafficCollector) snapshot() ([]flowSample, error) {
	samples := make([]flowSample, 0, 256)
	var (
		key bpf.TrafficFlowKey
		val bpf.TrafficFlowStats
	)
	iter := c.objs.Flows.Iterate()
	for iter.Next(&key, &val) {
		direction := "ingress"
		if key.Direction == 1 {
			direction = "egress"
		}
		samples = append(samples, flowSample{
			remoteIP:  ipToString(key.RemoteIpv4),
			direction: direction,
			bytes:     float64(val.Bytes),
			packets:   float64(val.Packets),
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("ebpf_traffic: iterate flows map: %w", err)
	}
	return samples, nil
}

// errEBPFNotAvailable is returned from the factory when the runtime
// environment cannot support the collector. The exporter's factory wrapper
// already logs-and-skips on factory errors, but exporting a sentinel makes
// callers and tests able to distinguish "missing capability" from a real bug.
var errEBPFNotAvailable = errors.New("ebpf_traffic: not available on this host")
```

Notes about the generated names:

- `bpf2go` derives Go type names from the C type names: `flow_key` → `TrafficFlowKey`, `flow_stats` → `TrafficFlowStats` (the `Traffic` prefix is the bpf2go program name).
- The objects struct field names track the C `SEC("…")` names: `count_egress` → `CountEgress`, `count_ingress` → `CountIngress`, `flows` map → `Flows`.
- If your `bpf2go` produced different names (e.g. an older version), adjust the snapshot loop and `link.AttachCgroup` calls to match what the generated `traffic_bpfel.go` exposes.

- [ ] **Step 2: Compile**

```bash
GOOS=linux GOARCH=amd64 go build ./...
```

Expected: succeeds.

- [ ] **Step 3: Re-run the unit tests**

```bash
go test ./exporter/ -run "TestIPToString|TestTopNByBytes" -v
```

Expected: both tests still pass. The eBPF-loading code is not exercised by these tests; that is intentional — it requires root and a kernel, and is covered by the smoke test in Task 9.

- [ ] **Step 4: Commit**

```bash
git add exporter/ebpf_traffic.go
git commit -m "feat(exporter): implement eBPF traffic collector loader and Update"
```

---

## Task 8: Register the collector with the exporter

The collector self-registers via `init()` just like every other Linux collector. Factory failures (no cgroup v2, no CAP_NET_ADMIN, kernel too old) cause `defaultLinuxCollectors()` (`exporter/exporter_linux.go:36`) to skip the collector and log nothing — but for an opt-in capability that requires elevated privileges it is worth logging *why* the collector failed, otherwise the operator has no way to know it tried.

**Files:**

- Modify: `exporter/ebpf_traffic.go`
- Modify: `exporter/exporter_linux.go`

- [ ] **Step 1: Add the init() in `exporter/ebpf_traffic.go`**

Append at the bottom of `exporter/ebpf_traffic.go`:

```go
func init() {
	registerLinuxCollector(ebpfTrafficCollectorName, newEBPFTrafficCollector)
}
```

- [ ] **Step 2: Improve factory error visibility in `exporter/exporter_linux.go`**

The current loop silently drops failed factories. For an eBPF collector that needs root, silent skipping makes operational issues invisible. Change `defaultLinuxCollectors` to log factory failures.

Locate:

```go
func defaultLinuxCollectors() map[string]Collector {
	out := make(map[string]Collector, len(linuxCollectorFactories))
	for name, factory := range linuxCollectorFactories {
		c, err := factory()
		if err != nil {
			// A factory failure usually means the collector cannot read its
			// data source on this kernel/build. We log and skip rather than
			// fail the entire exporter so the rest still works.
			continue
		}
		out[name] = c
	}
	return out
}
```

Replace with:

```go
func defaultLinuxCollectors() map[string]Collector {
	out := make(map[string]Collector, len(linuxCollectorFactories))
	for name, factory := range linuxCollectorFactories {
		c, err := factory()
		if err != nil {
			// A factory failure usually means the collector cannot read its
			// data source on this kernel/build. We log and skip rather than
			// fail the entire exporter so the rest still works.
			log.Printf("exporter: skipping collector %s: %v", name, err)
			continue
		}
		out[name] = c
	}
	return out
}
```

Add `"log"` to the file's import list if it is not already present.

- [ ] **Step 3: Build**

```bash
GOOS=linux GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```

Expected: both succeed.

- [ ] **Step 4: Run the full exporter test suite**

```bash
go test ./exporter/... -v
```

Expected: every existing test still passes. New tests pass. No new test files yet exercise the BPF runtime path (that is the smoke test).

- [ ] **Step 5: Commit**

```bash
git add exporter/ebpf_traffic.go exporter/exporter_linux.go
git commit -m "feat(exporter): register ebpf_traffic collector and log factory skips"
```

---

## Task 9: Smoke test on Linux (manual)

The eBPF program cannot be unit-tested in any meaningful way without a kernel and root. This task is a manual recipe to verify the collector works end-to-end on a Linux host (WSL2 with cgroup v2 enabled is sufficient, as are typical Linux VMs).

**Prerequisites on the target host:**

- Linux kernel 5.4 or newer (`uname -r`)
- cgroup v2 mounted at `/sys/fs/cgroup` (`stat /sys/fs/cgroup/cgroup.controllers` succeeds)
- Run as root, or grant the binary `CAP_BPF` + `CAP_NET_ADMIN`

- [ ] **Step 1: Build the server binary for Linux**

From WSL or a Linux shell:

```bash
GOOS=linux GOARCH=amd64 go build -o bbx-server ./cmd/bbx-server
```

Expected: produces `bbx-server`.

- [ ] **Step 2: Run the server with capabilities**

```bash
sudo ./bbx-server -config conf/config.yaml
```

Expected log line within the first second of startup:

```
exporter: collector ebpf_traffic loaded
```

(That line does not exist yet — what you should actually see is the *absence* of `exporter: skipping collector ebpf_traffic: ...`. If a skip line appears, the factory failed; read its error and fix before continuing.)

- [ ] **Step 3: Generate some traffic from another shell**

```bash
curl -s https://www.google.com > /dev/null
ping -c 4 1.1.1.1
```

- [ ] **Step 4: Scrape `/metrics` and grep for the new metrics**

```bash
curl -s -u admin:<password> http://127.0.0.1:57575/metrics | grep -E "node_ebpf_traffic_(bytes|packets|truncated)"
```

Expected: at minimum, one line each for `node_ebpf_traffic_bytes_total{remote_ip="1.1.1.1",direction="egress"}` and the corresponding `ingress`. `node_ebpf_traffic_truncated_flows 0` should also appear (assuming the host has fewer than 1000 distinct flows since boot).

- [ ] **Step 5: Scrape the JSON endpoint and confirm grouping**

```bash
curl -s -u admin:<password> http://127.0.0.1:57575/api/resources/system | jq '.ebpf_traffic | keys'
```

Expected:

```json
[
  "node_ebpf_traffic_bytes_total",
  "node_ebpf_traffic_packets_total",
  "node_ebpf_traffic_truncated_flows"
]
```

- [ ] **Step 6: Document the result**

If anything was off (missing metrics, factory skipped, label values mangled), debug before declaring the smoke test complete. Otherwise, no commit needed — this task produces operational evidence, not code.

---

## Task 10: Document the new collector

Operators need to know the metric names, the capability requirements, and the loopback exclusion.

**Files:**

- Modify: `docs/superpowers/specs/2026-05-16-node-exporter-design.md` (append section)
- Modify: `docs/api/README.md` (if it lists endpoints — add a one-liner about the new metrics)

- [ ] **Step 1: Append to `docs/superpowers/specs/2026-05-16-node-exporter-design.md`**

Add at the bottom of the file:

```markdown
## Additional collectors

### `ebpf_traffic` (Linux 5.4+)

Per-remote-IPv4 byte and packet counters collected by two `cgroup_skb` BPF
programs (ingress + egress) attached to the root cgroup v2 hierarchy.

Metrics:

- `node_ebpf_traffic_bytes_total{remote_ip,direction}` — counter
- `node_ebpf_traffic_packets_total{remote_ip,direction}` — counter
- `node_ebpf_traffic_truncated_flows` — gauge, number of map entries dropped
  from the top-N at the last scrape

Defaults (hardcoded in MVP):

- cgroup path: `/sys/fs/cgroup`
- top-N: 1000 flows by byte count
- loopback (127.0.0.0/8) excluded
- IPv4 only
- BPF map: `BPF_MAP_TYPE_LRU_HASH`, `max_entries=8192`

Capability requirements: `CAP_BPF` + `CAP_NET_ADMIN` (or root). Without these
the collector fails to load and is silently dropped from the registry; the
log line `exporter: skipping collector ebpf_traffic: ...` records the reason.

To rebuild the BPF object after modifying `exporter/bpf/traffic.bpf.c`,
run `go generate ./exporter/bpf/...` inside WSL or a Linux shell with
`clang` and `libbpf-dev` installed. The generated `_bpfel`/`_bpfeb` files
and `.o` objects are committed to the repository.
```

- [ ] **Step 2: If `docs/api/README.md` indexes endpoints, add a line**

Find the section that lists `/metrics`. Add or append:

```markdown
- `node_ebpf_traffic_bytes_total` / `node_ebpf_traffic_packets_total` — per-remote-IPv4 byte and packet counters (Linux only, requires CAP_BPF + CAP_NET_ADMIN)
```

If no such index exists in `docs/api/README.md`, skip this step.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/specs/2026-05-16-node-exporter-design.md docs/api/README.md
git commit -m "docs(exporter): document ebpf_traffic collector"
```

---

## Self-review

**Spec coverage (against the four decisions from the conversation):**

| Decision | Implemented by |
| --- | --- |
| 1. Aggregation: remote_ip + direction | BPF map key `flow_key { remote_ipv4, direction }` (Task 5); labels `remote_ip`, `direction` (Task 7) |
| 2. No PID | No `bpf_get_current_pid_tgid` call in `traffic.bpf.c` (Task 5) |
| 3. No local IP | Only one IP stored per packet — `daddr` on egress, `saddr` on ingress (Task 5) |
| 4. IP-level + bytes/packets | `flow_stats { bytes, packets }` map value (Task 5); `bytes_total` + `packets_total` metric families (Task 7); no port/proto labels |

**Architecture concerns covered:**

- Lifecycle: documented "no explicit Close in MVP, kernel reclaims at process exit" in the Architecture header and again in Task 7's struct comment.
- Top-N cardinality cap: `topNByBytes` (Task 4), wired into `Update()` (Task 7), reported via truncated gauge.
- Cross-platform: `//go:build linux` on `ebpf_traffic.go` and its test; `bpf/doc.go` keeps the package present on Windows (Task 2); build verified for linux/amd64, linux/arm64, windows/amd64 in Task 6 and Task 8.
- Capability failure: factory returns error → `defaultLinuxCollectors()` now logs and skips (Task 8).
- BPF object portability: bpf2go produces both `_bpfel` and `_bpfeb` (Task 6); generated artifacts committed so end users do not need clang.

**Placeholder scan:** No "TBD", no "TODO", no "implement later", no "add validation" without showing how, no "similar to Task N" references. All code blocks contain complete, runnable code or commands.

**Type consistency:**

- `flowSample` defined in Task 4, used unchanged in Task 7's `snapshot()` and `Update()`.
- `bpf.TrafficObjects`, `bpf.TrafficFlowKey`, `bpf.TrafficFlowStats`, `bpf.LoadTrafficObjects` — these names are derived by bpf2go from the C `traffic` program name and the `flow_key`/`flow_stats` C struct names declared in Task 5. Task 7 notes that the generator's actual output should be inspected; if the generator version emits different names the loader code must be adjusted to match (one place: the `snapshot()` loop and the `link.AttachCgroup` `Program:` fields).
- `topNByBytes(samples, n)` signature in Task 3's tests matches the implementation in Task 4 and the caller in Task 7.
- `ipToString(raw [4]byte)` signature in Task 3's tests matches Task 4 and the caller in Task 7.

**Gaps acknowledged (deliberately out of MVP scope, listed in the header):**

- No IPv6 — `skb->family != AF_INET` short-circuits in the BPF program.
- No config wiring — `topN`, `cgroupPath`, loopback exclusion are constants. Adding `conf/config.yaml` integration is straightforward later: thread `*config.Config` into `NewNodeExporter`, then into the factory closure.
- No explicit `Close()` lifecycle — relies on process-exit fd cleanup. Documented in Task 7's struct comment.
- No automated kernel-side integration test in CI — Task 9 is manual. Adding a kind/qemu-based job is future work.

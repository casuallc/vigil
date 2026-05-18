# BPF Object Build Guide

This package contains eBPF programs and their Go bindings for the
`ebpf_traffic` collector. The `.o` files are pre-compiled ELF objects that
are embedded into the Go binary via `//go:embed`.

## Files

| File | Purpose |
|---|---|
| `traffic.bpf.c` | cgroup_skb BPF programs (cgroup v2 mode) |
| `traffic_tc.bpf.c` | TC BPF programs (cgroup v1 fallback mode) |
| `traffic_*.o` | Compiled BPF ELF objects (embedded in Go binary) |
| `traffic_*_bpfel.go` | Little-endian Go bindings (generated) |
| `traffic_*_bpfeb.go` | Big-endian Go bindings (generated) |
| `gen.go` | Code generation driver (`go generate` entry point) |
| `exports.go` | Exported type aliases and loader wrappers |

## Prerequisites

- Linux with **clang** and **libbpf-dev** installed
- Go toolchain

```bash
sudo apt-get install -y clang libbpf-dev
```

## Regenerating BPF Objects

After modifying any `.bpf.c` file, regenerate the bindings and objects:

```bash
go generate ./exporter/bpf/gen.go
```

This invokes `bpf2go` to:
1. Compile each `.bpf.c` to `.o` (for little-endian and big-endian targets)
2. Generate corresponding Go loader files (`*_bpfel.go`, `*_bpfeb.go`)

The generated `.o` and `.go` files are **committed to the repository** so that
a plain `go build` does not require clang.

## Build Flags

The `gen.go` cflags include `-I/usr/include/x86_64-linux-gnu` for Ubuntu's
multi-arch layout. On ARM64 Linux, replace it with
`-I/usr/include/aarch64-linux-gnu` before running `go generate`.

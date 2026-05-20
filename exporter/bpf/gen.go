// Package bpf — code generation driver.
//
// To regenerate the bindings and BPF objects after modifying the .bpf.c files,
// run inside WSL or a Linux shell with clang + libbpf headers installed:
//
//     sudo apt-get install -y clang libbpf-dev
//     go generate ./exporter/bpf/gen.go
//
// NOTE: target the file directly (`gen.go`), NOT the package wildcard
// (`./exporter/bpf/...`). The `//go:build ignore` tag below causes
// `go generate` to silently skip this file when matched via `...`.
//
// Architecture note: the cflags below include
// `-I/usr/include/x86_64-linux-gnu` to locate `asm/types.h` on Ubuntu's
// multi-arch layout. On arm64 Linux replace it with
// `-I/usr/include/aarch64-linux-gnu`. Long-term we should switch to a
// vendored `vmlinux.h` (CO-RE) to remove the kernel/asm header dependency.
//
// The generated _bpfel and _bpfeb files plus the compiled .o objects are
// committed to the repository, so a plain `go build` does NOT require clang.

//go:build ignore

package bpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -I/usr/include/bpf -I/usr/include/x86_64-linux-gnu" -type flow_key -type flow_stats traffic _traffic.bpf.c
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -I/usr/include/bpf -I/usr/include/x86_64-linux-gnu" -type flow_key -type flow_stats traffic_tc _traffic_tc.bpf.c

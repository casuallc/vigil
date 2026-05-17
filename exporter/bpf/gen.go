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

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -I/usr/include/bpf -I/usr/include/x86_64-linux-gnu" -type flow_key -type flow_stats traffic traffic.bpf.c

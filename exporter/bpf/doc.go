// Package bpf holds the eBPF traffic collector's compiled BPF object and
// the bpf2go-generated Go loader for the exporter package. The actual
// loader code lives in build-tagged generated files (traffic_bpfel.go,
// traffic_bpfeb.go) so this file simply anchors the package on platforms
// where no generated file matches.
package bpf

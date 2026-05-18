/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

//go:build linux

package bpf

import "github.com/cilium/ebpf"

// Objects exposes the bpf2go-generated objects struct under an exported
// name so external packages can declare values of this type. The
// underlying type embeds the (unexported) trafficPrograms and trafficMaps
// structs whose fields (`CountEgress`, `CountIngress`, `Flows`) are
// already exported and thus accessible via field promotion.
type Objects = trafficObjects

// FlowKey and FlowStats are exported aliases for the bpf2go-generated
// map key and value types. The C struct field names (remote_ipv4,
// direction, _pad, bytes, packets) appear as RemoteIpv4, Direction, Pad,
// Bytes, Packets on the Go side per bpf2go's naming convention.
type FlowKey = trafficFlowKey
type FlowStats = trafficFlowStats

// LoadObjects loads the eBPF programs and maps from the embedded BPF ELF
// into the kernel and populates *o with the loaded handles. It is a thin
// wrapper around the bpf2go-generated `loadTrafficObjects`.
func LoadObjects(o *Objects, opts *ebpf.CollectionOptions) error {
	return loadTrafficObjects(o, opts)
}

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

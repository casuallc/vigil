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
	"errors"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/casuallc/vigil/exporter/bpf"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/prometheus/client_golang/prometheus"
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
	objs    bpf.Objects
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

	var objs bpf.Objects
	if err := bpf.LoadObjects(&objs, nil); err != nil {
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
		key bpf.FlowKey
		val bpf.FlowStats
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

//go:build linux

/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/casuallc/vigil/exporter/bpf"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func init() {
	registerLinuxCollector(ebpfTrafficCollectorName, newEBPFTrafficCollector)
}

// flowSample is the per-(remote_ip, ifindex, direction) snapshot the collector
// pulls from the BPF map at scrape time.
type flowSample struct {
	remoteIP  string
	ifaceName string
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

	// directionEgress matches DIRECTION_EGRESS in the BPF C code.
	// The BPF program stores 0 for ingress and 1 for egress in flow_key.direction;
	// keep these two definitions in sync.
	directionEgress uint8 = 1
)

// flowKey mirrors the BPF struct flow_key layout (12 bytes).
type flowKey struct {
	RemoteIpv4 [4]uint8
	Ifindex    uint32
	Direction  uint8
	Pad        [3]uint8
}

// flowStats mirrors the BPF struct flow_stats layout (16 bytes).
type flowStats struct {
	Bytes   uint64
	Packets uint64
}

// ebpfTrafficCollector observes per-remote-IPv4 byte and packet counts
// using either cgroup_skb BPF programs (cgroup v2) or TC BPF programs
// (cgroup v1 fallback). The mode is auto-detected at construction time.
type ebpfTrafficCollector struct {
	flows *ebpf.Map
	topN  int
	close func() error
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
		"Bytes seen per remote IPv4 address, interface and direction since program load.",
		[]string{"remote_ip", "interface", "direction"}, nil,
	)
	packetsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ebpf_traffic", "packets_total"),
		"Packets seen per remote IPv4 address, interface and direction since program load.",
		[]string{"remote_ip", "interface", "direction"}, nil,
	)
	truncatedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ebpf_traffic", "truncated_flows"),
		"Number of map entries that were dropped from the top-N at this scrape.",
		nil, nil,
	)

	for _, s := range kept {
		ch <- prometheus.MustNewConstMetric(bytesDesc, prometheus.CounterValue, s.bytes, s.remoteIP, s.ifaceName, s.direction)
		ch <- prometheus.MustNewConstMetric(packetsDesc, prometheus.CounterValue, s.packets, s.remoteIP, s.ifaceName, s.direction)
	}
	ch <- prometheus.MustNewConstMetric(truncatedDesc, prometheus.GaugeValue, float64(truncated))
	return nil
}

// snapshot iterates the BPF map once and converts every entry into a flowSample.
func (c *ebpfTrafficCollector) snapshot() ([]flowSample, error) {
	samples := make([]flowSample, 0, 256)
	var key flowKey
	var val flowStats

	iter := c.flows.Iterate()
	for iter.Next(&key, &val) {
		direction := "ingress"
		if key.Direction == directionEgress {
			direction = "egress"
		}
		ifaceName := ""
		if iface, err := net.InterfaceByIndex(int(key.Ifindex)); err == nil {
			ifaceName = iface.Name
		}
		samples = append(samples, flowSample{
			remoteIP:  ipToString(key.RemoteIpv4),
			ifaceName: ifaceName,
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

// kernelVersionAtLeast returns true if the running kernel is >= major.minor.patch.
// It parses the release string from unix.Uname (e.g. "3.10.0-1160.119.1.el7.x86_64").
func kernelVersionAtLeast(major, minor, patch int) bool {
	var utsname unix.Utsname
	if err := unix.Uname(&utsname); err != nil {
		return false
	}
	release := byteSliceToString(utsname.Release[:])
	parts := strings.Split(release, ".")
	if len(parts) < 2 {
		return false
	}
	km, _ := strconv.Atoi(parts[0])
	kmi, _ := strconv.Atoi(parts[1])

	kpatch := 0
	if len(parts) >= 3 {
		patchPart := strings.Split(parts[2], "-")[0]
		kpatch, _ = strconv.Atoi(patchPart)
	}

	if km != major {
		return km > major
	}
	if kmi != minor {
		return kmi > minor
	}
	return kpatch >= patch
}

func byteSliceToString(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n == -1 {
		n = len(b)
	}
	return string(b[:n])
}

// newEBPFTrafficCollector tries cgroup v2 first, then falls back to TC eBPF.
func newEBPFTrafficCollector() (Collector, error) {
	// BPF_MAP_TYPE_LRU_HASH was introduced in kernel 4.10.  The pre-compiled
	// objects also rely on BPF features that may not exist on older kernels.
	// Skip gracefully rather than fail at load time.
	if !kernelVersionAtLeast(4, 10, 0) {
		return nil, fmt.Errorf("ebpf_traffic requires kernel >= 4.10 (BPF_MAP_TYPE_LRU_HASH)")
	}

	// BPF maps need locked memory above the default rlimit on older kernels.
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("ebpf_traffic: remove memlock rlimit: %w", err)
	}

	// Try cgroup v2 first.
	cgroupPath := ebpfTrafficDefaultCgroup
	if _, err := os.Stat(cgroupPath + "/cgroup.controllers"); err == nil {
		c, err := newCgroupTrafficCollector(cgroupPath)
		if err == nil {
			log.Printf("exporter: ebpf_traffic using cgroup_skb mode")
			return c, nil
		}
		log.Printf("exporter: ebpf_traffic cgroup_skb failed (%v), trying TC fallback", err)
	}

	// Fallback to TC eBPF.
	c, err := newTCTrafficCollector()
	if err == nil {
		log.Printf("exporter: ebpf_traffic using TC mode")
		return c, nil
	}

	return nil, fmt.Errorf("ebpf_traffic: cgroup and TC both failed, last error: %w", err)
}

// newCgroupTrafficCollector loads cgroup_skb programs and attaches them to
// the root cgroup v2 hierarchy.
func newCgroupTrafficCollector(cgroupPath string) (Collector, error) {
	var objs bpf.Objects
	if err := bpf.LoadObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("load BPF objects: %w", err)
	}

	egressLink, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetEgress,
		Program: objs.CountEgress,
	})
	if err != nil {
		_ = objs.Close()
		return nil, fmt.Errorf("attach egress cgroup_skb: %w", err)
	}

	ingressLink, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetIngress,
		Program: objs.CountIngress,
	})
	if err != nil {
		_ = egressLink.Close()
		_ = objs.Close()
		return nil, fmt.Errorf("attach ingress cgroup_skb: %w", err)
	}

	return &ebpfTrafficCollector{
		flows: objs.Flows,
		topN:  ebpfTrafficDefaultTopN,
		close: func() error {
			var errs []error
			if err := ingressLink.Close(); err != nil {
				errs = append(errs, err)
			}
			if err := egressLink.Close(); err != nil {
				errs = append(errs, err)
			}
			if err := objs.Close(); err != nil {
				errs = append(errs, err)
			}
			if len(errs) > 0 {
				return errs[0]
			}
			return nil
		},
	}, nil
}

// tcFilterInfo tracks a TC filter for cleanup.
type tcFilterInfo struct {
	ifaceIndex int
	parent     uint32
	handle     uint32
}

// newTCTrafficCollector loads TC BPF programs and attaches them to all
// eligible network interfaces via clsact qdisc.
func newTCTrafficCollector() (Collector, error) {
	var objs bpf.TcObjects
	if err := bpf.LoadTcObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("load TC BPF objects: %w", err)
	}

	interfaces, err := eligibleInterfaces()
	if err != nil {
		_ = objs.Close()
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	if len(interfaces) == 0 {
		_ = objs.Close()
		return nil, fmt.Errorf("no eligible network interfaces")
	}

	var filters []tcFilterInfo
	for _, iface := range interfaces {
		if err := cleanupVigilFilters(iface.Index); err != nil {
			log.Printf("exporter: ebpf_traffic warning: cleanup old filters on %s: %v", iface.Name, err)
		}

		qdisc := &netlink.Clsact{
			QdiscAttrs: netlink.QdiscAttrs{
				LinkIndex: iface.Index,
				Handle:    netlink.MakeHandle(0xffff, 0),
				Parent:    netlink.HANDLE_INGRESS,
			},
		}
		if err := netlink.QdiscAdd(qdisc); err != nil {
			if !isEEXIST(err) {
				cleanupTCFilters(filters)
				_ = objs.Close()
				return nil, fmt.Errorf("add clsact qdisc on %s: %w", iface.Name, err)
			}
		}

		ingressFilter := &netlink.BpfFilter{
			FilterAttrs: netlink.FilterAttrs{
				LinkIndex: iface.Index,
				Parent:    netlink.MakeHandle(0xffff, 0xfff2),
				Handle:    netlink.MakeHandle(0, 0x100),
				Protocol:  unix.ETH_P_ALL,
			},
			Fd:           objs.TcCountIngress.FD(),
			Name:         "vigil_tc_ingress",
			DirectAction: true,
		}
		if err := netlink.FilterAdd(ingressFilter); err != nil {
			cleanupTCFilters(filters)
			_ = objs.Close()
			return nil, fmt.Errorf("attach ingress filter on %s: %w", iface.Name, err)
		}
		filters = append(filters, tcFilterInfo{
			ifaceIndex: iface.Index,
			parent:     netlink.MakeHandle(0xffff, 0xfff2),
			handle:     netlink.MakeHandle(0, 0x100),
		})

		egressFilter := &netlink.BpfFilter{
			FilterAttrs: netlink.FilterAttrs{
				LinkIndex: iface.Index,
				Parent:    netlink.MakeHandle(0xffff, 0xfff3),
				Handle:    netlink.MakeHandle(0, 0x101),
				Protocol:  unix.ETH_P_ALL,
			},
			Fd:           objs.TcCountEgress.FD(),
			Name:         "vigil_tc_egress",
			DirectAction: true,
		}
		if err := netlink.FilterAdd(egressFilter); err != nil {
			cleanupTCFilters(filters)
			_ = objs.Close()
			return nil, fmt.Errorf("attach egress filter on %s: %w", iface.Name, err)
		}
		filters = append(filters, tcFilterInfo{
			ifaceIndex: iface.Index,
			parent:     netlink.MakeHandle(0xffff, 0xfff3),
			handle:     netlink.MakeHandle(0, 0x101),
		})
	}

	return &ebpfTrafficCollector{
		flows: objs.Flows,
		topN:  ebpfTrafficDefaultTopN,
		close: func() error {
			cleanupTCFilters(filters)
			return objs.Close()
		},
	}, nil
}

// eligibleInterfaces returns all non-loopback interfaces that are UP.
func eligibleInterfaces() ([]net.Interface, error) {
	all, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []net.Interface
	for _, iface := range all {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Name == "lo" {
			continue
		}
		out = append(out, iface)
	}
	return out, nil
}

// cleanupVigilFilters removes any TC filters with names starting with "vigil_"
// from the given interface's clsact ingress and egress hooks.
func cleanupVigilFilters(ifaceIndex int) error {
	link, err := netlink.LinkByIndex(ifaceIndex)
	if err != nil {
		return err
	}
	for _, parent := range []uint32{
		netlink.MakeHandle(0xffff, 0xfff2), // clsact ingress
		netlink.MakeHandle(0xffff, 0xfff3), // clsact egress
	} {
		filters, err := netlink.FilterList(link, parent)
		if err != nil {
			return err
		}
		for _, f := range filters {
			if bpfFilter, ok := f.(*netlink.BpfFilter); ok {
				if strings.HasPrefix(bpfFilter.Name, "vigil_") {
					if err := netlink.FilterDel(bpfFilter); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// cleanupTCFilters deletes the tracked TC filters. Errors are ignored.
func cleanupTCFilters(filters []tcFilterInfo) {
	for _, f := range filters {
		filter := &netlink.BpfFilter{
			FilterAttrs: netlink.FilterAttrs{
				LinkIndex: f.ifaceIndex,
				Parent:    f.parent,
				Handle:    f.handle,
				Protocol:  unix.ETH_P_ALL,
			},
		}
		_ = netlink.FilterDel(filter)
	}
}

// isEEXIST reports whether err is an EEXIST error from a netlink operation.
func isEEXIST(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, unix.EEXIST) || errors.Is(err, syscall.EEXIST) {
		return true
	}
	return strings.Contains(err.Error(), "file exists") ||
		strings.Contains(err.Error(), "object already exists")
}

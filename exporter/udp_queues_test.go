/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestUdpQueuesCollector_SumsAcrossSockets reproduces a bug where the
// collector emitted one metric per socket line, all with the same
// (ip, queue) labels, causing prometheus.Gather to reject the family
// with "was collected before with the same name and label values". It
// also exercises parseQueueLen, which previously always returned 0
// because of a misuse of fmt.Sscanf with `new(uint32)`.
func TestUdpQueuesCollector_SumsAcrossSockets(t *testing.T) {
	c := newUdpQueuesCollector("./testdata/proc/net/udp", "./testdata/proc/net/udp6")
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v (udp_queues must aggregate across sockets, not emit duplicates)", err)
	}

	type key struct{ ip, queue string }
	got := map[key]float64{}
	for _, fam := range families {
		if fam.GetName() != "node_udp_queue_length" {
			continue
		}
		for _, m := range fam.GetMetric() {
			var ip, queue string
			for _, lp := range m.GetLabel() {
				switch lp.GetName() {
				case "ip":
					ip = lp.GetValue()
				case "queue":
					queue = lp.GetValue()
				}
			}
			got[key{ip, queue}] = m.GetGauge().GetValue()
		}
	}

	// testdata/proc/net/udp has two sockets:
	//   sl=0  tx=0x00000000  rx=0x00000000
	//   sl=1  tx=0x00000010  rx=0x00000020
	// Expected sums: tx=16, rx=32 for ip=4. udp6 fixture is header-only, so
	// ip=6 should not appear (no socket lines => no metric emitted).
	if v, ok := got[key{"4", "tx"}]; !ok || v != 16 {
		t.Errorf("ip=4 queue=tx = %v (present=%v), want 16", v, ok)
	}
	if v, ok := got[key{"4", "rx"}]; !ok || v != 32 {
		t.Errorf("ip=4 queue=rx = %v (present=%v), want 32", v, ok)
	}
}

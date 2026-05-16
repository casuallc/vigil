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
	"github.com/prometheus/procfs"
)

func TestNetdevCollector_EmitsNetworkMetrics(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}
	c, err := newNetdevCollector(fs)
	if err != nil {
		t.Fatalf("newNetdevCollector: %v", err)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_network_receive_bytes_total" {
			found = true
		}
	}
	if !found {
		t.Error("node_network_receive_bytes_total not found")
	}
}

func TestSoftnetCollector_EmitsSoftnetMetrics(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}
	c, err := newSoftnetCollector(fs)
	if err != nil {
		t.Fatalf("newSoftnetCollector: %v", err)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_softnet_processed_total" {
			found = true
		}
	}
	if !found {
		t.Error("node_softnet_processed_total not found")
	}
}

func TestVmstatCollector_EmitsVmstatMetrics(t *testing.T) {
	c := newVmstatCollector("./testdata/proc/vmstat")
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_vmstat_pgfault" {
			found = true
			if fam.GetMetric()[0].GetCounter().GetValue() != 5000 {
				t.Errorf("pgfault = %v, want 5000", fam.GetMetric()[0].GetCounter().GetValue())
			}
		}
	}
	if !found {
		t.Error("node_vmstat_pgfault not found")
	}
}

func TestFilefdCollector_EmitsFilefdMetrics(t *testing.T) {
	c := newFilefdCollector("./testdata/proc/sys/fs/file-nr")
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_filefd_allocated" {
			found = true
			if fam.GetMetric()[0].GetGauge().GetValue() != 1500 {
				t.Errorf("allocated = %v, want 1500", fam.GetMetric()[0].GetGauge().GetValue())
			}
		}
	}
	if !found {
		t.Error("node_filefd_allocated not found")
	}
}

func TestEntropyCollector_EmitsEntropyMetric(t *testing.T) {
	c := newEntropyCollector("./testdata/proc/sys/kernel/random/entropy_avail")
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_entropy_available_bits" {
			found = true
			if fam.GetMetric()[0].GetGauge().GetValue() != 256 {
				t.Errorf("entropy = %v, want 256", fam.GetMetric()[0].GetGauge().GetValue())
			}
		}
	}
	if !found {
		t.Error("node_entropy_available_bits not found")
	}
}

func TestDiskstatsCollector_EmitsDiskMetrics(t *testing.T) {
	c := newDiskstatsCollector("./testdata/proc/diskstats")
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_disk_reads_completed_total" {
			found = true
		}
	}
	if !found {
		t.Error("node_disk_reads_completed_total not found")
	}
}

func TestStatCollector_EmitsStatMetrics(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}
	c, err := newStatCollector(fs)
	if err != nil {
		t.Fatalf("newStatCollector: %v", err)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	found := false
	for _, fam := range families {
		if fam.GetName() == "node_context_switches_total" {
			found = true
			if fam.GetMetric()[0].GetCounter().GetValue() != 987654321 {
				t.Errorf("context_switches = %v, want 987654321", fam.GetMetric()[0].GetCounter().GetValue())
			}
		}
	}
	if !found {
		t.Error("node_context_switches_total not found")
	}
}

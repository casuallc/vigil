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

func TestMeminfoCollector_EmitsMemTotalAndMemAvailable(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}

	c, err := newMeminfoCollector(fs)
	if err != nil {
		t.Fatalf("newMeminfoCollector: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	values := map[string]float64{}
	for _, fam := range families {
		if len(fam.GetMetric()) > 0 {
			values[fam.GetName()] = fam.GetMetric()[0].GetGauge().GetValue()
		}
	}

	if got, want := values["node_memory_MemTotal_bytes"], float64(16384*1024); got != want {
		t.Errorf("MemTotal = %v, want %v", got, want)
	}
	if got, want := values["node_memory_MemAvailable_bytes"], float64(12288*1024); got != want {
		t.Errorf("MemAvailable = %v, want %v", got, want)
	}
	if got, want := values["node_memory_MemFree_bytes"], float64(8192*1024); got != want {
		t.Errorf("MemFree = %v, want %v", got, want)
	}
	if got, want := values["node_memory_Buffers_bytes"], float64(256*1024); got != want {
		t.Errorf("Buffers = %v, want %v", got, want)
	}
	if got, want := values["node_memory_Cached_bytes"], float64(1024*1024); got != want {
		t.Errorf("Cached = %v, want %v", got, want)
	}
}

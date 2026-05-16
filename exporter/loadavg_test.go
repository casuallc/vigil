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

// TestLoadavgCollector_GathersLoad1Load5Load15 verifies that the collector
// reads /proc/loadavg and emits the three load metrics.
func TestLoadavgCollector_GathersLoad1Load5Load15(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}

	c, err := newLoadavgCollector(fs)
	if err != nil {
		t.Fatalf("newLoadavgCollector: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	values := map[string]float64{}
	for _, fam := range families {
		if len(fam.GetMetric()) == 0 {
			continue
		}
		name := fam.GetName()
		values[name] = fam.GetMetric()[0].GetGauge().GetValue()
	}

	if got, want := values["node_load1"], 0.42; got != want {
		t.Errorf("node_load1 = %v, want %v", got, want)
	}
	if got, want := values["node_load5"], 0.51; got != want {
		t.Errorf("node_load5 = %v, want %v", got, want)
	}
	if got, want := values["node_load15"], 0.48; got != want {
		t.Errorf("node_load15 = %v, want %v", got, want)
	}
}

// testCollectorAdapter adapts our Collector to prometheus.Collector for
// unit-test harnesses.
type testCollectorAdapter struct{ c Collector }

func (a *testCollectorAdapter) Describe(ch chan<- *prometheus.Desc) {
	// Unchecked collection — no pre-declaration required.
}

func (a *testCollectorAdapter) Collect(ch chan<- prometheus.Metric) {
	_ = a.c.Update(ch)
}

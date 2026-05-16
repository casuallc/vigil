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

func TestCpuCollector_EmitsPerCpuModeMetrics(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}

	c, err := newCpuCollector(fs)
	if err != nil {
		t.Fatalf("newCpuCollector: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	var foundMetric bool
	for _, fam := range families {
		if fam.GetName() != "node_cpu_seconds_total" {
			continue
		}
		foundMetric = true
		for _, m := range fam.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			cpu := labels["cpu"]
			mode := labels["mode"]
			val := m.GetCounter().GetValue()
			// procfs divides jiffies by USER_HZ (100), so fixture values
			// 1000/500/8000 become 10/5/80.
			if cpu == "0" && mode == "user" && val != 10 {
				t.Errorf("cpu0 user = %v, want 10", val)
			}
			if cpu == "0" && mode == "idle" && val != 80 {
				t.Errorf("cpu0 idle = %v, want 80", val)
			}
			if cpu == "0" && mode == "system" && val != 5 {
				t.Errorf("cpu0 system = %v, want 5", val)
			}
		}
	}
	if !foundMetric {
		t.Fatal("node_cpu_seconds_total metric family not found")
	}
}

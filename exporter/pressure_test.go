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

// TestPressureCollector_GathersWithoutDuplicateNameConflict reproduces a bug
// where the collector emitted three metrics with the same FQName but different
// help text, causing prometheus Gather to fail with "has help X but should
// have Y". After the fix, the three windows (avg10/avg60/avg300) share a
// single descriptor and are distinguished by a "window" label.
func TestPressureCollector_GathersWithoutDuplicateNameConflict(t *testing.T) {
	fs, err := procfs.NewFS("./testdata/proc")
	if err != nil {
		t.Fatalf("procfs.NewFS: %v", err)
	}
	c, err := newPressureCollector(fs)
	if err != nil {
		t.Fatalf("newPressureCollector: %v", err)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v (pressure collector must not emit duplicate name+labels)", err)
	}

	// Verify per-window labels appear on the avg gauge series for memory/some,
	// which has non-zero values in the testdata fixture.
	var found bool
	for _, fam := range families {
		if fam.GetName() != "node_pressure_memory_some" {
			continue
		}
		found = true
		windows := map[string]float64{}
		for _, m := range fam.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "window" {
					windows[lp.GetValue()] = m.GetGauge().GetValue()
				}
			}
		}
		// Expected values come from testdata/proc/pressure/memory:
		//   some avg10=1.23 avg60=0.50 avg300=0.10 total=987654321
		if got, want := windows["avg10"], 1.23; got != want {
			t.Errorf("window=avg10: %v, want %v", got, want)
		}
		if got, want := windows["avg60"], 0.50; got != want {
			t.Errorf("window=avg60: %v, want %v", got, want)
		}
		if got, want := windows["avg300"], 0.10; got != want {
			t.Errorf("window=avg300: %v, want %v", got, want)
		}
	}
	if !found {
		t.Error("node_pressure_memory_some not found in gathered families")
	}
}

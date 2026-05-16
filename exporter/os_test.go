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

func TestOsCollector_ReadsOSRelease(t *testing.T) {
	c, err := newOsCollector("./testdata/etc/os-release")
	if err != nil {
		t.Fatalf("newOsCollector: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	var found bool
	for _, fam := range families {
		if fam.GetName() != "node_os_info" {
			continue
		}
		found = true
		m := fam.GetMetric()[0]
		labels := map[string]string{}
		for _, lp := range m.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		if labels["id"] != "ubuntu" {
			t.Errorf("id = %q, want ubuntu", labels["id"])
		}
		if labels["version_id"] != "22.04" {
			t.Errorf("version_id = %q, want 22.04", labels["version_id"])
		}
	}
	if !found {
		t.Error("node_os_info not found")
	}
}

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

func TestUnameCollector_ReadsProcSysKernel(t *testing.T) {
	c, err := newUnameCollector("./testdata/proc")
	if err != nil {
		t.Fatalf("newUnameCollector: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&testCollectorAdapter{c: c})

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	var found bool
	for _, fam := range families {
		if fam.GetName() != "node_uname_info" {
			continue
		}
		found = true
		m := fam.GetMetric()[0]
		labels := map[string]string{}
		for _, lp := range m.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		if labels["sysname"] != "Linux" {
			t.Errorf("sysname = %q, want Linux", labels["sysname"])
		}
		if labels["release"] != "5.15.0-105-generic" {
			t.Errorf("release = %q, want 5.15.0-105-generic", labels["release"])
		}
	}
	if !found {
		t.Error("node_uname_info not found")
	}
}

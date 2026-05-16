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

// TestGatherJSON_GroupsBySourceCollector verifies metrics are bucketed by the
// collector that emitted them, identified by the "vigil_collector" internal
// label that NodeExporter strips before exposition.
func TestGatherJSON_GroupsBySourceCollector(t *testing.T) {
	cpuDesc := prometheus.NewDesc(
		"node_cpu_seconds_total",
		"cpu test",
		[]string{"cpu", "mode"}, nil,
	)
	memDesc := prometheus.NewDesc(
		"node_memory_MemTotal_bytes",
		"mem test",
		nil, nil,
	)

	collectors := map[string]Collector{
		"cpu": &fakeCollector{name: "cpu", metrics: []prometheus.Metric{
			prometheus.MustNewConstMetric(cpuDesc, prometheus.CounterValue, 1234.5, "0", "user"),
			prometheus.MustNewConstMetric(cpuDesc, prometheus.CounterValue, 5678.9, "0", "idle"),
		}},
		"meminfo": &fakeCollector{name: "meminfo", metrics: []prometheus.Metric{
			prometheus.MustNewConstMetric(memDesc, prometheus.GaugeValue, 16777216000),
		}},
	}

	n, err := newNodeExporterWithCollectors(collectors)
	if err != nil {
		t.Fatalf("newNodeExporterWithCollectors: %v", err)
	}

	out, err := n.GatherJSON()
	if err != nil {
		t.Fatalf("GatherJSON: %v", err)
	}

	cpu, ok := out["cpu"]
	if !ok {
		t.Fatalf("cpu group missing in JSON output: %+v", out)
	}
	samples := cpu["node_cpu_seconds_total"]
	if len(samples) != 2 {
		t.Errorf("node_cpu_seconds_total samples = %d, want 2", len(samples))
	}
	var foundUser, foundIdle bool
	for _, s := range samples {
		mode := s.Labels["mode"]
		switch mode {
		case "user":
			foundUser = true
			if s.Value != 1234.5 {
				t.Errorf("user value = %v, want 1234.5", s.Value)
			}
		case "idle":
			foundIdle = true
			if s.Value != 5678.9 {
				t.Errorf("idle value = %v, want 5678.9", s.Value)
			}
		}
	}
	if !foundUser || !foundIdle {
		t.Errorf("missing label values: foundUser=%v foundIdle=%v", foundUser, foundIdle)
	}

	mem, ok := out["meminfo"]
	if !ok {
		t.Fatalf("meminfo group missing")
	}
	memSamples := mem["node_memory_MemTotal_bytes"]
	if len(memSamples) != 1 || memSamples[0].Value != 16777216000 {
		t.Errorf("meminfo samples = %+v, want one with value 16777216000", memSamples)
	}
}

// TestGatherJSON_IncludesScrapeMetaUnderScrapeGroup confirms node_scrape_*
// metrics are exposed too, but bucketed under the "scrape" pseudo-group
// rather than under any single collector.
func TestGatherJSON_IncludesScrapeMetaUnderScrapeGroup(t *testing.T) {
	desc := prometheus.NewDesc("vigil_x", "x", nil, nil)
	m := prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1)
	collectors := map[string]Collector{
		"x": &fakeCollector{name: "x", metrics: []prometheus.Metric{m}},
	}

	n, err := newNodeExporterWithCollectors(collectors)
	if err != nil {
		t.Fatalf("newNodeExporterWithCollectors: %v", err)
	}

	out, err := n.GatherJSON()
	if err != nil {
		t.Fatalf("GatherJSON: %v", err)
	}

	scrape, ok := out["scrape"]
	if !ok {
		t.Fatalf("scrape group missing: %+v", out)
	}
	if _, ok := scrape["node_scrape_collector_success"]; !ok {
		t.Error("scrape.node_scrape_collector_success missing")
	}
	if _, ok := scrape["node_scrape_collector_duration_seconds"]; !ok {
		t.Error("scrape.node_scrape_collector_duration_seconds missing")
	}
}

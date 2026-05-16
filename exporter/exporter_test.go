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

// fakeCollector emits a fixed set of metrics for testing.
type fakeCollector struct {
	name    string
	metrics []prometheus.Metric
	err     error
}

func (f *fakeCollector) Name() string { return f.name }

func (f *fakeCollector) Update(ch chan<- prometheus.Metric) error {
	for _, m := range f.metrics {
		ch <- m
	}
	return f.err
}

// TestNodeExporter_GathersRegisteredCollectorMetrics verifies that a metric
// emitted by a registered collector appears in the Registry's Gather output.
func TestNodeExporter_GathersRegisteredCollectorMetrics(t *testing.T) {
	desc := prometheus.NewDesc("vigil_test_value", "test help", nil, nil)
	metric := prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 42)
	fake := &fakeCollector{name: "fake", metrics: []prometheus.Metric{metric}}

	n, err := newNodeExporterWithCollectors(map[string]Collector{"fake": fake})
	if err != nil {
		t.Fatalf("newNodeExporterWithCollectors: %v", err)
	}

	families, err := n.Registry().Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	var got float64
	var found bool
	for _, fam := range families {
		if fam.GetName() == "vigil_test_value" {
			found = true
			got = fam.GetMetric()[0].GetGauge().GetValue()
		}
	}
	if !found {
		t.Fatalf("vigil_test_value not in gathered families")
	}
	if got != 42 {
		t.Errorf("vigil_test_value = %v, want 42", got)
	}
}

// TestNodeExporter_EmitsScrapeSuccessMetricPerCollector verifies that the
// exporter emits node_scrape_collector_success{collector="X"} = 1 for each
// successful collector and 0 for failing ones.
func TestNodeExporter_EmitsScrapeSuccessMetricPerCollector(t *testing.T) {
	okMetric := prometheus.MustNewConstMetric(
		prometheus.NewDesc("vigil_ok", "ok", nil, nil),
		prometheus.GaugeValue, 1,
	)
	collectors := map[string]Collector{
		"good": &fakeCollector{name: "good", metrics: []prometheus.Metric{okMetric}},
		"bad":  &fakeCollector{name: "bad", err: errCollectorFailed("simulated")},
	}
	n, err := newNodeExporterWithCollectors(collectors)
	if err != nil {
		t.Fatalf("newNodeExporterWithCollectors: %v", err)
	}

	families, err := n.Registry().Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	got := map[string]float64{}
	for _, fam := range families {
		if fam.GetName() != "node_scrape_collector_success" {
			continue
		}
		for _, m := range fam.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "collector" {
					got[lp.GetValue()] = m.GetGauge().GetValue()
				}
			}
		}
	}

	if got["good"] != 1 {
		t.Errorf("node_scrape_collector_success{collector=good} = %v, want 1", got["good"])
	}
	if got["bad"] != 0 {
		t.Errorf("node_scrape_collector_success{collector=bad} = %v, want 0", got["bad"])
	}
}

// TestNodeExporter_FailingCollectorDoesNotPanic checks that a collector
// returning an error does not propagate the failure to other collectors.
func TestNodeExporter_FailingCollectorDoesNotPanic(t *testing.T) {
	okMetric := prometheus.MustNewConstMetric(
		prometheus.NewDesc("vigil_ok2", "ok", nil, nil),
		prometheus.GaugeValue, 7,
	)
	collectors := map[string]Collector{
		"good": &fakeCollector{name: "good", metrics: []prometheus.Metric{okMetric}},
		"bad":  &fakeCollector{name: "bad", err: errCollectorFailed("boom")},
	}
	n, err := newNodeExporterWithCollectors(collectors)
	if err != nil {
		t.Fatalf("newNodeExporterWithCollectors: %v", err)
	}

	families, err := n.Registry().Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	var foundGood bool
	for _, fam := range families {
		if fam.GetName() == "vigil_ok2" {
			foundGood = true
		}
	}
	if !foundGood {
		t.Error("good collector metric missing; bad collector should not poison others")
	}
}

// errCollectorFailed returns a sentinel error matching the message style we
// expect; used to make the test's intent obvious.
func errCollectorFailed(msg string) error {
	return &collectorError{msg: msg}
}

type collectorError struct{ msg string }

func (e *collectorError) Error() string { return "collector failed: " + e.msg }

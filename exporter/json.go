/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MetricSample is one observation of a metric, optionally with labels.
type MetricSample struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

// CollectorMetrics is the per-metric-name samples produced by a collector.
type CollectorMetrics map[string][]MetricSample

// GatherJSON runs every registered collector once and returns the results
// bucketed by the collector that produced them. The pseudo-group "scrape"
// holds the meta-metrics (success and duration per collector).
//
// Each collector runs in its own goroutine and its failures are isolated
// from the others, mirroring the behavior of Collect on Registry().
func (n *NodeExporter) GatherJSON() (map[string]CollectorMetrics, error) {
	out := map[string]CollectorMetrics{}
	scrape := CollectorMetrics{
		"node_scrape_collector_success":          nil,
		"node_scrape_collector_duration_seconds": nil,
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, c := range n.collectors {
		wg.Add(1)
		go func(name string, c Collector) {
			defer wg.Done()

			begin := time.Now()
			success := 1.0
			families := runCollectorInTempRegistry(c, &success)
			duration := time.Since(begin).Seconds()

			grouped := familiesToCollectorMetrics(families)

			mu.Lock()
			defer mu.Unlock()

			if _, ok := out[name]; !ok {
				out[name] = CollectorMetrics{}
			}
			for metricName, samples := range grouped {
				out[name][metricName] = append(out[name][metricName], samples...)
			}
			scrape["node_scrape_collector_success"] = append(
				scrape["node_scrape_collector_success"],
				MetricSample{Labels: map[string]string{"collector": name}, Value: success},
			)
			scrape["node_scrape_collector_duration_seconds"] = append(
				scrape["node_scrape_collector_duration_seconds"],
				MetricSample{Labels: map[string]string{"collector": name}, Value: duration},
			)
		}(name, c)
	}
	wg.Wait()

	out["scrape"] = scrape
	return out, nil
}

// runCollectorInTempRegistry executes the collector against a fresh registry
// and returns the gathered metric families. Errors and panics set *success
// to 0 and yield whatever metrics did make it onto the channel before the
// failure.
func runCollectorInTempRegistry(c Collector, success *float64) []*dto.MetricFamily {
	tmp := prometheus.NewRegistry()
	wrapper := &singleCollectorAdapter{c: c, success: success}
	if err := tmp.Register(wrapper); err != nil {
		*success = 0
		return nil
	}
	families, err := tmp.Gather()
	if err != nil {
		*success = 0
		return families
	}
	return families
}

// singleCollectorAdapter adapts our Collector interface to prometheus.Collector
// for the purposes of GatherJSON. It does not emit scrape meta metrics —
// that bookkeeping is done by GatherJSON itself.
type singleCollectorAdapter struct {
	c       Collector
	success *float64
}

func (a *singleCollectorAdapter) Describe(ch chan<- *prometheus.Desc) {
	// Unchecked collection: leaving Describe empty allows arbitrary metrics
	// from Update without a-priori description, which matches how the
	// upstream node_exporter collectors work.
}

func (a *singleCollectorAdapter) Collect(ch chan<- prometheus.Metric) {
	defer func() {
		if r := recover(); r != nil {
			*a.success = 0
		}
	}()
	if err := a.c.Update(ch); err != nil {
		*a.success = 0
	}
}

// familiesToCollectorMetrics converts gathered MetricFamily protos to the
// JSON-friendly per-metric-name samples used by the API.
func familiesToCollectorMetrics(families []*dto.MetricFamily) CollectorMetrics {
	out := CollectorMetrics{}
	for _, fam := range families {
		name := fam.GetName()
		for _, m := range fam.GetMetric() {
			sample := MetricSample{Value: metricValue(m)}
			if labels := m.GetLabel(); len(labels) > 0 {
				sample.Labels = make(map[string]string, len(labels))
				for _, lp := range labels {
					sample.Labels[lp.GetName()] = lp.GetValue()
				}
			}
			out[name] = append(out[name], sample)
		}
	}
	return out
}

// metricValue extracts the numeric value from a metric proto, defaulting to
// the Gauge/Counter/Untyped value. Histogram and Summary types are flattened
// to their sample sum (a common simplification — full distribution support
// can be added later).
func metricValue(m *dto.Metric) float64 {
	switch {
	case m.Gauge != nil:
		return m.Gauge.GetValue()
	case m.Counter != nil:
		return m.Counter.GetValue()
	case m.Untyped != nil:
		return m.Untyped.GetValue()
	case m.Summary != nil:
		return m.Summary.GetSampleSum()
	case m.Histogram != nil:
		return m.Histogram.GetSampleSum()
	}
	return 0
}

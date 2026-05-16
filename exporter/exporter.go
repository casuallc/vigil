/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "node"

// NodeExporter holds the Prometheus registry and the collectors that feed it.
type NodeExporter struct {
	registry   *prometheus.Registry
	collectors map[string]Collector

	scrapeDuration *prometheus.Desc
	scrapeSuccess  *prometheus.Desc
}

// Registry returns the underlying Prometheus registry, suitable for handing
// to promhttp.HandlerFor.
func (n *NodeExporter) Registry() *prometheus.Registry {
	return n.registry
}

// newNodeExporterWithCollectors builds a NodeExporter from an explicit
// collector map. Used by tests; production code goes through NewNodeExporter
// in the platform-specific file.
func newNodeExporterWithCollectors(collectors map[string]Collector) (*NodeExporter, error) {
	n := &NodeExporter{
		registry:   prometheus.NewRegistry(),
		collectors: collectors,
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
			"node_exporter: Duration of a collector scrape.",
			[]string{"collector"}, nil,
		),
		scrapeSuccess: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "collector_success"),
			"node_exporter: Whether a collector succeeded.",
			[]string{"collector"}, nil,
		),
	}
	if err := n.registry.Register(n); err != nil {
		return nil, err
	}
	return n, nil
}

// Describe implements prometheus.Collector.
func (n *NodeExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- n.scrapeDuration
	ch <- n.scrapeSuccess
}

// Collect implements prometheus.Collector. Each registered Collector runs
// concurrently; failures are isolated and reported as meta-metrics so a
// single broken collector cannot fail the entire scrape.
func (n *NodeExporter) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	for name, c := range n.collectors {
		wg.Add(1)
		go func(name string, c Collector) {
			defer wg.Done()
			n.execCollector(ch, name, c)
		}(name, c)
	}
	wg.Wait()
}

// execCollector runs a single collector and records duration + success.
// A panic inside the collector is recovered and recorded as failure.
func (n *NodeExporter) execCollector(ch chan<- prometheus.Metric, name string, c Collector) {
	begin := time.Now()
	success := 1.0

	defer func() {
		if r := recover(); r != nil {
			log.Printf("exporter: collector %s panicked: %v", name, r)
			success = 0
		}
		duration := time.Since(begin).Seconds()
		ch <- prometheus.MustNewConstMetric(n.scrapeDuration, prometheus.GaugeValue, duration, name)
		ch <- prometheus.MustNewConstMetric(n.scrapeSuccess, prometheus.GaugeValue, success, name)
	}()

	if err := c.Update(ch); err != nil {
		log.Printf("exporter: collector %s failed: %v", name, err)
		success = 0
	}
}

/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

// Package exporter provides node_exporter-style Linux host metrics. It
// re-implements roughly 30 of the upstream collectors using prometheus/procfs
// for /proc parsing and prometheus/client_golang for metric registration and
// exposition.
package exporter

import "github.com/prometheus/client_golang/prometheus"

// Collector is the interface every system metric collector implements. It is
// adapted to prometheus.Collector by the NodeExporter so failures and
// durations can be tracked uniformly.
type Collector interface {
	// Name returns the collector identifier (e.g. "cpu", "meminfo").
	Name() string

	// Update produces metrics into ch. It must return any error encountered
	// while reading the underlying source; the NodeExporter records success
	// or failure as a meta-metric and never propagates the error to the
	// scrape itself.
	Update(ch chan<- prometheus.Metric) error
}

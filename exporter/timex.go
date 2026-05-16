/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("timex", func() (Collector, error) {
		return newTimexCollector(), nil
	})
}

type timexCollector struct{}

func newTimexCollector() Collector {
	return &timexCollector{}
}

func (c *timexCollector) Name() string { return "timex" }

func (c *timexCollector) Update(ch chan<- prometheus.Metric) error {
	// timex requires adjtimex syscall (Linux-only).
	// Full implementation deferred; stub to satisfy registration.
	return nil
}

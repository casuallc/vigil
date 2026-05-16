/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("time", func() (Collector, error) {
		return newTimeCollector()
	})
}

type timeCollector struct {
	desc *prometheus.Desc
}

func newTimeCollector() (Collector, error) {
	return &timeCollector{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "time_seconds"),
			"System time in seconds since epoch (1970).",
			nil, nil,
		),
	}, nil
}

func (c *timeCollector) Name() string { return "time" }

func (c *timeCollector) Update(ch chan<- prometheus.Metric) error {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(time.Now().Unix()))
	return nil
}

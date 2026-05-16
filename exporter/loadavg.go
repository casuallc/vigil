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
	"github.com/prometheus/procfs"
)

type loadavgCollector struct {
	fs procfs.FS

	load1  *prometheus.Desc
	load5  *prometheus.Desc
	load15 *prometheus.Desc
}

func newLoadavgCollector(fs procfs.FS) (Collector, error) {
	return &loadavgCollector{
		fs: fs,
		load1: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "load1"),
			"1m load average.", nil, nil,
		),
		load5: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "load5"),
			"5m load average.", nil, nil,
		),
		load15: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "load15"),
			"15m load average.", nil, nil,
		),
	}, nil
}

func (c *loadavgCollector) Name() string { return "loadavg" }

func (c *loadavgCollector) Update(ch chan<- prometheus.Metric) error {
	la, err := c.fs.LoadAvg()
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(c.load1, prometheus.GaugeValue, la.Load1)
	ch <- prometheus.MustNewConstMetric(c.load5, prometheus.GaugeValue, la.Load5)
	ch <- prometheus.MustNewConstMetric(c.load15, prometheus.GaugeValue, la.Load15)
	return nil
}

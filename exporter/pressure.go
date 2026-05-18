/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	registerLinuxCollector("pressure", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newPressureCollector(fs)
	})
}

type pressureCollector struct {
	fs procfs.FS
}

func newPressureCollector(fs procfs.FS) (Collector, error) {
	return &pressureCollector{fs: fs}, nil
}

func (c *pressureCollector) Name() string { return "pressure" }

func (c *pressureCollector) Update(ch chan<- prometheus.Metric) error {
	for _, res := range []string{"cpu", "memory", "io"} {
		if err := c.updateResource(ch, res); err != nil {
			// PSI may not be available on older kernels; skip silently.
			continue
		}
	}
	return nil
}

func (c *pressureCollector) updateResource(ch chan<- prometheus.Metric, res string) error {
	stats, err := c.fs.PSIStatsForResource(res)
	if err != nil {
		return err
	}
	if stats.Some != nil {
		c.emit(ch, res, "some", stats.Some)
	}
	if stats.Full != nil {
		c.emit(ch, res, "full", stats.Full)
	}
	return nil
}

func (c *pressureCollector) emit(ch chan<- prometheus.Metric, res, typ string, line *procfs.PSILine) {
	avgDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "pressure", fmt.Sprintf("%s_%s", res, typ)),
		fmt.Sprintf("%s %s pressure ratio averaged over the indicated window.", res, typ),
		[]string{"window"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(avgDesc, prometheus.GaugeValue, line.Avg10, "avg10")
	ch <- prometheus.MustNewConstMetric(avgDesc, prometheus.GaugeValue, line.Avg60, "avg60")
	ch <- prometheus.MustNewConstMetric(avgDesc, prometheus.GaugeValue, line.Avg300, "avg300")

	totalDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "pressure", fmt.Sprintf("%s_%s_total", res, typ)),
		fmt.Sprintf("%s %s pressure total stall time in seconds.", res, typ),
		[]string{}, nil,
	)
	ch <- prometheus.MustNewConstMetric(totalDesc, prometheus.CounterValue, float64(line.Total)/1000000.0)
}

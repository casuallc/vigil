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

func init() {
	registerLinuxCollector("stat", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newStatCollector(fs)
	})
}

type statCollector struct {
	fs procfs.FS
}

func newStatCollector(fs procfs.FS) (Collector, error) {
	return &statCollector{fs: fs}, nil
}

func (c *statCollector) Name() string { return "stat" }

func (c *statCollector) Update(ch chan<- prometheus.Metric) error {
	stat, err := c.fs.Stat()
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "context_switches_total"), "Total number of context switches.", nil, nil),
		prometheus.CounterValue, float64(stat.ContextSwitches),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "forks_total"), "Total number of forks.", nil, nil),
		prometheus.CounterValue, float64(stat.ProcessCreated),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "interrupts_total"), "Total number of interrupts serviced.", nil, nil),
		prometheus.CounterValue, float64(stat.IRQTotal),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "procs_running"), "Number of processes in runnable state.", nil, nil),
		prometheus.GaugeValue, float64(stat.ProcessesRunning),
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "procs_blocked"), "Number of processes blocked waiting for I/O.", nil, nil),
		prometheus.GaugeValue, float64(stat.ProcessesBlocked),
	)
	return nil
}

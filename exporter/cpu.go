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
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var cpuModes = []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guest_nice"}

func init() {
	registerLinuxCollector("cpu", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newCpuCollector(fs)
	})
}

// cpuCollector produces node_cpu_seconds_total matching node_exporter.
type cpuCollector struct {
	fs   procfs.FS
	desc *prometheus.Desc
}

func newCpuCollector(fs procfs.FS) (Collector, error) {
	return &cpuCollector{
		fs: fs,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cpu_seconds_total"),
			"Seconds the CPUs spent in each mode.",
			[]string{"cpu", "mode"}, nil,
		),
	}, nil
}

func (c *cpuCollector) Name() string { return "cpu" }

func (c *cpuCollector) Update(ch chan<- prometheus.Metric) error {
	stat, err := c.fs.Stat()
	if err != nil {
		return err
	}

	// Emit per-CPU metrics (cpu0, cpu1, ...) and a "cpu" total aggregate.
	for cpu, s := range stat.CPU {
		if err := c.emit(ch, strconv.FormatInt(cpu, 10), s); err != nil {
			return err
		}
	}
	if err := c.emit(ch, "cpu", stat.CPUTotal); err != nil {
		return err
	}
	return nil
}

func (c *cpuCollector) emit(ch chan<- prometheus.Metric, cpu string, s procfs.CPUStat) error {
	// procfs.CPUStat already returns values in seconds (divided by USER_HZ).
	values := []float64{s.User, s.Nice, s.System, s.Idle, s.Iowait, s.IRQ, s.SoftIRQ, s.Steal, s.Guest, s.GuestNice}
	for i, val := range values {
		mode := cpuModes[i]
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, val, cpu, mode)
	}
	return nil
}

func (c *cpuCollector) errCPU(str string, e error) error {
	return fmt.Errorf("cpu collector error %s: %w", str, e)
}

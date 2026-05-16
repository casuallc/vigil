/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	registerLinuxCollector("softnet", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newSoftnetCollector(fs)
	})
}

type softnetCollector struct {
	fs procfs.FS
}

func newSoftnetCollector(fs procfs.FS) (Collector, error) {
	return &softnetCollector{fs: fs}, nil
}

func (c *softnetCollector) Name() string { return "softnet" }

func (c *softnetCollector) Update(ch chan<- prometheus.Metric) error {
	stats, err := c.fs.NetSoftnetStat()
	if err != nil {
		return err
	}
	for cpu, s := range stats {
		c.emit(ch, "processed", float64(s.Processed), cpu)
		c.emit(ch, "dropped", float64(s.Dropped), cpu)
		c.emit(ch, "times_squeezed", float64(s.TimeSqueezed), cpu)
	}
	return nil
}

func (c *softnetCollector) emit(ch chan<- prometheus.Metric, name string, value float64, cpu int) {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "softnet", name+"_total"),
		"Number of "+name+" softnet events.",
		[]string{"cpu"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, strconv.Itoa(cpu))
}

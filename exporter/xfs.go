/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("xfs", func() (Collector, error) {
		return newXfsCollector("/sys/fs/xfs"), nil
	})
}

type xfsCollector struct {
	sysPath string
}

func newXfsCollector(sysPath string) Collector {
	return &xfsCollector{sysPath: sysPath}
}

func (c *xfsCollector) Name() string { return "xfs" }

func (c *xfsCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dev := e.Name()
		devPath := filepath.Join(c.sysPath, dev, "stats")
		stats, err := os.ReadDir(devPath)
		if err != nil {
			continue
		}
		for _, s := range stats {
			if s.IsDir() {
				continue
			}
			name := s.Name()
			val := readSysfsInt(filepath.Join(devPath, name))
			if val == 0 {
				continue
			}
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "xfs", name+"_total"),
				"XFS statistic "+name+".",
				[]string{"device"}, nil,
			)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(val), dev)
		}
	}
	return nil
}

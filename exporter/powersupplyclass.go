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
	registerLinuxCollector("powersupplyclass", func() (Collector, error) {
		return newPowersupplyclassCollector("/sys/class/power_supply"), nil
	})
}

type powersupplyclassCollector struct {
	sysPath string
}

func newPowersupplyclassCollector(sysPath string) Collector {
	return &powersupplyclassCollector{sysPath: sysPath}
}

func (c *powersupplyclassCollector) Name() string { return "powersupplyclass" }

func (c *powersupplyclassCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		supply := e.Name()
		supplyPath := filepath.Join(c.sysPath, supply)
		stype := readSysfsString(filepath.Join(supplyPath, "type"))
		status := readSysfsString(filepath.Join(supplyPath, "status"))
		present := readSysfsInt(filepath.Join(supplyPath, "present"))

		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "powersupply", "present"),
			"Whether a power supply is present (1) or not (0).",
			[]string{"powersupply"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(present), supply)

		if stype == "Battery" && status != "" {
			statusDesc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "powersupply", "status"),
				"Current status of the power supply.",
				[]string{"powersupply", "status"}, nil,
			)
			ch <- prometheus.MustNewConstMetric(statusDesc, prometheus.GaugeValue, 1, supply, status)
		}
	}
	return nil
}

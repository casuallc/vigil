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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("thermal_zone", func() (Collector, error) {
		return newThermalZoneCollector("/sys/class/thermal"), nil
	})
}

type thermalZoneCollector struct {
	sysPath string
}

func newThermalZoneCollector(sysPath string) Collector {
	return &thermalZoneCollector{sysPath: sysPath}
}

func (c *thermalZoneCollector) Name() string { return "thermal_zone" }

func (c *thermalZoneCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "thermal_zone") {
			continue
		}
		zone := e.Name()
		zonePath := filepath.Join(c.sysPath, zone)
		ztype := readSysfsString(filepath.Join(zonePath, "type"))
		temp := readSysfsInt(filepath.Join(zonePath, "temp"))
		if temp == 0 {
			continue
		}
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "thermal_zone", "temp"),
			"Zone temperature in Celsius.",
			[]string{"zone", "type"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(temp)/1000, zone, ztype)
	}
	return nil
}

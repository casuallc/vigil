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
	registerLinuxCollector("hwmon", func() (Collector, error) {
		return newHwmonCollector("/sys/class/hwmon"), nil
	})
}

type hwmonCollector struct {
	sysPath string
}

func newHwmonCollector(sysPath string) Collector {
	return &hwmonCollector{sysPath: sysPath}
}

func (c *hwmonCollector) Name() string { return "hwmon" }

func (c *hwmonCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return nil // hwmon may not be present
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		chip := e.Name()
		chipPath := filepath.Join(c.sysPath, chip)
		name := readSysfsString(filepath.Join(chipPath, "name"))
		if name == "" {
			continue
		}
		c.readSensors(ch, chipPath, name)
	}
	return nil
}

func (c *hwmonCollector) readSensors(ch chan<- prometheus.Metric, chipPath, chipName string) {
	entries, _ := os.ReadDir(chipPath)
	for _, e := range entries {
		fname := e.Name()
		if !strings.HasPrefix(fname, "temp") || !strings.HasSuffix(fname, "_input") {
			continue
		}
		sensor := strings.TrimSuffix(fname, "_input")
		label := readSysfsString(filepath.Join(chipPath, sensor+"_label"))
		if label == "" {
			label = sensor
		}
		val := readSysfsInt(filepath.Join(chipPath, fname))
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "hwmon", "temp_celsius"),
			"Hardware monitor for temperature (in Celsius).",
			[]string{"chip", "sensor"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(val)/1000, chipName, label)
	}
}

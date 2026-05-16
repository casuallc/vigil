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
	registerLinuxCollector("rapl", func() (Collector, error) {
		return newRaplCollector("/sys/class/powercap/intel-rapl"), nil
	})
}

type raplCollector struct {
	sysPath string
}

func newRaplCollector(sysPath string) Collector {
	return &raplCollector{sysPath: sysPath}
}

func (c *raplCollector) Name() string { return "rapl" }

func (c *raplCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "intel-rapl") {
			continue
		}
		pkg := e.Name()
		pkgPath := filepath.Join(c.sysPath, pkg)
		name := readSysfsString(filepath.Join(pkgPath, "name"))
		if name == "" {
			name = pkg
		}
		energy := readSysfsInt(filepath.Join(pkgPath, "energy_uj"))
		if energy == 0 {
			continue
		}
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "rapl", "joules_total"),
			"Current RAPL value in joules.",
			[]string{"index", "package", "domain"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(energy)/1e6, "0", name, name)
	}
	return nil
}

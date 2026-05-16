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
	registerLinuxCollector("cpufreq", func() (Collector, error) {
		return newCpufreqCollector("/sys/devices/system/cpu"), nil
	})
}

type cpufreqCollector struct {
	sysPath string
}

func newCpufreqCollector(sysPath string) Collector {
	return &cpufreqCollector{sysPath: sysPath}
}

func (c *cpufreqCollector) Name() string { return "cpufreq" }

func (c *cpufreqCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "cpu") {
			continue
		}
		cpu := e.Name()
		freqPath := filepath.Join(c.sysPath, cpu, "cpufreq", "scaling_cur_freq")
		freq := readSysfsInt(freqPath)
		if freq == 0 {
			continue
		}
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cpufreq", "scaling_frequency_hertz"),
			"Current scaled CPU thread frequency in hertz.",
			[]string{"cpu"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(freq)*1000, cpu)
	}
	return nil
}

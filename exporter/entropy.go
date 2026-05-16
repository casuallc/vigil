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
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("entropy", func() (Collector, error) {
		return newEntropyCollector("/proc/sys/kernel/random/entropy_avail"), nil
	})
}

type entropyCollector struct {
	procPath string
}

func newEntropyCollector(procPath string) Collector {
	return &entropyCollector{procPath: procPath}
}

func (c *entropyCollector) Name() string { return "entropy" }

func (c *entropyCollector) Update(ch chan<- prometheus.Metric) error {
	data, err := os.ReadFile(c.procPath)
	if err != nil {
		return err
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "entropy_available_bits"), "Bits of available entropy.", nil, nil),
		prometheus.GaugeValue, val,
	)
	return nil
}

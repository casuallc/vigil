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
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("filefd", func() (Collector, error) {
		return newFilefdCollector("/proc/sys/fs/file-nr"), nil
	})
}

type filefdCollector struct {
	procPath string
}

func newFilefdCollector(procPath string) Collector {
	return &filefdCollector{procPath: procPath}
}

func (c *filefdCollector) Name() string { return "filefd" }

func (c *filefdCollector) Update(ch chan<- prometheus.Metric) error {
	data, err := os.ReadFile(c.procPath)
	if err != nil {
		return err
	}
	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return fmt.Errorf("unexpected file-nr format")
	}
	allocated, _ := strconv.ParseFloat(parts[0], 64)
	maximum, _ := strconv.ParseFloat(parts[2], 64)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "filefd", "allocated"), "Number of allocated file descriptors.", nil, nil),
		prometheus.GaugeValue, allocated,
	)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "filefd", "maximum"), "Maximum number of file descriptors.", nil, nil),
		prometheus.GaugeValue, maximum,
	)
	return nil
}

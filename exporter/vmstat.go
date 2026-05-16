/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("vmstat", func() (Collector, error) {
		return newVmstatCollector("/proc/vmstat"), nil
	})
}

type vmstatCollector struct {
	procPath string
}

func newVmstatCollector(procPath string) Collector {
	return &vmstatCollector{procPath: procPath}
}

func (c *vmstatCollector) Name() string { return "vmstat" }

func (c *vmstatCollector) Update(ch chan<- prometheus.Metric) error {
	f, err := os.Open(c.procPath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "vmstat", name),
			fmt.Sprintf("/proc/vmstat information field %s.", name),
			nil, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, val)
	}
	return scanner.Err()
}

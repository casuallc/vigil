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
	registerLinuxCollector("diskstats", func() (Collector, error) {
		return newDiskstatsCollector("/proc/diskstats"), nil
	})
}

type diskstatsCollector struct {
	procPath string
}

func newDiskstatsCollector(procPath string) Collector {
	return &diskstatsCollector{procPath: procPath}
}

func (c *diskstatsCollector) Name() string { return "diskstats" }

func (c *diskstatsCollector) Update(ch chan<- prometheus.Metric) error {
	f, err := os.Open(c.procPath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}
		device := fields[2]
		c.emit(ch, device, "reads_completed", 3, fields)
		c.emit(ch, device, "reads_merged", 4, fields)
		c.emit(ch, device, "sectors_read", 5, fields)
		c.emit(ch, device, "read_time_seconds", 6, fields)
		c.emit(ch, device, "writes_completed", 7, fields)
		c.emit(ch, device, "writes_merged", 8, fields)
		c.emit(ch, device, "sectors_written", 9, fields)
		c.emit(ch, device, "write_time_seconds", 10, fields)
		c.emit(ch, device, "io_now", 11, fields)
		c.emit(ch, device, "io_time_seconds", 12, fields)
		c.emit(ch, device, "io_time_weighted_seconds", 13, fields)
	}
	return scanner.Err()
}

func (c *diskstatsCollector) emit(ch chan<- prometheus.Metric, device, name string, idx int, fields []string) {
	val, _ := strconv.ParseFloat(fields[idx], 64)
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "disk", name+"_total"),
		fmt.Sprintf("The number of %s.", name),
		[]string{"device"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, val, device)
}

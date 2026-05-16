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
	registerLinuxCollector("conntrack", func() (Collector, error) {
		return newConntrackCollector("/proc/sys/net/netfilter/nf_conntrack_count"), nil
	})
}

type conntrackCollector struct {
	procPath string
}

func newConntrackCollector(procPath string) Collector {
	return &conntrackCollector{procPath: procPath}
}

func (c *conntrackCollector) Name() string { return "conntrack" }

func (c *conntrackCollector) Update(ch chan<- prometheus.Metric) error {
	data, err := os.ReadFile(c.procPath)
	if err != nil {
		return err
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "nf_conntrack_entries"), "Number of currently allocated flow entries for connection tracking.", nil, nil),
		prometheus.GaugeValue, val,
	)
	return nil
}

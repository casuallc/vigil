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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("selinux", func() (Collector, error) {
		return newSelinuxCollector("/sys/fs/selinux/enforce"), nil
	})
}

type selinuxCollector struct {
	procPath string
}

func newSelinuxCollector(procPath string) Collector {
	return &selinuxCollector{procPath: procPath}
}

func (c *selinuxCollector) Name() string { return "selinux" }

func (c *selinuxCollector) Update(ch chan<- prometheus.Metric) error {
	data, err := os.ReadFile(c.procPath)
	if err != nil {
		// SELinux may not be enabled; emit 0.
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "selinux_enabled"), "Whether SELinux is enabled.", nil, nil),
			prometheus.GaugeValue, 0,
		)
		return nil
	}
	val := 0.0
	if strings.TrimSpace(string(data)) == "1" {
		val = 1
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "selinux_enabled"), "Whether SELinux is enabled.", nil, nil),
		prometheus.GaugeValue, val,
	)
	return nil
}

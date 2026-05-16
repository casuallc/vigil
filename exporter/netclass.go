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
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("netclass", func() (Collector, error) {
		return newNetclassCollector("/sys/class/net"), nil
	})
}

type netclassCollector struct {
	sysPath string
}

func newNetclassCollector(sysPath string) Collector {
	return &netclassCollector{sysPath: sysPath}
}

func (c *netclassCollector) Name() string { return "netclass" }

func (c *netclassCollector) Update(ch chan<- prometheus.Metric) error {
	entries, err := os.ReadDir(c.sysPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		iface := e.Name()
		if iface == "lo" {
			continue
		}
		c.emitInfo(ch, iface)
	}
	return nil
}

func (c *netclassCollector) emitInfo(ch chan<- prometheus.Metric, iface string) {
	base := filepath.Join(c.sysPath, iface)
	mtu := readSysfsInt(filepath.Join(base, "mtu"))
	state := readSysfsString(filepath.Join(base, "operstate"))
	carrier := readSysfsInt(filepath.Join(base, "carrier"))

	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "network", "info"),
		"Non-numeric data from /sys/class/net/<iface>, value is always 1.",
		[]string{"device", "address", "broadcast", "duplex", "operstate", "ifalias"},
		nil,
	)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1,
		iface, "", "", "", state, "",
	)

	mtuDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "network", "mtu_bytes"),
		"Network interface MTU in bytes.",
		[]string{"device"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(mtuDesc, prometheus.GaugeValue, float64(mtu), iface)

	carrierDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "network", "carrier"),
		"1 if the physical network link is up, 0 if down.",
		[]string{"device"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(carrierDesc, prometheus.GaugeValue, float64(carrier), iface)
}

func readSysfsString(path string) string {
	data, _ := os.ReadFile(path)
	return strings.TrimSpace(string(data))
}

func readSysfsInt(path string) int64 {
	s := readSysfsString(path)
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

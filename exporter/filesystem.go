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
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("filesystem", func() (Collector, error) {
		return newFilesystemCollector("/proc/1/mounts"), nil
	})
}

type filesystemCollector struct {
	procPath string
}

func newFilesystemCollector(procPath string) Collector {
	return &filesystemCollector{procPath: procPath}
}

func (c *filesystemCollector) Name() string { return "filesystem" }

func (c *filesystemCollector) Update(ch chan<- prometheus.Metric) error {
	f, err := os.Open(c.procPath)
	if err != nil {
		return err
	}
	defer f.Close()

	ignoredFSTypes := map[string]bool{
		"proc": true, "sysfs": true, "tmpfs": true, "devpts": true,
		"cgroup": true, "cgroup2": true, "devtmpfs": true, "fusectl": true,
		"overlay": true, "squashfs": true, "aufs": true, "tracefs": true,
		"debugfs": true, "securityfs": true, "pstore": true, "bpf": true,
		"hugetlbfs": true, "mqueue": true, "rpc_pipefs": true, "configfs": true,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 4 {
			continue
		}
		device := parts[0]
		mountpoint := parts[1]
		fstype := parts[2]

		if ignoredFSTypes[fstype] || strings.HasPrefix(device, "sunrpc") {
			continue
		}

		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "filesystem", "device_error"),
			"Whether an error occurred while getting statistics for the given device.",
			[]string{"device", "fstype", "mountpoint"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 0, device, fstype, mountpoint)
	}
	return scanner.Err()
}

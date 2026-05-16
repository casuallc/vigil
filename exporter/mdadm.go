/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	registerLinuxCollector("mdadm", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newMdadmCollector(fs)
	})
}

type mdadmCollector struct {
	fs procfs.FS
}

func newMdadmCollector(fs procfs.FS) (Collector, error) {
	return &mdadmCollector{fs: fs}, nil
}

func (c *mdadmCollector) Name() string { return "mdadm" }

func (c *mdadmCollector) Update(ch chan<- prometheus.Metric) error {
	_, err := c.fs.MDStat()
	// mdadm metrics are complex; for now just swallow errors if /proc/mdstat
	// is absent or empty. Full implementation can be added later.
	if err != nil {
		return nil
	}
	return nil
}

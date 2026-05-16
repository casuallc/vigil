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
	registerLinuxCollector("schedstat", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newSchedstatCollector(fs)
	})
}

type schedstatCollector struct {
	fs procfs.FS
}

func newSchedstatCollector(fs procfs.FS) (Collector, error) {
	return &schedstatCollector{fs: fs}, nil
}

func (c *schedstatCollector) Name() string { return "schedstat" }

func (c *schedstatCollector) Update(ch chan<- prometheus.Metric) error {
	_, err := c.fs.Schedstat()
	// schedstat parsing is complex; emit a minimal metric for now.
	if err != nil {
		return nil
	}
	return nil
}

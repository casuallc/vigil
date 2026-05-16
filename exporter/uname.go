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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("uname", func() (Collector, error) {
		return newUnameCollector("/proc")
	})
}

type unameCollector struct {
	procRoot string
	desc     *prometheus.Desc
}

func newUnameCollector(procRoot string) (Collector, error) {
	return &unameCollector{
		procRoot: procRoot,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "uname_info"),
			"Labeled system information as provided by the uname system call.",
			[]string{"sysname", "release", "version", "machine"},
			nil,
		),
	}, nil
}

func (c *unameCollector) Name() string { return "uname" }

func (c *unameCollector) Update(ch chan<- prometheus.Metric) error {
	info := c.readUname()
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1,
		info["sysname"], info["release"], info["version"], info["machine"],
	)
	return nil
}

func (c *unameCollector) readUname() map[string]string {
	sysname, _ := os.ReadFile(filepath.Join(c.procRoot, "sys/kernel/ostype"))
	release, _ := os.ReadFile(filepath.Join(c.procRoot, "sys/kernel/osrelease"))
	version, _ := os.ReadFile(filepath.Join(c.procRoot, "sys/kernel/version"))
	machine, _ := os.ReadFile(filepath.Join(c.procRoot, "sys/kernel/hardware"))
	if machine == nil {
		machine = []byte("")
	}
	return map[string]string{
		"sysname":  strings.TrimSpace(string(sysname)),
		"release":  strings.TrimSpace(string(release)),
		"version":  strings.TrimSpace(string(version)),
		"machine":  strings.TrimSpace(string(machine)),
	}
}

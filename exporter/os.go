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
	registerLinuxCollector("os", func() (Collector, error) {
		return newOsCollector("/etc/os-release")
	})
}

type osCollector struct {
	path string
	desc *prometheus.Desc
}

func newOsCollector(path string) (Collector, error) {
	return &osCollector{
		path: path,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "os_info"),
			"A metric with a constant '1' value labeled by id, id_like, name, pretty_name and variant.",
			[]string{"id", "id_like", "name", "pretty_name", "variant", "variant_id", "version", "version_id", "build_id"},
			nil,
		),
	}, nil
}

func (c *osCollector) Name() string { return "os" }

func (c *osCollector) Update(ch chan<- prometheus.Metric) error {
	info := readOSRelease(c.path)
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1,
		info["ID"],
		info["ID_LIKE"],
		info["NAME"],
		info["PRETTY_NAME"],
		info["VARIANT"],
		info["VARIANT_ID"],
		info["VERSION"],
		info["VERSION_ID"],
		info["BUILD_ID"],
	)
	return nil
}

func readOSRelease(path string) map[string]string {
	defaults := map[string]string{
		"ID": "", "ID_LIKE": "", "NAME": "", "PRETTY_NAME": "",
		"VARIANT": "", "VARIANT_ID": "", "VERSION": "", "VERSION_ID": "", "BUILD_ID": "",
	}
	f, err := os.Open(path)
	if err != nil {
		return defaults
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"`)
		defaults[key] = val
	}
	return defaults
}

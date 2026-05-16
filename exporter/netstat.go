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
	registerLinuxCollector("netstat", func() (Collector, error) {
		return newNetstatCollector("/proc/net/netstat"), nil
	})
}

type netstatCollector struct {
	procPath string
}

func newNetstatCollector(procPath string) Collector {
	return &netstatCollector{procPath: procPath}
}

func (c *netstatCollector) Name() string { return "netstat" }

func (c *netstatCollector) Update(ch chan<- prometheus.Metric) error {
	f, err := os.Open(c.procPath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var keys []string
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := strings.TrimSpace(parts[0])
		values := strings.Fields(parts[1])

		if len(keys) == 0 || keys[0] != prefix {
			keys = append([]string{prefix}, values...)
			continue
		}

		for i, v := range values {
			if i+1 >= len(keys) {
				break
			}
			key := fmt.Sprintf("%s_%s", strings.ToLower(keys[0]), strings.ToLower(keys[i+1]))
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "netstat", key),
				fmt.Sprintf("%s %s statistic.", keys[0], keys[i+1]),
				nil, nil,
			)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, val)
		}
		keys = nil
	}
	return scanner.Err()
}

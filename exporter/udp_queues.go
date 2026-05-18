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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerLinuxCollector("udp_queues", func() (Collector, error) {
		return newUdpQueuesCollector("/proc/net/udp", "/proc/net/udp6"), nil
	})
}

type udpQueuesCollector struct {
	ipv4Path string
	ipv6Path string
}

func newUdpQueuesCollector(ipv4Path, ipv6Path string) Collector {
	return &udpQueuesCollector{ipv4Path: ipv4Path, ipv6Path: ipv6Path}
}

func (c *udpQueuesCollector) Name() string { return "udp_queues" }

func (c *udpQueuesCollector) Update(ch chan<- prometheus.Metric) error {
	for _, path := range []string{c.ipv4Path, c.ipv6Path} {
		if err := c.update(ch, path); err != nil {
			continue
		}
	}
	return nil
}

func (c *udpQueuesCollector) update(ch chan<- prometheus.Metric, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ipVer := "4"
	if strings.Contains(path, "udp6") {
		ipVer = "6"
	}

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil // skip header
	}
	var txTotal, rxTotal float64
	var sockets int
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}
		// Fields: sl local_address rem_address st tx_queue:rx_queue tr:tm->rems ...
		queues := strings.Split(fields[4], ":")
		if len(queues) != 2 {
			continue
		}
		txTotal += parseQueueLen(queues[0])
		rxTotal += parseQueueLen(queues[1])
		sockets++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if sockets == 0 {
		// No sockets in this address family; skip emission to avoid a misleading
		// zero series for an inactive family (e.g. IPv6 disabled).
		return nil
	}

	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "udp", "queue_length"),
		"Total bytes queued in all UDP sockets, by IP version and direction.",
		[]string{"ip", "queue"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, txTotal, ipVer, "tx")
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, rxTotal, ipVer, "rx")
	return nil
}

func parseQueueLen(s string) float64 {
	var v uint32
	if _, err := fmt.Sscanf(s, "%x", &v); err != nil {
		return 0
	}
	return float64(v)
}

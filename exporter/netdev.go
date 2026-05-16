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
	registerLinuxCollector("netdev", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newNetdevCollector(fs)
	})
}

type netdevCollector struct {
	fs procfs.FS
}

func newNetdevCollector(fs procfs.FS) (Collector, error) {
	return &netdevCollector{fs: fs}, nil
}

func (c *netdevCollector) Name() string { return "netdev" }

func (c *netdevCollector) Update(ch chan<- prometheus.Metric) error {
	dev, err := c.fs.NetDev()
	if err != nil {
		return err
	}
	for iface, stats := range dev {
		c.emit(ch, "receive_bytes", float64(stats.RxBytes), iface)
		c.emit(ch, "receive_packets", float64(stats.RxPackets), iface)
		c.emit(ch, "receive_errors", float64(stats.RxErrors), iface)
		c.emit(ch, "receive_dropped", float64(stats.RxDropped), iface)
		c.emit(ch, "receive_multicast", float64(stats.RxMulticast), iface)
		c.emit(ch, "transmit_bytes", float64(stats.TxBytes), iface)
		c.emit(ch, "transmit_packets", float64(stats.TxPackets), iface)
		c.emit(ch, "transmit_errors", float64(stats.TxErrors), iface)
		c.emit(ch, "transmit_dropped", float64(stats.TxDropped), iface)
		c.emit(ch, "transmit_colls", float64(stats.TxCollisions), iface)
	}
	return nil
}

func (c *netdevCollector) emit(ch chan<- prometheus.Metric, name string, value float64, iface string) {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "network", name+"_total"),
		"Network device statistic "+name+".",
		[]string{"device"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, iface)
}

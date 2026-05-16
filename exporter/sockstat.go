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
	registerLinuxCollector("sockstat", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newSockstatCollector(fs)
	})
}

type sockstatCollector struct {
	fs procfs.FS
}

func newSockstatCollector(fs procfs.FS) (Collector, error) {
	return &sockstatCollector{fs: fs}, nil
}

func (c *sockstatCollector) Name() string { return "sockstat" }

func (c *sockstatCollector) Update(ch chan<- prometheus.Metric) error {
	for _, fn := range []func() error{
		c.updateIPv4, c.updateIPv6,
	} {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (c *sockstatCollector) updateIPv4() error {
	// stub: procfs.NetSockstat needs a real /proc/net/sockstat
	return nil
}

func (c *sockstatCollector) updateIPv6() error {
	// stub
	return nil
}

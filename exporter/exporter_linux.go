/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

//go:build linux

package exporter

import (
	"errors"
	"fmt"
	"log"
)

// ErrUnsupported is returned by GatherJSON on platforms where the exporter
// has no collectors (e.g. non-Linux).
var ErrUnsupported = errors.New("exporter is only supported on Linux")

// NewNodeExporter constructs an exporter with the default Linux collector
// set. It reads /proc and /sys at scrape time via prometheus/procfs and
// direct sysfs reads.
func NewNodeExporter() (*NodeExporter, error) {
	return newNodeExporterWithCollectors(defaultLinuxCollectors())
}

// defaultLinuxCollectors returns the set of collectors enabled by default.
// Individual collectors register themselves into this map via their own
// init() functions in *_linux.go files. The map is populated at package
// initialization time, so callers should not modify it concurrently.
func defaultLinuxCollectors() map[string]Collector {
	out := make(map[string]Collector, len(linuxCollectorFactories))
	for name, factory := range linuxCollectorFactories {
		c, err := safeFactory(factory)
		if err != nil {
			// A factory failure usually means the collector cannot read its
			// data source on this kernel/build. We log and skip rather than
			// fail the entire exporter so the rest still works.
			log.Printf("exporter: skipping collector %s: %v", name, err)
			continue
		}
		out[name] = c
	}
	return out
}

// safeFactory runs a collector factory, converting panics into errors.
// Some collectors (e.g. cilium/ebpf) panic instead of returning errors on
// incompatible kernels/ABIs — e.g. raw rt_sigprocmask with a mismatched
// sigset size on LoongArch old-world kernels. A metrics collector must
// never take down the whole server.
func safeFactory(factory func() (Collector, error)) (c Collector, err error) {
	defer func() {
		if r := recover(); r != nil {
			c = nil
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return factory()
}

// linuxCollectorFactories is populated by individual collector init()s.
// Each entry is a constructor that may inspect the runtime environment
// (e.g. presence of /sys/class/hwmon) before returning a working collector.
var linuxCollectorFactories = map[string]func() (Collector, error){}

// registerLinuxCollector is the entry point each *_linux.go file uses in its
// init() to add itself to the default set.
func registerLinuxCollector(name string, factory func() (Collector, error)) {
	linuxCollectorFactories[name] = factory
}

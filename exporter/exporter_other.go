/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

//go:build !linux

package exporter

import "errors"

// ErrUnsupported is returned by GatherJSON on platforms where the exporter
// has no collectors (e.g. non-Linux).
var ErrUnsupported = errors.New("exporter is only supported on Linux")

// NewNodeExporter on non-Linux platforms returns an empty exporter so the
// API surface compiles. /metrics will be valid but contain only meta-metrics
// and /api/resources/system handlers should detect the empty collector set
// and respond accordingly.
func NewNodeExporter() (*NodeExporter, error) {
	return newNodeExporterWithCollectors(map[string]Collector{})
}

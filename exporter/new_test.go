/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"runtime"
	"testing"
)

// TestNewNodeExporter_ReturnsUsableRegistry verifies that the public
// constructor yields an exporter with a non-nil Registry on every platform.
// On non-Linux platforms the registry has no collectors registered, but it
// must still be safe to gather from.
func TestNewNodeExporter_ReturnsUsableRegistry(t *testing.T) {
	n, err := NewNodeExporter()
	if err != nil {
		t.Fatalf("NewNodeExporter: %v", err)
	}
	if n == nil {
		t.Fatal("NewNodeExporter returned nil")
	}
	if n.Registry() == nil {
		t.Fatal("Registry() returned nil")
	}
	// Gather must not error even with zero collectors.
	if _, err := n.Registry().Gather(); err != nil {
		t.Errorf("Gather: %v", err)
	}
	// GatherJSON should always succeed, returning at minimum the scrape group.
	out, err := n.GatherJSON()
	if err != nil {
		t.Fatalf("GatherJSON: %v", err)
	}
	if _, ok := out["scrape"]; !ok {
		t.Errorf("GatherJSON missing scrape group: %+v", out)
	}
}

// TestNewNodeExporter_LinuxRegistersDefaultCollectors confirms that on Linux
// the default collector set is wired up. On other platforms this is a noop.
func TestNewNodeExporter_LinuxRegistersDefaultCollectors(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("collectors only ship on Linux (running on %s)", runtime.GOOS)
	}
	n, err := NewNodeExporter()
	if err != nil {
		t.Fatalf("NewNodeExporter: %v", err)
	}
	if len(n.collectors) == 0 {
		t.Error("expected at least one collector to be registered on Linux")
	}
}

/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exporter

import (
	"sort"

	"github.com/casuallc/vigil/models"
)

// GatherToResourceStats converts the Prometheus-style output of GatherJSON
// into the flat ResourceStats structure expected by /api/resources/system.
// Fields that cannot be derived from exporter metrics are left at zero/nil.
func GatherToResourceStats(data map[string]CollectorMetrics) models.ResourceStats {
	var stats models.ResourceStats

	// Flatten all collector groups into a single metric lookup table.
	// Metric names are unique across collectors, so this is safe.
	allMetrics := CollectorMetrics{}
	for _, group := range data {
		for name, samples := range group {
			allMetrics[name] = append(allMetrics[name], samples...)
		}
	}

	// ---------- CPU ----------
	stats.CPUUsage = calcCPUUsage(allMetrics)

	// ---------- Memory ----------
	stats.MemoryTotal = getUint64(allMetrics, "node_memory_MemTotal_bytes")
	stats.MemoryAvailable = getUint64(allMetrics, "node_memory_MemAvailable_bytes")
	memFree := getUint64(allMetrics, "node_memory_MemFree_bytes")

	// memory_usage = total - available (preferred) or total - free
	if stats.MemoryAvailable > 0 {
		stats.MemoryUsage = stats.MemoryTotal - stats.MemoryAvailable
	} else if memFree > 0 {
		stats.MemoryUsage = stats.MemoryTotal - memFree
	}
	if stats.MemoryTotal > 0 {
		stats.MemoryUsedPercent = float64(stats.MemoryUsage) / float64(stats.MemoryTotal) * 100
	}

	// Swap
	stats.SwapTotal = getUint64(allMetrics, "node_memory_SwapTotal_bytes")
	swapFree := getUint64(allMetrics, "node_memory_SwapFree_bytes")
	if stats.SwapTotal > 0 {
		stats.SwapUsed = stats.SwapTotal - swapFree
		stats.SwapFree = swapFree
	}

	// ---------- Load ----------
	stats.Load = models.LoadAvg{
		Load1:  getFloat(allMetrics, "node_load1"),
		Load5:  getFloat(allMetrics, "node_load5"),
		Load15: getFloat(allMetrics, "node_load15"),
	}

	// ---------- File Descriptors ----------
	allocated := getUint64(allMetrics, "node_filefd_allocated")
	max := getUint64(allMetrics, "node_filefd_maximum")
	if max > 0 {
		stats.FD = models.FDCheck{
			CurrentAllocated: allocated,
			InUse:            allocated,
			Max:              max,
			UsagePercent:     float64(allocated) / float64(max) * 100,
		}
	}

	// ---------- Disk IO ----------
	stats.DiskIODevices = buildDiskIODevices(allMetrics)
	var totalDiskIO uint64
	for _, d := range stats.DiskIODevices {
		totalDiskIO += d.ReadBytes + d.WriteBytes
	}
	stats.DiskIO = totalDiskIO

	// ---------- Network (aggregated counters) ----------
	var rxBytes, txBytes, rxPkts, txPkts, rxErrs, txErrs, rxDrop, txDrop uint64
	for _, s := range allMetrics["node_network_receive_bytes_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		rxBytes += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_transmit_bytes_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		txBytes += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_receive_packets_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		rxPkts += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_transmit_packets_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		txPkts += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_receive_errors_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		rxErrs += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_transmit_errors_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		txErrs += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_receive_dropped_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		rxDrop += uint64(s.Value)
	}
	for _, s := range allMetrics["node_network_transmit_dropped_total"] {
		if iface := s.Labels["device"]; iface == "lo" {
			continue
		}
		txDrop += uint64(s.Value)
	}
	stats.NetRxPackets = rxPkts
	stats.NetTxPackets = txPkts
	stats.NetRxErrors = rxErrs
	stats.NetTxErrors = txErrs
	stats.NetRxDropped = rxDrop
	stats.NetTxDropped = txDrop

	// network_io is a rough aggregate of all bytes sent+received
	stats.NetworkIO = rxBytes + txBytes

	// ---------- Memory Pressure (PSI) ----------
	stats.MemoryPressure = buildMemoryPressure(allMetrics)

	// disk_usages, tcp_state_counts, system_uptime_seconds, and
	// net_*_per_sec rates cannot be derived from the current exporter
	// metrics and are intentionally left at zero / nil.

	return stats
}

// calcCPUUsage computes the average CPU usage since boot from
// node_cpu_seconds_total counters (cpu="cpu" aggregate).
func calcCPUUsage(allMetrics CollectorMetrics) float64 {
	var total, idle float64
	for _, s := range allMetrics["node_cpu_seconds_total"] {
		if s.Labels["cpu"] != "cpu" {
			continue // per-CPU entries; skip them
		}
		total += s.Value
		if s.Labels["mode"] == "idle" {
			idle = s.Value
		}
	}
	if total <= 0 {
		return 0
	}
	return (total - idle) / total * 100
}

// buildDiskIODevices maps node_disk_* metrics into DiskIOInfo structs.
func buildDiskIODevices(allMetrics CollectorMetrics) []models.DiskIOInfo {
	// Group samples by device label.
	byDevice := map[string]*models.DiskIOInfo{}

	for _, s := range allMetrics["node_disk_reads_completed_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).ReadCount = uint64(s.Value)
	}
	for _, s := range allMetrics["node_disk_sectors_read_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).ReadBytes = uint64(s.Value) * 512
	}
	for _, s := range allMetrics["node_disk_read_time_seconds_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).ReadTimeMS = uint64(s.Value * 1000)
	}
	for _, s := range allMetrics["node_disk_writes_completed_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).WriteCount = uint64(s.Value)
	}
	for _, s := range allMetrics["node_disk_sectors_written_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).WriteBytes = uint64(s.Value) * 512
	}
	for _, s := range allMetrics["node_disk_write_time_seconds_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).WriteTimeMS = uint64(s.Value * 1000)
	}
	for _, s := range allMetrics["node_disk_io_time_seconds_total"] {
		dev := s.Labels["device"]
		getOrCreate(byDevice, dev).BusyTimeMS = uint64(s.Value * 1000)
	}

	// Compute derived fields.
	for dev, d := range byDevice {
		d.Device = dev
		if d.ReadCount > 0 {
			d.AvgReadLatencyMS = float64(d.ReadTimeMS) / float64(d.ReadCount)
		}
		if d.WriteCount > 0 {
			d.AvgWriteLatencyMS = float64(d.WriteTimeMS) / float64(d.WriteCount)
		}
	}

	// Stable sort by device name.
	devices := make([]string, 0, len(byDevice))
	for dev := range byDevice {
		devices = append(devices, dev)
	}
	sort.Strings(devices)

	out := make([]models.DiskIOInfo, 0, len(devices))
	for _, dev := range devices {
		out = append(out, *byDevice[dev])
	}
	return out
}

// buildMemoryPressure extracts memory PSI from node_pressure_memory_some.
func buildMemoryPressure(allMetrics CollectorMetrics) models.PressureStallInfo {
	var psi models.PressureStallInfo
	for _, s := range allMetrics["node_pressure_memory_some"] {
		window := s.Labels["window"]
		switch window {
		case "avg10":
			psi.Avg10 = s.Value
		case "avg60":
			psi.Avg60 = s.Value
		case "avg300":
			psi.Avg300 = s.Value
		}
	}
	for _, s := range allMetrics["node_pressure_memory_some_total"] {
		// total is stored as seconds (float64); convert to microseconds -> uint64
		psi.Total = uint64(s.Value * 1_000_000)
	}
	return psi
}

// getFloat returns the first sample value for a metric name, or 0.
func getFloat(allMetrics CollectorMetrics, name string) float64 {
	if samples := allMetrics[name]; len(samples) > 0 {
		return samples[0].Value
	}
	return 0
}

// getUint64 returns the first sample value as uint64, or 0.
func getUint64(allMetrics CollectorMetrics, name string) uint64 {
	return uint64(getFloat(allMetrics, name))
}

// getOrCreate returns the DiskIOInfo for a device, creating it if needed.
func getOrCreate(m map[string]*models.DiskIOInfo, device string) *models.DiskIOInfo {
	if d, ok := m[device]; ok {
		return d
	}
	m[device] = &models.DiskIOInfo{}
	return m[device]
}

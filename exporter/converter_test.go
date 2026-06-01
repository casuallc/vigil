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
	"testing"
)

func TestGatherToResourceStats_MemoryAndLoad(t *testing.T) {
	data := map[string]CollectorMetrics{
		"meminfo": {
			"node_memory_MemTotal_bytes":     {{Value: 32_000_000_000}},
			"node_memory_MemAvailable_bytes": {{Value: 16_000_000_000}},
			"node_memory_SwapTotal_bytes":    {{Value: 8_000_000_000}},
			"node_memory_SwapFree_bytes":     {{Value: 4_000_000_000}},
		},
		"loadavg": {
			"node_load1":  {{Value: 0.5}},
			"node_load5":  {{Value: 0.7}},
			"node_load15": {{Value: 0.9}},
		},
	}

	stats := GatherToResourceStats(data)

	if stats.MemoryTotal != 32_000_000_000 {
		t.Errorf("MemoryTotal = %d, want 32_000_000_000", stats.MemoryTotal)
	}
	if stats.MemoryAvailable != 16_000_000_000 {
		t.Errorf("MemoryAvailable = %d, want 16_000_000_000", stats.MemoryAvailable)
	}
	if stats.MemoryUsage != 16_000_000_000 {
		t.Errorf("MemoryUsage = %d, want 16_000_000_000", stats.MemoryUsage)
	}
	if stats.MemoryUsedPercent != 50.0 {
		t.Errorf("MemoryUsedPercent = %f, want 50.0", stats.MemoryUsedPercent)
	}
	if stats.SwapTotal != 8_000_000_000 {
		t.Errorf("SwapTotal = %d, want 8_000_000_000", stats.SwapTotal)
	}
	if stats.SwapUsed != 4_000_000_000 {
		t.Errorf("SwapUsed = %d, want 4_000_000_000", stats.SwapUsed)
	}
	if stats.Load.Load1 != 0.5 {
		t.Errorf("Load1 = %f, want 0.5", stats.Load.Load1)
	}
	if stats.Load.Load5 != 0.7 {
		t.Errorf("Load5 = %f, want 0.7", stats.Load.Load5)
	}
	if stats.Load.Load15 != 0.9 {
		t.Errorf("Load15 = %f, want 0.9", stats.Load.Load15)
	}
}

func TestGatherToResourceStats_CPUUsage(t *testing.T) {
	data := map[string]CollectorMetrics{
		"cpu": {
			"node_cpu_seconds_total": {
				{Labels: map[string]string{"cpu": "cpu", "mode": "user"}, Value: 100},
				{Labels: map[string]string{"cpu": "cpu", "mode": "system"}, Value: 50},
				{Labels: map[string]string{"cpu": "cpu", "mode": "idle"}, Value: 850},
				// per-cpu entries should be ignored
				{Labels: map[string]string{"cpu": "0", "mode": "user"}, Value: 50},
			},
		},
	}

	stats := GatherToResourceStats(data)

	want := 15.0 // (1000 - 850) / 1000 * 100
	if stats.CPUUsage != want {
		t.Errorf("CPUUsage = %f, want %f", stats.CPUUsage, want)
	}
}

func TestGatherToResourceStats_DiskIO(t *testing.T) {
	data := map[string]CollectorMetrics{
		"diskstats": {
			"node_disk_reads_completed_total": {
				{Labels: map[string]string{"device": "vda"}, Value: 1000},
				{Labels: map[string]string{"device": "vdb"}, Value: 100},
			},
			"node_disk_sectors_read_total": {
				{Labels: map[string]string{"device": "vda"}, Value: 2048},
				{Labels: map[string]string{"device": "vdb"}, Value: 512},
			},
			"node_disk_read_time_seconds_total":    {{Labels: map[string]string{"device": "vda"}, Value: 10}},
			"node_disk_writes_completed_total":     {{Labels: map[string]string{"device": "vda"}, Value: 500}},
			"node_disk_sectors_written_total":      {{Labels: map[string]string{"device": "vda"}, Value: 4096}},
			"node_disk_write_time_seconds_total":   {{Labels: map[string]string{"device": "vda"}, Value: 20}},
			"node_disk_io_time_seconds_total":      {{Labels: map[string]string{"device": "vda"}, Value: 30}},
		},
	}

	stats := GatherToResourceStats(data)

	if stats.DiskIO != (2048+4096+512)*512 {
		t.Errorf("DiskIO = %d, want %d", stats.DiskIO, (2048+4096+512)*512)
	}
	if len(stats.DiskIODevices) != 2 {
		t.Fatalf("DiskIODevices len = %d, want 2", len(stats.DiskIODevices))
	}

	// Sorted by device name: vda first
	vda := stats.DiskIODevices[0]
	if vda.Device != "vda" {
		t.Errorf("Device[0].Device = %s, want vda", vda.Device)
	}
	if vda.ReadCount != 1000 {
		t.Errorf("Device[0].ReadCount = %d, want 1000", vda.ReadCount)
	}
	if vda.ReadBytes != 2048*512 {
		t.Errorf("Device[0].ReadBytes = %d, want %d", vda.ReadBytes, 2048*512)
	}
	if vda.ReadTimeMS != 10000 {
		t.Errorf("Device[0].ReadTimeMS = %d, want 10000", vda.ReadTimeMS)
	}
	if vda.WriteCount != 500 {
		t.Errorf("Device[0].WriteCount = %d, want 500", vda.WriteCount)
	}
	if vda.WriteBytes != 4096*512 {
		t.Errorf("Device[0].WriteBytes = %d, want %d", vda.WriteBytes, 4096*512)
	}
	if vda.WriteTimeMS != 20000 {
		t.Errorf("Device[0].WriteTimeMS = %d, want 20000", vda.WriteTimeMS)
	}
	if vda.BusyTimeMS != 30000 {
		t.Errorf("Device[0].BusyTimeMS = %d, want 30000", vda.BusyTimeMS)
	}
	if vda.AvgReadLatencyMS != 10.0 {
		t.Errorf("Device[0].AvgReadLatencyMS = %f, want 10.0", vda.AvgReadLatencyMS)
	}
	if vda.AvgWriteLatencyMS != 40.0 {
		t.Errorf("Device[0].AvgWriteLatencyMS = %f, want 40.0", vda.AvgWriteLatencyMS)
	}
}

func TestGatherToResourceStats_FD(t *testing.T) {
	data := map[string]CollectorMetrics{
		"filefd": {
			"node_filefd_allocated": {{Value: 5000}},
			"node_filefd_maximum":   {{Value: 1_000_000}},
		},
	}

	stats := GatherToResourceStats(data)

	if stats.FD.CurrentAllocated != 5000 {
		t.Errorf("FD.CurrentAllocated = %d, want 5000", stats.FD.CurrentAllocated)
	}
	if stats.FD.InUse != 5000 {
		t.Errorf("FD.InUse = %d, want 5000", stats.FD.InUse)
	}
	if stats.FD.Max != 1_000_000 {
		t.Errorf("FD.Max = %d, want 1_000_000", stats.FD.Max)
	}
	wantPct := 5000.0 / 1_000_000.0 * 100
	if stats.FD.UsagePercent != wantPct {
		t.Errorf("FD.UsagePercent = %f, want %f", stats.FD.UsagePercent, wantPct)
	}
}

func TestGatherToResourceStats_Network(t *testing.T) {
	data := map[string]CollectorMetrics{
		"netdev": {
			"node_network_receive_bytes_total":  {
				{Labels: map[string]string{"device": "eth0"}, Value: 1000},
				{Labels: map[string]string{"device": "lo"}, Value: 9999}, // should be skipped
				{Labels: map[string]string{"device": "eth1"}, Value: 2000},
			},
			"node_network_transmit_bytes_total": {
				{Labels: map[string]string{"device": "eth0"}, Value: 3000},
				{Labels: map[string]string{"device": "eth1"}, Value: 4000},
			},
			"node_network_receive_packets_total": {
				{Labels: map[string]string{"device": "eth0"}, Value: 10},
				{Labels: map[string]string{"device": "eth1"}, Value: 20},
			},
			"node_network_transmit_packets_total": {
				{Labels: map[string]string{"device": "eth0"}, Value: 30},
				{Labels: map[string]string{"device": "eth1"}, Value: 40},
			},
			"node_network_receive_errors_total": {
				{Labels: map[string]string{"device": "eth0"}, Value: 1},
				{Labels: map[string]string{"device": "eth1"}, Value: 2},
			},
			"node_network_receive_dropped_total": {
				{Labels: map[string]string{"device": "eth0"}, Value: 3},
				{Labels: map[string]string{"device": "eth1"}, Value: 4},
			},
		},
	}

	stats := GatherToResourceStats(data)

	if stats.NetworkIO != 10000 { // 1000+2000+3000+4000
		t.Errorf("NetworkIO = %d, want 10000", stats.NetworkIO)
	}
	if stats.NetRxPackets != 30 {
		t.Errorf("NetRxPackets = %d, want 30", stats.NetRxPackets)
	}
	if stats.NetTxPackets != 70 {
		t.Errorf("NetTxPackets = %d, want 70", stats.NetTxPackets)
	}
	if stats.NetRxErrors != 3 {
		t.Errorf("NetRxErrors = %d, want 3", stats.NetRxErrors)
	}
	if stats.NetRxDropped != 7 {
		t.Errorf("NetRxDropped = %d, want 7", stats.NetRxDropped)
	}
}

func TestGatherToResourceStats_Pressure(t *testing.T) {
	data := map[string]CollectorMetrics{
		"pressure": {
			"node_pressure_memory_some": {
				{Labels: map[string]string{"window": "avg10"}, Value: 1.5},
				{Labels: map[string]string{"window": "avg60"}, Value: 2.5},
				{Labels: map[string]string{"window": "avg300"}, Value: 3.5},
			},
			"node_pressure_memory_some_total": {
				{Value: 10.5}, // seconds
			},
		},
	}

	stats := GatherToResourceStats(data)

	if stats.MemoryPressure.Avg10 != 1.5 {
		t.Errorf("MemoryPressure.Avg10 = %f, want 1.5", stats.MemoryPressure.Avg10)
	}
	if stats.MemoryPressure.Avg60 != 2.5 {
		t.Errorf("MemoryPressure.Avg60 = %f, want 2.5", stats.MemoryPressure.Avg60)
	}
	if stats.MemoryPressure.Avg300 != 3.5 {
		t.Errorf("MemoryPressure.Avg300 = %f, want 3.5", stats.MemoryPressure.Avg300)
	}
	if stats.MemoryPressure.Total != 10_500_000 {
		t.Errorf("MemoryPressure.Total = %d, want 10_500_000", stats.MemoryPressure.Total)
	}
}

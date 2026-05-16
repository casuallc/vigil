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

// meminfoMetricDef pairs a metric name with a getter that returns the byte
// value from procfs.Meminfo. If the getter returns nil the metric is skipped.
type meminfoMetricDef struct {
	name string
	fn   func(procfs.Meminfo) *uint64
}

var meminfoMetrics = []meminfoMetricDef{
	{"MemTotal", func(m procfs.Meminfo) *uint64 { return m.MemTotalBytes }},
	{"MemFree", func(m procfs.Meminfo) *uint64 { return m.MemFreeBytes }},
	{"MemAvailable", func(m procfs.Meminfo) *uint64 { return m.MemAvailableBytes }},
	{"Buffers", func(m procfs.Meminfo) *uint64 { return m.BuffersBytes }},
	{"Cached", func(m procfs.Meminfo) *uint64 { return m.CachedBytes }},
	{"SwapCached", func(m procfs.Meminfo) *uint64 { return m.SwapCachedBytes }},
	{"Active", func(m procfs.Meminfo) *uint64 { return m.ActiveBytes }},
	{"Inactive", func(m procfs.Meminfo) *uint64 { return m.InactiveBytes }},
	{"ActiveAnon", func(m procfs.Meminfo) *uint64 { return m.ActiveAnonBytes }},
	{"InactiveAnon", func(m procfs.Meminfo) *uint64 { return m.InactiveAnonBytes }},
	{"ActiveFile", func(m procfs.Meminfo) *uint64 { return m.ActiveFileBytes }},
	{"InactiveFile", func(m procfs.Meminfo) *uint64 { return m.InactiveFileBytes }},
	{"Unevictable", func(m procfs.Meminfo) *uint64 { return m.UnevictableBytes }},
	{"Mlocked", func(m procfs.Meminfo) *uint64 { return m.MlockedBytes }},
	{"SwapTotal", func(m procfs.Meminfo) *uint64 { return m.SwapTotalBytes }},
	{"SwapFree", func(m procfs.Meminfo) *uint64 { return m.SwapFreeBytes }},
	{"Dirty", func(m procfs.Meminfo) *uint64 { return m.DirtyBytes }},
	{"Writeback", func(m procfs.Meminfo) *uint64 { return m.WritebackBytes }},
	{"AnonPages", func(m procfs.Meminfo) *uint64 { return m.AnonPagesBytes }},
	{"Mapped", func(m procfs.Meminfo) *uint64 { return m.MappedBytes }},
	{"Shmem", func(m procfs.Meminfo) *uint64 { return m.ShmemBytes }},
	{"Slab", func(m procfs.Meminfo) *uint64 { return m.SlabBytes }},
	{"SReclaimable", func(m procfs.Meminfo) *uint64 { return m.SReclaimableBytes }},
	{"SUnreclaim", func(m procfs.Meminfo) *uint64 { return m.SUnreclaimBytes }},
	{"KernelStack", func(m procfs.Meminfo) *uint64 { return m.KernelStackBytes }},
	{"PageTables", func(m procfs.Meminfo) *uint64 { return m.PageTablesBytes }},
	{"NFSUnstable", func(m procfs.Meminfo) *uint64 { return m.NFSUnstableBytes }},
	{"Bounce", func(m procfs.Meminfo) *uint64 { return m.BounceBytes }},
	{"WritebackTmp", func(m procfs.Meminfo) *uint64 { return m.WritebackTmpBytes }},
	{"CommitLimit", func(m procfs.Meminfo) *uint64 { return m.CommitLimitBytes }},
	{"CommittedAS", func(m procfs.Meminfo) *uint64 { return m.CommittedASBytes }},
	{"VmallocTotal", func(m procfs.Meminfo) *uint64 { return m.VmallocTotalBytes }},
	{"VmallocUsed", func(m procfs.Meminfo) *uint64 { return m.VmallocUsedBytes }},
	{"VmallocChunk", func(m procfs.Meminfo) *uint64 { return m.VmallocChunkBytes }},
	{"Percpu", func(m procfs.Meminfo) *uint64 { return m.PercpuBytes }},
	{"HardwareCorrupted", func(m procfs.Meminfo) *uint64 { return m.HardwareCorruptedBytes }},
	{"AnonHugePages", func(m procfs.Meminfo) *uint64 { return m.AnonHugePagesBytes }},
	{"ShmemHugePages", func(m procfs.Meminfo) *uint64 { return m.ShmemHugePagesBytes }},
	{"ShmemPmdMapped", func(m procfs.Meminfo) *uint64 { return m.ShmemPmdMappedBytes }},
	{"CmaTotal", func(m procfs.Meminfo) *uint64 { return m.CmaTotalBytes }},
	{"CmaFree", func(m procfs.Meminfo) *uint64 { return m.CmaFreeBytes }},
	// HugePages fields are in pages, not bytes — keep raw values.
	{"HugePages_Total", func(m procfs.Meminfo) *uint64 { return m.HugePagesTotal }},
	{"HugePages_Free", func(m procfs.Meminfo) *uint64 { return m.HugePagesFree }},
	{"HugePages_Rsvd", func(m procfs.Meminfo) *uint64 { return m.HugePagesRsvd }},
	{"HugePages_Surp", func(m procfs.Meminfo) *uint64 { return m.HugePagesSurp }},
	{"Hugepagesize", func(m procfs.Meminfo) *uint64 { return m.HugepagesizeBytes }},
	{"DirectMap4k", func(m procfs.Meminfo) *uint64 { return m.DirectMap4kBytes }},
	{"DirectMap2M", func(m procfs.Meminfo) *uint64 { return m.DirectMap2MBytes }},
	{"DirectMap1G", func(m procfs.Meminfo) *uint64 { return m.DirectMap1GBytes }},
}

func init() {
	registerLinuxCollector("meminfo", func() (Collector, error) {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			return nil, err
		}
		return newMeminfoCollector(fs)
	})
}

type meminfoCollector struct {
	fs      procfs.FS
	descMap map[string]*prometheus.Desc
}

func newMeminfoCollector(fs procfs.FS) (Collector, error) {
	descMap := make(map[string]*prometheus.Desc, len(meminfoMetrics))
	for _, m := range meminfoMetrics {
		suffix := "_bytes"
		if m.name == "HugePages_Total" || m.name == "HugePages_Free" || m.name == "HugePages_Rsvd" || m.name == "HugePages_Surp" {
			suffix = "" // pages
		}
		name := prometheus.BuildFQName(namespace, "memory", m.name+suffix)
		descMap[m.name] = prometheus.NewDesc(name, "Memory information field "+m.name+".", nil, nil)
	}
	return &meminfoCollector{fs: fs, descMap: descMap}, nil
}

func (c *meminfoCollector) Name() string { return "meminfo" }

func (c *meminfoCollector) Update(ch chan<- prometheus.Metric) error {
	meminfo, err := c.fs.Meminfo()
	if err != nil {
		return err
	}
	for _, m := range meminfoMetrics {
		val := m.fn(meminfo)
		if val == nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.descMap[m.name], prometheus.GaugeValue, float64(*val))
	}
	return nil
}

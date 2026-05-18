# Vigil Exporter Metrics Reference

All metrics use the `node_` prefix (namespace = `"node"`). There are **31 collectors**, of which **4 are stubs** (do not yet emit metrics). Each collector runs concurrently; a failure in one does not affect the others.

## Meta Metrics (from the Exporter itself)

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_scrape_collector_duration_seconds` | Gauge | `collector` | Duration of a single collector scrape (seconds) |
| `node_scrape_collector_success` | Gauge | `collector` | Whether the scrape succeeded (1 = success, 0 = failure) |

---

## Collectors

### 1. ebpf_traffic (eBPF Traffic)
**Source**: two BPF programs (ingress/egress) that auto-detect the best attachment mode:
- **cgroup_skb mode** (preferred): attaches to the root cgroup v2.
- **TC mode** (fallback): attaches to all non-loopback interfaces via clsact qdisc when cgroup v2 is unavailable.

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_ebpf_traffic_bytes_total` | Counter | `remote_ip`, `direction` | Bytes per remote IPv4 and direction since BPF program load |
| `node_ebpf_traffic_packets_total` | Counter | `remote_ip`, `direction` | Packets per remote IPv4 and direction since BPF program load |
| `node_ebpf_traffic_truncated_flows` | Gauge | — | Flows dropped by the top-N limit (default 1000) in this scrape |

> Notes: Loopback (127.0.0.0/8) and non-IPv4 traffic are skipped. The BPF map is an LRU hash capped at 8192 entries.

### 2. udp_queues (UDP Queue Lengths)
**Source**: `/proc/net/udp`, `/proc/net/udp6`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_udp_queue_length` | Gauge | `ip` (4/6), `queue` (tx/rx) | Total queued bytes across all UDP sockets |

### 3. cpu (CPU Time)
**Source**: `/proc/stat`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_cpu_seconds_total` | Counter | `cpu` (cpu0/cpu1/.../cpu), `mode` | Seconds each CPU spent in each mode. Modes: user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice |

### 4. meminfo (Memory Information)
**Source**: `/proc/meminfo` (45+ fields)

| Metric | Type | Description |
|---|---|---|
| `node_memory_<field>_bytes` | Gauge | Memory fields in bytes. Fields: MemTotal, MemFree, MemAvailable, Buffers, Cached, SwapCached, Active, Inactive, ActiveAnon, InactiveAnon, ActiveFile, InactiveFile, Unevictable, Mlocked, SwapTotal, SwapFree, Dirty, Writeback, AnonPages, Mapped, Shmem, Slab, SReclaimable, SUnreclaim, KernelStack, PageTables, NFSUnstable, Bounce, WritebackTmp, CommitLimit, CommittedAS, VmallocTotal, VmallocUsed, VmallocChunk, Percpu, HardwareCorrupted, AnonHugePages, ShmemHugePages, ShmemPmdMapped, CmaTotal, CmaFree, Hugepagesize, DirectMap4k, DirectMap2M, DirectMap1G |
| `node_memory_HugePages_Total` | Gauge | Total huge pages (in pages, not bytes) |
| `node_memory_HugePages_Free` | Gauge | Free huge pages |
| `node_memory_HugePages_Rsvd` | Gauge | Reserved huge pages |
| `node_memory_HugePages_Surp` | Gauge | Surplus huge pages |

### 5. netdev (Network Device Statistics)
**Source**: `/proc/net/dev`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_network_receive_bytes_total` | Counter | `device` | Received bytes |
| `node_network_receive_packets_total` | Counter | `device` | Received packets |
| `node_network_receive_errors_total` | Counter | `device` | Receive errors |
| `node_network_receive_dropped_total` | Counter | `device` | Receive drops |
| `node_network_receive_multicast_total` | Counter | `device` | Received multicast |
| `node_network_transmit_bytes_total` | Counter | `device` | Transmitted bytes |
| `node_network_transmit_packets_total` | Counter | `device` | Transmitted packets |
| `node_network_transmit_errors_total` | Counter | `device` | Transmit errors |
| `node_network_transmit_dropped_total` | Counter | `device` | Transmit drops |
| `node_network_transmit_colls_total` | Counter | `device` | Transmit collisions |

### 6. diskstats (Disk I/O Statistics)
**Source**: `/proc/diskstats`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_disk_reads_completed_total` | Counter | `device` | Read completions |
| `node_disk_reads_merged_total` | Counter | `device` | Merged reads |
| `node_disk_sectors_read_total` | Counter | `device` | Sectors read |
| `node_disk_read_time_seconds_total` | Counter | `device` | Read time (seconds) |
| `node_disk_writes_completed_total` | Counter | `device` | Write completions |
| `node_disk_writes_merged_total` | Counter | `device` | Merged writes |
| `node_disk_sectors_written_total` | Counter | `device` | Sectors written |
| `node_disk_write_time_seconds_total` | Counter | `device` | Write time (seconds) |
| `node_disk_io_now_total` | Counter | `device` | Current I/O operations |
| `node_disk_io_time_seconds_total` | Counter | `device` | I/O time (seconds) |
| `node_disk_io_time_weighted_seconds_total` | Counter | `device` | Weighted I/O time |

### 7. filesystem (Filesystem Mounts)
**Source**: `/proc/1/mounts`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_filesystem_device_error` | Gauge | `device`, `fstype`, `mountpoint` | Whether an error occurred reading stats (0 = no error) |

> Virtual filesystems (proc, sysfs, tmpfs, cgroup, overlay, etc.) are skipped.

### 8. loadavg (Load Average)
**Source**: `/proc/loadavg`

| Metric | Type | Description |
|---|---|---|
| `node_load1` | Gauge | 1-minute load average |
| `node_load5` | Gauge | 5-minute load average |
| `node_load15` | Gauge | 15-minute load average |

### 9. pressure (PSI Pressure Statistics)
**Source**: `/proc/pressure/cpu`, `/proc/pressure/memory`, `/proc/pressure/io`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_pressure_cpu_some{window}` | Gauge | `window` (avg10/avg60/avg300) | CPU some pressure ratio |
| `node_pressure_cpu_some_total` | Counter | — | CPU some total stall time (seconds) |
| `node_pressure_memory_some{window}` | Gauge | `window` | Memory some pressure ratio |
| `node_pressure_memory_some_total` | Counter | — | Memory some total stall time |
| `node_pressure_memory_full{window}` | Gauge | `window` | Memory full pressure ratio |
| `node_pressure_memory_full_total` | Counter | — | Memory full total stall time |
| `node_pressure_io_some{window}` | Gauge | `window` | I/O some pressure ratio |
| `node_pressure_io_some_total` | Counter | — | I/O some total stall time |
| `node_pressure_io_full{window}` | Gauge | `window` | I/O full pressure ratio |
| `node_pressure_io_full_total` | Counter | — | I/O full total stall time |

### 10. stat (System Statistics)
**Source**: `/proc/stat`

| Metric | Type | Description |
|---|---|---|
| `node_context_switches_total` | Counter | Total context switches |
| `node_forks_total` | Counter | Total forks |
| `node_interrupts_total` | Counter | Total interrupts serviced |
| `node_procs_running` | Gauge | Processes in runnable state |
| `node_procs_blocked` | Gauge | Processes blocked waiting for I/O |

### 11. vmstat (Virtual Memory Statistics)
**Source**: `/proc/vmstat`

| Metric | Type | Description |
|---|---|---|
| `node_vmstat_<field>` | Counter | Every field in `/proc/vmstat`, e.g. `pgpgin`, `pgpgout`, `pswpin`, `pswpout` |

### 12. conntrack (Connection Tracking)
**Source**: `/proc/sys/net/netfilter/nf_conntrack_count`

| Metric | Type | Description |
|---|---|---|
| `node_nf_conntrack_entries` | Gauge | Currently allocated conntrack flow entries |

### 13. entropy (Entropy Pool)
**Source**: `/proc/sys/kernel/random/entropy_avail`

| Metric | Type | Description |
|---|---|---|
| `node_entropy_available_bits` | Gauge | Available entropy bits |

### 14. filefd (File Descriptors)
**Source**: `/proc/sys/fs/file-nr`

| Metric | Type | Description |
|---|---|---|
| `node_filefd_allocated` | Gauge | Allocated file descriptors |
| `node_filefd_maximum` | Gauge | Maximum file descriptors |

### 15. hwmon (Hardware Monitoring)
**Source**: `/sys/class/hwmon`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_hwmon_temp_celsius` | Gauge | `chip`, `sensor` | Temperature sensor reading (Celsius) |

### 16. netclass (Network Interface Information)
**Source**: `/sys/class/net`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_network_info` | Gauge | `device`, `address`, `broadcast`, `duplex`, `operstate`, `ifalias` | Constant value 1, carries interface metadata |
| `node_network_mtu_bytes` | Gauge | `device` | Interface MTU (bytes) |
| `node_network_carrier` | Gauge | `device` | Physical link state (1 = up, 0 = down) |

### 17. netstat (Network Protocol Statistics)
**Source**: `/proc/net/netstat`

| Metric | Type | Description |
|---|---|---|
| `node_netstat_<prefix>_<key>` | Counter | e.g. `TcpExt_syncookiesfailed`, `IpExt_inoctets` |

### 18. os (Operating System Information)
**Source**: `/etc/os-release`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_os_info` | Gauge | `id`, `id_like`, `name`, `pretty_name`, `variant`, `variant_id`, `version`, `version_id`, `build_id` | Constant value 1 |

### 19. softnet (Softnet Statistics)
**Source**: `/proc/net/softnet_stat`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_softnet_processed_total` | Counter | `cpu` | Packets processed |
| `node_softnet_dropped_total` | Counter | `cpu` | Packets dropped |
| `node_softnet_times_squeezed_total` | Counter | `cpu` | Times squeezed events |

### 20. thermal_zone (Thermal Zone Temperature)
**Source**: `/sys/class/thermal`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_thermal_zone_temp` | Gauge | `zone`, `type` | Zone temperature (Celsius) |

### 21. time (System Time)

| Metric | Type | Description |
|---|---|---|
| `node_time_seconds` | Gauge | System time in seconds since epoch |

### 22. cpufreq (CPU Frequency)
**Source**: `/sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_cpufreq_scaling_frequency_hertz` | Gauge | `cpu` | Current scaled CPU frequency (Hz) |

### 23. rapl (Intel RAPL Energy)
**Source**: `/sys/class/powercap/intel-rapl`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_rapl_joules_total` | Counter | `index`, `package`, `domain` | RAPL energy consumption (joules) |

### 24. selinux (SELinux Status)
**Source**: `/sys/fs/selinux/enforce`

| Metric | Type | Description |
|---|---|---|
| `node_selinux_enabled` | Gauge | Whether SELinux is enabled (1 = enabled, 0 = not) |

### 25. uname (System Information)
**Source**: `/proc/sys/kernel/*`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_uname_info` | Gauge | `sysname`, `release`, `version`, `machine` | Constant value 1 |

### 26. powersupplyclass (Power Supply)
**Source**: `/sys/class/power_supply`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_powersupply_present` | Gauge | `powersupply` | Whether the power supply is present (1/0) |
| `node_powersupply_status` | Gauge | `powersupply`, `status` | Current status (Battery type only) |

### 27. xfs (XFS Filesystem Statistics)
**Source**: `/sys/fs/xfs/*/stats`

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_xfs_<stat>_total` | Gauge | `device` | Per-statistic for each XFS device |

### 28-31. Stub Collectors (no metrics yet)

| Collector | Description |
|---|---|
| `sockstat` | `/proc/net/sockstat` parsing — stub |
| `timex` | Requires `adjtimex` syscall — stub |
| `schedstat` | `/proc/schedstat` parsing — stub |
| `mdadm` | `/proc/mdstat` parsing — stub |

---

## BPF Metrics: How They Work

BPF metrics (`node_ebpf_traffic_*`) are **not** read from `/proc`. They are collected directly in the kernel by eBPF programs.

### Architecture

1. **Program Load**: `ebpf_traffic.go:100` calls `bpf.LoadObjects()` to load pre-compiled BPF ELF objects (`traffic_bpfel.o` / `traffic_bpfeb.o`).
2. **Attachment Points**: Two `cgroup_skb` programs attach to the root cgroup v2:
   - `BPF_CGROUP_INET_EGRESS` — counts outbound traffic
   - `BPF_CGROUP_INET_INGRESS` — counts inbound traffic
3. **Data Store**: Counters are written to an **LRU hash BPF map** (max 8192 entries). Key = `(remote_ipv4, direction)`, Value = `(bytes, packets)`.
4. **Scrape**: Go code iterates the BPF map via `c.objs.Flows.Iterate()` and converts entries to Prometheus metrics.

### Prerequisites

- **Linux kernel**: must support BPF (4.1+ for TC mode, 4.10+ for cgroup_skb mode)
- **cgroup v2 mounted**: only required for cgroup_skb mode; TC mode works on cgroup v1
- **Privileges**: `CAP_BPF` + `CAP_NET_ADMIN` (for TC mode) or root
- **IPv4 only**: IPv6 and loopback (127.0.0.0/8) are skipped

### Mode Selection

At startup the collector tries modes in order:

1. **cgroup_skb** — checks `/sys/fs/cgroup/cgroup.controllers`, loads cgroup_skb programs, attaches to root cgroup v2
2. **TC** — loads TC BPF programs, creates `clsact` qdisc on each non-loopback interface, attaches ingress/egress filters
3. If both fail, the collector is skipped with a log message

The active mode is logged at startup:
```
exporter: ebpf_traffic using cgroup_skb mode
exporter: ebpf_traffic using TC mode
```

### Accessing BPF Metrics

#### Via bbx-server /metrics endpoint

```bash
./bbx-server
# Metrics are exposed on the server's /metrics endpoint
curl http://localhost:8080/metrics | grep ebpf_traffic
```

#### Directly from code

```go
import "github.com/casuallc/vigil/exporter"

exp, err := exporter.NewNodeExporter()
if err != nil {
    // handle
}
registry := exp.Registry()
// Pass registry to promhttp.HandlerFor or call Gather()
```

### Troubleshooting

If metrics are missing entirely, check the collector was not skipped:

```bash
journalctl -u bbx-server | grep ebpf_traffic
# or
cat logs/app.log | grep ebpf_traffic
```

Expected messages:
- `exporter: ebpf_traffic using cgroup_skb mode` — running on cgroup v2
- `exporter: ebpf_traffic using TC mode` — fell back to TC (cgroup v1)
- `exporter: skipping collector ebpf_traffic: ...` — both modes failed

If both modes fail, diagnose:

```bash
# Kernel BPF support
cat /proc/version
sysctl kernel.unprivileged_bpf_disabled

# cgroup state
mount | grep cgroup
ls /sys/fs/cgroup/

# TC / netlink (for TC mode)
ip link show
# Must have CAP_NET_ADMIN or root for TC filter manipulation
```

Common scenarios:

| Scenario | cgroup_skb | TC | Fix |
|---|---|---|---|
| cgroup v2 mounted | ✅ | — | Normal |
| cgroup v1 only | ❌ | ✅ | Normal — TC fallback activates |
| cgroup v1 + no `CAP_NET_ADMIN` | ❌ | ❌ | Run as root or add `CAP_NET_ADMIN` |
| Kernel < 4.1 | ❌ | ❌ | Upgrade kernel |

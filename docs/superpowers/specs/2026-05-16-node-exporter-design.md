# Node Exporter Integration Design

## Overview

Integrate `node_exporter`-style Linux system metrics collection directly into `bbx-server`. A new `exporter/` package re-implements ~30 of node_exporter's default-enabled collectors (referencing upstream code but rewriting in vigil's style), exposes them at `GET /metrics` in Prometheus text exposition format, and powers a redesigned `GET /api/resources/system` JSON endpoint. The existing `proc.ResourceMonitor` (system-level periodic collection + cache) is removed entirely.

## Motivation

Vigil currently has a simple `proc.ResourceMonitor` that collects flat CPU/memory numbers via `gopsutil` and caches them with a 5s TTL. This covers only a tiny slice of what operators expect on a Linux host. By adopting node_exporter's metric set and exposition format we get:

- Industry-standard metric names and labels — Grafana's "Node Exporter Full" dashboard and similar work out of the box
- A `/metrics` endpoint that any Prometheus server can scrape with zero adapter code
- A consistent collector pattern that makes adding new metrics trivial

We deliberately rewrite rather than depend on `github.com/prometheus/node_exporter` because that module is not designed as an importable library (`main` package, internal types, vendored build tooling).

## Scope

### In scope

- Linux-only system metrics collection
- ~30 default-enabled node_exporter collectors (list below)
- `GET /metrics` Prometheus text exposition (auth required, same as other endpoints)
- `GET /api/resources/system` redesigned JSON endpoint backed by the same collectors
- Complete removal of `proc.ResourceMonitor`

### Out of scope

- Cross-platform (Windows/macOS): `exporter` package is Linux-only via build tag; non-Linux builds return 501 from `/metrics` and `/api/resources/system`
- Per-collector enable/disable configuration (all default collectors always on)
- `textfile` collector (custom user metrics from a directory)
- Public/unauthenticated `/metrics` endpoint (uses existing `BasicAuthMiddleware`)
- Per-process metrics in node_exporter style (per-process is handled separately via `GET /api/resources/process/{pid}`)
- Backwards-compatible JSON format for `/api/resources/system` — this is an intentional breaking change

## Architecture

### Package layout

```
exporter/
  exporter.go           # NodeExporter struct, public API
  collector.go          # Collector interface, factory registration
  json.go               # GatherJSON: registry -> grouped JSON
  exporter_linux.go     # //go:build linux: real implementation
  exporter_stub.go      # //go:build !linux: stub returning ErrUnsupported
  collectors/
    cpu.go              # /proc/stat CPU times
    cpufreq.go          # /sys/devices/system/cpu/cpu*/cpufreq
    loadavg.go          # /proc/loadavg
    meminfo.go          # /proc/meminfo
    vmstat.go           # /proc/vmstat
    stat.go             # /proc/stat (non-CPU: intr, ctxt, processes, btime)
    uname.go            # uname() syscall
    time.go             # node_time_seconds
    timex.go            # adjtimex()
    os.go               # /etc/os-release
    diskstats.go        # /proc/diskstats
    filesystem.go       # /proc/mounts + statfs
    filefd.go           # /proc/sys/fs/file-nr
    xfs.go              # /sys/fs/xfs/*
    mdadm.go            # /proc/mdstat
    netdev.go           # /proc/net/dev
    netstat.go          # /proc/net/netstat + /proc/net/snmp
    netclass.go         # /sys/class/net/*
    sockstat.go         # /proc/net/sockstat
    softnet.go          # /proc/net/softnet_stat
    udp_queues.go       # /proc/net/udp{,6}
    conntrack.go        # /proc/sys/net/netfilter/nf_conntrack_count
    hwmon.go            # /sys/class/hwmon
    thermal_zone.go     # /sys/class/thermal/thermal_zone*
    powersupplyclass.go # /sys/class/power_supply/*
    rapl.go             # /sys/class/powercap/intel-rapl
    entropy.go          # /proc/sys/kernel/random/entropy_avail
    pressure.go         # /proc/pressure/{cpu,memory,io}
    schedstat.go        # /proc/schedstat
    selinux.go          # /sys/fs/selinux/enforce
  testdata/proc/        # fixture /proc tree borrowed from node_exporter
```

### Collector interface

Each collector lives in its own file under `exporter/collectors/`, defines a `XxxCollector` type, and registers itself via package `init()`:

```go
// exporter/collector.go
type Collector interface {
    // Name returns the collector identifier (e.g. "cpu", "meminfo").
    Name() string

    // Update emits all metrics this collector produces. Errors are logged
    // but should not propagate — collectors must isolate failures.
    Update(ch chan<- prometheus.Metric) error
}

// Registered factories. Each collectors/*.go calls register("name", factory)
// from its init().
var factories = map[string]func() (Collector, error){}

func register(name string, factory func() (Collector, error)) {
    factories[name] = factory
}
```

The `Collector` interface differs from `prometheus.Collector` (which has both `Describe` and `Collect`); we wrap each of our collectors in a small adapter that implements `prometheus.Collector` and forwards `Collect` to `Update`. This matches node_exporter's `NodeCollector` pattern.

### NodeExporter public API

```go
// exporter/exporter.go (Linux build)
type NodeExporter struct {
    registry   *prometheus.Registry
    collectors map[string]Collector
}

func NewNodeExporter() (*NodeExporter, error)

// Registry returns the underlying prometheus.Registry. The /metrics handler
// hands this to promhttp.HandlerFor.
func (n *NodeExporter) Registry() *prometheus.Registry

// GatherJSON returns the current metric snapshot as a JSON-friendly structure,
// grouped by collector name. Used by /api/resources/system.
func (n *NodeExporter) GatherJSON() (map[string]CollectorMetrics, error)

type CollectorMetrics map[string][]MetricSample

type MetricSample struct {
    Labels map[string]string `json:"labels,omitempty"`
    Value  float64           `json:"value"`
}
```

On non-Linux platforms, `exporter_stub.go` provides:

```go
// //go:build !linux
type NodeExporter struct{}

var ErrUnsupported = errors.New("exporter is only supported on Linux")

func NewNodeExporter() (*NodeExporter, error) { return &NodeExporter{}, nil }
func (n *NodeExporter) Registry() *prometheus.Registry { return prometheus.NewRegistry() }
func (n *NodeExporter) GatherJSON() (map[string]CollectorMetrics, error) {
    return nil, ErrUnsupported
}
```

### JSON output shape

`GatherJSON` walks `registry.Gather()` and groups metric families by the prefix-implicit collector. We tag each registered metric with the collector name via a registry wrapper so the grouping is unambiguous (no string-splitting `node_cpu_seconds_total` → `cpu`).

Example output:

```json
{
  "cpu": {
    "node_cpu_seconds_total": [
      {"labels": {"cpu":"0","mode":"user"}, "value": 1234.5},
      {"labels": {"cpu":"0","mode":"system"}, "value": 87.2},
      {"labels": {"cpu":"0","mode":"idle"}, "value": 9876.5}
    ]
  },
  "meminfo": {
    "node_memory_MemTotal_bytes":     [{"value": 16777216000}],
    "node_memory_MemAvailable_bytes": [{"value": 8388608000}]
  },
  "loadavg": {
    "node_load1":  [{"value": 0.42}],
    "node_load5":  [{"value": 0.51}],
    "node_load15": [{"value": 0.48}]
  },
  "scrape": {
    "node_scrape_collector_success_total": [
      {"labels": {"collector":"cpu"},     "value": 1},
      {"labels": {"collector":"meminfo"}, "value": 1},
      {"labels": {"collector":"xfs"},     "value": 0}
    ]
  }
}
```

## Collector list

All collectors are implemented and registered. None are runtime-toggleable.

| Group           | Collectors                                                                 |
| --------------- | -------------------------------------------------------------------------- |
| Core system     | cpu, cpufreq, loadavg, meminfo, vmstat, stat, uname, time, timex, os       |
| Disk/filesystem | diskstats, filesystem, filefd, xfs, mdadm                                  |
| Network         | netdev, netstat, netclass, sockstat, softnet, udp_queues, conntrack        |
| Hardware/power  | hwmon, thermal_zone, powersupplyclass, rapl                                |
| Kernel          | entropy, pressure, schedstat, selinux                                      |

**~30 collectors total.** Each collector returns 1–N `prometheus.Metric` values per `Update()` call. Metric names and labels follow node_exporter conventions exactly so existing Grafana dashboards render correctly.

`prometheus/procfs` is used to parse most `/proc` files (cpu times, meminfo, diskstats, netdev, netstat, sockstat, etc.). For sysfs paths (`/sys/class/hwmon`, `/sys/class/thermal`) we use `os.ReadDir` + `os.ReadFile` directly.

## HTTP integration

### Routes (`api/routes.go`)

```go
// New
r.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

// Existing routes preserved; handler bodies change
r.HandleFunc("/api/resources/system", s.handleGetSystemResources).Methods("GET")
r.HandleFunc("/api/resources/process/{pid}", s.handleGetProcessResources).Methods("GET")
```

`/metrics` flows through the standard `LoggingMiddleware` and `BasicAuthMiddleware`. No middleware bypass.

### Handlers

```go
// api/handlers_core.go
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
    promhttp.HandlerFor(s.exporter.Registry(), promhttp.HandlerOpts{
        ErrorHandling: promhttp.ContinueOnError,
    }).ServeHTTP(w, r)
}

// api/handlers_process.go (replaced)
func (s *Server) handleGetSystemResources(w http.ResponseWriter, r *http.Request) {
    data, err := s.exporter.GatherJSON()
    if err != nil {
        if errors.Is(err, exporter.ErrUnsupported) {
            writeError(w, http.StatusNotImplemented, err.Error())
            return
        }
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, data)
}

// api/handlers_process.go (cache removed; direct collection)
func (s *Server) handleGetProcessResources(w http.ResponseWriter, r *http.Request) {
    pid, err := strconv.Atoi(mux.Vars(r)["pid"])
    if err != nil {
        writeError(w, http.StatusBadRequest, "Invalid PID")
        return
    }
    resources, err := proc.GetUnixProcessResourceUsage(pid)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resources)
}
```

### Server struct (`api/server.go`)

```go
type Server struct {
    // ...existing fields except resourceMonitor
    exporter *exporter.NodeExporter   // NEW: replaces resourceMonitor
    // ...
}

// NewServerWithManager: replace the resourceMonitor block with:
exp, err := exporter.NewNodeExporter()
if err != nil {
    log.Printf("Warning: failed to initialize node exporter: %v", err)
    // continue without it; handlers will return 500/501
}

// Drop resourceMonitor.Start() and resourceMonitor.Stop() calls.
```

The exporter is purely on-demand — no background goroutine, no cache. Each `/metrics` scrape and each `/api/resources/system` call triggers a fresh collection. Typical Prometheus scrape interval is 15s, and `/proc` reads are microsecond-cheap, so this is fine.

## Error handling

Collector failures are isolated:

1. Each `Update()` runs inside a `defer recover()` wrapper so a panic in one collector cannot crash the scrape
2. When `Update()` returns a non-nil error, the wrapper logs it and emits `node_scrape_collector_success{collector="X"} 0` instead of failing the whole scrape
3. Successful collectors emit `node_scrape_collector_success{collector="X"} 1` and `node_scrape_collector_duration_seconds{collector="X"} <duration>`
4. `/metrics` always returns HTTP 200 with whatever metrics succeeded (Prometheus standard behavior)
5. `/api/resources/system` likewise returns 200 with partial data on partial failures; only top-level errors (e.g. `ErrUnsupported`) become non-200

## Non-Linux behavior

`exporter/exporter_linux.go` and `exporter/exporter_stub.go` use build tags to provide platform-specific implementations.

- On Linux: full `NodeExporter` with all collectors
- On other platforms: `NodeExporter` exists but `GatherJSON()` returns `ErrUnsupported`, and `Registry()` returns an empty registry
- `handleMetrics` on non-Linux returns 200 with effectively empty output (just process/go runtime metrics from prometheus.DefaultRegisterer if any)
- `handleGetSystemResources` returns 501 Not Implemented on non-Linux

This keeps the API surface consistent across platforms while making it obvious the data only exists on Linux.

## Testing

### Per-collector unit tests

`exporter/collectors/<name>_test.go` for each collector:

- Use `prometheus/procfs`'s test fixtures path (`procfs.NewFS("./testdata/proc")`) — fixtures borrowed from upstream node_exporter's `collector/fixtures/proc/`
- Assert produced metrics by gathering into a `prometheus.Registry` and comparing against a golden file
- For sysfs-reading collectors (hwmon, thermal_zone, etc.), put fixture trees under `testdata/sys/`
- Build tag `//go:build linux` since collectors are Linux-only

### Integration test

`exporter/exporter_test.go`:

- Instantiate `NodeExporter`, scrape via `httptest.NewServer(promhttp.HandlerFor(...))`
- Assert presence of key metric names (`node_cpu_seconds_total`, `node_memory_MemTotal_bytes`, `node_load1`, `node_filesystem_size_bytes`, `node_network_receive_bytes_total`)
- Assert `node_scrape_collector_success` exists for every registered collector
- Verify `Content-Type: text/plain; version=0.0.4` header

### JSON test

`exporter/json_test.go`:

- Call `GatherJSON()` on a populated registry, assert shape: `map[collector]map[metric_name][]MetricSample`
- Verify a label-bearing metric (e.g. cpu) groups into multiple samples
- Verify a label-less metric (e.g. loadavg) has a single sample

## Files to modify

| File                                    | Change                                                                       |
| --------------------------------------- | ---------------------------------------------------------------------------- |
| `exporter/` (new package)               | All collector files, exporter.go, json.go, build-tag variants, tests         |
| `go.mod` / `go.sum`                     | Promote `prometheus/client_golang` and `prometheus/procfs` to direct deps    |
| `api/server.go`                         | Replace `resourceMonitor *proc.ResourceMonitor` with `exporter *exporter.NodeExporter`; drop `.Start()`/`.Stop()` calls |
| `api/routes.go`                         | Register `GET /metrics` route                                                |
| `api/handlers_core.go`                  | Add `handleMetrics`                                                          |
| `api/handlers_process.go`               | Rewrite `handleGetSystemResources` and `handleGetProcessResources` to bypass cache and call new APIs |
| `proc/resource_monitor.go`              | Delete                                                                       |
| `proc/resource_monitor_test.go`         | Delete                                                                       |
| `proc/manager.go`                       | No change (uses `MonitorProcess` which uses `GetUnixProcessResourceUsage` directly) |
| `docs/api/resources.md`                 | Update to reflect new JSON shape and `/metrics` endpoint                     |
| `docs/api/README.md`                    | Add `/metrics` to endpoint index                                             |

## Breaking changes

- `GET /api/resources/system` JSON response shape changes from flat `models.ResourceStats` to grouped collector metric families. Any frontend, CLI command, or external script consuming this endpoint must be updated. No legacy compatibility endpoint is retained.
- `proc.ResourceMonitor` and its `NewResourceMonitor` constructor are removed. External imports (none expected since vigil is the only consumer) would break.

## Open questions

None — all major decisions resolved in brainstorming.

## References

- node_exporter source: `https://github.com/prometheus/node_exporter` (specifically `collector/` package patterns)
- Prometheus text exposition format: `https://prometheus.io/docs/instrumenting/exposition_formats/`
- `prometheus/procfs`: `https://github.com/prometheus/procfs`

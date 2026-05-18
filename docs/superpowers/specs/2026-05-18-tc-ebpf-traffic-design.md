# TC eBPF Traffic Collector Design

## Problem

The `ebpf_traffic` collector requires cgroup v2 (via `cgroup_skb` BPF programs), but many systems (e.g., CentOS 7) run cgroup v1 only. On these systems the collector is skipped entirely, leaving a gap in per-remote-IP traffic visibility.

## Goal

Add a TC-based eBPF fallback so that `ebpf_traffic` works on both cgroup v2 and cgroup v1 systems, emitting the same Prometheus metric names.

## Metrics (unchanged)

| Metric | Type | Labels | Description |
|---|---|---|---|
| `node_ebpf_traffic_bytes_total` | Counter | `remote_ip`, `direction` | Bytes per remote IPv4 and direction |
| `node_ebpf_traffic_packets_total` | Counter | `remote_ip`, `direction` | Packets per remote IPv4 and direction |
| `node_ebpf_traffic_truncated_flows` | Gauge | — | Flows dropped by top-N limit |

## Architecture

```
+-------------------------------------------+
|  Go: exporter/ebpf_traffic.go             |
|  --------------------------------         |
|  1. Enumerate interfaces (skip lo, down)  |
|  2. Create clsact qdisc on each           |
|  3. Attach TC ingress/egress BPF programs |
|  4. Scrape: iterate BPF LRU map           |
|  5. Emit Prometheus metrics               |
+--------------------+----------------------+
                     |
                     v
+-------------------------------------------+
|  BPF: exporter/bpf/traffic.bpf.c          |
|  --------------------------------         |
|  tc_count_ingress: parse eth+ip hdr       |
|    -> map[remote_ip=src, ingress] += 1    |
|  tc_count_egress:  parse eth+ip hdr       |
|    -> map[remote_ip=dst, egress]  += 1    |
|  Filter: non-IPv4, loopback, malformed    |
+-------------------------------------------+
```

## Fallback Strategy

`newEBPFTrafficCollector` tries modes in order:

1. **cgroup v2** (existing): check `/sys/fs/cgroup/cgroup.controllers` → load cgroup_skb programs → attach to root cgroup
2. **TC** (new): enumerate interfaces → create clsact qdisc → attach TC programs
3. If both fail → return error, collector is skipped

The mode (cgroup vs TC) is logged at construction so operators know which path is active.

## BPF Program Design

### Map

```c
struct flow_key {
    __u32 remote_ipv4;   // network byte order
    __u8  direction;     // 0=ingress, 1=egress
};

struct flow_stats {
    __u64 bytes;
    __u64 packets;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 8192);
    __type(key, struct flow_key);
    __type(value, struct flow_stats);
} flows SEC(".maps");
```

### Programs

```c
SEC("tc/ingress")
int tc_count_ingress(struct __sk_buff *skb) {
    // Parse ethernet header
    // If ethertype != ETH_P_IP (0x0800) -> TC_ACT_OK
    // Parse IPv4 header, extract src IP
    // If src IP starts with 127 -> TC_ACT_OK
    // Lookup/update map entry (direction = 0)
    // Increment bytes by skb->len, packets by 1
    return TC_ACT_OK;
}

SEC("tc/egress")
int tc_count_egress(struct __sk_buff *skb) {
    // Same as ingress but extract dst IP, direction = 1
    return TC_ACT_OK;
}
```

### Header Parsing in TC

TC programs receive `struct __sk_buff` with the packet data accessible via `skb->data` and `skb->data_end`. The parsing boundary must be checked to pass the verifier:

```c
void *data = (void *)(long)skb->data;
void *data_end = (void *)(long)skb->data_end;

struct ethhdr *eth = data;
if ((void *)(eth + 1) > data_end)
    return TC_ACT_OK;

if (eth->h_proto != __constant_htons(ETH_P_IP))
    return TC_ACT_OK;

struct iphdr *ip = (void *)(eth + 1);
if ((void *)(ip + 1) > data_end)
    return TC_ACT_OK;
```

## Go Collector Design

### Interface Selection

```go
func eligibleInterfaces() ([]net.Interface, error) {
    all, err := net.Interfaces()
    // Filter: Flags&Up != 0, name != "lo"
}
```

### TC Attachment

Use `github.com/vishvananda/netlink` (or raw netlink) to:

1. Create `clsact` qdisc on each interface:
   ```go
   qdisc := &netlink.Clsact{
       QdiscAttrs: netlink.QdiscAttrs{
           LinkIndex: iface.Index,
           Handle:    netlink.MakeHandle(0xffff, 0),
           Parent:    netlink.HANDLE_INGRESS,
       },
   }
   netlink.QdiscAdd(qdisc)
   ```

2. Attach BPF program as TC filter:
   ```go
   filter := &netlink.BpfFilter{
       FilterAttrs: netlink.FilterAttrs{
           LinkIndex: iface.Index,
           Parent:    netlink.MakeHandle(0xffff, 0xfff2), // clsact ingress
           Handle:    1,
           Protocol:  syscall.ETH_P_ALL,
       },
       Fd:           progFD,
       Name:         "tc_count_ingress",
       DirectAction: true,
   }
   netlink.FilterAdd(filter)
   ```

3. Egress uses parent `netlink.MakeHandle(0xffff, 0xfff3)`.

### Scrape

Same as existing: iterate the BPF map, convert to `flowSample`, apply `topNByBytes`, emit metrics.

## Build Changes

- Add `github.com/vishvananda/netlink` dependency
- `go generate` (or `bpf2go`) to regenerate Go bindings from the updated C code
- The BPF ELF objects (`traffic_bpfel.o`, `traffic_bpfeb.o`) are rebuilt via existing build tooling

### Separate ELF Objects per Mode

To prevent a cgroup_skb verification failure from blocking the TC fallback (or vice versa), compile two separate BPF objects:
- `traffic_cgroup.bpf.c` → `traffic_cgroup_bpfel.o` (cgroup_skb programs only)
- `traffic_tc.bpf.c` → `traffic_tc_bpfel.o` (TC programs only, same map layout)

The Go loader tries each object independently. Both objects declare the same `flows` map layout so the Go struct definitions are compatible.

## Kernel Requirements

| Mode | Minimum Kernel | Required Features |
|---|---|---|
| cgroup_skb (existing) | 4.10+ | `BPF_PROG_TYPE_CGROUP_SKB`, cgroup v2 |
| TC (new fallback) | 4.1+ | `BPF_PROG_TYPE_SCHED_CLS`, `clsact` qdisc |

The TC path uses `bpf_skb_load_bytes()` for packet parsing, which is available since kernel 4.1. On kernels older than 4.1, neither path works and the collector is skipped.

## Testing Plan

1. **Unit tests**: `topNByBytes` and `ipToString` already tested; add TC-specific helpers if any
2. **Integration test**: Run `smoke_ebpf.sh` on a cgroup v1 system, verify metrics appear
3. **Manual verification**:
   ```bash
   curl -s http://localhost:8080/metrics | grep ebpf_traffic
   ```

## Open Questions

- Should we dynamically watch for interface changes (hotplug)? **Decision**: no, attach once at startup. A server restart picks up new interfaces.
- Should VLAN-tagged traffic be parsed? **Decision**: skip for first version; 802.1Q ethertype (0x8100) is passed through.
- Should the TC qdisc be cleaned up on process exit? **Decision**: detach filters on `Close()`, leave qdisc (idempotent recreation).

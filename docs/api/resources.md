# 资源监控 API

## 接口列表

| 接口路径 | 请求方法 | 功能描述 |
|---------|----------|----------|
| /api/resources/system | GET | 获取系统资源信息 |
| /api/resources/process/{pid} | GET | 获取进程资源信息 |

---

## GET /api/resources/system

**功能描述**：获取系统资源信息

**请求参数**：无

**响应格式**：
```json
{
  "cpu_usage": 3.810118689274953,
  "cpu_usage_human": "3.8%",
  "memory_usage": 11411128320,
  "memory_usage_human": "10.63GiB",
  "memory_total": 33724628992,
  "memory_used_percent": 33.83618637496915,
  "memory_available": 15649669120,
  "memory_pressure": {},
  "disk_io": 23794880850944,
  "disk_io_human": "21.64TiB",
  "network_io": 0,
  "disk_usages": [
    {
      "device": "/dev/dm-0",
      "mountpoint": "/",
      "fstype": "xfs",
      "total": 526731812864,
      "used": 178061602816,
      "free": 348670210048,
      "used_percent": 33.804983573675806,
      "inodes_total": 257318912,
      "inodes_used": 956437,
      "inodes_free": 256362475,
      "inodes_used_percent": 0.3716932395548136
    },
    {
      "device": "/dev/vda2",
      "mountpoint": "/boot",
      "fstype": "xfs",
      "total": 1063256064,
      "used": 212705280,
      "free": 850550784,
      "used_percent": 20.0050850591716,
      "inodes_total": 524288,
      "inodes_used": 70,
      "inodes_free": 524218,
      "inodes_used_percent": 0.0133514404296875
    },
    {
      "device": "/dev/vda1",
      "mountpoint": "/boot/efi",
      "fstype": "vfat",
      "total": 209489920,
      "used": 9424896,
      "free": 200065024,
      "used_percent": 4.498973506696647
    }
  ],
  "disk_io_devices": [
    {
      "device": "vda",
      "read_bytes": 226265508864,
      "write_bytes": 7705374139392,
      "read_count": 1730471,
      "write_count": 470693144,
      "read_time_ms": 1889098,
      "write_time_ms": 472766393,
      "busy_time_ms": 483262260,
      "utilization_percent": 0.9991085254360093,
      "avg_read_latency_ms": 1.0916669507896983,
      "avg_write_latency_ms": 1.0044046721870248,
      "write_throughput_bps": 286464.3964130125
    },
    {
      "device": "vda1",
      "read_bytes": 3324416,
      "write_bytes": 0,
      "read_count": 306,
      "read_time_ms": 810,
      "busy_time_ms": 150,
      "avg_read_latency_ms": 2.6470588235294117
    },
    {
      "device": "vda2",
      "read_bytes": 13260800,
      "write_bytes": 15415808,
      "read_count": 160,
      "write_count": 131,
      "read_time_ms": 107,
      "write_time_ms": 413,
      "busy_time_ms": 1090,
      "avg_read_latency_ms": 0.66875,
      "avg_write_latency_ms": 3.1526717557251906
    },
    {
      "device": "vda3",
      "read_bytes": 226246105600,
      "write_bytes": 7705358723584,
      "read_count": 1729968,
      "write_count": 470693013,
      "read_time_ms": 1888150,
      "write_time_ms": 472765979,
      "busy_time_ms": 483261300,
      "utilization_percent": 0.9991085254360093,
      "avg_read_latency_ms": 1.0914363733895656,
      "avg_write_latency_ms": 1.0044040721717702,
      "write_throughput_bps": 286464.3964130125
    },
    {
      "device": "sr0",
      "read_bytes": 1116160,
      "write_bytes": 0,
      "read_count": 38,
      "read_time_ms": 14,
      "busy_time_ms": 50,
      "avg_read_latency_ms": 0.3684210526315789
    },
    {
      "device": "dm-0",
      "read_bytes": 226242042368,
      "write_bytes": 7705358723584,
      "read_count": 1653347,
      "write_count": 480635377,
      "read_time_ms": 1534190,
      "write_time_ms": 433022320,
      "busy_time_ms": 483322930,
      "utilization_percent": 0.9991085254360093,
      "avg_read_latency_ms": 0.9279298296122955,
      "avg_write_latency_ms": 0.9009372608042541,
      "write_throughput_bps": 286464.3964130125
    },
    {
      "device": "dm-1",
      "read_bytes": 2490368,
      "write_bytes": 0,
      "read_count": 28,
      "read_time_ms": 20,
      "busy_time_ms": 40,
      "avg_read_latency_ms": 0.7142857142857143
    }
  ],
  "load": {
    "load1": 0.06,
    "load5": 0.13,
    "load15": 0.17
  },
  "fd": {
    "current_allocated": 3488,
    "in_use": 3488,
    "max": 3282483,
    "usage_percent": 0.1062610225247168
  },
  "net_rx_bytes_per_sec": 10604.537888977802,
  "net_tx_bytes_per_sec": 10084.00234722564,
  "net_rx_packets": 4579953555,
  "net_tx_packets": 4273632665,
  "net_rx_dropped": 102531,
  "tcp_state_counts": {
    "CLOSE_WAIT": 2,
    "ESTABLISHED": 29,
    "LISTEN": 29,
    "TIME_WAIT": 2
  },
  "system_uptime_seconds": 22069505
}

```

---

## GET /api/resources/process/{pid}

**功能描述**：获取进程资源信息

**请求参数**：
- `pid`：进程 ID（路径参数）

**响应格式**：
```json
{
  "pid": 1234,
  "cpu_usage": 5.2,
  "memory_usage": 20.5,
  "disk_usage": 10.1,
  "network_stats": {
    "rx_bytes": 512,
    "tx_bytes": 1024
  }
}
```

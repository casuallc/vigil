// SPDX-License-Identifier: Apache-2.0
//
// cgroup_skb programs that count IPv4 bytes and packets per remote IP and
// direction. Attach the egress program with BPF_CGROUP_INET_EGRESS and
// the ingress program with BPF_CGROUP_INET_INGRESS to the root cgroup v2
// hierarchy (typically /sys/fs/cgroup).
//
// Loopback traffic (127.0.0.0/8) is skipped to keep the LRU map focused on
// useful entries. IPv6 packets are skipped: this is an MVP and IPv6 support
// is tracked as future work.

#include <linux/bpf.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

#ifndef AF_INET
#define AF_INET 2
#endif

#define DIRECTION_INGRESS 0
#define DIRECTION_EGRESS  1

struct flow_key {
    __u8 remote_ipv4[4]; /* network byte order, first-octet-first */
    __u8 direction;
    __u8 _pad[3];
};

struct flow_stats {
    __u64 bytes;
    __u64 packets;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, struct flow_key);
    __type(value, struct flow_stats);
    __uint(max_entries, 8192);
} flows SEC(".maps");

static __always_inline int count(struct __sk_buff *skb, __u8 direction)
{
    if (skb->family != AF_INET) {
        return 1; /* pass non-IPv4 unchanged */
    }

    void *data     = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct iphdr *ip = data;
    if ((void *)(ip + 1) > data_end) {
        return 1; /* malformed / truncated, pass */
    }

    __be32 remote_be = (direction == DIRECTION_EGRESS) ? ip->daddr : ip->saddr;

    struct flow_key key = {};
    __builtin_memcpy(&key.remote_ipv4, &remote_be, sizeof(remote_be));
    key.direction = direction;

    /* Skip 127.0.0.0/8 — first octet equals 127. The remote_ipv4 array
     * preserves network byte order, so element [0] is always the leading
     * octet regardless of host endianness. */
    if (key.remote_ipv4[0] == 127) {
        return 1;
    }

    struct flow_stats *st = bpf_map_lookup_elem(&flows, &key);
    if (st) {
        __sync_fetch_and_add(&st->bytes, skb->len);
        __sync_fetch_and_add(&st->packets, 1);
    } else {
        struct flow_stats init = { .bytes = skb->len, .packets = 1 };
        bpf_map_update_elem(&flows, &key, &init, BPF_NOEXIST);
    }
    return 1; /* always allow the packet */
}

SEC("cgroup_skb/egress")
int count_egress(struct __sk_buff *skb)
{
    return count(skb, DIRECTION_EGRESS);
}

SEC("cgroup_skb/ingress")
int count_ingress(struct __sk_buff *skb)
{
    return count(skb, DIRECTION_INGRESS);
}

char LICENSE[] SEC("license") = "Apache-2.0";

// SPDX-License-Identifier: Apache-2.0
//
// TC (Traffic Control) BPF programs that count IPv4 bytes and packets per
// remote IP and direction. These attach to network interfaces via clsact
// qdisc and serve as a fallback when cgroup v2 is not available.
//
// Two programs are provided:
//   - tc_count_ingress : attached to TC ingress hook
//   - tc_count_egress  : attached to TC egress hook
//
// Loopback traffic (127.0.0.0/8) is skipped. IPv6 packets are skipped.

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>

#ifndef ETH_P_IP
#define ETH_P_IP 0x0800
#endif

#ifndef TC_ACT_OK
#define TC_ACT_OK 0
#endif

#define DIRECTION_INGRESS 0
#define DIRECTION_EGRESS  1

struct flow_key {
	__u8  remote_ipv4[4];
	__u32 ifindex;
	__u8  direction;
	__u8  _pad[3];
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

static __always_inline int count_tc(struct __sk_buff *skb, __u8 direction)
{
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

	__be32 remote_be = (direction == DIRECTION_EGRESS) ? ip->daddr : ip->saddr;

	struct flow_key key = {};
	__builtin_memcpy(&key.remote_ipv4, &remote_be, sizeof(remote_be));
	key.ifindex   = skb->ifindex;
	key.direction = direction;

	/* Skip 127.0.0.0/8 */
	if (key.remote_ipv4[0] == 127)
		return TC_ACT_OK;

	struct flow_stats *st = bpf_map_lookup_elem(&flows, &key);
	if (st) {
		__sync_fetch_and_add(&st->bytes, skb->len);
		__sync_fetch_and_add(&st->packets, 1);
	} else {
		struct flow_stats init = { .bytes = skb->len, .packets = 1 };
		bpf_map_update_elem(&flows, &key, &init, BPF_NOEXIST);
	}
	return TC_ACT_OK;
}

SEC("tc")
int tc_count_ingress(struct __sk_buff *skb)
{
	return count_tc(skb, DIRECTION_INGRESS);
}

SEC("tc")
int tc_count_egress(struct __sk_buff *skb)
{
	return count_tc(skb, DIRECTION_EGRESS);
}

char LICENSE[] SEC("license") = "Apache-2.0";

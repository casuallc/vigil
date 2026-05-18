#!/usr/bin/env bash
# Manual smoke test for the eBPF traffic collector (Task 9).
# Run from WSL as a user that can sudo. Cleans up after itself.
#
#   chmod +x smoke_ebpf.sh
#   ./smoke_ebpf.sh
#
# Expected: a handful of `node_ebpf_traffic_*` lines printed at the end.

set -uo pipefail

cd "$(dirname "$0")"
BIN=./bbx-server-linux
CONFIG=./conf/config.yaml
AUTH="bbx:Flzx3qL@ysyhl9t"
URL="http://127.0.0.1:57575/metrics"
LOG=/tmp/bbx-server-smoke.log

if [[ ! -x "$BIN" ]]; then
    echo "ERROR: $BIN not found. Build first:"
    echo "    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o bbx-server-linux ./cmd/bbx-server"
    exit 1
fi

echo "==> Starting bbx-server (sudo)..."
sudo "$BIN" -config "$CONFIG" > "$LOG" 2>&1 &
SERVER_PID=$!
trap 'echo "==> Stopping server"; sudo kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null' EXIT INT TERM

echo "==> Waiting for server to come up..."
for i in {1..20}; do
    if curl -s -u "$AUTH" "$URL" > /dev/null 2>&1; then
        echo "    Server is responding."
        break
    fi
    sleep 0.5
done

echo
echo "==> Checking that ebpf_traffic was NOT skipped:"
grep -E "skipping collector ebpf_traffic|cgroup_skb" "$LOG" || echo "    (no skip / no attach message — that's OK if startup just logged differently)"
echo

echo "==> Generating some external traffic..."
curl -s -o /dev/null https://www.google.com  || true
curl -s -o /dev/null https://1.1.1.1         || true
ping -c 3 1.1.1.1 > /dev/null 2>&1            || true
sleep 1

echo
echo "==> Scraping /metrics for node_ebpf_traffic_*:"
echo "----------------------------------------------"
curl -s -u "$AUTH" "$URL" | grep -E "^node_ebpf_traffic_" | head -40
echo "----------------------------------------------"
echo

echo "==> Checking JSON endpoint /api/resources/system:"
echo "----------------------------------------------"
curl -s -u "$AUTH" "http://127.0.0.1:57575/api/resources/system" \
    | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps({k: list(v.keys()) for k,v in d.items() if 'ebpf' in k}, indent=2))" 2>/dev/null \
    || curl -s -u "$AUTH" "http://127.0.0.1:57575/api/resources/system" | grep -o '"ebpf_traffic":{[^}]*}' | head -1
echo "----------------------------------------------"
echo

echo "==> Server log tail (last 20 lines):"
tail -20 "$LOG" || true

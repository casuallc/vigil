# License Feature — Multi-IP per Interface Enhancement

## Summary
Extend the `/api/license` endpoint to emit one `LicenseInfo` entry per valid IP address on each physical interface, instead of only one entry per interface.

## Problem
When a single network interface is bound to multiple IP addresses (e.g. both an IPv4 and an IPv6 address, or multiple IPv4 addresses), the current implementation silently drops all but one address. This causes users to miss IP addresses that they may need for license binding or inventory.

## Solution
In `api/handlers_license.go`, change `getLicenseCodes()` so that:
- After selecting all valid IP addresses on an interface (same filtering rules as before: non-loopback, non-multicast, non-unspecified, etc.),
- Emit a separate `LicenseInfo` entry for **each** valid IP, using the same `Interface` name and the same `Code` (derived from MAC).

The `LicenseInfo` struct, API route, and CLI behavior remain unchanged.

## Example
Interface `eth0` with MAC `00-1A-2B-3C-4D-5E` and IPs `192.168.1.10`, `10.0.0.5`:

```json
[
  { "code": "SZTY123456789", "interface": "eth0", "ip": "192.168.1.10" },
  { "code": "SZTY123456789", "interface": "eth0", "ip": "10.0.0.5" }
]
```

## Files Modified
- `api/handlers_license.go` — update `getLicenseCodes()` loop logic.

## Backward Compatibility
Fully backward-compatible. Response is still `[]LicenseInfo`; existing consumers will simply see more rows when interfaces have multiple IPs.

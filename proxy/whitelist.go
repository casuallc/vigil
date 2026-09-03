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

package proxy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Whitelist decides whether an upstream target may be contacted.
// Entry syntax is auto-detected from the string form:
//
//	"10.0.0.0/8"      -> CIDR (matches resolved IPs)
//	"*.internal.corp" -> domain suffix (also matches the bare domain)
//	"db01" / "db01:5432" / "1.2.3.4:8080" -> exact host or host:port
//
// The whitelist is deny-by-default: an empty list allows nothing.
// Well-known cloud metadata endpoints are always denied, even when
// allowPrivate is set.
type Whitelist struct {
	cidrs        []*net.IPNet
	suffixes     []string
	hosts        map[string]bool // exact host without port (any port allowed)
	hostPorts    map[string]bool // exact host:port
	allowPrivate bool
	cidrsOnly    bool // every entry parsed as a CIDR
}

// metadataIP is the link-local cloud metadata endpoint (AWS/Aliyun/Tencent).
var metadataIP = net.ParseIP("169.254.169.254")

// metadataHost is the GCP metadata hostname.
const metadataHost = "metadata.google.internal"

// ParseWhitelist parses entries into a Whitelist. allowPrivate controls
// whether loopback / RFC1918 / link-local / ULA targets are permitted.
func ParseWhitelist(entries []string, allowPrivate bool) (*Whitelist, error) {
	w := &Whitelist{
		hosts:        make(map[string]bool),
		hostPorts:    make(map[string]bool),
		allowPrivate: allowPrivate,
		cidrsOnly:    len(entries) > 0,
	}
	for _, raw := range entries {
		e := strings.TrimSpace(raw)
		if e == "" {
			continue
		}
		switch {
		case strings.Contains(e, "/"):
			_, cidr, err := net.ParseCIDR(e)
			if err != nil {
				return nil, fmt.Errorf("proxy: invalid whitelist CIDR %q: %w", e, err)
			}
			w.cidrs = append(w.cidrs, cidr)
		case strings.HasPrefix(e, "*."):
			suffix := strings.ToLower(strings.TrimPrefix(e, "*."))
			if suffix == "" || strings.Contains(suffix, "*") {
				return nil, fmt.Errorf("proxy: invalid whitelist suffix %q", raw)
			}
			w.suffixes = append(w.suffixes, suffix)
			w.cidrsOnly = false
		default:
			host := e
			if h, p, err := net.SplitHostPort(e); err == nil {
				if h == "" || p == "" {
					return nil, fmt.Errorf("proxy: invalid whitelist entry %q", raw)
				}
				w.hostPorts[strings.ToLower(h)+":"+p] = true
				w.cidrsOnly = false
				continue
			}
			if strings.Contains(e, ":") && net.ParseIP(e) == nil {
				// A colon that is neither host:port nor an IPv6 literal.
				return nil, fmt.Errorf("proxy: invalid whitelist entry %q", raw)
			}
			host = strings.ToLower(host)
			if host == "" {
				return nil, fmt.Errorf("proxy: invalid whitelist entry %q", raw)
			}
			if ip := net.ParseIP(host); ip != nil {
				// Exact IP entries behave like a /32 (or /128) CIDR so the
				// post-dial CIDR-only check stays consistent.
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				w.cidrs = append(w.cidrs, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
				continue
			}
			w.hosts[host] = true
			w.cidrsOnly = false
		}
	}
	return w, nil
}

// Allowed reports whether the target host[:port] may be contacted.
func (w *Whitelist) Allowed(hostport string) bool {
	host, port := splitHostPort(hostport)
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return w.allowedIP(ip, port)
	}
	h := strings.ToLower(host)
	if h == metadataHost {
		return false
	}
	if port != "" && w.hostPorts[h+":"+port] {
		return true
	}
	if w.hosts[h] {
		return true
	}
	for _, s := range w.suffixes {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

// allowedIP applies the IP-based rules: always-deny list, private-address
// guard, then exact host / host:port / CIDR matching.
func (w *Whitelist) allowedIP(ip net.IP, port string) bool {
	if isAlwaysDeniedIP(ip) {
		return false
	}
	if !w.allowPrivate && isPrivateIP(ip) {
		return false
	}
	if port != "" && w.hostPorts[ip.String()+":"+port] {
		return true
	}
	if w.hosts[ip.String()] {
		return true
	}
	for _, c := range w.cidrs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}

// postDialIPAllowed re-checks the actually-connected IP after dialing.
// It mitigates DNS rebinding: a whitelisted hostname that resolves to a
// denied address is rejected here. When the whitelist is purely CIDR-based
// the connected IP must additionally fall inside one of the CIDRs.
func (w *Whitelist) postDialIPAllowed(ip net.IP, port string) bool {
	if isAlwaysDeniedIP(ip) {
		return false
	}
	if !w.allowPrivate && isPrivateIP(ip) {
		return false
	}
	if w.cidrsOnly {
		for _, c := range w.cidrs {
			if c.Contains(ip) {
				return true
			}
		}
		return false
	}
	return true
}

// CheckingDialer wraps base (or a default dialer) so that every established
// connection's remote address is re-validated via postDialIPAllowed.
func (w *Whitelist) CheckingDialer(base *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if base == nil {
		base = &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := base.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		host, port, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		ip := net.ParseIP(host)
		if splitErr != nil || ip == nil || !w.postDialIPAllowed(ip, port) {
			conn.Close()
			return nil, fmt.Errorf("proxy: connection to %s rejected by whitelist policy", addr)
		}
		return conn, nil
	}
}

// isAlwaysDeniedIP reports whether ip is a well-known sensitive endpoint
// that is rejected regardless of configuration.
func isAlwaysDeniedIP(ip net.IP) bool {
	return ip.Equal(metadataIP)
}

// isPrivateIP covers loopback, RFC1918, link-local and ULA (fc00::/7).
func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// splitHostPort tolerates a bare host (returns an empty port).
func splitHostPort(hostport string) (host, port string) {
	if h, p, err := net.SplitHostPort(hostport); err == nil {
		return h, p
	}
	// Bare IPv6 literal without brackets parses fine as IP.
	return strings.Trim(hostport, "[]"), ""
}

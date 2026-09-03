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
	"net"
	"testing"
)

func TestWhitelistEmptyDeniesAll(t *testing.T) {
	wl, err := ParseWhitelist(nil, true)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	if wl.Allowed("example.com") || wl.Allowed("10.0.0.1:80") {
		t.Fatal("empty whitelist must deny everything")
	}
}

func TestWhitelistCIDR(t *testing.T) {
	wl, err := ParseWhitelist([]string{"10.0.0.0/8"}, true)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	cases := map[string]bool{
		"10.0.0.1":        true,
		"10.255.255.254":  true,
		"10.0.0.1:9000":   true,
		"11.0.0.1":        false,
		"192.168.1.1":     false,
		"10.0.0.0/8-host": false,
	}
	for target, want := range cases {
		if got := wl.Allowed(target); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", target, got, want)
		}
	}
}

func TestWhitelistSuffix(t *testing.T) {
	wl, err := ParseWhitelist([]string{"*.internal.corp"}, false)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	cases := map[string]bool{
		"db.internal.corp":      true,
		"a.b.internal.corp":     true,
		"internal.corp":         true, // bare domain matches its own suffix entry
		"evilinternal.corp":     false,
		"internal.corp.evil.io": false,
		"INTERNAL.corp":         true, // case-insensitive
	}
	for target, want := range cases {
		if got := wl.Allowed(target); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", target, got, want)
		}
	}
}

func TestWhitelistExactHostAndPort(t *testing.T) {
	wl, err := ParseWhitelist([]string{"db01", "web01:8443"}, false)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	cases := map[string]bool{
		"db01":       true,
		"db01:5432":  true, // host entry without port allows any port
		"db01.evil":  false,
		"web01":      false, // host:port entry requires the port
		"web01:8443": true,
		"web01:443":  false,
	}
	for target, want := range cases {
		if got := wl.Allowed(target); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", target, got, want)
		}
	}
}

func TestWhitelistPrivateGuard(t *testing.T) {
	strict, _ := ParseWhitelist([]string{"192.168.0.0/16", "127.0.0.0/8", "10.0.0.0/8"}, false)
	for _, target := range []string{"192.168.1.10", "127.0.0.1", "10.1.2.3", "169.254.1.1"} {
		if strict.Allowed(target) {
			t.Errorf("allow_private=false must deny %q even when whitelisted", target)
		}
	}
	open, _ := ParseWhitelist([]string{"192.168.0.0/16"}, true)
	if !open.Allowed("192.168.1.10") {
		t.Error("allow_private=true must permit whitelisted private targets")
	}
}

func TestWhitelistMetadataAlwaysDenied(t *testing.T) {
	wl, err := ParseWhitelist([]string{"0.0.0.0/0", "metadata.google.internal", "*.internal"}, true)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	if wl.Allowed("169.254.169.254") {
		t.Error("cloud metadata IP must always be denied")
	}
	if wl.Allowed("metadata.google.internal") {
		t.Error("GCP metadata host must always be denied")
	}
}

func TestWhitelistInvalidEntries(t *testing.T) {
	for _, entries := range [][]string{
		{"10.0.0.0/33"},
		{"*."},
		{"*.*.corp"},
		{"db01:"},
		{"not-an-ip:port:extra"},
	} {
		if _, err := ParseWhitelist(entries, false); err == nil {
			t.Errorf("ParseWhitelist(%v) should fail", entries)
		}
	}
}

func TestCheckingDialerRejectsResolvedIP(t *testing.T) {
	// CIDR-only whitelist: the connected IP must fall inside the CIDR,
	// even if the dialed hostname was whitelisted by name elsewhere.
	wl, err := ParseWhitelist([]string{"203.0.113.0/24"}, false)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	dial := wl.CheckingDialer(&net.Dialer{})
	// Dial a real local listener to get a connection to 127.0.0.1.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()
	conn, err := dial(context.Background(), "tcp", ln.Addr().String())
	if err == nil {
		conn.Close()
		t.Fatal("dialer must reject a connection whose IP is outside the CIDR whitelist")
	}
}

func TestCheckingDialerPermitsWhitelistedIP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	wl, err := ParseWhitelist([]string{"127.0.0.0/8"}, true)
	if err != nil {
		t.Fatalf("ParseWhitelist: %v", err)
	}
	dial := wl.CheckingDialer(&net.Dialer{})
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()
	conn, err := dial(context.Background(), "tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("dialer must permit whitelisted IPs: %v", err)
	}
	conn.Close()
}

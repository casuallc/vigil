//go:build linux

/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package exporter

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func TestIPToString(t *testing.T) {
	cases := []struct {
		name string
		in   [4]byte
		want string
	}{
		{"loopback", [4]byte{127, 0, 0, 1}, "127.0.0.1"},
		{"public", [4]byte{8, 8, 8, 8}, "8.8.8.8"},
		{"private", [4]byte{10, 0, 0, 5}, "10.0.0.5"},
		{"zero", [4]byte{0, 0, 0, 0}, "0.0.0.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ipToString(tc.in)
			if got != tc.want {
				t.Fatalf("ipToString(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTopNByBytes(t *testing.T) {
	samples := []flowSample{
		{remoteIP: "1.1.1.1", direction: "ingress", bytes: 100, packets: 1},
		{remoteIP: "2.2.2.2", direction: "ingress", bytes: 500, packets: 5},
		{remoteIP: "3.3.3.3", direction: "egress", bytes: 50, packets: 1},
		{remoteIP: "4.4.4.4", direction: "egress", bytes: 200, packets: 2},
	}

	t.Run("under cap", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 10)
		if truncated != 0 {
			t.Fatalf("truncated = %d, want 0", truncated)
		}
		if len(kept) != 4 {
			t.Fatalf("len(kept) = %d, want 4", len(kept))
		}
		// Verify descending byte order.
		for i := 1; i < len(kept); i++ {
			if kept[i-1].bytes < kept[i].bytes {
				t.Fatalf("kept not sorted descending: %v", kept)
			}
		}
	})

	t.Run("over cap", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 2)
		if truncated != 2 {
			t.Fatalf("truncated = %d, want 2", truncated)
		}
		if len(kept) != 2 {
			t.Fatalf("len(kept) = %d, want 2", len(kept))
		}
		// Top two by bytes are 500 then 200.
		if kept[0].remoteIP != "2.2.2.2" || kept[1].remoteIP != "4.4.4.4" {
			t.Fatalf("unexpected top-N: %v", kept)
		}
	})

	t.Run("cap zero keeps nothing", func(t *testing.T) {
		kept, truncated := topNByBytes(append([]flowSample(nil), samples...), 0)
		if len(kept) != 0 {
			t.Fatalf("len(kept) = %d, want 0", len(kept))
		}
		if truncated != 4 {
			t.Fatalf("truncated = %d, want 4", truncated)
		}
	})
}

func TestEligibleInterfaces(t *testing.T) {
	ifaces, err := eligibleInterfaces()
	if err != nil {
		t.Fatalf("eligibleInterfaces() error: %v", err)
	}
	// Loopback should never appear.
	for _, iface := range ifaces {
		if iface.Name == "lo" {
			t.Fatalf("eligibleInterfaces() included loopback")
		}
		if iface.Flags&net.FlagUp == 0 {
			t.Fatalf("eligibleInterfaces() included down interface %s", iface.Name)
		}
	}
}

func TestIsEEXIST(t *testing.T) {
	if isEEXIST(nil) {
		t.Fatal("isEEXIST(nil) = true, want false")
	}
	if !isEEXIST(os.ErrExist) {
		t.Fatal("isEEXIST(os.ErrExist) = false, want true")
	}
	if !isEEXIST(fmt.Errorf("wrapped: %w", os.ErrExist)) {
		t.Fatal("isEEXIST(wrapped os.ErrExist) = false, want true")
	}
	if isEEXIST(os.ErrNotExist) {
		t.Fatal("isEEXIST(os.ErrNotExist) = true, want false")
	}
}

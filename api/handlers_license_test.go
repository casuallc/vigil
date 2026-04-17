package api

import (
	"testing"
)

func TestComputeLicenseCode(t *testing.T) {
	// MAC "00-1A-2B-3C-4D-5E" with salt "AAS" should produce a deterministic SZTY... code
	code, err := computeLicenseCode("00-1A-2B-3C-4D-5E")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code")
	}
	prefix := "SZTY"
	if len(code) <= len(prefix) || code[:len(prefix)] != prefix {
		t.Fatalf("expected code to start with %s, got %s", prefix, code)
	}
}

func TestIsVirtualInterface(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"eth0", false},
		{"docker0", true},
		{"veth1234", true},
		{"virbr0", true},
		{"dummy0", true},
		{"tun0", true},
		{"tap0", true},
		{"vEthernet", true},
		{"VirtualBox Host-Only", true},
		{"Hyper-V Virtual", true},
		{"lo", false},
	}
	for _, c := range cases {
		got := isVirtualInterface(c.name)
		if got != c.expected {
			t.Errorf("isVirtualInterface(%q) = %v, want %v", c.name, got, c.expected)
		}
	}
}

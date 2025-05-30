package asinfo

import (
	"net/netip"
	"testing"
)

func TestIpRangeToCIDRs(t *testing.T) {
	tests := []struct {
		name     string
		startIP  string
		endIP    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single IPv4 address",
			startIP:  "192.168.1.1",
			endIP:    "192.168.1.1",
			expected: []string{"192.168.1.1/32"},
			wantErr:  false,
		},
		{
			name:     "single IPv6 address",
			startIP:  "2001:db8::1",
			endIP:    "2001:db8::1",
			expected: []string{"2001:db8::1/128"},
			wantErr:  false,
		},
		{
			name:     "perfect /24 IPv4 range",
			startIP:  "192.168.1.0",
			endIP:    "192.168.1.255",
			expected: []string{"192.168.1.0/24"},
			wantErr:  false,
		},
		{
			name:     "perfect /16 IPv4 range",
			startIP:  "192.168.0.0",
			endIP:    "192.168.255.255",
			expected: []string{"192.168.0.0/16"},
			wantErr:  false,
		},
		{
			name:     "small IPv4 range requiring multiple CIDRs",
			startIP:  "192.168.1.100",
			endIP:    "192.168.1.103",
			expected: []string{"192.168.1.100/30"},
			wantErr:  false,
		},
		{
			name:     "IPv4 range crossing /24 boundary",
			startIP:  "192.168.1.254",
			endIP:    "192.168.2.1",
			expected: []string{"192.168.1.254/31", "192.168.2.0/31"},
			wantErr:  false,
		},
		{
			name:     "IPv6 /64 range",
			startIP:  "2001:db8::",
			endIP:    "2001:db8::ffff:ffff:ffff:ffff",
			expected: []string{"2001:db8::/64"},
			wantErr:  false,
		},
		{
			name:    "invalid range - start > end",
			startIP: "192.168.1.10",
			endIP:   "192.168.1.5",
			wantErr: true,
		},
		{
			name:    "mixed address families",
			startIP: "192.168.1.1",
			endIP:   "2001:db8::1",
			wantErr: true,
		},
		{
			name:     "two consecutive IPv4 addresses",
			startIP:  "192.168.1.10",
			endIP:    "192.168.1.11",
			expected: []string{"192.168.1.10/31"},
			wantErr:  false,
		},
		{
			name:     "large IPv4 range requiring multiple CIDRs",
			startIP:  "10.0.0.1",
			endIP:    "10.0.1.254",
			expected: []string{"10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/30", "10.0.0.8/29", "10.0.0.16/28", "10.0.0.32/27", "10.0.0.64/26", "10.0.0.128/25", "10.0.1.0/25", "10.0.1.128/26", "10.0.1.192/27", "10.0.1.224/28", "10.0.1.240/29", "10.0.1.248/30", "10.0.1.252/31", "10.0.1.254/32"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startIP, err := netip.ParseAddr(tt.startIP)
			if err != nil {
				t.Fatalf("Failed to parse start IP: %v", err)
			}

			endIP, err := netip.ParseAddr(tt.endIP)
			if err != nil {
				t.Fatalf("Failed to parse end IP: %v", err)
			}

			result, err := ipRangeToCIDRs(startIP, endIP)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d CIDRs, got %d", len(tt.expected), len(result))
				return
			}

			for i, prefix := range result {
				if prefix.String() != tt.expected[i] {
					t.Errorf("Expected CIDR %d to be %s, got %s", i, tt.expected[i], prefix.String())
				}
			}
		})
	}
}

func TestGetLastAddr(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:     "IPv4 /32 host route",
			prefix:   "192.168.1.1/32",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 /24 network",
			prefix:   "192.168.1.0/24",
			expected: "192.168.1.255",
		},
		{
			name:     "IPv4 /16 network",
			prefix:   "192.168.0.0/16",
			expected: "192.168.255.255",
		},
		{
			name:     "IPv4 /30 subnet",
			prefix:   "192.168.1.100/30",
			expected: "192.168.1.103",
		},
		{
			name:     "IPv4 /31 point-to-point",
			prefix:   "192.168.1.10/31",
			expected: "192.168.1.11",
		},
		{
			name:     "IPv4 /8 class A",
			prefix:   "10.0.0.0/8",
			expected: "10.255.255.255",
		},
		{
			name:     "IPv6 /128 host route",
			prefix:   "2001:db8::1/128",
			expected: "2001:db8::1",
		},
		{
			name:     "IPv6 /64 network",
			prefix:   "2001:db8::/64",
			expected: "2001:db8::ffff:ffff:ffff:ffff",
		},
		{
			name:     "IPv6 /48 network",
			prefix:   "2001:db8::/48",
			expected: "2001:db8::ffff:ffff:ffff:ffff:ffff",
		},
		{
			name:     "IPv6 /56 network",
			prefix:   "2001:db8:1::/56",
			expected: "2001:db8:1:ff:ffff:ffff:ffff:ffff",
		},
		{
			name:     "IPv6 /127 point-to-point",
			prefix:   "2001:db8::10/127",
			expected: "2001:db8::11",
		},
		{
			name:     "IPv6 /126 subnet",
			prefix:   "2001:db8::100/126",
			expected: "2001:db8::103",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := netip.ParsePrefix(tt.prefix)
			if err != nil {
				t.Fatalf("Failed to parse prefix: %v", err)
			}

			expected, err := netip.ParseAddr(tt.expected)
			if err != nil {
				t.Fatalf("Failed to parse expected address: %v", err)
			}

			result := getLastAddr(prefix)

			if result.Compare(expected) != 0 {
				t.Errorf("Expected last address %s, got %s", expected, result)
			}
		})
	}
}

package lib

import (
	"net"
	"net/netip"
	"slices"
	"testing"
)

func TestEntryMappedCIDR(t *testing.T) {
	for _, tc := range []struct{ cidr, want string }{
		{"::ffff:10.0.0.1/104", "10.0.0.0/8"},
		{"::ffff:0.0.0.0/96", "0.0.0.0/0"},
		{"::ffff:192.0.2.1/128", "192.0.2.1/32"},
		{"::ffff:c000:201/120", "192.0.2.0/24"},
		{"192.0.2.1/24", "192.0.2.0/24"},
		{"2001:db8::1/32", "2001:db8::/32"},
	} {
		_, network, err := net.ParseCIDR(tc.cidr)
		if err != nil {
			t.Fatal(err)
		}
		prefix := netip.MustParsePrefix(tc.cidr)
		for _, source := range []struct {
			name  string
			value any
		}{
			{"string", tc.cidr}, {"IPNet", network}, {"Prefix", prefix}, {"PrefixPointer", &prefix},
		} {
			t.Run(tc.cidr+"/"+source.name, func(t *testing.T) {
				entry := NewEntry("test")
				if err := entry.AddPrefix(source.value); err != nil {
					t.Fatal(err)
				}
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{tc.want}) {
					t.Fatalf("MarshalText = %v, %v; want %s", got, err, tc.want)
				}
			})
		}
	}
}

func TestEntryInvalidCIDRDoesNotPoisonList(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"mapped string below 96", "::ffff:192.0.2.1/95"},
		{"mapped IPNet below 96", &net.IPNet{IP: net.ParseIP("::ffff:192.0.2.1"), Mask: net.CIDRMask(95, 128)}},
		{"noncontiguous mask", &net.IPNet{IP: net.ParseIP("192.0.2.1"), Mask: net.IPMask{255, 0, 255, 0}}},
		{"IPv6 with IPv4 mask", &net.IPNet{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(24, 32)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := NewEntry("test")
			if err := entry.AddPrefix(tc.value); err == nil {
				t.Fatal("invalid input was accepted")
			}
			if err := entry.AddPrefix("192.168.0.0/16"); err != nil {
				t.Fatal(err)
			}
			got, err := entry.MarshalText()
			if err != nil || !slices.Equal(got, []string{"192.168.0.0/16"}) {
				t.Fatalf("valid list was poisoned: %v, %v", got, err)
			}
		})
	}
}

func TestEntryMappedCIDRMutationsInvalidateCache(t *testing.T) {
	entry := NewEntry("test")
	if err := entry.AddPrefix("10.0.0.0/8"); err != nil {
		t.Fatal(err)
	}
	if _, err := entry.MarshalText(); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("::ffff:192.0.2.0/120 # comment"); err != nil {
		t.Fatal(err)
	}
	got, err := entry.MarshalText()
	if err != nil || !slices.Equal(got, []string{"10.0.0.0/8", "192.0.2.0/24"}) {
		t.Fatalf("cached list after add = %v, %v", got, err)
	}
	if err := entry.RemovePrefix("::ffff:10.0.0.0/104"); err != nil {
		t.Fatal(err)
	}
	got, err = entry.MarshalText()
	if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) {
		t.Fatalf("cached list after remove = %v, %v", got, err)
	}
}

package lib

import (
	"errors"
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
			{"string", tc.cidr}, {"StringPointer", &tc.cidr},
			{"IPNet", network}, {"IPNetValue", *network},
			{"Prefix", prefix}, {"PrefixPointer", &prefix},
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

func TestEntryIPInputTypes(t *testing.T) {
	for _, tc := range []struct{ address, want string }{
		{"192.0.2.1", "192.0.2.1/32"},
		{"::ffff:192.0.2.1", "192.0.2.1/32"},
		{"2001:db8::1", "2001:db8::1/128"},
	} {
		for _, kind := range []string{"IP", "IPPointer", "Addr", "AddrPointer", "string", "StringPointer"} {
			t.Run(tc.address+"/"+kind, func(t *testing.T) {
				ip := net.ParseIP(tc.address)
				originalIP := slices.Clone(ip)
				addr := netip.MustParseAddr(tc.address)
				originalAddr := addr
				text := tc.address
				var source any
				switch kind {
				case "IP":
					source = ip
				case "IPPointer":
					source = &ip
				case "Addr":
					source = addr
				case "AddrPointer":
					source = &addr
				case "string":
					source = text
				case "StringPointer":
					source = &text
				}
				entry := NewEntry("test")
				if err := entry.AddPrefix(source); err != nil {
					t.Fatal(err)
				}
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{tc.want}) {
					t.Fatalf("MarshalText = %v, %v; want %s", got, err, tc.want)
				}
				if addr != originalAddr || !slices.Equal(ip, originalIP) || text != tc.address {
					t.Fatal("AddPrefix modified the caller's address")
				}
			})
		}
	}
}

func TestEntryNilAndInvalidInput(t *testing.T) {
	var emptyIP net.IP
	for _, tc := range []struct {
		name      string
		value     any
		wantError error
	}{
		{"nil interface", nil, ErrInvalidPrefixType},
		{"nil IP pointer", (*net.IP)(nil), ErrInvalidPrefixType},
		{"nil IPNet pointer", (*net.IPNet)(nil), ErrInvalidPrefixType},
		{"nil Addr pointer", (*netip.Addr)(nil), ErrInvalidPrefixType},
		{"nil Prefix pointer", (*netip.Prefix)(nil), ErrInvalidPrefixType},
		{"nil string pointer", (*string)(nil), ErrInvalidPrefixType},
		{"unsupported type", 123, ErrInvalidPrefixType},
		{"nil IP", emptyIP, ErrInvalidIP},
		{"pointer to nil IP", &emptyIP, ErrInvalidIP},
		{"invalid IP length", net.IP{192, 0, 2}, ErrInvalidIP},
		{"zero IPNet", net.IPNet{}, ErrInvalidIPNet},
		{"zero IPNet pointer", &net.IPNet{}, ErrInvalidIPNet},
		{"zero Addr", netip.Addr{}, ErrInvalidIPType},
		{"zero Prefix", netip.Prefix{}, ErrInvalidIPType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("AddPrefix panicked: %v", p)
				}
			}()
			entry := NewEntry("test")
			if err := entry.AddPrefix(tc.value); !errors.Is(err, tc.wantError) {
				t.Fatalf("AddPrefix error = %v; want %v", err, tc.wantError)
			}
			if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
				t.Fatal(err)
			}
			got, err := entry.MarshalText()
			if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) {
				t.Fatalf("invalid input poisoned the list: %v, %v", got, err)
			}
		})
	}
}

func TestEntryIPNetMaskWidths(t *testing.T) {
	for _, tc := range []struct {
		name    string
		network net.IPNet
		want    string
	}{
		{"four byte IPv4", net.IPNet{IP: net.IP{192, 0, 2, 1}, Mask: net.CIDRMask(24, 32)}, "192.0.2.0/24"},
		{"mapped IPv4 with 32 bit mask", net.IPNet{IP: net.ParseIP("192.0.2.1"), Mask: net.CIDRMask(24, 32)}, "192.0.2.0/24"},
		{"mapped IPv4 with 128 bit mask", net.IPNet{IP: net.ParseIP("192.0.2.1"), Mask: net.CIDRMask(120, 128)}, "192.0.2.0/24"},
		{"IPv6", net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(32, 128)}, "2001:db8::/32"},
	} {
		for _, source := range []struct {
			name  string
			value any
		}{{"value", tc.network}, {"pointer", &tc.network}} {
			t.Run(tc.name+"/"+source.name, func(t *testing.T) {
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
	for _, network := range []net.IPNet{
		{IP: net.ParseIP("192.0.2.1"), Mask: net.IPMask{255, 0, 255, 0}},
		{IP: net.ParseIP("192.0.2.1"), Mask: net.CIDRMask(95, 128)},
		{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(24, 32)},
		{IP: net.IP{192, 0, 2, 1}, Mask: net.CIDRMask(120, 128)},
	} {
		for _, source := range []any{network, &network} {
			entry := NewEntry("test")
			if err := entry.AddPrefix(source); err == nil {
				t.Fatalf("accepted invalid IPNet: %v", network)
			}
			if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
				t.Fatal(err)
			}
			if _, err := entry.MarshalText(); err != nil {
				t.Fatalf("invalid IPNet poisoned the list: %v", err)
			}
		}
	}
}

func TestEntryStringPointerComments(t *testing.T) {
	for _, line := range []string{"", " ", "# comment", "// comment", "/* comment"} {
		entry := NewEntry("test")
		if err := entry.AddPrefix(&line); err != nil {
			t.Fatalf("comment %q: %v", line, err)
		}
		if _, err := entry.MarshalText(); !errors.Is(err, ErrEmptyPrefix) {
			t.Fatalf("comment created a prefix: %v", err)
		}
	}
	for _, line := range []string{
		" 192.0.2.1 # comment", "192.0.2.1 // comment", "192.0.2.1 /* comment",
	} {
		entry := NewEntry("test")
		if err := entry.AddPrefix(&line); err != nil {
			t.Fatal(err)
		}
		got, err := entry.MarshalText()
		if err != nil || !slices.Equal(got, []string{"192.0.2.1/32"}) {
			t.Fatalf("IP with comment = %v, %v", got, err)
		}
	}
}

func TestEntryCIDRStringNormalization(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"192.0.2.129/024", "192.0.2.0/24"},
		{"192.0.2.1/000", "0.0.0.0/0"},
		{"2001:db8::1/032", "2001:db8::/32"},
		{"2001:db8::1/000", "::/0"},
		{" 	::FFFF:192.0.2.129/0120 # comment", "192.0.2.0/24"},
		{"::ffff:192.0.2.1/0096 // comment", "0.0.0.0/0"},
		{"::ffff:192.0.2.1/0128 /* comment", "192.0.2.1/32"},
	} {
		for _, pointer := range []bool{false, true} {
			t.Run(tc.input+map[bool]string{false: "/string", true: "/pointer"}[pointer], func(t *testing.T) {
				input := tc.input
				var source any = input
				if pointer {
					source = &input
				}
				entry := NewEntry("test")
				if err := entry.AddPrefix(source); err != nil {
					t.Fatal(err)
				}
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{tc.want}) {
					t.Fatalf("normalized CIDR = %v, %v; want %s", got, err, tc.want)
				}
				if err := entry.RemovePrefix(input); err != nil {
					t.Fatal(err)
				}
				if _, err := entry.MarshalText(); !errors.Is(err, ErrEmptyPrefix) {
					t.Fatalf("normalized removal left prefixes: %v", err)
				}
			})
		}
	}
}

func TestEntryInvalidCIDRPreservesCachedPrefixes(t *testing.T) {
	for _, input := range []string{
		"192.0.2.1/33", "2001:db8::1/129", "192.0.2.1/-1",
		"192.0.2.1/+24", "192.0.2.1/", "192.0.2.1/24/32",
		"fe80::1%eth0/64", "192.00.2.1/24", "::ffff:192.0.2.1/095 # comment",
	} {
		t.Run(input, func(t *testing.T) {
			entry := NewEntry("test")
			if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
				t.Fatal(err)
			}
			if _, err := entry.MarshalText(); err != nil {
				t.Fatal(err)
			}
			for _, action := range []struct {
				name string
				run  func() error
			}{
				{"add", func() error { return entry.AddPrefix(input) }},
				{"add pointer", func() error { return entry.AddPrefix(&input) }},
				{"remove", func() error { return entry.RemovePrefix(input) }},
			} {
				if err := action.run(); !errors.Is(err, ErrInvalidCIDR) {
					t.Fatalf("%s invalid CIDR error = %v", action.name, err)
				}
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) {
					t.Fatalf("%s changed cached prefixes: %v, %v", action.name, got, err)
				}
			}
			if err := entry.AddPrefix("198.51.100.0/24"); err != nil {
				t.Fatal(err)
			}
			got, err := entry.MarshalText()
			if err != nil || !slices.Equal(got, []string{"192.0.2.0/24", "198.51.100.0/24"}) {
				t.Fatalf("invalid CIDR poisoned later mutations: %v, %v", got, err)
			}
		})
	}
}

package maxmind

import (
	"github.com/xtls/geoip/lib"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCountryCSVFallbackAndWantedList(t *testing.T) {
	dir := t.TempDir()
	countries := filepath.Join(dir, "countries.csv")
	blocks := filepath.Join(dir, "blocks.csv")
	if err := os.WriteFile(countries, []byte("geoname_id,locale_code,continent_code,continent_name,country_iso_code\n1,en,NA,North America,US\n2,en,AS,Asia,CN\n3,en,AS,Asia,JP\n6255147,en,AS,Asia,\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocks, []byte("network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n192.0.2.0/24,6255147,2,\n198.51.100.0/24,1,2,\n203.0.113.0/24,,3,2\n10.0.0.0/24,999,998,3\n10.1.0.0/24,999,998,997\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		want     map[string]bool
		expected []string
	}{
		{"all", nil, []string{"CN", "US", "JP", "JP", ""}},
		{"only CN", map[string]bool{"CN": true}, []string{"CN", "", "", "", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &GeoLite2CountryCSVIn{Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, CountryCodeFile: countries, IPv4File: blocks, Want: tc.want}
			container, err := input.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			for i, ip := range []string{"192.0.2.1", "198.51.100.1", "203.0.113.1", "10.0.0.1", "10.1.0.1"} {
				names, found, err := container.Lookup(ip)
				if err != nil || found != (tc.expected[i] != "") {
					t.Fatalf("Lookup(%s) = %v, %v, %v", ip, names, found, err)
				}
				if found && !slices.Equal(names, []string{tc.expected[i]}) {
					t.Fatalf("Lookup(%s) = %v; want %s", ip, names, tc.expected[i])
				}
			}
		})
	}
	input := &GeoLite2CountryCSVIn{Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, CountryCodeFile: countries, IPv4File: blocks, Want: map[string]bool{"DE": true}}
	if _, err := input.Input(lib.NewContainer()); err == nil || !strings.Contains(err.Error(), "no entry is generated") {
		t.Fatalf("unmatched wantedList: %v", err)
	}
	if err := os.WriteFile(countries, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := input.Input(lib.NewContainer()); err == nil || !strings.Contains(err.Error(), "empty country code file") {
		t.Fatalf("empty CSV: %v", err)
	}
}

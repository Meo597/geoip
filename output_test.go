package main

import (
	"encoding/json"
	"github.com/xtls/geoip/lib"
	"github.com/xtls/geoip/plugin/xray"
	"google.golang.org/protobuf/proto"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOutputWantedAndExcludedLists(t *testing.T) {
	formats := []struct {
		kind, extension string
		aggregate       bool
	}{
		{kind: "text", extension: ".rules"},
		{kind: "clashRuleSet", extension: ".rules"},
		{kind: "clashRuleSetClassical", extension: ".rules"},
		{kind: "surgeRuleSet", extension: ".rules"},
		{kind: "singboxSRS", extension: ".srs"},
		{kind: "mihomoMRS", extension: ".mrs"},
		{kind: "xrayGeoIPDat", extension: ".dat"},
		{kind: "maxmindMMDB", aggregate: true},
		{kind: "dbipCountryMMDB", aggregate: true},
		{kind: "ipinfoCountryMMDB", aggregate: true},
		{kind: "stdout"},
	}
	cases := []struct {
		name                 string
		want, exclude, names []string
		stdout               string
	}{
		{name: "all wanted excluded", want: []string{" cn "}, exclude: []string{"CN"}},
		{name: "partial exclusion", want: []string{"cn", " us "}, exclude: []string{"CN"}, names: []string{"us"}, stdout: "198.51.100.0/24\n"},
		{name: "wanted CN", want: []string{"cn"}, names: []string{"cn"}, stdout: "192.0.2.0/24\n"},
		{name: "retain empty category", want: []string{"empty"}, names: []string{"empty"}},
		{name: "blank wanted entries", want: []string{"", " "}, names: []string{"cn", "empty", "us"}, stdout: "192.0.2.0/24\n198.51.100.0/24\n"},
		{name: "exclusion without wanted", exclude: []string{"cn"}, names: []string{"empty", "us"}, stdout: "198.51.100.0/24\n"},
	}
	for _, format := range formats {
		for _, tc := range cases {
			t.Run(format.kind+"/"+tc.name, func(t *testing.T) {
				container := lib.NewContainer()
				for _, item := range []struct{ name, cidr string }{{"cn", "192.0.2.0/24"}, {"us", "198.51.100.0/24"}, {"empty", ""}} {
					entry := lib.NewEntry(item.name)
					if item.cidr != "" {
						if err := entry.AddPrefix(item.cidr); err != nil {
							t.Fatal(err)
						}
					}
					if err := container.Add(entry); err != nil {
						t.Fatal(err)
					}
				}
				dir := t.TempDir()
				config, err := json.Marshal(map[string]any{"output": []any{map[string]any{"type": format.kind, "args": map[string]any{
					"outputDir": dir, "outputName": "test.mmdb", "outputExtension": ".rules", "oneFilePerList": true, "wantedList": tc.want, "excludedList": tc.exclude,
				}}}})
				if err != nil {
					t.Fatal(err)
				}
				instance, err := lib.NewInstance()
				if err != nil {
					t.Fatal(err)
				}
				if err := instance.InitConfigFromBytes(config); err != nil {
					t.Fatal(err)
				}
				capturePath := filepath.Join(t.TempDir(), "stdout.txt")
				capture, err := os.Create(capturePath)
				if err != nil {
					t.Fatal(err)
				}
				err = func() error {
					previous := os.Stdout
					os.Stdout = capture
					defer func() { os.Stdout = previous; capture.Close() }()
					return instance.RunOutput(container)
				}()
				if err != nil {
					t.Fatal(err)
				}
				output, err := os.ReadFile(capturePath)
				if err != nil {
					t.Fatal(err)
				}
				if format.kind == "stdout" {
					if string(output) != tc.stdout {
						t.Fatalf("stdout = %q; want %q", output, tc.stdout)
					}
				} else if len(output) != 0 {
					t.Fatalf("unexpected stdout: %q", output)
				}
				files, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				var got, want []string
				for _, file := range files {
					got = append(got, file.Name())
				}
				if format.kind != "stdout" {
					if format.aggregate {
						if len(tc.names) != 0 {
							want = []string{"test.mmdb"}
						}
					} else {
						for _, name := range tc.names {
							want = append(want, name+format.extension)
						}
					}
				}
				if !slices.Equal(got, want) {
					t.Fatalf("exported files = %v; want %v", got, want)
				}
				if format.kind == "xrayGeoIPDat" && tc.name == "retain empty category" {
					data, err := os.ReadFile(filepath.Join(dir, "empty.dat"))
					if err != nil {
						t.Fatal(err)
					}
					var lists xray.GeoIPList
					if err := proto.Unmarshal(data, &lists); err != nil {
						t.Fatal(err)
					}
					if len(lists.Entry) != 1 || lists.Entry[0].CountryCode != "EMPTY" || len(lists.Entry[0].Cidr) != 0 {
						t.Fatalf("empty Xray category was not preserved: %v", &lists)
					}
				}
			})
		}
	}
}

func TestGeoIPDatAggregateSelection(t *testing.T) {
	cn := &xray.GeoIP{CountryCode: "CN", Cidr: []*xray.CIDR{{Ip: netip.MustParseAddr("192.0.2.0").AsSlice(), Prefix: 24}}}
	us := &xray.GeoIP{CountryCode: "US", Cidr: []*xray.CIDR{{Ip: netip.MustParseAddr("2001:db8::").AsSlice(), Prefix: 32}}}
	empty := &xray.GeoIP{CountryCode: "EMPTY"}
	for _, tc := range []struct {
		name          string
		want, exclude []string
		onlyIPType    lib.IPType
		entries       []*xray.GeoIP
	}{
		{name: "partial exclusion", want: []string{" us ", "empty", " cn "}, exclude: []string{"US"}, entries: []*xray.GeoIP{cn, empty}},
		{name: "all wanted excluded", want: []string{"cn"}, exclude: []string{"CN"}},
		{name: "missing wanted category", want: []string{"missing"}},
		{name: "IPv6 filter retains empty categories", want: []string{"cn", "empty", "us"}, exclude: []string{"us"}, onlyIPType: lib.IPv6, entries: []*xray.GeoIP{{CountryCode: "CN"}, empty}},
		{name: "exclusion without wanted", exclude: []string{"cn"}, entries: []*xray.GeoIP{empty, us}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container := lib.NewContainer()
			for _, item := range []struct{ name, cidr string }{
				{"cn", "::ffff:192.0.2.1/120"}, {"us", "2001:db8::1/32"}, {"empty", ""},
			} {
				entry := lib.NewEntry(item.name)
				if item.cidr != "" {
					if err := entry.AddPrefix(item.cidr); err != nil {
						t.Fatal(err)
					}
				}
				if err := container.Add(entry); err != nil {
					t.Fatal(err)
				}
			}
			dir := t.TempDir()
			config, err := json.Marshal(map[string]any{"output": []any{map[string]any{
				"type": "xrayGeoIPDat", "args": map[string]any{
					"outputDir": dir, "outputName": "selection.dat",
					"wantedList": tc.want, "excludedList": tc.exclude, "onlyIPType": tc.onlyIPType,
				},
			}}})
			if err != nil {
				t.Fatal(err)
			}
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.InitConfigFromBytes(config); err != nil {
				t.Fatal(err)
			}
			if err := instance.RunOutput(container); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.entries) == 0 {
				if len(files) != 0 {
					t.Fatalf("empty selection exported %v", files)
				}
				return
			}
			if len(files) != 1 || files[0].Name() != "selection.dat" {
				t.Fatalf("aggregate output files = %v", files)
			}
			data, err := os.ReadFile(filepath.Join(dir, "selection.dat"))
			if err != nil {
				t.Fatal(err)
			}
			var got xray.GeoIPList
			if err := proto.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			want := &xray.GeoIPList{Entry: tc.entries}
			if !proto.Equal(&got, want) {
				t.Fatalf("aggregate DAT = %v; want %v", &got, want)
			}
		})
	}
}

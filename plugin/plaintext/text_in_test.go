package plaintext

import (
	"encoding/json"
	"github.com/xtls/geoip/lib"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTextInputWantedList(t *testing.T) {
	for _, format := range []struct{ kind, body string }{
		{TypeTextIn, "192.0.2.0/24\n"},
		{TypeJSONIn, "{\"prefixes\":[\"192.0.2.0/24\"]}"},
		{TypeClashRuleSetIPCIDRIn, "payload:\n  - '192.0.2.0/24'\n"},
		{TypeClashRuleSetClassicalIn, "payload:\n  - IP-CIDR,192.0.2.0/24\n"},
		{TypeSurgeRuleSetIn, "IP-CIDR,192.0.2.0/24\n"},
	} {
		for _, wanted := range []string{"CN", "US"} {
			t.Run(format.kind+"/"+wanted, func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "cn.txt")
				if err := os.WriteFile(file, []byte(format.body), 0600); err != nil {
					t.Fatal(err)
				}
				args, err := json.Marshal(map[string]any{"name": "cn", "uri": file, "wantedList": []string{wanted}, "jsonPath": []string{"prefixes"}})
				if err != nil {
					t.Fatal(err)
				}
				input, err := newTextIn(format.kind, "", lib.ActionAdd, args)
				if err != nil {
					t.Fatal(err)
				}
				container := lib.NewContainer()
				_, err = input.Input(container)
				if wanted == "US" {
					if err == nil || !strings.Contains(err.Error(), "no entry is generated") || container.Len() != 0 {
						t.Fatalf("excluded list was created: Len=%d err=%v", container.Len(), err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				entry, ok := container.GetEntry("CN")
				if !ok {
					t.Fatal("wanted list missing")
				}
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) {
					t.Fatalf("wanted data = %v, %v", got, err)
				}
			})
		}
	}
}

func TestTextInlineWantedList(t *testing.T) {
	for _, withURI := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline", true: "URI and inline"}[withURI], func(t *testing.T) {
			input := &TextIn{Type: TypeTextIn, Action: lib.ActionAdd, Name: "cn", IPOrCIDR: []string{"192.0.2.0/24"}, Want: map[string]bool{"US": true}}
			if withURI {
				input.URI = filepath.Join(t.TempDir(), "cn.txt")
				if err := os.WriteFile(input.URI, []byte("198.51.100.0/24\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			container := lib.NewContainer()
			if _, err := input.Input(container); err == nil || !strings.Contains(err.Error(), "no entry is generated") || container.Len() != 0 {
				t.Fatalf("inline list bypassed wantedList: Len=%d err=%v", container.Len(), err)
			}
		})
	}
}

func TestTextInputNormalizedName(t *testing.T) {
	for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
		for _, withURI := range []bool{false, true} {
			for _, wanted := range []string{" cn ", "US"} {
				name := string(action) + "/" + map[bool]string{false: "inline", true: "URI and inline"}[withURI] + "/" + strings.TrimSpace(wanted)
				t.Run(name, func(t *testing.T) {
					args := map[string]any{
						"name": " 	 cN \n ", "wantedList": []string{wanted},
						"ipOrCIDR": []string{"192.0.2.0/24"},
					}
					if withURI {
						file := filepath.Join(t.TempDir(), "cn.txt")
						if err := os.WriteFile(file, []byte("198.51.100.0/24\n"), 0600); err != nil {
							t.Fatal(err)
						}
						args["uri"] = file
					}
					config, err := json.Marshal(map[string]any{"input": []any{map[string]any{
						"type": TypeTextIn, "action": action, "args": args,
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
					container := lib.NewContainer()
					initial := []string{"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24"}
					if action == lib.ActionRemove {
						entry := lib.NewEntry("cn")
						for _, cidr := range initial {
							if err := entry.AddPrefix(cidr); err != nil {
								t.Fatal(err)
							}
						}
						if err := container.Add(entry); err != nil {
							t.Fatal(err)
						}
					}
					err = instance.RunInput(container)
					if wanted == "US" {
						if err == nil || !strings.Contains(err.Error(), "no entry is generated") {
							t.Fatalf("excluded input error = %v", err)
						}
						if action == lib.ActionAdd {
							if container.Len() != 0 {
								t.Fatalf("excluded input created %d categories", container.Len())
							}
							return
						}
					} else if err != nil {
						t.Fatal(err)
					}
					entry, found := container.GetEntry("CN")
					if !found || container.Len() != 1 {
						t.Fatalf("CN missing or duplicated: found=%v Len=%d", found, container.Len())
					}
					want := []string{"192.0.2.0/24"}
					if withURI {
						want = append(want, "198.51.100.0/24")
					}
					if action == lib.ActionRemove {
						want = []string{"203.0.113.0/24"}
						if !withURI {
							want = append([]string{"198.51.100.0/24"}, want...)
						}
						if wanted == "US" {
							want = initial
						}
					}
					got, err := entry.MarshalText()
					if err != nil || !slices.Equal(got, want) {
						t.Fatalf("CN prefixes = %v, %v; want %v", got, err, want)
					}
				})
			}
		}
	}
}

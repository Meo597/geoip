package special

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/xtls/geoip/lib"
)

func TestCutterRemovalDuringIteration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		onlyIPType lib.IPType
		keep       []string
	}{
		{name: "whole categories"},
		{name: "IPv4 only", onlyIPType: lib.IPv4, keep: []string{"2001:db8::/32"}},
		{name: "IPv6 only", onlyIPType: lib.IPv6, keep: []string{"192.0.2.0/24"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const count = 1000 // Exceed the old Loop channel's 300-entry buffer.
			container := lib.NewContainer()
			both := []string{"192.0.2.0/24", "2001:db8::/32"}
			var wanted []string
			for i := 0; i <= count; i++ {
				name := fmt.Sprintf("entry-%d", i)
				if i == count {
					name = "retained"
				} else {
					wanted = append(wanted, " "+name+" ")
				}
				entry := lib.NewEntry(name)
				for _, cidr := range both {
					if err := entry.AddPrefix(cidr); err != nil {
						t.Fatal(err)
					}
				}
				if err := container.Add(entry); err != nil {
					t.Fatal(err)
				}
			}
			if err := container.Add(lib.NewEntry("empty")); err != nil {
				t.Fatal(err)
			}
			wanted = append(wanted, "empty")
			args, err := json.Marshal(map[string]any{"wantedList": wanted, "onlyIPType": tc.onlyIPType})
			if err != nil {
				t.Fatal(err)
			}
			input, err := newCutter(lib.ActionRemove, args)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := input.Input(container); err != nil {
				t.Fatal(err)
			}
			wantLen := 1
			if tc.onlyIPType != "" {
				wantLen = count + 2
			}
			if container.Len() != wantLen {
				t.Fatalf("Len = %d; want %d", container.Len(), wantLen)
			}
			for i := 0; i < count; i++ {
				entry, found := container.GetEntry(fmt.Sprintf("entry-%d", i))
				if found != (tc.onlyIPType != "") {
					t.Fatalf("entry-%d membership = %v", i, found)
				}
				if found {
					got, err := entry.MarshalText()
					if err != nil || !slices.Equal(got, tc.keep) {
						t.Fatalf("entry-%d prefixes = %v, %v; want %v", i, got, err, tc.keep)
					}
				}
			}
			if entry, found := container.GetEntry("empty"); found != (tc.onlyIPType != "") {
				t.Fatalf("empty category membership = %v", found)
			} else if found {
				if _, err := entry.MarshalText(); !errors.Is(err, lib.ErrEmptyPrefix) {
					t.Fatalf("empty category changed: %v", err)
				}
			}
			entry, found := container.GetEntry("retained")
			if !found {
				t.Fatal("unselected category was removed")
			}
			got, err := entry.MarshalText()
			if err != nil || !slices.Equal(got, both) {
				t.Fatalf("unselected category changed: %v, %v", got, err)
			}
		})
	}
}

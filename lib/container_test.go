package lib

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestContainerLoopSnapshot(t *testing.T) {
	c := NewContainer()
	for i := 0; i < 1000; i++ {
		if err := c.Add(NewEntry(fmt.Sprint(i))); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := c.Loop()
	removed, _ := c.GetEntry("0")
	if err := c.Remove(removed, CaseRemoveEntry); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(NewEntry("later")); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for entry := range snapshot {
		seen[entry.GetName()] = true
	}
	if len(seen) != 1000 || !seen["0"] || seen["LATER"] {
		t.Fatalf("iteration did not retain its original entries: count=%d removed=%v later=%v", len(seen), seen["0"], seen["LATER"])
	}
	for entry := range c.Loop() {
		if err := c.Remove(entry, CaseRemoveEntry); err != nil {
			t.Fatal(err)
		}
	}
	if c.Len() != 0 {
		t.Fatalf("entries remain after removing during iteration: %d", c.Len())
	}
	for entry := range c.Loop() {
		t.Fatalf("empty container returned %s", entry.GetName())
	}
}

func TestContainerLookupMappedCIDR(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("cn")
	if err := entry.AddPrefix("1.1.1.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(NewEntry("empty")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		found bool
		err   error
	}{
		{"1.1.1.0/25", true, nil},
		{"::ffff:1.1.1.0/121", true, nil},
		{"::ffff:1.1.1.255/128", true, nil},
		{"::ffff:1.1.1.1", true, nil},
		{"::ffff:1.1.0.0/112", false, nil},
		{"::ffff:1.1.1.0/95", false, ErrInvalidCIDR},
	} {
		t.Run(tc.query, func(t *testing.T) {
			names, found, err := c.Lookup(tc.query, " cn ")
			if found != tc.found || !errors.Is(err, tc.err) {
				t.Fatalf("Lookup = %v, %v, %v", names, found, err)
			}
			if found && !slices.Equal(names, []string{"CN"}) {
				t.Fatalf("unexpected lists: %v", names)
			}
		})
	}
}

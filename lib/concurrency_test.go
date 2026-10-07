package lib

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"
)

func runConcurrent(t *testing.T, actions ...func()) {
	t.Helper()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, action := range actions {
		wg.Go(func() {
			<-start
			action()
		})
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent operations did not finish; possible deadlock")
	}
}

func TestContainerConcurrentAccess(t *testing.T) {
	c := NewContainer()
	base := NewEntry("base")
	if err := base.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(base); err != nil {
		t.Fatal(err)
	}

	var actions []func()
	for worker := 0; worker < 4; worker++ {
		actions = append(actions, func() {
			for i := 0; i < 100; i++ {
				entry := NewEntry(fmt.Sprintf("worker-%d-%d", worker, i))
				if err := entry.AddPrefix("10.0.0.0/8"); err != nil {
					t.Error(err)
					return
				}
				if err := c.Add(entry); err != nil {
					t.Error(err)
					return
				}
				if err := c.Remove(entry, CaseRemoveEntry); err != nil {
					t.Error(err)
					return
				}
			}
		})
		actions = append(actions, func() {
			for i := 0; i < 100; i++ {
				if c.Len() < 1 {
					t.Error("base entry disappeared")
					return
				}
				if entry, ok := c.GetEntry(" base "); !ok || entry != base {
					t.Error("GetEntry lost the base entry")
					return
				}
				seen := make(map[string]bool)
				for entry := range c.Loop() {
					if seen[entry.GetName()] {
						t.Errorf("duplicate snapshot entry %s", entry.GetName())
						return
					}
					seen[entry.GetName()] = true
				}
				if !seen["BASE"] {
					t.Error("snapshot lost the base entry")
					return
				}
				names, found, err := c.Lookup("192.0.2.1")
				if err != nil || !found || !slices.Equal(names, []string{"BASE"}) {
					t.Errorf("Lookup = %v, %v, %v", names, found, err)
					return
				}
			}
		})
	}
	runConcurrent(t, actions...)
	if c.Len() != 1 {
		t.Fatalf("Len = %d; want 1", c.Len())
	}
}

func TestContainerConcurrentPrefixUpdates(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("shared")
	if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}
	published, _ := c.GetEntry("shared")
	if published != entry {
		t.Fatal("Add changed the stored Entry pointer")
	}

	const workers, iterations = 4, 50
	var actions []func()
	for worker := 0; worker < workers; worker++ {
		actions = append(actions, func() {
			for i := 0; i < iterations; i++ {
				delta := NewEntry("shared")
				v4 := fmt.Sprintf("10.%d.%d.1", worker, i)
				v6 := fmt.Sprintf("2001:db9:%x::%x", worker, i)
				for _, prefix := range []string{v4, v6} {
					if err := delta.AddPrefix(prefix); err != nil {
						t.Error(err)
						return
					}
				}
				if err := c.Add(delta); err != nil {
					t.Error(err)
					return
				}
				if err := c.Remove(delta, CaseRemovePrefix); err != nil {
					t.Error(err)
					return
				}
				if err := c.Add(delta); err != nil {
					t.Error(err)
					return
				}
				// Mutate the Entry returned by GetEntry directly as well.
				direct := fmt.Sprintf("172.%d.%d.1", 16+worker, i)
				if err := published.AddPrefix(direct); err != nil {
					t.Error(err)
					return
				}
				if err := published.RemovePrefix(direct); err != nil {
					t.Error(err)
					return
				}
			}
		})
		actions = append(actions, func() {
			for i := 0; i < iterations; i++ {
				if _, err := published.MarshalText(); err != nil {
					t.Error(err)
					return
				}
				if _, err := published.MarshalPrefix(); err != nil {
					t.Error(err)
					return
				}
				if _, err := published.MarshalIPRange(); err != nil {
					t.Error(err)
					return
				}
				for _, query := range []string{"192.0.2.1", "2001:db8::1"} {
					names, found, err := c.Lookup(query, "shared")
					if err != nil || !found || !slices.Equal(names, []string{"SHARED"}) {
						t.Errorf("Lookup(%s) = %v, %v, %v", query, names, found, err)
						return
					}
				}
				// Exercise self-merging without recursively locking the Entry.
				if err := c.Add(published); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	runConcurrent(t, actions...)

	ipv4, err := entry.GetIPv4Set()
	if err != nil {
		t.Fatal(err)
	}
	ipv6, err := entry.GetIPv6Set()
	if err != nil {
		t.Fatal(err)
	}
	for worker := 0; worker < workers; worker++ {
		for i := 0; i < iterations; i++ {
			v4 := netip.MustParseAddr(fmt.Sprintf("10.%d.%d.1", worker, i))
			v6 := netip.MustParseAddr(fmt.Sprintf("2001:db9:%x::%x", worker, i))
			if !ipv4.Contains(v4) || !ipv6.Contains(v6) {
				t.Fatalf("concurrent merge lost %s or %s", v4, v6)
			}
			direct := netip.MustParseAddr(fmt.Sprintf("172.%d.%d.1", 16+worker, i))
			if ipv4.Contains(direct) {
				t.Fatalf("concurrent removal left %s", direct)
			}
		}
	}
}

func TestContainerSharedEntries(t *testing.T) {
	for _, sameEntry := range []bool{false, true} {
		t.Run(fmt.Sprintf("sameEntry=%v", sameEntry), func(t *testing.T) {
			a, b := NewContainer(), NewContainer()
			left, right := NewEntry("shared"), NewEntry("shared")
			if err := left.AddPrefix("192.0.2.0/24"); err != nil {
				t.Fatal(err)
			}
			if sameEntry {
				right = left
			}
			if err := right.AddPrefix("2001:db8::/32"); err != nil {
				t.Fatal(err)
			}
			if err := a.Add(left); err != nil {
				t.Fatal(err)
			}
			if err := b.Add(right); err != nil {
				t.Fatal(err)
			}
			merge := func(c Container, source *Entry) func() {
				return func() {
					for i := 0; i < 100; i++ {
						if err := c.Add(source); err != nil {
							t.Error(err)
							return
						}
					}
				}
			}
			runConcurrent(t, merge(a, right), merge(b, left), merge(a, left), merge(b, right))
			for _, entry := range []*Entry{left, right} {
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{"192.0.2.0/24", "2001:db8::/32"}) {
					t.Fatalf("shared entry = %v, %v", got, err)
				}
			}
		})
	}
}

func TestContainerConcurrentFamilyRemoval(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("shared")
	if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}
	ipv6 := NewEntry("shared")
	if err := ipv6.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	runConcurrent(t, func() {
		for i := 0; i < 100; i++ {
			if err := c.Add(ipv6, IgnoreIPv4); err != nil {
				t.Error(err)
				return
			}
			if err := c.Remove(entry, CaseRemoveEntry, IgnoreIPv4); err != nil {
				t.Error(err)
				return
			}
		}
	}, func() {
		for i := 0; i < 100; i++ {
			if _, _, err := c.Lookup("2001:db8::1"); err != nil {
				t.Error(err)
				return
			}
			set, err := entry.GetIPv4Set()
			if err != nil || !set.Contains(netip.MustParseAddr("192.0.2.1")) {
				t.Errorf("IPv4 set = %v, %v", set, err)
				return
			}
		}
	})
	if err := c.Add(ipv6); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(entry, CaseRemoveEntry, IgnoreIPv4); err != nil {
		t.Fatal(err)
	}
	got, err := entry.MarshalText()
	if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) || c.Len() != 1 {
		t.Fatalf("family removal = %v, %v, Len %d", got, err, c.Len())
	}
}

func TestEntryIPSetSnapshot(t *testing.T) {
	entry := NewEntry("snapshot")
	if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := entry.GetIPv4Set()
	if err != nil {
		t.Fatal(err)
	}
	runConcurrent(t, func() {
		for i := 0; i < 100; i++ {
			if err := entry.AddPrefix("198.51.100.0/24"); err != nil {
				t.Error(err)
				return
			}
			if _, err := entry.GetIPv4Set(); err != nil {
				t.Error(err)
				return
			}
			if err := entry.RemovePrefix("198.51.100.0/24"); err != nil {
				t.Error(err)
				return
			}
			if _, err := entry.MarshalText(); err != nil {
				t.Error(err)
				return
			}
		}
	}, func() {
		for i := 0; i < 100; i++ {
			if !snapshot.Contains(netip.MustParseAddr("192.0.2.1")) ||
				snapshot.Contains(netip.MustParseAddr("198.51.100.1")) {
				t.Error("published IPSet snapshot was modified")
				return
			}
		}
	})
	if err := entry.RemovePrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	if _, err := entry.MarshalText(); !errors.Is(err, ErrEmptyPrefix) {
		t.Fatalf("empty entry = %v", err)
	}
}

func TestContainerFamilyOptions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option IgnoreIPOption
	}{
		{"both", nil},
		{"IPv6 only", IgnoreIPv4},
		{"IPv4 only", IgnoreIPv6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			makeEntry := func(v4, v6 string) *Entry {
				t.Helper()
				entry := NewEntry("shared")
				for _, prefix := range []string{v4, v6} {
					if err := entry.AddPrefix(prefix); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := entry.MarshalText(); err != nil {
					t.Fatal(err)
				}
				return entry
			}
			c := NewContainer()
			entry := makeEntry("192.0.2.0/24", "2001:db8::/32")
			if err := c.Add(entry, tc.option); err != nil {
				t.Fatal(err)
			}
			initial, err := entry.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			delta := makeEntry("198.51.100.0/24", "2001:dba::/32")
			added, err := delta.MarshalText(tc.option)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Add(delta, tc.option); err != nil {
				t.Fatal(err)
			}
			merged, err := entry.MarshalText()
			want := append(slices.Clone(initial), added...)
			slices.Sort(want)
			ordered := slices.Clone(merged)
			slices.Sort(ordered)
			if err != nil || !slices.Equal(ordered, want) {
				t.Fatalf("merged = %v, %v; want %v", merged, err, want)
			}
			if err := c.Remove(delta, CaseRemovePrefix, tc.option); err != nil {
				t.Fatal(err)
			}
			got, err := entry.MarshalText()
			if err != nil || !slices.Equal(got, initial) {
				t.Fatalf("prefix removal = %v, %v; want %v", got, err, initial)
			}
			if err := c.Remove(entry, CaseRemoveEntry, tc.option); err != nil {
				t.Fatal(err)
			}
			if tc.option == nil {
				if c.Len() != 0 {
					t.Fatal("unfiltered removal retained the entry")
				}
			} else {
				stored, found := c.GetEntry("shared")
				if !found || stored != entry {
					t.Fatal("filtered removal lost the empty category")
				}
				if _, err := stored.MarshalText(); !errors.Is(err, ErrEmptyPrefix) {
					t.Fatalf("filtered removal did not clear the family: %v", err)
				}
			}
		})
	}
}

func TestContainerOptionsCanCallMethods(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("shared")
	if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	option := func() IPType {
		c.Len()
		c.GetEntry("shared")
		for range c.Loop() {
		}
		if _, _, err := c.Lookup("192.0.2.1"); err != nil {
			t.Error(err)
		}
		return ""
	}
	runConcurrent(t, func() {
		if err := c.Add(entry, option); err != nil {
			t.Error(err)
			return
		}
		if _, err := entry.MarshalText(func() IPType {
			if err := entry.AddPrefix("198.51.100.0/24"); err != nil {
				t.Error(err)
			}
			return ""
		}); err != nil {
			t.Error(err)
			return
		}
		if err := c.Remove(entry, CaseRemoveEntry, option); err != nil {
			t.Error(err)
		}
	})
}

func TestContainerSelfPrefixRemoval(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option IgnoreIPOption
		keep4  bool
		keep6  bool
	}{
		{"both", nil, false, false},
		{"IPv6 only", IgnoreIPv4, true, false},
		{"IPv4 only", IgnoreIPv6, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer()
			entry := NewEntry("shared")
			for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
				if err := entry.AddPrefix(prefix); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.Add(entry); err != nil {
				t.Fatal(err)
			}
			if err := c.Remove(entry, CaseRemovePrefix, tc.option); err != nil {
				t.Fatal(err)
			}
			if c.Len() != 1 {
				t.Fatal("prefix removal lost the category")
			}
			v4, err := entry.GetIPv4Set()
			if err != nil {
				t.Fatal(err)
			}
			v6, err := entry.GetIPv6Set()
			if err != nil {
				t.Fatal(err)
			}
			if v4.Contains(netip.MustParseAddr("192.0.2.1")) != tc.keep4 ||
				v6.Contains(netip.MustParseAddr("2001:db8::1")) != tc.keep6 {
				t.Fatal("self-removal did not respect the ignored family")
			}
		})
	}
}

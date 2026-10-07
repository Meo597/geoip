package lib

import (
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"go4.org/netipx"
)

// Container supports concurrent membership and prefix operations.
// Loop snapshots membership; the returned Entry pointers remain shared.
type Container interface {
	GetEntry(name string) (*Entry, bool)
	Len() int
	Add(entry *Entry, opts ...IgnoreIPOption) error
	Remove(entry *Entry, rCase CaseRemove, opts ...IgnoreIPOption) error
	Loop() <-chan *Entry
	Lookup(ipOrCidr string, searchList ...string) ([]string, bool, error)
}

type container struct {
	// Acquire the container lock before any Entry lock. Never hold two Entry locks.
	mu      sync.RWMutex
	entries map[string]*Entry
}

func NewContainer() Container {
	return &container{
		entries: make(map[string]*Entry),
	}
}

func (c *container) isValid() bool {
	return c.entries != nil
}

func (c *container) GetEntry(name string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.isValid() {
		return nil, false
	}
	val, ok := c.entries[strings.ToUpper(strings.TrimSpace(name))]
	if !ok {
		return nil, false
	}
	return val, true
}

func (c *container) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.isValid() {
		return 0
	}
	return len(c.entries)
}

func (c *container) Loop() <-chan *Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ch := make(chan *Entry, len(c.entries))
	for _, val := range c.entries {
		ch <- val
	}
	close(ch)
	return ch
}

func (c *container) Add(entry *Entry, opts ...IgnoreIPOption) error {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}

	name := entry.GetName()
	c.mu.Lock()
	defer c.mu.Unlock()
	val, found := c.entries[name]

	switch found {
	case true:
		if val == entry {
			entry.mu.Lock()
			defer entry.mu.Unlock()
			// Self-merging is a no-op; do not re-add a stale source snapshot.
			return entry.buildIPSet()
		}
		// Release the source lock before locking the destination.
		ipv4set, ipv6set, err := entry.ipSets()
		if err != nil {
			return err
		}
		val.mu.Lock()
		defer val.mu.Unlock()
		switch ignoreIPType {
		case IPv4:
			if ipv6set != nil {
				if !val.hasIPv6Builder() {
					val.ipv6Builder = new(netipx.IPSetBuilder)
				}
				val.ipv6Builder.AddSet(ipv6set)
				val.ipv6Set = nil
			}
		case IPv6:
			if ipv4set != nil {
				if !val.hasIPv4Builder() {
					val.ipv4Builder = new(netipx.IPSetBuilder)
				}
				val.ipv4Builder.AddSet(ipv4set)
				val.ipv4Set = nil
			}
		default:
			if ipv4set != nil {
				if !val.hasIPv4Builder() {
					val.ipv4Builder = new(netipx.IPSetBuilder)
				}
				val.ipv4Builder.AddSet(ipv4set)
				val.ipv4Set = nil
			}
			if ipv6set != nil {
				if !val.hasIPv6Builder() {
					val.ipv6Builder = new(netipx.IPSetBuilder)
				}
				val.ipv6Builder.AddSet(ipv6set)
				val.ipv6Set = nil
			}
		}

	case false:
		entry.mu.Lock()
		defer entry.mu.Unlock()
		switch ignoreIPType {
		case IPv4:
			entry.ipv4Builder = nil
			entry.ipv4Set = nil
		case IPv6:
			entry.ipv6Builder = nil
			entry.ipv6Set = nil
		}
		c.entries[name] = entry
	}

	return nil
}

func (c *container) Remove(entry *Entry, rCase CaseRemove, opts ...IgnoreIPOption) error {
	name := entry.GetName()
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	val, found := c.entries[name]
	if !found {
		return fmt.Errorf("entry %s not found", name)
	}

	switch rCase {
	case CaseRemovePrefix:
		var ipv4set, ipv6set *netipx.IPSet
		if val == entry {
			val.mu.Lock()
			defer val.mu.Unlock()
			if err := val.buildIPSet(); err != nil {
				return err
			}
			ipv4set, ipv6set = val.ipv4Set, val.ipv6Set
		} else {
			// Release the source lock before locking the destination.
			var err error
			ipv4set, ipv6set, err = entry.ipSets()
			if err != nil {
				return err
			}
			val.mu.Lock()
			defer val.mu.Unlock()
		}

		switch ignoreIPType {
		case IPv4:
			if ipv6set != nil {
				if !val.hasIPv6Builder() {
					val.ipv6Builder = new(netipx.IPSetBuilder)
				}
				val.ipv6Builder.RemoveSet(ipv6set)
				val.ipv6Set = nil
			}
		case IPv6:
			if ipv4set != nil {
				if !val.hasIPv4Builder() {
					val.ipv4Builder = new(netipx.IPSetBuilder)
				}
				val.ipv4Builder.RemoveSet(ipv4set)
				val.ipv4Set = nil
			}
		default:
			if ipv4set != nil {
				if !val.hasIPv4Builder() {
					val.ipv4Builder = new(netipx.IPSetBuilder)
				}
				val.ipv4Builder.RemoveSet(ipv4set)
				val.ipv4Set = nil
			}
			if ipv6set != nil {
				if !val.hasIPv6Builder() {
					val.ipv6Builder = new(netipx.IPSetBuilder)
				}
				val.ipv6Builder.RemoveSet(ipv6set)
				val.ipv6Set = nil
			}
		}

	case CaseRemoveEntry:
		val.mu.Lock()
		defer val.mu.Unlock()
		switch ignoreIPType {
		case IPv4:
			val.ipv6Builder = nil
			val.ipv6Set = nil
		case IPv6:
			val.ipv4Builder = nil
			val.ipv4Set = nil
		default:
			delete(c.entries, name)
		}

	default:
		return fmt.Errorf("unknown remove case %d", rCase)
	}

	return nil
}

func (c *container) Lookup(ipOrCidr string, searchList ...string) ([]string, bool, error) {
	switch strings.Contains(ipOrCidr, "/") {
	case true: // CIDR
		prefix, err := netip.ParsePrefix(ipOrCidr)
		if err != nil {
			return nil, false, err
		}
		addr := prefix.Addr()
		if addr.Is4In6() {
			if prefix.Bits() < 96 {
				return nil, false, ErrInvalidCIDR
			}
			addr = addr.Unmap()
			prefix = netip.PrefixFrom(addr, prefix.Bits()-96)
		}
		switch {
		case addr.Is4():
			return c.lookup(prefix, IPv4, searchList...)
		case addr.Is6():
			return c.lookup(prefix, IPv6, searchList...)
		}

	case false: // IP
		addr, err := netip.ParseAddr(ipOrCidr)
		if err != nil {
			return nil, false, err
		}
		addr = addr.Unmap()
		switch {
		case addr.Is4():
			return c.lookup(addr, IPv4, searchList...)
		case addr.Is6():
			return c.lookup(addr, IPv6, searchList...)
		}
	}

	return nil, false, nil
}

func (c *container) lookup(addrOrPrefix any, iptype IPType, searchList ...string) ([]string, bool, error) {
	searchMap := make(map[string]bool)
	for _, name := range searchList {
		if name = strings.ToUpper(strings.TrimSpace(name)); name != "" {
			searchMap[name] = true
		}
	}

	isfound := false
	result := make([]string, 0, 8)

	for entry := range c.Loop() {
		if len(searchMap) > 0 && !searchMap[entry.GetName()] {
			continue
		}

		ipv4set, ipv6set, err := entry.ipSets()
		if err != nil {
			return nil, false, err
		}
		var ipset *netipx.IPSet
		switch iptype {
		case IPv4:
			ipset = ipv4set
		case IPv6:
			ipset = ipv6set
		}
		if ipset == nil {
			continue
		}

		switch addrOrPrefix := addrOrPrefix.(type) {
		case netip.Prefix:
			if found := ipset.ContainsPrefix(addrOrPrefix); found {
				isfound = true
				result = append(result, entry.GetName())
			}
		case netip.Addr:
			if found := ipset.Contains(addrOrPrefix); found {
				isfound = true
				result = append(result, entry.GetName())
			}
		}
	}

	return result, isfound, nil
}

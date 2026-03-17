package provider

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Provider{}
)

// Register adds a provider to the global registry.
// Providers call this from init() to self-register.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name] = p
}

// Get returns a registered provider by name.
func Get(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q (available: %v)", name, Available())
	}
	return p, nil
}

// Available returns the sorted list of registered provider names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

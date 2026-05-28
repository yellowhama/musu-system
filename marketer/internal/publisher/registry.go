package publisher

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Publisher)
	mu       sync.RWMutex
)

// Register adds a new publisher adapter to the global registry.
func Register(name string, p Publisher) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = p
}

// Get retrieves a publisher adapter by its name.
func Get(name string) (Publisher, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("publisher '%s' not registered", name)
	}
	return p, nil
}

// List returns the names of all registered publishers.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	var names []string
	for k := range registry {
		names = append(names, k)
	}
	return names
}

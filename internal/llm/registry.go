package llm

import (
	"errors"
	"sync"
)

// Registry stores protocol adapters and is safe for concurrent access.
type Registry struct {
	mu       sync.RWMutex
	adapters map[Protocol]ProtocolAdapter
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[Protocol]ProtocolAdapter),
	}
}

// Register adds or replaces an adapter for its protocol.
func (r *Registry) Register(adapter ProtocolAdapter) error {
	if adapter == nil {
		return errors.New("protocol adapter is nil")
	}

	protocol := adapter.Protocol()
	if protocol == "" {
		return errors.New("protocol adapter protocol is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.adapters[protocol] = adapter
	return nil
}

// Get returns the adapter registered for the protocol.
func (r *Registry) Get(protocol Protocol) (ProtocolAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[protocol]
	return adapter, ok
}

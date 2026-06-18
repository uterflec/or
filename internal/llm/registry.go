package llm

import (
	"errors"
	"sync"
)

// Registry stores providers by protocol and is safe for concurrent access.
type Registry struct {
	mu        sync.RWMutex
	providers map[Protocol]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[Protocol]Provider),
	}
}

// Register adds or replaces the provider for its protocol.
func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return errors.New("provider is nil")
	}

	protocol := provider.Protocol()
	if protocol == "" {
		return errors.New("provider protocol is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[protocol] = provider
	return nil
}

// Get returns the provider registered for the protocol.
func (r *Registry) Get(protocol Protocol) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[protocol]
	return provider, ok
}

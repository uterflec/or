package llm

import (
	"context"
	"errors"
	"sync"
)

// ProviderEnv contains request-scoped environment overrides. Non-empty values
// take precedence over process environment variables during credential lookup.
type ProviderEnv map[string]string

// StreamOptions contains provider-specific settings for a stream request.
type StreamOptions struct {
	APIKey string
	Env    ProviderEnv
	// Temperature overrides the model's default sampling temperature when set.
	Temperature *float64
	// MaxTokens caps the output tokens for this request. Zero leaves it unset.
	MaxTokens int64
	// Headers are merged into the request, overriding model default headers.
	Headers map[string]string
	// Reasoning requests a thinking level. The provider clamps it to what the
	// model supports. Empty leaves the model's default; "off" disables thinking.
	Reasoning ModelThinkingLevel
}

// ProtocolAdapter translates between a concrete LLM protocol and the package streaming interface.
type ProtocolAdapter interface {
	// Protocol returns the registry key used to select this adapter.
	Protocol() Protocol

	// Stream emits response events for the given model and conversation context.
	Stream(ctx context.Context, model Model, input Context, options StreamOptions) (<-chan Event, error)
}

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
func (registry *Registry) Register(adapter ProtocolAdapter) error {
	if adapter == nil {
		return errors.New("protocol adapter is nil")
	}

	protocol := adapter.Protocol()
	if protocol == "" {
		return errors.New("protocol adapter protocol is empty")
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.adapters[protocol] = adapter
	return nil
}

// Get returns the adapter registered for the protocol.
func (registry *Registry) Get(protocol Protocol) (ProtocolAdapter, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	adapter, ok := registry.adapters[protocol]
	return adapter, ok
}

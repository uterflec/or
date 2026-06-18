package llm

import "context"

// StreamOptions contains provider-specific settings for a stream request.
type StreamOptions struct {
	APIKey string
}

// ProtocolAdapter translates between a concrete LLM protocol and the package streaming interface.
type ProtocolAdapter interface {
	// Protocol returns the registry key used to select this provider.
	Protocol() Protocol

	// Stream emits response events for the given model and conversation context.
	Stream(
		ctx context.Context,
		model Model,
		input Context,
		options StreamOptions,
	) (<-chan Event, error)
}

package fake

import (
	"context"
	"fmt"

	"github.com/ktsoator/or/internal/llm"
)

const Protocol llm.Protocol = "fake"

// Provider is an in-memory provider useful for tests and local development.
type Provider struct {
	response string
}

// NewProvider creates a fake provider that streams the given response.
func NewProvider(response string) *Provider {
	return &Provider{
		response: response,
	}
}

// Protocol returns the registry key for the fake provider.
func (p *Provider) Protocol() llm.Protocol {
	return Protocol
}

// Stream emits a deterministic response without calling an external service.
func (p *Provider) Stream(
	ctx context.Context,
	model llm.Model,
	input llm.Context,
	options llm.StreamOptions,
) (<-chan llm.Event, error) {
	if model.Protocol != p.Protocol() {
		return nil, fmt.Errorf(
			"model protocol %q does not match provider protocol %q",
			model.Protocol,
			p.Protocol(),
		)
	}

	events := make(chan llm.Event, 4)

	go func() {
		defer close(events)

		partial := llm.AssistantMessage{
			Model: model.ID,
		}

		events <- llm.Event{
			Type:    llm.EventStart,
			Partial: &partial,
		}

		select {
		case <-ctx.Done():
			failed := llm.AssistantMessage{
				Model:      model.ID,
				StopReason: "aborted",
			}

			events <- llm.Event{
				Type:    llm.EventError,
				Message: &failed,
				Err:     ctx.Err(),
			}
			return

		default:
		}

		partial = llm.AssistantMessage{
			Model: model.ID,
			Content: []llm.Content{
				{
					Type: llm.ContentText,
					Text: p.response,
				},
			},
		}

		events <- llm.Event{
			Type:    llm.EventTextDelta,
			Delta:   p.response,
			Partial: &partial,
		}

		finalMessage := partial
		finalMessage.StopReason = "stop"

		events <- llm.Event{
			Type:    llm.EventDone,
			Message: &finalMessage,
		}
	}()

	_ = input
	_ = options

	return events, nil
}

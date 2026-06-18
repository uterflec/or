package fake

import (
	"context"
	"fmt"

	"github.com/ktsoator/or/internal/llm"
)

const Protocol llm.Protocol = "fake"

// Adapter is an in-memory protocol adapter useful for tests and local development.
type Adapter struct {
	response string
}

// NewAdapter creates a fake adapter that streams the given response.
func NewAdapter(response string) *Adapter {
	return &Adapter{
		response: response,
	}
}

// Protocol returns the registry key for the fake provider.
func (a *Adapter) Protocol() llm.Protocol {
	return Protocol
}

// Stream emits a deterministic response without calling an external service.
func (a *Adapter) Stream(
	ctx context.Context,
	model llm.Model,
	input llm.Context,
	options llm.StreamOptions,
) (<-chan llm.Event, error) {
	if model.Protocol != a.Protocol() {
		return nil, fmt.Errorf(
			"model protocol %q does not match adapter protocol %q",
			model.Protocol,
			a.Protocol(),
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
			Content: []llm.AssistantContent{
				&llm.TextContent{Text: a.response},
			},
		}

		events <- llm.Event{
			Type:    llm.EventTextDelta,
			Delta:   a.response,
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

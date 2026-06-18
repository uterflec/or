// Package agent runs a general-purpose agent loop on top of the llm layer:
// stream an assistant turn, execute any tool calls it requests, feed the results
// back, and repeat until the model produces a final answer (or a limit is hit).
//
// The loop mechanism (loop) is separate from the stateful coordinator (Agent):
// the loop reaches steering and follow-up messages only through hooks, and the
// Agent wires those hooks to its message queues. This keeps the loop independent
// of how queued messages are stored, and is what lets Steer/FollowUp be called
// from another goroutine while a run is in flight.
package agent

import (
	"context"

	"github.com/ktsoator/or/internal/llm"
)

const defaultMaxTurns = 16

// Config configures an Agent.
type Config struct {
	Client       *llm.Client
	Model        llm.Model
	Options      llm.StreamOptions
	SystemPrompt string
	Tools        []Tool
	// MaxTurns bounds the loop. Zero uses a default.
	MaxTurns int
}

// Agent is the stateful coordinator: it owns the transcript and the steering and
// follow-up queues, and drives the loop. Run and Continue are not safe to call
// concurrently with each other; Steer and FollowUp are safe to call at any time.
type Agent struct {
	cfg      loopConfig
	messages []llm.Message
	steering messageQueue
	followUp messageQueue
}

// New builds an Agent from its config.
func New(cfg Config) *Agent {
	tools := make(map[string]Tool, len(cfg.Tools))
	definitions := make([]llm.ToolDefinition, 0, len(cfg.Tools))
	for _, tool := range cfg.Tools {
		definition := tool.Definition()
		tools[definition.Name] = tool
		definitions = append(definitions, definition)
	}

	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	return &Agent{
		cfg: loopConfig{
			client:       cfg.Client,
			model:        cfg.Model,
			options:      cfg.Options,
			systemPrompt: cfg.SystemPrompt,
			tools:        tools,
			definitions:  definitions,
			maxTurns:     maxTurns,
		},
	}
}

// Run streams the agent loop for a new prompt. Events flow on the returned
// channel, which closes when the run ends. To stop early, cancel ctx and keep
// draining the channel until it closes.
func (a *Agent) Run(ctx context.Context, prompt string) <-chan Event {
	user := &llm.UserMessage{Content: []llm.UserContent{&llm.TextContent{Text: prompt}}}
	return a.start(ctx, []llm.Message{user})
}

// Continue resumes from the current transcript without a new prompt, draining any
// queued steering or follow-up messages. The last transcript message must be a
// user or tool result message, or the provider will reject the request.
func (a *Agent) Continue(ctx context.Context) <-chan Event {
	return a.start(ctx, nil)
}

func (a *Agent) start(ctx context.Context, prompts []llm.Message) <-chan Event {
	events := make(chan Event)

	go func() {
		defer close(events)

		emit := func(event Event) bool {
			select {
			case events <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}

		l := &loop{
			cfg: a.cfg,
			hooks: loopHooks{
				getSteering: a.steering.drain,
				getFollowUp: a.followUp.drain,
			},
		}
		l.run(ctx, &a.messages, prompts, emit)
	}()

	return events
}

// Steer queues a message to inject after the current turn's tools finish.
func (a *Agent) Steer(message llm.Message) { a.steering.enqueue(message) }

// FollowUp queues a message to run after the agent would otherwise stop.
func (a *Agent) FollowUp(message llm.Message) { a.followUp.enqueue(message) }

// ClearQueues removes all queued steering and follow-up messages.
func (a *Agent) ClearQueues() {
	a.steering.clear()
	a.followUp.clear()
}

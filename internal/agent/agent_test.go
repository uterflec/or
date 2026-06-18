package agent_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ktsoator/or/internal/agent"
	"github.com/ktsoator/or/internal/llm"
)

// scriptedAdapter emits a fixed sequence of llm events per Stream call, one
// entry per turn. It lets the agent loop run without a real provider.
type scriptedAdapter struct {
	turns [][]llm.Event
	call  int
}

func (s *scriptedAdapter) Protocol() llm.Protocol { return llm.Protocol("scripted") }

func (s *scriptedAdapter) Stream(
	_ context.Context,
	_ llm.Model,
	_ llm.Context,
	_ llm.StreamOptions,
) (<-chan llm.Event, error) {
	turn := s.turns[s.call]
	s.call++

	events := make(chan llm.Event, len(turn))
	go func() {
		defer close(events)
		for _, event := range turn {
			events <- event
		}
	}()
	return events, nil
}

// echoTool records that it ran and echoes its arguments back.
type echoTool struct {
	executed *bool
	gotArgs  *string
}

func (t echoTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        "echo",
		Description: "Echo the arguments back",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
	}
}

func (t echoTool) Execute(_ context.Context, arguments string) (agent.Result, error) {
	*t.executed = true
	*t.gotArgs = arguments
	return agent.Result{Content: []llm.ToolResultContent{&llm.TextContent{Text: "echoed"}}}, nil
}

func TestAgentRunsToolThenAnswers(t *testing.T) {
	toolTurn := &llm.AssistantMessage{
		Content:    []llm.AssistantContent{&llm.ToolCall{ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`}},
		StopReason: llm.StopReasonToolUse,
	}
	answerTurn := &llm.AssistantMessage{
		Content:    []llm.AssistantContent{&llm.TextContent{Text: "all done"}},
		StopReason: llm.StopReasonStop,
	}
	adapter := &scriptedAdapter{turns: [][]llm.Event{
		{{Type: llm.EventDone, Message: toolTurn}},
		{{Type: llm.EventDone, Message: answerTurn}},
	}}

	registry := llm.NewRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	executed := false
	gotArgs := ""
	a := agent.New(agent.Config{
		Client:       llm.NewClient(registry),
		Model:        llm.Model{ID: "m", Protocol: adapter.Protocol()},
		SystemPrompt: "be helpful",
		Tools:        []agent.Tool{echoTool{executed: &executed, gotArgs: &gotArgs}},
	})

	var seen []agent.EventType
	var end agent.Event
	for event := range a.Run(context.Background(), "hello") {
		seen = append(seen, event.Type)
		if event.Type == agent.EventAgentEnd {
			end = event
		}
	}

	if !executed {
		t.Fatal("tool was not executed")
	}
	if gotArgs != `{"text":"hi"}` {
		t.Fatalf("tool received unexpected arguments: %q", gotArgs)
	}
	if !contains(seen, agent.EventToolStart) || !contains(seen, agent.EventToolEnd) {
		t.Fatalf("missing tool execution events: %v", seen)
	}
	if adapter.call != 2 {
		t.Fatalf("expected 2 turns, got %d", adapter.call)
	}

	if len(end.Messages) == 0 {
		t.Fatal("agent_end carried no messages")
	}
	last, ok := end.Messages[len(end.Messages)-1].(*llm.AssistantMessage)
	if !ok || len(last.Content) != 1 {
		t.Fatalf("unexpected final message: %#v", end.Messages[len(end.Messages)-1])
	}
	text, ok := last.Content[0].(*llm.TextContent)
	if !ok || text.Text != "all done" {
		t.Fatalf("unexpected final answer: %#v", last.Content[0])
	}
}

func TestAgentFollowUpContinuesAfterStop(t *testing.T) {
	firstAnswer := &llm.AssistantMessage{
		Content:    []llm.AssistantContent{&llm.TextContent{Text: "first answer"}},
		StopReason: llm.StopReasonStop,
	}
	secondAnswer := &llm.AssistantMessage{
		Content:    []llm.AssistantContent{&llm.TextContent{Text: "second answer"}},
		StopReason: llm.StopReasonStop,
	}
	adapter := &scriptedAdapter{turns: [][]llm.Event{
		{{Type: llm.EventDone, Message: firstAnswer}},
		{{Type: llm.EventDone, Message: secondAnswer}},
	}}

	registry := llm.NewRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	a := agent.New(agent.Config{
		Client: llm.NewClient(registry),
		Model:  llm.Model{ID: "m", Protocol: adapter.Protocol()},
	})

	// Queued before the run: picked up after the first turn would otherwise stop.
	followUp := &llm.UserMessage{Content: []llm.UserContent{&llm.TextContent{Text: "now do more"}}}
	a.FollowUp(followUp)

	var end agent.Event
	for event := range a.Run(context.Background(), "start") {
		if event.Type == agent.EventAgentEnd {
			end = event
		}
	}

	if adapter.call != 2 {
		t.Fatalf("expected 2 turns (initial + follow-up), got %d", adapter.call)
	}

	foundFollowUp := false
	for _, message := range end.Messages {
		if user, ok := message.(*llm.UserMessage); ok && user == followUp {
			foundFollowUp = true
		}
	}
	if !foundFollowUp {
		t.Fatal("follow-up message was not injected into the transcript")
	}
}

func contains(types []agent.EventType, target agent.EventType) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

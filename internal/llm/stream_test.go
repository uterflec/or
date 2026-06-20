package llm

import (
	"context"
	"errors"
	"testing"
)

func TestStreamWriterEmitsExactlyOneTerminalEvent(t *testing.T) {
	events := make(chan Event, 10)
	message := AssistantMessage{}
	writer := NewStreamWriter(context.Background(), events, &message)

	writer.Emit(Event{Type: EventTextDelta, Delta: "hello"})
	writer.Done()
	writer.Fail(errors.New("late failure"))
	writer.Emit(Event{Type: EventTextDelta, Delta: "late delta"})
	writer.Done()

	got := drainEvents(events)
	want := []EventType{EventStart, EventTextDelta, EventDone}
	if len(got) != len(want) {
		t.Fatalf("events = %#v, want types %v", got, want)
	}
	for i, eventType := range want {
		if got[i].Type != eventType {
			t.Fatalf("event[%d].Type = %q, want %q", i, got[i].Type, eventType)
		}
	}
}

func TestStreamWriterFailWithoutError(t *testing.T) {
	events := make(chan Event, 2)
	message := AssistantMessage{}
	writer := NewStreamWriter(context.Background(), events, &message)

	writer.Fail(nil)

	got := drainEvents(events)
	if len(got) != 2 || got[0].Type != EventStart || got[1].Type != EventError {
		t.Fatalf("events = %#v, want start then error", got)
	}
	terminal := got[1]
	if terminal.Err == nil || terminal.Message == nil || terminal.Message.StopReason != StopReasonError {
		t.Fatalf("terminal event = %#v, want ordinary stream error", terminal)
	}
}

func TestStreamWriterDoneAfterCancellationEmitsAbortedError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	events := make(chan Event, 2)
	message := AssistantMessage{}
	writer := NewStreamWriter(ctx, events, &message)

	writer.Done()

	got := drainEvents(events)
	if len(got) != 2 || got[0].Type != EventStart || got[1].Type != EventError {
		t.Fatalf("events = %#v, want start then error", got)
	}
	terminal := got[1]
	if !errors.Is(terminal.Err, context.Canceled) || terminal.Message == nil ||
		terminal.Message.StopReason != StopReasonAborted {
		t.Fatalf("terminal event = %#v, want cancelled stream", terminal)
	}
}

func drainEvents(events chan Event) []Event {
	result := make([]Event, 0, len(events))
	for len(events) > 0 {
		result = append(result, <-events)
	}
	return result
}

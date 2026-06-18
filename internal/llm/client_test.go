package llm_test

import (
	"context"
	"testing"

	"github.com/ktsoator/or/internal/llm"
	"github.com/ktsoator/or/internal/llm/providers/fake"
)

func TestClientCompleteWithFakeProvider(t *testing.T) {
	registry := llm.NewRegistry()

	adapter := fake.NewAdapter("hello from fake provider")
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	client := llm.NewClient(registry)

	model := llm.Model{
		ID:       "fake-1",
		Name:     "Fake Model",
		Protocol: fake.Protocol,
		Provider: "fake",
	}

	input := llm.Context{
		Messages: []llm.Message{
			&llm.UserMessage{
				Content: []llm.UserContent{
					&llm.TextContent{Text: "hello"},
				},
			},
		},
	}

	message, err := client.Complete(
		context.Background(),
		model,
		input,
		llm.StreamOptions{},
	)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if message.StopReason != "stop" {
		t.Fatalf(
			"expected stop reason %q, got %q",
			"stop",
			message.StopReason,
		)
	}

	if len(message.Content) != 1 {
		t.Fatalf(
			"expected one content block, got %d",
			len(message.Content),
		)
	}

	content, ok := message.Content[0].(*llm.TextContent)
	if !ok || content == nil || content.Text != "hello from fake provider" {
		t.Fatalf(
			"unexpected response: %q",
			message.Content[0],
		)
	}
}

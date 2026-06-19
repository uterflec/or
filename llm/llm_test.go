package llm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ktsoator/or/llm"
)

func TestDefaultClientIncludesBuiltInAdapters(t *testing.T) {
	tests := []struct {
		name     string
		protocol llm.Protocol
		want     string
	}{
		{name: "openai", protocol: llm.ProtocolOpenAICompletions, want: "OpenAI API key is empty"},
		{name: "anthropic", protocol: llm.ProtocolAnthropicMessages, want: "Anthropic API key is empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := llm.Stream(context.Background(), llm.Model{
				ID:       "test-model",
				Protocol: test.protocol,
				Provider: "facade-test",
			}, llm.Context{}, llm.StreamOptions{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Stream() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDefaultClientRejectsUnknownProtocol(t *testing.T) {
	_, err := llm.Stream(context.Background(), llm.Model{
		ID:       "test-model",
		Protocol: "unknown",
		Provider: "facade-test",
	}, llm.Context{}, llm.StreamOptions{})
	if err == nil || !strings.Contains(err.Error(), "no adapter registered") {
		t.Fatalf("Stream() error = %v", err)
	}
}

func TestGetModelUsesBuiltInCatalog(t *testing.T) {
	model := llm.GetModel("deepseek", "deepseek-chat")
	if model.Provider != "deepseek" || model.Protocol != llm.ProtocolOpenAICompletions {
		t.Fatalf("model = %#v", model)
	}
}

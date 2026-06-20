package llm

import (
	"strings"
	"testing"
)

func TestStreamOptionsValidateAcceptsMatchingAnthropicExtension(t *testing.T) {
	options := StreamOptions{
		ProtocolOptions: &AnthropicStreamOptions{ThinkingDisplay: ThinkingDisplayOmitted},
	}
	if err := options.Validate(ProtocolAnthropicMessages); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestStreamOptionsValidateRejectsAnthropicExtensionForOtherProtocol(t *testing.T) {
	options := StreamOptions{ProtocolOptions: &AnthropicStreamOptions{}}
	err := options.Validate(ProtocolOpenAICompletions)
	if err == nil || !strings.Contains(err.Error(), string(ProtocolAnthropicMessages)) {
		t.Fatalf("Validate() error = %v, want Anthropic protocol mismatch", err)
	}
}

func TestStreamOptionsValidateRejectsUnknownThinkingDisplay(t *testing.T) {
	options := StreamOptions{
		ProtocolOptions: &AnthropicStreamOptions{ThinkingDisplay: ThinkingDisplay("verbatim")},
	}
	err := options.Validate(ProtocolAnthropicMessages)
	if err == nil || !strings.Contains(err.Error(), "thinking display") {
		t.Fatalf("Validate() error = %v, want unsupported display", err)
	}
}

func TestStreamOptionsValidateRejectsTypedNilProtocolOptions(t *testing.T) {
	var anthropicOptions *AnthropicStreamOptions
	options := StreamOptions{ProtocolOptions: anthropicOptions}
	err := options.Validate(ProtocolAnthropicMessages)
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("Validate() error = %v, want typed nil error", err)
	}
}

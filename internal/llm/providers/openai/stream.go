package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ktsoator/or/internal/llm"
	oai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/respjson"
)

// ensureAssistantToolCall finds or creates the streaming tool call block for a
// delta, appending a new block to the message content on first sight and
// backfilling the id and name as they arrive across chunks. It tracks blocks by
// both stream index and id: some providers stream the index on every fragment,
// others only send the id once, so keying on a single field would merge distinct
// tool calls (or split one) depending on the provider.
func ensureAssistantToolCall(
	message *llm.AssistantMessage,
	byIndex map[int64]*llm.ToolCall,
	byID map[string]*llm.ToolCall,
	delta oai.ChatCompletionChunkChoiceDeltaToolCall,
) (*llm.ToolCall, int, bool) {
	hasIndex := delta.JSON.Index.Valid()

	var block *llm.ToolCall
	var ok bool
	if hasIndex {
		block, ok = byIndex[delta.Index]
	}
	if !ok && delta.ID != "" {
		block, ok = byID[delta.ID]
	}

	started := !ok
	if !ok {
		block = &llm.ToolCall{
			ID:        delta.ID,
			Name:      delta.Function.Name,
			Arguments: make(map[string]any),
		}
		message.Content = append(message.Content, block)
	}
	if hasIndex {
		byIndex[delta.Index] = block
	}
	if delta.ID != "" {
		byID[delta.ID] = block
	}
	if block.ID == "" && delta.ID != "" {
		block.ID = delta.ID
	}
	if block.Name == "" && delta.Function.Name != "" {
		block.Name = delta.Function.Name
	}
	return block, assistantContentIndex(message.Content, block), started
}

func ensureAssistantText(message *llm.AssistantMessage) (*llm.TextContent, int, bool) {
	for i, rawContent := range message.Content {
		if content, ok := rawContent.(*llm.TextContent); ok && content != nil {
			return content, i, false
		}
	}
	content := &llm.TextContent{}
	message.Content = append(message.Content, content)
	return content, len(message.Content) - 1, true
}

func ensureAssistantThinking(message *llm.AssistantMessage) (*llm.ThinkingContent, int, bool) {
	for i, rawContent := range message.Content {
		if content, ok := rawContent.(*llm.ThinkingContent); ok && content != nil {
			return content, i, false
		}
	}
	content := &llm.ThinkingContent{}
	message.Content = append(message.Content, content)
	return content, len(message.Content) - 1, true
}

func assistantContentIndex(content []llm.AssistantContent, target llm.AssistantContent) int {
	for i, candidate := range content {
		if candidate == target {
			return i
		}
	}
	return -1
}

func cloneAssistantMessage(message llm.AssistantMessage) *llm.AssistantMessage {
	clone := message
	clone.Content = make([]llm.AssistantContent, len(message.Content))
	for i, rawContent := range message.Content {
		switch content := rawContent.(type) {
		case *llm.TextContent:
			if content != nil {
				copied := *content
				clone.Content[i] = &copied
			}
		case *llm.ThinkingContent:
			if content != nil {
				copied := *content
				clone.Content[i] = &copied
			}
		case *llm.ToolCall:
			clone.Content[i] = cloneToolCall(content)
		}
	}
	return &clone
}

func cloneToolCall(toolCall *llm.ToolCall) *llm.ToolCall {
	if toolCall == nil {
		return nil
	}
	clone := *toolCall
	clone.Arguments = cloneJSONObject(toolCall.Arguments)
	return &clone
}

// parseToolArguments converts completed streamed JSON into the public argument
// object. OpenAI function arguments should be an object; malformed or incomplete
// JSON follows pi's tolerant behavior and becomes an empty object.
func parseToolArguments(raw string) map[string]any {
	if raw == "" {
		return make(map[string]any)
	}
	var arguments map[string]any
	if err := json.Unmarshal([]byte(raw), &arguments); err != nil || arguments == nil {
		return make(map[string]any)
	}
	return arguments
}

func cloneJSONObject(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	clone := make(map[string]any, len(value))
	for key, item := range value {
		clone[key] = cloneJSONValue(item)
	}
	return clone
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONObject(typed)
	case []any:
		clone := make([]any, len(typed))
		for index, item := range typed {
			clone[index] = cloneJSONValue(item)
		}
		return clone
	default:
		return value
	}
}

func emitError(events chan<- llm.Event, output llm.AssistantMessage, ctx context.Context, err error) {
	if ctx.Err() != nil {
		output.StopReason = llm.StopReasonAborted
		err = ctx.Err()
	} else {
		output.StopReason = llm.StopReasonError
	}
	output.ErrorMessage = err.Error()
	events <- llm.Event{Type: llm.EventError, Message: cloneAssistantMessage(output), Err: err}
}

func usageFrom(usage oai.CompletionUsage, model llm.Model) llm.Usage {
	cacheRead := usage.PromptTokensDetails.CachedTokens
	input := max(0, usage.PromptTokens-cacheRead)
	result := llm.Usage{
		Input:       input,
		Output:      usage.CompletionTokens,
		CacheRead:   cacheRead,
		TotalTokens: input + usage.CompletionTokens + cacheRead,
	}
	result.Cost = llm.CalculateCost(model, result)
	return result
}

// usageFromExtra reads usage from a non-standard extra field. Some providers
// (e.g. Moonshot) report token usage under choice.usage instead of the standard
// top-level chunk.usage. ok is false when the field is absent or null.
func usageFromExtra(fields map[string]respjson.Field, name string, model llm.Model) (llm.Usage, bool, error) {
	field, ok := fields[name]
	if !ok || field.Raw() == "" || field.Raw() == "null" {
		return llm.Usage{}, false, nil
	}

	var usage oai.CompletionUsage
	if err := json.Unmarshal([]byte(field.Raw()), &usage); err != nil {
		return llm.Usage{}, false, fmt.Errorf("decode OpenAI %s field: %w", name, err)
	}
	return usageFrom(usage, model), true, nil
}

// reasoningFieldNames lists the delta fields that carry reasoning content across
// OpenAI-compatible providers, in precedence order. The first non-empty value
// wins so providers that echo the same content under several names (e.g. both
// reasoning_content and reasoning) do not duplicate it.
var reasoningFieldNames = []string{"reasoning_content", "reasoning", "reasoning_text"}

// extraReasoning returns the first non-empty reasoning delta among the known
// provider-specific field names.
func extraReasoning(fields map[string]respjson.Field) (string, error) {
	for _, name := range reasoningFieldNames {
		value, err := extraString(fields, name)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
	}
	return "", nil
}

func extraString(fields map[string]respjson.Field, name string) (string, error) {
	field, ok := fields[name]
	if !ok || field.Raw() == "" || field.Raw() == "null" {
		return "", nil
	}

	var value string
	if err := json.Unmarshal([]byte(field.Raw()), &value); err != nil {
		return "", fmt.Errorf("decode OpenAI %s field: %w", name, err)
	}
	return value, nil
}

func mapStopReason(reason string) (llm.StopReason, error) {
	switch reason {
	case "stop", "end":
		return llm.StopReasonStop, nil
	case "length":
		return llm.StopReasonLength, nil
	case "tool_calls", "function_call":
		return llm.StopReasonToolUse, nil
	case "content_filter":
		return "", errors.New("OpenAI response was blocked by the content filter")
	case "":
		return "", errors.New("OpenAI stream ended without a finish reason")
	default:
		return "", fmt.Errorf("unsupported OpenAI finish reason %q", reason)
	}
}

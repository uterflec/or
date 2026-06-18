package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ktsoator/or/internal/llm"
	oai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/respjson"
	"github.com/openai/openai-go/v3/shared"
)

// Adapter translates the OpenAI-compatible Chat Completions protocol.
type Adapter struct {
	httpClient *http.Client
}

// NewAdapter creates an adapter that uses httpClient for requests.
// A nil client uses http.DefaultClient.
func NewAdapter(httpClient *http.Client) *Adapter {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Adapter{httpClient: httpClient}
}

// Protocol returns the registry key for the Chat Completions protocol.
func (a *Adapter) Protocol() llm.Protocol {
	return llm.ProtocolOpenAICompletions
}

// Stream starts a Chat Completions request and translates SDK chunks into
// package events. It supports text, reasoning, and tool call content.
func (a *Adapter) Stream(
	ctx context.Context,
	model llm.Model,
	input llm.Context,
	options llm.StreamOptions,
) (<-chan llm.Event, error) {
	if model.Protocol != a.Protocol() {
		return nil, fmt.Errorf("model protocol %q does not match adapter protocol %q", model.Protocol, a.Protocol())
	}
	if model.ID == "" {
		return nil, errors.New("model ID is empty")
	}
	if options.APIKey == "" {
		return nil, errors.New("OpenAI API key is empty")
	}

	messages, err := convertMessages(input)
	if err != nil {
		return nil, err
	}

	tools, err := convertTools(input.Tools)
	if err != nil {
		return nil, err
	}

	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
		option.WithHTTPClient(a.httpClient),
	}
	if model.BaseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(model.BaseURL))
	}
	client := oai.NewClient(clientOptions...)

	events := make(chan llm.Event)
	go func() {
		defer close(events)

		output := llm.AssistantMessage{Model: model.ID}
		events <- llm.Event{Type: llm.EventStart, Partial: cloneAssistantMessage(output)}

		params := oai.ChatCompletionNewParams{
			Model:    model.ID,
			Messages: messages,
		}
		if len(tools) > 0 {
			params.Tools = tools
		}
		stream := client.Chat.Completions.NewStreaming(ctx, params)
		defer stream.Close()

		toolCallsByIndex := make(map[int64]*llm.ToolCall)
		finishReason := ""
		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			reasoningDelta, err := extraString(choice.Delta.JSON.ExtraFields, "reasoning_content")
			if err != nil {
				emitError(events, output, ctx, err)
				return
			}
			if reasoningDelta != "" {
				appendAssistantThinking(&output, reasoningDelta)
				events <- llm.Event{
					Type:    llm.EventThinkingDelta,
					Delta:   reasoningDelta,
					Partial: cloneAssistantMessage(output),
				}
			}
			if choice.Delta.Content != "" {
				appendAssistantText(&output, choice.Delta.Content)
				events <- llm.Event{
					Type:    llm.EventTextDelta,
					Delta:   choice.Delta.Content,
					Partial: cloneAssistantMessage(output),
				}
			}
			for _, toolDelta := range choice.Delta.ToolCalls {
				block := ensureAssistantToolCall(&output, toolCallsByIndex, toolDelta)
				if toolDelta.Function.Arguments != "" {
					block.Arguments += toolDelta.Function.Arguments
				}
				events <- llm.Event{
					Type:     llm.EventToolCallDelta,
					Delta:    toolDelta.Function.Arguments,
					ToolCall: cloneToolCall(block),
					Partial:  cloneAssistantMessage(output),
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}

		if err := stream.Err(); err != nil {
			emitError(events, output, ctx, err)
			return
		}

		stopReason, err := mapStopReason(finishReason)
		if err != nil {
			emitError(events, output, ctx, err)
			return
		}
		output.StopReason = stopReason
		for _, content := range output.Content {
			if toolCall, ok := content.(*llm.ToolCall); ok && toolCall != nil {
				events <- llm.Event{
					Type:     llm.EventToolCallEnd,
					ToolCall: cloneToolCall(toolCall),
					Partial:  cloneAssistantMessage(output),
				}
			}
		}
		events <- llm.Event{Type: llm.EventDone, Message: cloneAssistantMessage(output)}
	}()

	return events, nil
}

func convertMessages(input llm.Context) ([]oai.ChatCompletionMessageParamUnion, error) {
	messages := make([]oai.ChatCompletionMessageParamUnion, 0, len(input.Messages)+1)
	if input.SystemPrompt != "" {
		messages = append(messages, oai.SystemMessage(input.SystemPrompt))
	}

	for i := 0; i < len(input.Messages); i++ {
		rawMessage := input.Messages[i]
		switch message := rawMessage.(type) {
		case *llm.UserMessage:
			if message == nil {
				return nil, errors.New("user message is nil")
			}
			userMessage, err := convertUserMessage(message)
			if err != nil {
				return nil, err
			}
			if userMessage == nil {
				continue
			}
			messages = append(messages, *userMessage)
		case *llm.AssistantMessage:
			if message == nil {
				return nil, errors.New("assistant message is nil")
			}
			assistant, err := convertAssistantMessage(message)
			if err != nil {
				return nil, err
			}
			if assistant == nil {
				continue
			}
			messages = append(messages, oai.ChatCompletionMessageParamUnion{OfAssistant: assistant})
		case *llm.ToolResultMessage:
			var images []oai.ChatCompletionContentPartUnionParam
			for ; i < len(input.Messages); i++ {
				toolResult, ok := input.Messages[i].(*llm.ToolResultMessage)
				if !ok {
					break
				}
				toolMessage, resultImages, err := convertToolResultMessage(toolResult)
				if err != nil {
					return nil, err
				}
				messages = append(messages, toolMessage)
				images = append(images, resultImages...)
			}
			i--
			if len(images) > 0 {
				parts := []oai.ChatCompletionContentPartUnionParam{
					oai.TextContentPart("Attached image(s) from tool result:"),
				}
				parts = append(parts, images...)
				messages = append(messages, oai.UserMessage(parts))
			}
		default:
			return nil, fmt.Errorf("unsupported message type %T", rawMessage)
		}
	}

	return messages, nil
}

func convertUserMessage(message *llm.UserMessage) (*oai.ChatCompletionMessageParamUnion, error) {
	if len(message.Content) == 1 {
		if content, ok := message.Content[0].(*llm.TextContent); ok && content != nil {
			converted := oai.UserMessage(content.Text)
			return &converted, nil
		}
	}

	parts := make([]oai.ChatCompletionContentPartUnionParam, 0, len(message.Content))
	for _, rawContent := range message.Content {
		switch content := rawContent.(type) {
		case *llm.TextContent:
			if content == nil {
				return nil, errors.New("user text content is nil")
			}
			parts = append(parts, oai.TextContentPart(content.Text))
		case *llm.ImageContent:
			image, err := convertImageContent(content)
			if err != nil {
				return nil, err
			}
			parts = append(parts, image)
		default:
			return nil, fmt.Errorf("unsupported user content type %T", rawContent)
		}
	}
	if len(parts) == 0 {
		return nil, nil
	}

	converted := oai.UserMessage(parts)
	return &converted, nil
}

func convertToolResultMessage(message *llm.ToolResultMessage) (
	oai.ChatCompletionMessageParamUnion,
	[]oai.ChatCompletionContentPartUnionParam,
	error,
) {
	if message == nil {
		return oai.ChatCompletionMessageParamUnion{}, nil, errors.New("tool result message is nil")
	}
	if message.ToolCallID == "" {
		return oai.ChatCompletionMessageParamUnion{}, nil, errors.New("tool result message is missing tool call ID")
	}

	var textParts []string
	var images []oai.ChatCompletionContentPartUnionParam
	for _, rawContent := range message.Content {
		switch content := rawContent.(type) {
		case *llm.TextContent:
			if content == nil {
				return oai.ChatCompletionMessageParamUnion{}, nil, errors.New("tool result text content is nil")
			}
			textParts = append(textParts, content.Text)
		case *llm.ImageContent:
			image, err := convertImageContent(content)
			if err != nil {
				return oai.ChatCompletionMessageParamUnion{}, nil, err
			}
			images = append(images, image)
		default:
			return oai.ChatCompletionMessageParamUnion{}, nil,
				fmt.Errorf("unsupported tool result content type %T", rawContent)
		}
	}

	result := strings.Join(textParts, "\n")
	if result == "" && len(images) > 0 {
		result = "(see attached image)"
	}
	return oai.ToolMessage(result, message.ToolCallID), images, nil
}

func convertImageContent(content *llm.ImageContent) (oai.ChatCompletionContentPartUnionParam, error) {
	if content == nil {
		return oai.ChatCompletionContentPartUnionParam{}, errors.New("image content is nil")
	}
	if content.MIMEType == "" {
		return oai.ChatCompletionContentPartUnionParam{}, errors.New("image content is missing MIME type")
	}
	if content.Data == "" {
		return oai.ChatCompletionContentPartUnionParam{}, errors.New("image content is missing data")
	}
	return oai.ImageContentPart(oai.ChatCompletionContentPartImageImageURLParam{
		URL: "data:" + content.MIMEType + ";base64," + content.Data,
	}), nil
}

// convertAssistantMessage serializes an assistant message, including any tool
// calls, into an OpenAI assistant message param. It returns nil for an empty
// message (no text and no tool calls), which the caller skips: some providers
// reject assistant messages that carry neither content nor tool calls.
func convertAssistantMessage(message *llm.AssistantMessage) (*oai.ChatCompletionAssistantMessageParam, error) {
	assistant := &oai.ChatCompletionAssistantMessageParam{}
	var text strings.Builder
	var reasoning strings.Builder
	for _, rawContent := range message.Content {
		switch content := rawContent.(type) {
		case *llm.TextContent:
			if content == nil {
				return nil, errors.New("assistant text content is nil")
			}
			text.WriteString(content.Text)
		case *llm.ThinkingContent:
			if content == nil {
				return nil, errors.New("assistant thinking content is nil")
			}
			reasoning.WriteString(content.Thinking)
		case *llm.ToolCall:
			if content == nil {
				return nil, errors.New("assistant tool call content is missing tool call data")
			}
			assistant.ToolCalls = append(assistant.ToolCalls, oai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &oai.ChatCompletionMessageFunctionToolCallParam{
					ID: content.ID,
					Function: oai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      content.Name,
						Arguments: content.Arguments,
					},
				},
			})
		default:
			return nil, fmt.Errorf("unsupported assistant content type %T", rawContent)
		}
	}

	hasText := false
	if value := text.String(); value != "" {
		assistant.Content.OfString = oai.String(value)
		hasText = true
	}
	if value := reasoning.String(); value != "" {
		assistant.SetExtraFields(map[string]any{
			"reasoning_content": value,
		})
	}
	if !hasText && len(assistant.ToolCalls) == 0 {
		return nil, nil
	}
	return assistant, nil
}

// convertTools maps tool definitions to OpenAI function tool params.
func convertTools(tools []llm.ToolDefinition) ([]oai.ChatCompletionToolUnionParam, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	converted := make([]oai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			return nil, errors.New("tool definition is missing a name")
		}

		function := shared.FunctionDefinitionParam{Name: tool.Name}
		if tool.Description != "" {
			function.Description = oai.String(tool.Description)
		}
		if len(tool.Parameters) > 0 {
			var parameters map[string]any
			if err := json.Unmarshal(tool.Parameters, &parameters); err != nil {
				return nil, fmt.Errorf("decode parameters for tool %q: %w", tool.Name, err)
			}
			function.Parameters = parameters
		}

		converted = append(converted, oai.ChatCompletionFunctionTool(function))
	}

	return converted, nil
}

// ensureAssistantToolCall finds or creates the streaming tool call block for a
// delta's index, appending a new block to the message content on first sight and
// backfilling the id and name as they arrive across chunks.
func ensureAssistantToolCall(
	message *llm.AssistantMessage,
	byIndex map[int64]*llm.ToolCall,
	delta oai.ChatCompletionChunkChoiceDeltaToolCall,
) *llm.ToolCall {
	block, ok := byIndex[delta.Index]
	if !ok {
		block = &llm.ToolCall{ID: delta.ID, Name: delta.Function.Name}
		byIndex[delta.Index] = block
		message.Content = append(message.Content, block)
	}
	if block.ID == "" && delta.ID != "" {
		block.ID = delta.ID
	}
	if block.Name == "" && delta.Function.Name != "" {
		block.Name = delta.Function.Name
	}
	return block
}

func appendAssistantText(message *llm.AssistantMessage, delta string) {
	for _, rawContent := range message.Content {
		if content, ok := rawContent.(*llm.TextContent); ok && content != nil {
			content.Text += delta
			return
		}
	}
	message.Content = append(message.Content, &llm.TextContent{Text: delta})
}

func appendAssistantThinking(message *llm.AssistantMessage, delta string) {
	for _, rawContent := range message.Content {
		if content, ok := rawContent.(*llm.ThinkingContent); ok && content != nil {
			content.Thinking += delta
			return
		}
	}
	message.Content = append(message.Content, &llm.ThinkingContent{Thinking: delta})
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
	return &clone
}

func emitError(events chan<- llm.Event, output llm.AssistantMessage, ctx context.Context, err error) {
	if ctx.Err() != nil {
		output.StopReason = "aborted"
		err = ctx.Err()
	} else {
		output.StopReason = "error"
	}
	events <- llm.Event{Type: llm.EventError, Message: cloneAssistantMessage(output), Err: err}
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

func mapStopReason(reason string) (string, error) {
	switch reason {
	case "stop":
		return "stop", nil
	case "length":
		return "length", nil
	case "tool_calls", "function_call":
		return "toolUse", nil
	case "content_filter":
		return "", errors.New("OpenAI response was blocked by the content filter")
	case "":
		return "", errors.New("OpenAI stream ended without a finish reason")
	default:
		return "", fmt.Errorf("unsupported OpenAI finish reason %q", reason)
	}
}

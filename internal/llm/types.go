package llm

import "encoding/json"

// Protocol identifies the API protocol used to communicate with a model.
type Protocol string

const (
	ProtocolOpenAICompletions Protocol = "openai-completions"
)

// UserContent is content that can appear in a user message.
type UserContent interface {
	isUserContent()
}

// AssistantContent is content that can appear in an assistant message.
type AssistantContent interface {
	isAssistantContent()
}

// ToolResultContent is content that can appear in a tool result message.
type ToolResultContent interface {
	isToolResultContent()
}

// TextContent represents plain text.
type TextContent struct {
	Text string `json:"text"`
}

func (*TextContent) isUserContent()       {}
func (*TextContent) isAssistantContent()  {}
func (*TextContent) isToolResultContent() {}

// ThinkingContent represents model reasoning content.
type ThinkingContent struct {
	Thinking string `json:"thinking"`
}

func (*ThinkingContent) isAssistantContent() {}

// ImageContent represents a base64-encoded image.
type ImageContent struct {
	Data     string `json:"data"`
	MIMEType string `json:"mimeType"`
}

func (*ImageContent) isUserContent()       {}
func (*ImageContent) isToolResultContent() {}

// ToolCall describes a request to invoke a named tool with JSON arguments.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (*ToolCall) isAssistantContent() {}

// Message is one item in the conversation context.
type Message interface {
	isMessage()
}

// UserMessage contains content supplied by the user.
type UserMessage struct {
	Content []UserContent `json:"content"`
}

func (*UserMessage) isMessage() {}

// ToolResultMessage contains the result of an assistant tool call.
type ToolResultMessage struct {
	ToolCallID string              `json:"toolCallId"`
	ToolName   string              `json:"toolName"`
	Content    []ToolResultContent `json:"content"`
	IsError    bool                `json:"isError"`
}

func (*ToolResultMessage) isMessage() {}

// ToolDefinition describes a tool that the model may call.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Context contains the prompt, conversation history, and available tools.
type Context struct {
	SystemPrompt string           `json:"systemPrompt,omitempty"`
	Messages     []Message        `json:"messages"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
}

// Model identifies the model and provider endpoint to use.
type Model struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol Protocol `json:"protocol"`
	Provider string   `json:"provider"`
	BaseURL  string   `json:"baseUrl"`
}

// AssistantMessage is the final or partial response returned by a provider.
type AssistantMessage struct {
	Content    []AssistantContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stopReason"`
}

func (*AssistantMessage) isMessage() {}

package agent

import "github.com/ktsoator/or/internal/llm"

// EventType identifies an agent loop update. Agent events wrap the lower-level
// llm stream events: a message_update carries the underlying llm.Event.
type EventType string

const (
	// EventAgentStart marks the beginning of a run.
	EventAgentStart EventType = "agent_start"
	// EventAgentEnd marks the end of a run. Messages holds this run's new messages.
	EventAgentEnd EventType = "agent_end"
	// EventTurnStart marks the beginning of a turn (one assistant response plus its tool calls).
	EventTurnStart EventType = "turn_start"
	// EventTurnEnd marks the end of a turn. Message is the assistant message; ToolResults are its results.
	EventTurnEnd EventType = "turn_end"
	// EventMessageStart marks a message (user, assistant, or tool result) entering the transcript.
	EventMessageStart EventType = "message_start"
	// EventMessageUpdate carries a streaming update for the current assistant message.
	// LLMEvent holds the underlying llm stream event.
	EventMessageUpdate EventType = "message_update"
	// EventMessageEnd marks a completed message.
	EventMessageEnd EventType = "message_end"
	// EventToolStart marks the start of a tool execution.
	EventToolStart EventType = "tool_execution_start"
	// EventToolEnd marks the end of a tool execution.
	EventToolEnd EventType = "tool_execution_end"
)

// Event is a single agent loop update. Only the fields relevant to Type are set.
type Event struct {
	Type EventType

	// Message is set for message_* events and turn_end (the assistant message).
	Message llm.Message
	// LLMEvent is the underlying stream event, set for message_update.
	LLMEvent *llm.Event

	// ToolCallID, ToolName, Args, Result, IsError are set for tool_execution_* events.
	ToolCallID string
	ToolName   string
	Args       map[string]any
	Result     *Result
	IsError    bool

	// ToolResults holds the tool result messages for a turn_end event.
	ToolResults []llm.Message
	// Messages holds this run's new messages for an agent_end event.
	Messages []llm.Message
}

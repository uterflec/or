package agent

import (
	"context"

	"github.com/ktsoator/or/internal/llm"
)

// Result is what a tool returns to the agent loop.
type Result struct {
	// Content is the text/image returned to the model as the tool result.
	Content []llm.ToolResultContent
	// Details is arbitrary structured data for logs or UI rendering.
	Details any
	// Terminate hints that the agent should stop after the current tool batch.
	// The loop only stops early when every tool result in the batch sets it.
	Terminate bool
}

// Tool is a model-facing tool definition plus its executor.
//
// Execute should return an error on failure rather than encoding the failure in
// Content; the loop turns the error into an error tool result so the model can
// recover. It must honor ctx cancellation.
type Tool interface {
	Definition() llm.ToolDefinition
	Execute(ctx context.Context, arguments map[string]any) (Result, error)
}

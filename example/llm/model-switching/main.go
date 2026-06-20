// Command model-switching reuses one provider-neutral conversation across two
// models that speak different wire protocols.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ktsoator/or/llm"
)

func main() {
	ctx := context.Background()
	draftModel := llm.GetModel("deepseek", "deepseek-v4-flash")
	reviewModel := llm.GetModel("minimax-cn", "MiniMax-M3")

	history := []llm.Message{
		&llm.UserMessage{Content: []llm.UserContent{
			&llm.TextContent{Text: "Draft a concise explanation of Go interfaces."},
		}},
	}

	draft, err := llm.Complete(
		ctx,
		draftModel,
		llm.Context{Messages: history},
		llm.StreamOptions{Reasoning: llm.ModelThinkingHigh},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"Draft from %s/%s (%s):\n%s\n\n",
		draftModel.Provider,
		draftModel.ID,
		draftModel.Protocol,
		assistantText(draft),
	)

	history = append(history, &draft)
	history = append(history, &llm.UserMessage{Content: []llm.UserContent{
		&llm.TextContent{Text: "Review the explanation for accuracy and improve it."},
	}})

	review, err := llm.Complete(
		ctx,
		reviewModel,
		llm.Context{Messages: history},
		llm.StreamOptions{Reasoning: llm.ModelThinkingHigh},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"Review from %s/%s (%s):\n%s\n",
		reviewModel.Provider,
		reviewModel.ID,
		reviewModel.Protocol,
		assistantText(review),
	)
}

func assistantText(message llm.AssistantMessage) string {
	var parts []string
	for _, content := range message.Content {
		if text, ok := content.(*llm.TextContent); ok && text != nil {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "")
}

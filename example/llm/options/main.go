// Command options sends a prompt with a system message and per-request options.
//
// It shows the next step after the basic example: shape the model's behavior
// with a system prompt, control generation with StreamOptions, and inspect
// response metadata such as token usage, cost, and stop reason.
//
// The API key is read from the provider's environment variable when
// StreamOptions.APIKey is empty:
//
//	DEEPSEEK_API_KEY=sk-... go run ./example/llm/options
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
	_ "github.com/ktsoator/or/llm/openai" // register the OpenAI-compatible protocol (DeepSeek speaks it)
)

func ptr[T any](v T) *T {
	return &v
}

func main() {
	model := llm.GetModel("deepseek", "deepseek-v4-flash")

	input := llm.PromptWithSystem(
		"You are a concise Go expert.",
		"How should I choose between channels and mutexes?",
	)

	msg, err := llm.Complete(context.Background(), model, input, llm.StreamOptions{
		Temperature: ptr(0.2),
		MaxTokens:   500,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(msg.Text())
	fmt.Printf(
		"\ntokens in=%d out=%d cost=$%.5f stop=%s\n",
		msg.Usage.Input,
		msg.Usage.Output,
		msg.Usage.Cost.Total,
		msg.StopReason,
	)
}

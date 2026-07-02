// Command basic sends a single prompt to a model and prints the reply.
//
// It is the smallest possible use of the llm package: resolve a model from the
// built-in catalog, build a one-message Context with Prompt, and let Complete
// run the request and return the final assistant message.
//
// The API key is read from the provider's environment variable when
// StreamOptions.APIKey is empty:
//
//	DEEPSEEK_API_KEY=sk-... go run ./example/llm/basic
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
	_ "github.com/ktsoator/or/llm/openai" // register the OpenAI-compatible protocol (DeepSeek speaks it)
)

func main() {
	model := llm.GetModel("deepseek", "deepseek-v4-flash")

	msg, err := llm.Complete(context.Background(), model,
		llm.Prompt("Explain goroutines in one sentence."),
		llm.StreamOptions{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(msg.Text())
}

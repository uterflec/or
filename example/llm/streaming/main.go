// Command streaming consumes a response as a live event stream instead of
// waiting for the final message.
//
// It shows the difference from the basic example: Stream returns a channel of
// Event values that report incremental progress. Here it prints each text delta
// as it arrives, then stops on the single terminal EventDone (or EventError).
// Complete is just this loop wrapped up when you only want the final message.
//
// The API key is read from the provider's environment variable when
// StreamOptions.APIKey is empty:
//
//	DEEPSEEK_API_KEY=sk-... go run ./example/llm/streaming
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

	events, err := llm.Stream(context.Background(), model,
		llm.Prompt("Write a short haiku about Go."),
		llm.StreamOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for event := range events {
		switch event.Type {
		case llm.EventTextDelta:
			fmt.Print(event.Delta) // incremental text, printed as it streams
		case llm.EventDone:
			fmt.Println() // terminal event: event.Message holds the final message
		case llm.EventError:
			log.Fatal(event.Err)
		}
	}
}

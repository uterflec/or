// Command complete demonstrates the smallest non-streaming model request.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
)

func main() {
	model := llm.GetModel("deepseek", "deepseek-v4-flash")
	response, err := llm.Complete(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{
			&llm.UserMessage{Content: []llm.UserContent{
				&llm.TextContent{Text: "Explain Go channels in three sentences."},
			}},
		}},
		llm.StreamOptions{},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, content := range response.Content {
		if text, ok := content.(*llm.TextContent); ok {
			fmt.Println(text.Text)
		}
	}
}

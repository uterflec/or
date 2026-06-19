# or
Choose the path from intent to action.

## Quick start

The `llm` package includes the built-in OpenAI-compatible and Anthropic protocol
adapters and can call `Complete` directly:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
)

func main() {
	model := llm.GetModel("deepseek", "deepseek-chat")
	response, err := llm.Complete(
		context.Background(),
		model,
		llm.Context{Messages: []llm.Message{
			&llm.UserMessage{Content: []llm.UserContent{
				&llm.TextContent{Text: "Explain Go channels briefly."},
			}},
		}},
		llm.StreamOptions{},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, block := range response.Content {
		if text, ok := block.(*llm.TextContent); ok {
			fmt.Println(text.Text)
		}
	}
}
```

Set the provider API key in the environment, for example
`DEEPSEEK_API_KEY`. Use `llm.NewClient` when an isolated built-in client is
needed.

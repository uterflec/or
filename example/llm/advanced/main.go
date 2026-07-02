// Command advanced shows two lower-level controls layered on a normal request:
//
//   - Observe the exact serialized HTTP request with the OnRequest hook. It fires
//     once per attempt, including retries.
//   - Force the model to call a tool with a protocol-specific ToolChoice, carried
//     on ProtocolOptions and validated against the target protocol before sending.
//
// The model comes from the built-in catalog. To reach an OpenAI-compatible
// endpoint that is not in the catalog, construct an llm.Model by hand instead and
// point its BaseURL at that endpoint (Provider still drives the API key lookup).
//
// The API key is read from the provider's environment variable when
// StreamOptions.APIKey is empty:
//
//	DEEPSEEK_API_KEY=sk-... go run ./example/llm/advanced
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
	_ "github.com/ktsoator/or/llm/openai" // register the OpenAI-compatible protocol (DeepSeek speaks it)
)

// WeatherArgs is the tool's argument schema.
type WeatherArgs struct {
	City string `json:"city" jsonschema:"description=City name,minLength=1"`
	Days int    `json:"days" jsonschema:"minimum=1,maximum=10"`
}

func main() {
	model := llm.GetModel("deepseek", "deepseek-v4-flash")

	weather := llm.MustTool[WeatherArgs]("get_weather", "Get a weather forecast for a city")

	input := llm.NewContext(llm.UserText("What's the weather in Beijing for the next 3 days?"))
	input.Tools = []llm.ToolDefinition{weather}

	msg, err := llm.Complete(context.Background(), model, input, llm.StreamOptions{
		// Observe the exact request body sent to the provider.
		OnRequest: func(method, url string, body []byte) {
			fmt.Printf(">> %s %s\n%s\n\n", method, url, body)
		},
		// Protocol-specific option: force the model to call a tool this turn.
		ProtocolOptions: &llm.OpenAICompletionsStreamOptions{
			ToolChoice: llm.OpenAIToolChoiceRequired,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, call := range msg.ToolCalls() {
		args, err := llm.DecodeToolCall[WeatherArgs](weather, call)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("model chose: get_weather(city=%q, days=%d)\n", args.City, args.Days)
	}
}

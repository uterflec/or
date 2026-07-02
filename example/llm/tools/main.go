// Command tools runs a streaming tool loop with reasoning: the model thinks,
// optionally calls a typed tool, sees the result, and continues until it gives a
// final answer.
//
// It combines the pieces the agent package automates into one loop:
//   - MustTool derives a JSON Schema from a Go struct.
//   - Stream with Reasoning surfaces the model's thinking (EventThinking*) and
//     answer (EventText*) live, turn by turn.
//   - EventDone carries the final assistant message; DecodeToolCall validates and
//     decodes any tool calls, and ToolResult feeds each outcome back.
//
// Wrap this loop with state, steering, and persistence and you have an agent.
//
// The API key is read from the provider's environment variable when
// StreamOptions.APIKey is empty:
//
//	DEEPSEEK_API_KEY=sk-... go run ./example/llm/tools
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ktsoator/or/llm"
	_ "github.com/ktsoator/or/llm/openai" // register the OpenAI-compatible protocol (DeepSeek speaks it)
)

// WeatherArgs is the tool's argument schema. A json field without omitempty is
// required; jsonschema tags add descriptions and constraints to the schema.
type WeatherArgs struct {
	City string `json:"city" jsonschema:"description=City name,minLength=1"`
	Days int    `json:"days" jsonschema:"minimum=1,maximum=10"`
}

func main() {
	model := llm.GetModel("deepseek", "deepseek-v4-flash")

	weather := llm.MustTool[WeatherArgs]("get_weather", "Get a weather forecast for a city")

	input := llm.NewContext(llm.UserText("What should I pack for a trip to Beijing over the next 3 days?"))
	input.Tools = []llm.ToolDefinition{weather}

	for turn := 1; ; turn++ {
		fmt.Printf("\n===== turn %d =====\n", turn)

		events, err := llm.Stream(context.Background(), model, input, llm.StreamOptions{
			Reasoning: llm.ModelThinkingHigh,
		})
		if err != nil {
			log.Fatal(err)
		}

		var final llm.AssistantMessage
		for event := range events {
			switch event.Type {
			case llm.EventThinkingStart:
				fmt.Print("[thinking] ")
			case llm.EventThinkingDelta:
				fmt.Print(event.Delta)
			case llm.EventTextStart:
				fmt.Print("\n[answer] ")
			case llm.EventTextDelta:
				fmt.Print(event.Delta)
			case llm.EventDone:
				fmt.Println()
				final = *event.Message // the assembled assistant turn
			case llm.EventError:
				log.Fatal(event.Err)
			}
		}
		input.Messages = append(input.Messages, &final) // record the assistant turn

		calls := final.ToolCalls()
		if len(calls) == 0 {
			return // no tool calls: the streamed answer above was final
		}

		for _, call := range calls {
			args, err := llm.DecodeToolCall[WeatherArgs](weather, call)
			if err != nil {
				// Feed the error back so the model can correct its arguments.
				input.Messages = append(input.Messages,
					llm.ToolResult(call.ID, call.Name, "invalid arguments: "+err.Error()))
				continue
			}

			// A real tool would do work here; this one returns a canned result.
			result := fmt.Sprintf("%s: sunny, around 25C for the next %d days", args.City, args.Days)
			fmt.Printf("[tool] get_weather(city=%q, days=%d) -> %s\n", args.City, args.Days, result)
			input.Messages = append(input.Messages, llm.ToolResult(call.ID, call.Name, result))
		}
	}
}

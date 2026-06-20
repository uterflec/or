# Reasoning and thinking

`StreamOptions.Reasoning` is a provider-neutral effort level. Each adapter maps
it to the target provider's native form—Anthropic adaptive or budget thinking,
or OpenAI-compatible reasoning fields—and clamps it to the levels supported by
the selected model. Non-reasoning models ignore it.

```go
options := llm.StreamOptions{Reasoning: llm.ModelThinkingHigh}
```

The accepted levels are:

- `ModelThinkingOff`
- `ModelThinkingMinimal`
- `ModelThinkingLow`
- `ModelThinkingMedium`
- `ModelThinkingHigh`
- `ModelThinkingXHigh`

`SupportedThinkingLevels` reports the levels a model accepts.
`ClampThinkingLevel` adjusts a requested level to the nearest supported one.

## Anthropic thinking display

On the Anthropic protocol, `ThinkingDisplay` controls how reasoning is returned
without changing whether the model reasons. `ThinkingDisplayOmitted` withholds
thinking text while retaining the signature needed for multi-turn tool use. It
is useful for applications that must not display reasoning content.

```go
options := llm.StreamOptions{
	Reasoning: llm.ModelThinkingHigh,
	ProtocolOptions: &llm.AnthropicStreamOptions{
		ThinkingDisplay: llm.ThinkingDisplayOmitted,
	},
}
```

Use `ThinkingDisplaySummarized` to request summarized thinking. While streaming,
visible reasoning arrives through `EventThinkingStart`, `EventThinkingDelta`,
and `EventThinkingEnd` before the answer text.

## Conversation continuity

Reasoning metadata needed by a provider—such as Anthropic signatures and
OpenRouter encrypted reasoning—is retained in assistant messages and replayed
when required by later tool calls. When the target model changes, the library
preserves, downgrades, or omits reasoning content according to compatibility.
See [Conversations](conversations.md) for model switching and persistence.

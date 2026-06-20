# Providers and models

The package includes two protocol adapters:

- `openai-completions`
- `anthropic-messages`

The catalog and compatibility layer explicitly configure these providers:

| Provider | Provider ID | Protocol | Environment variable |
|---|---|---|---|
| DeepSeek | `deepseek` | `openai-completions` | `DEEPSEEK_API_KEY` |
| MiniMax Global | `minimax` | `anthropic-messages` | `MINIMAX_API_KEY` |
| MiniMax China | `minimax-cn` | `anthropic-messages` | `MINIMAX_CN_API_KEY` |
| Xiaomi MiMo | `xiaomi` | `openai-completions` | `XIAOMI_API_KEY` or `MIMO_API_KEY` |
| Z.AI Global | `zai` | `openai-completions` | `ZAI_API_KEY` |
| Zhipu Coding Plan China | `zai-coding-cn` | `openai-completions` | `ZAI_CODING_CN_API_KEY` |
| Moonshot AI Global | `moonshotai` | `openai-completions` | `MOONSHOT_API_KEY` |
| Moonshot AI China | `moonshotai-cn` | `openai-completions` | `MOONSHOT_API_KEY` |
| Kimi Coding | `kimi-coding` | `anthropic-messages` | `KIMI_API_KEY` |

The catalog also contains metadata for additional compatible providers and
models. Those entries can be queried and may work through one of the two
protocol adapters, but they have not all been verified against live provider
APIs and are not a support guarantee. Automated tests exercise both adapters
with local mock servers rather than live integration tests for every provider.

Only the key for the provider selected by `llm.GetModel` is read. Request-scoped
credentials can also be supplied with `StreamOptions.APIKey` or
`StreamOptions.Env`.

## Discover models

Query the catalog instead of hard-coding model IDs supplied dynamically:

```go
for _, provider := range llm.GetProviders() {
	fmt.Println(provider)
	for _, model := range llm.GetModels(provider) {
		fmt.Printf("  %s: %s\n", model.ID, model.Name)
	}
}

model, ok := llm.LookupModel("xiaomi", "mimo-v2-flash")
if !ok {
	log.Fatal("model not found")
}
```

`LookupModel` returns a model and a found flag. `GetModel` is convenient for a
known catalog entry and panics when the provider or model ID does not exist.
Model metadata includes reasoning and image support, context windows, output
limits, and pricing information.

## Custom and compatible endpoints

Any endpoint implementing one of the built-in protocols can be used by
constructing a `Model` directly and setting `BaseURL`. This covers local servers
such as Ollama, vLLM, and LM Studio, as well as private model gateways:

```go
model := llm.Model{
	ID:            "qwen2.5-coder:7b",
	Name:          "Qwen2.5 Coder 7B",
	Provider:      "ollama",
	Protocol:      llm.ProtocolOpenAICompletions,
	BaseURL:       "http://localhost:11434/v1",
	Input:         []llm.ModelInput{llm.Text},
	ContextWindow: 32768,
	MaxTokens:     4096,
}

events, err := llm.Stream(ctx, model, input, llm.StreamOptions{APIKey: "ollama"})
```

Endpoint-specific behavior—reasoning field names, cache-control support, and
similar differences—is configured through `Model.Compatibility` with
`OpenAICompletionsCompatibility` or `AnthropicMessagesCompatibility`.

For a wire protocol that is neither OpenAI-compatible nor
Anthropic-compatible, implement a [custom protocol adapter](extending.md).

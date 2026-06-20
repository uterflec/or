# LLM package

`github.com/ktsoator/or/llm` provides one Go API for streaming responses,
structured tools, reasoning content, multimodal messages, and conversation
history across OpenAI-compatible and Anthropic-compatible models.

## Install

```sh
go get github.com/ktsoator/or/llm@latest
```

## Documentation

- [Getting started](getting-started.md) — credentials and your first request
- [Providers and models](providers.md) — catalog discovery and custom endpoints
- [Streaming](streaming.md) — events, partial responses, diagnostics, and cancellation
- [Tools](tools.md) — typed tools and protocol-specific tool choice
- [Reasoning](reasoning.md) — effort levels and thinking display
- [Conversations](conversations.md) — images, model switching, and persistence
- [Configuration](configuration.md) — retries, timeouts, headers, and HTTP hooks
- [Custom protocols](extending.md) — adapters, registries, and `StreamWriter`

For exported types and functions, see the package documentation on
[pkg.go.dev](https://pkg.go.dev/github.com/ktsoator/or/llm).

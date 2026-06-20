# LLM examples

These programs demonstrate focused features of `github.com/ktsoator/or/llm`.
They call live model APIs and may incur provider charges.

Set the environment variables required by each example before running it. The
programs intentionally use the process environment directly so applications
can choose their own configuration or secret-management library.

| Example | What it demonstrates | Required environment variables |
|---|---|---|
| [`complete`](complete) | A minimal non-streaming completion | `DEEPSEEK_API_KEY` |
| [`stream`](stream) | Text, reasoning, terminal events, and usage | `DEEPSEEK_API_KEY` |
| [`tool`](tool) | Typed tools, validation, diagnostics, and a tool loop | `DEEPSEEK_API_KEY` |
| [`image`](image) | Sending a local image to a multimodal model | `XIAOMI_API_KEY` |
| [`model-switching`](model-switching) | Reusing one history across two protocols | `DEEPSEEK_API_KEY`, `MINIMAX_CN_API_KEY` |

For example:

```sh
export DEEPSEEK_API_KEY=your-deepseek-api-key
go run ./example/llm/complete
```

The image example also expects a local file path:

```sh
export XIAOMI_API_KEY=your-xiaomi-api-key
go run ./example/llm/image ./screenshot.png
```

See the [LLM documentation](../../docs/llm/README.md) for detailed guides and
API behavior.

# Conversations

Conversation messages are provider-neutral. The same history can be persisted,
extended, and sent to another compatible model without rebuilding it.

## Image input

Multimodal models accept images alongside text in a user message. Provide the
raw bytes as base64 with their MIME type:

```go
raw, err := os.ReadFile("screenshot.png")
if err != nil {
	log.Fatal(err)
}
input := llm.Context{Messages: []llm.Message{
	&llm.UserMessage{Content: []llm.UserContent{
		&llm.TextContent{Text: "Describe the problem shown in this screenshot."},
		&llm.ImageContent{
			MIMEType: "image/png",
			Data:     base64.StdEncoding.EncodeToString(raw),
		},
	}},
}}
```

A model declares image support through `Model.Input`. When a history containing
images is sent to a text-only model, images are replaced with a short
placeholder automatically.

## Switch models between turns

Before each request, the library adapts stored history for the target model. It
downgrades images for text-only models, preserves reasoning signatures where
compatible, downgrades or removes incompatible reasoning, and normalizes tool
call identifiers.

```go
ctx := context.Background()
draft := llm.GetModel("deepseek", "deepseek-v4-flash")
review := llm.GetModel("anthropic", "claude-opus-4-8")

messages := []llm.Message{
	&llm.UserMessage{Content: []llm.UserContent{
		&llm.TextContent{Text: "Compute 25 * 18 and explain the steps."},
	}},
}

first, err := llm.Complete(ctx, draft,
	llm.Context{Messages: messages}, llm.StreamOptions{})
if err != nil {
	log.Fatal(err)
}
messages = append(messages, &first)
messages = append(messages, &llm.UserMessage{Content: []llm.UserContent{
	&llm.TextContent{Text: "Check the calculation above for mistakes."},
}})

second, err := llm.Complete(ctx, review,
	llm.Context{Messages: messages}, llm.StreamOptions{})
if err != nil {
	log.Fatal(err)
}
```

`TransformMessages` performs this adaptation and is exported for callers that
need to inspect the exact history a model would receive.

## Save and restore conversations

`Context` serializes to self-describing JSON: messages carry a role and content
blocks carry a type, so JSON round-trips into concrete message and content types
without manual dispatch.

```go
data, err := json.MarshalIndent(llm.Context{Messages: messages}, "", "  ")
if err != nil {
	log.Fatal(err)
}
if err := os.WriteFile("conversation.json", data, 0o644); err != nil {
	log.Fatal(err)
}

raw, err := os.ReadFile("conversation.json")
if err != nil {
	log.Fatal(err)
}
var restored llm.Context
if err := json.Unmarshal(raw, &restored); err != nil {
	log.Fatal(err)
}
```

`restored.Messages` is ready to extend and replay against any model.

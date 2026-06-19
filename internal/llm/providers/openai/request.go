package openai

import (
	"maps"
	"net/http"

	"github.com/ktsoator/or/internal/llm"
	oai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// buildClient creates one SDK client for the target model endpoint. Model
// headers provide defaults and request headers override values with the same
// name.
func buildClient(httpClient *http.Client, model llm.Model, options llm.StreamOptions) oai.Client {
	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
		option.WithHTTPClient(httpClient),
		// Drop non-compliant SSE keep-alive heartbeats so providers like Xiaomi
		// MiMo do not break the SDK decoder while the model is thinking.
		option.WithMiddleware(sseHeartbeatFilter),
	}
	if model.BaseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(model.BaseURL))
	}
	if options.MaxRetries != nil {
		clientOptions = append(clientOptions, option.WithMaxRetries(*options.MaxRetries))
	}
	if options.Timeout > 0 {
		clientOptions = append(clientOptions, option.WithRequestTimeout(options.Timeout))
	}
	for name, value := range mergedHeaders(model, options) {
		clientOptions = append(clientOptions, option.WithHeader(name, value))
	}
	return oai.NewClient(clientOptions...)
}

// buildParams translates provider-independent request options into OpenAI Chat
// Completions parameters using the model's resolved compatibility dialect.
func buildParams(
	model llm.Model,
	messages []oai.ChatCompletionMessageParamUnion,
	tools []oai.ChatCompletionToolUnionParam,
	options llm.StreamOptions,
	compat resolvedCompat,
) oai.ChatCompletionNewParams {
	params := oai.ChatCompletionNewParams{
		Model:    model.ID,
		Messages: messages,
		StreamOptions: oai.ChatCompletionStreamOptionsParam{
			IncludeUsage: oai.Bool(true),
		},
	}
	if len(tools) > 0 {
		params.Tools = tools
	}
	if options.MaxTokens > 0 {
		if compat.maxTokensField == "max_tokens" {
			params.MaxTokens = oai.Int(options.MaxTokens)
		} else {
			params.MaxCompletionTokens = oai.Int(options.MaxTokens)
		}
	}
	if options.Temperature != nil {
		params.Temperature = oai.Float(*options.Temperature)
	}
	if compat.supportsStore {
		params.Store = oai.Bool(false)
	}
	applyThinking(&params, model, compat, resolveEffort(model, options.Reasoning))
	if len(tools) > 0 && compat.zaiToolStream {
		mergeExtraFields(&params, map[string]any{"tool_stream": true})
	}
	return params
}

// mergeExtraFields preserves provider-specific fields already installed by
// applyThinking. The SDK's SetExtraFields replaces rather than merges its map.
func mergeExtraFields(params *oai.ChatCompletionNewParams, fields map[string]any) {
	merged := maps.Clone(params.ExtraFields())
	if merged == nil {
		merged = make(map[string]any, len(fields))
	}
	maps.Copy(merged, fields)
	params.SetExtraFields(merged)
}

func mergedHeaders(model llm.Model, options llm.StreamOptions) map[string]string {
	if len(model.Headers) == 0 && len(options.Headers) == 0 {
		return nil
	}
	merged := make(map[string]string, len(model.Headers)+len(options.Headers))
	maps.Copy(merged, model.Headers)
	maps.Copy(merged, options.Headers)
	return merged
}

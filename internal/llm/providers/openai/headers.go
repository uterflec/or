package openai

import (
	"maps"

	"github.com/ktsoator/or/internal/llm"
)

// mergedHeaders combines model defaults with request-scoped headers. Request
// values win when both maps contain the same header name.
func mergedHeaders(model llm.Model, options llm.StreamOptions) map[string]string {
	if len(model.Headers) == 0 && len(options.Headers) == 0 {
		return nil
	}
	merged := make(map[string]string, len(model.Headers)+len(options.Headers))
	maps.Copy(merged, model.Headers)
	for name, value := range options.Headers {
		merged[name] = value
	}
	return merged
}

package cache

import (
	"encoding/json"
	"sort"

	"github.com/gaea/gaea/internal/gaea/provider"
)

type canonicalToolSpec struct {
	name        string
	description string
	schema      json.RawMessage
}

// NormalizeToolSchemas sorts and canonicalizes tool schemas for fingerprinting.
func NormalizeToolSchemas(tools []provider.ToolSchema) []canonicalToolSpec {
	out := make([]canonicalToolSpec, len(tools))
	for i, t := range tools {
		out[i] = canonicalToolSpec{
			name:        t.Name,
			description: t.Description,
			schema:      canonicalizeJSON(t.Parameters),
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// CanonicalizeValue recursively sorts JSON object keys for canonical output.
func CanonicalizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = CanonicalizeValue(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = CanonicalizeValue(val)
		}
		return out
	default:
		return v
	}
}

func canonicalizeJSON(raw json.RawMessage) json.RawMessage {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	canonical := CanonicalizeValue(value)
	result, err := json.Marshal(canonical)
	if err != nil {
		return raw
	}
	return result
}

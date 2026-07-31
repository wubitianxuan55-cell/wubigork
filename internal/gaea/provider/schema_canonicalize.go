package provider

import (
	"encoding/json"
	"sort"
)

// compressSchemaMaxDepth limits recursion when stripping redundant fields from
// deeply nested schemas (multi_edit / complete_step have nested arrays).
const compressSchemaMaxDepth = 5

// compressSchema strips redundant fields from a JSON Schema to reduce prompt
// token count.
func compressSchema(v any, depth int) any {
	if depth > compressSchemaMaxDepth {
		return v
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return v
	}
	// Strip all description fields — they mirror the tool-level Description()
	// and only inflate per-turn prompt tokens. Applies at every nesting level.
	delete(obj, "description")
	// Do NOT remove "type":"object" — DeepSeek API requires it.
	// Remove "type":"string" from property values (safe: implied when omitted)
	if prop, hasProps := obj["properties"]; hasProps {
		if props, ok := prop.(map[string]any); ok {
			for key, pv := range props {
				if pm, ok := pv.(map[string]any); ok {
					if pt, hasPT := pm["type"]; hasPT && pt == "string" {
						delete(pm, "type")
					}
					if len(pm) == 0 {
						props[key] = struct{}{}
					} else {
						props[key] = compressSchema(pm, depth+1)
					}
				}
			}
		}
		if items, hasItems := obj["items"]; hasItems {
			obj["items"] = compressSchema(items, depth+1)
		}
	}
	// Remove empty "required":[]
	if req, hasReq := obj["required"]; hasReq {
		if arr, ok := req.([]any); ok && len(arr) == 0 {
			delete(obj, "required")
		}
	}
	return obj
}

// CanonicalizeSchema recursively stabilizes a JSON Schema so the same logical
// fallbackObjectSchema 是非法 JSON Schema 的兜底：无参数对象。保证任何进入
// registry 的 schema 都是合法 JSON——否则 MCP 服务器返回的非法 inputSchema
// 会在发送消息时导致 json.Marshal 崩溃
// （"json: error calling MarshalJSON for type json.RawMessage"）。
var fallbackObjectSchema = json.RawMessage(`{"type":"object"}`)

func CanonicalizeSchema(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// 非法输入绝不原样透传：替换为无参数对象兜底，
		// 让对话继续而非整条消息失败。
		return fallbackObjectSchema
	}
	canon := canonicalizeSchemaValue(v)
	b, err := json.Marshal(canon)
	if err != nil {
		return fallbackObjectSchema
	}
	return json.RawMessage(b)
}

var setLikeSchemaArrays = map[string]bool{
	"required":          true,
	"dependentRequired": true,
}

func canonicalizeSchemaValue(v any) any {
	v = compressSchema(v, 0)
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			val[k] = canonicalizeSchemaValue(inner)
		}
		for key := range val {
			if setLikeSchemaArrays[key] {
				if arr, ok := val[key].([]any); ok {
					sort.SliceStable(arr, func(i, j int) bool {
						return schemaJSONString(arr[i]) < schemaJSONString(arr[j])
					})
				}
			}
		}
		if dr, ok := val["dependentRequired"]; ok {
			if drMap, ok := dr.(map[string]any); ok {
				for _, inner := range drMap {
					if arr, ok := inner.([]any); ok {
						sort.SliceStable(arr, func(i, j int) bool {
							return schemaJSONString(arr[i]) < schemaJSONString(arr[j])
						})
					}
				}
			}
		}
		return val
	case []any:
		for i, elem := range val {
			val[i] = canonicalizeSchemaValue(elem)
		}
		return val
	default:
		return v
	}
}

func schemaJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

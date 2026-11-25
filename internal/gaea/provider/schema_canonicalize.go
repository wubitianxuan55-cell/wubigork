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

// CanonicalizeSchemaVerbose 是 CanonicalizeSchema 的**保留描述**版：各层字段
// description 原样下发，其余规范化（required 稳定排序、空 required 删除）照做。
//
// 为什么需要它（2026-11-25 铜陵六表反馈 #13）：CanonicalizeSchema 的前提是
// 「字段描述与工具级 Description 重复」，对单动作工具成立；但多动作工具
// （一个 action 一套嵌套入参）的参数语义只写在字段描述里，剥掉后模型只能看
// 键名猜结构——实测把 dict/classify 塞进 quota、不知道 import_project 也吃
// specialty/region，单点浪费 9 轮。这类工具经 tool.SchemaDescriptionsKeeper
// 显式选用本函数；代价是 schema 提示词体积变大，所以不默认。
func CanonicalizeSchemaVerbose(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fallbackObjectSchema
	}
	b, err := json.Marshal(canonicalizeVerboseValue(v))
	if err != nil {
		return fallbackObjectSchema
	}
	return json.RawMessage(b)
}

// canonicalizeVerboseValue 递归稳定化但不动 description / type。
func canonicalizeVerboseValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			val[k] = canonicalizeVerboseValue(inner)
		}
		sortSetLikeArrays(val)
		return val
	case []any:
		for i, elem := range val {
			val[i] = canonicalizeVerboseValue(elem)
		}
		return val
	default:
		return v
	}
}

func canonicalizeSchemaValue(v any) any {
	v = compressSchema(v, 0)
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			val[k] = canonicalizeSchemaValue(inner)
		}
		sortSetLikeArrays(val)
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

// sortSetLikeArrays 把语义上是集合的数组（required / dependentRequired 的值）
// 排成稳定序，同一逻辑 schema 得到逐字节相同的下发文本。
func sortSetLikeArrays(val map[string]any) {
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
}

func schemaJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

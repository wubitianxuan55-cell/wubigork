package app

import (
	"encoding/json"
	"log/slog"
	"strings"
)

// classifyMainBrainIntent 规则识别主脑意图：关键词命中模块与意图；默认 gaea.chat。
// 主脑是可选编排入口，不经由任何模块的直接路径。
func classifyMainBrainIntent(msg string) (moduleID, intent string) {
	lower := strings.ToLower(msg)
	switch {
	case containsAny(lower, "标书", "招标", "方案", "报价", "proposal", "tender"):
		return "office", "create"
	case containsAny(lower, "章节", "小说", "大纲", "角色", "章", "chapter", "novel"):
		return "novel", "create_chapter"
	case containsAny(lower, "轻语", "聊天", "陪", "whisper"):
		return "whisper", "chat"
	case containsAny(lower, "画", "图", "生图", "绘梦", "image", "generate"):
		return "imagegen", "generate"
	default:
		return "gaea", "chat"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// MainBrainChat 主脑统一入口（可选编排）：规则识别意图 → BrainSearch 取两脑材料 → 派发模块 → 汇总。
func (a *App) MainBrainChat(message string) (string, error) {
	moduleID, intent := classifyMainBrainIntent(message)
	var materials []Hit
	if a.brain != nil {
		materials, _ = a.brain.Search(message)
		if len(materials) > 5 {
			materials = materials[:5]
		}
	}
	result := map[string]any{
		"module": moduleID, "intent": intent, "materials": materials,
	}
	if a.modules != nil && a.modules.Has(moduleID) {
		out, err := a.modules.Dispatch(moduleID, intent, map[string]any{
			"message": message, "title": message, "requirements": message, "prompt": message,
		})
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["output"] = out
			if text, ok := out["reply"].(string); ok {
				result["reply"] = text
			}
		}
	} else {
		// 缺陷 2 修复：模块未注册时不再静默跳过，记录告警便于排查（D8）。
		slog.Warn("主脑: 模块未注册，跳过派发", "module", moduleID, "intent", intent)
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

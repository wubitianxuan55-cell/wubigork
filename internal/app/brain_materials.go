package app

import "fmt"

// brainLabel 三脑显示名。
func brainLabel(brain string) string {
	switch brain {
	case BrainRight:
		return "右脑"
	case BrainLeft:
		return "左脑"
	default:
		return "主脑"
	}
}

// buildBrainMaterials 从三脑检索关键词相关记忆，格式化为注入文本（去重、最多 3 条）。
func buildBrainMaterials(b *BrainStore, keywords ...string) []string {
	if b == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		hits, _ := b.Search(kw)
		for _, h := range hits {
			key := h.Brain + "|" + h.Entity + "|" + h.Text
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, fmt.Sprintf("【跨脑记忆·%s】%s：%s", brainLabel(h.Brain), h.Entity, h.Text))
			if len(out) >= 3 {
				return out
			}
		}
	}
	return out
}

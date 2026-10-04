package strutil

import (
	"encoding/json"
	"strings"
)

// ParseTagsJSON parses a JSON string array into a tag slice, best-effort:
// empty / whitespace-only / "[]" / malformed input all yield nil (never an
// error — callers treat tags as decorative metadata, not data to validate).
//
// 收敛自两份逐字相同的本地实现（2026-10-04 第三轮审计 §3.2）：
// cost.parseTagsJSON 与 knowledge.parseTagsJSON。语义冻结：输入整体不合法
// （含尾部垃圾）时 Unmarshal 返回 error 且结果保持 nil，此处与旧实现一致
// 地忽略 error、按 nil 返回。
func ParseTagsJSON(raw string) []string {
	var tags []string
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

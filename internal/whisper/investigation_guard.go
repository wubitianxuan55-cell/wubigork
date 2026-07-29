// Package whisper — investigation_guard.go
// 100% 对齐 ackem desktop-agent/investigation/
// 调查子系统：幻觉守卫 + 检查清单 + 意图路由
package whisper

import (
	"fmt"
	"strings"
)

// ─── 幻觉守卫 ────────────────────────────────────────────────────

// vaguePhrases 敷衍句黑名单
var vaguePhrases = []string{
	"自己打开看看", "没帮你扫", "没找到任何", "什么都没有",
	"不太清楚", "不确定", "好像没有", "应该没有",
	"你自己去", "我不太了解", "没看到",
}

// DetectHallucination 检测 LLM 回复中的幻觉/敷衍
// 返回 (是否可疑, 原因)
func DetectHallucination(llmReply string, findings []string) (bool, string) {
	reply := strings.ToLower(llmReply)

	// 检查敷衍句
	for _, phrase := range vaguePhrases {
		if strings.Contains(reply, phrase) {
			return true, "检测到敷衍表述：" + phrase
		}
	}

	// 有 findings 但回复声称没有
	for _, f := range findings {
		if f != "" && strings.Contains(reply, "没找到") && !strings.Contains(reply, strings.ToLower(f)) {
			return true, "有实际发现但回复声称没找到"
		}
	}

	return false, ""
}

// FormatFindingsFallbackReply 当 LLM 输出不合格时，生成结构化 fallback
func FormatFindingsFallbackReply(findings []string, unscannedDirs []string) string {
	var sb strings.Builder
	sb.WriteString("根据电脑扫描结果：\n\n")

	if len(findings) > 0 {
		sb.WriteString("**已找到：**\n")
		for _, f := range findings {
			if f != "" {
				sb.WriteString("· " + f + "\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(unscannedDirs) > 0 {
		sb.WriteString("**未扫描的位置：**\n")
		for _, d := range unscannedDirs {
			sb.WriteString("· " + d + "\n")
		}
	}

	return sb.String()
}

// ─── 检查清单 ────────────────────────────────────────────────────

// InvestigationChecklist 调查清单
type InvestigationChecklist struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}

// CreateGamesChecklist 生成游戏调查清单
func CreateGamesChecklist() InvestigationChecklist {
	return InvestigationChecklist{
		Title: "游戏调查",
		Steps: []string{
			"扫描桌面快捷方式",
			"扫描开始菜单",
			"扫描 Program Files",
			"扫描 Program Files (x86)",
			"扫描本地 Programs",
			"扫描 Steam 游戏库",
			"扫描 Epic 游戏库",
		},
	}
}

// CreateDocumentsChecklist 生成文档调查清单
func CreateDocumentsChecklist() InvestigationChecklist {
	return InvestigationChecklist{
		Title: "文档调查",
		Steps: []string{
			"扫描桌面文件",
			"扫描文档文件夹",
			"扫描下载文件夹",
		},
	}
}

// ChecklistProgressLabel 生成调查进度标签
func ChecklistProgressLabel(checklist InvestigationChecklist, completed int) string {
	total := len(checklist.Steps)
	pct := 0
	if total > 0 {
		pct = completed * 100 / total
	}
	current := ""
	if completed < total {
		current = checklist.Steps[completed]
	}
	return fmt.Sprintf("电脑助手查找中 · %d/%d (%d%%) · %s", completed, total, pct, current)
}

// ─── 意图路由 ────────────────────────────────────────────────────

// InvestigationIntent 调查意图
type InvestigationIntent struct {
	Type      string `json:"type"`       // filesystem_inventory / filesystem_search / none
	Category  string `json:"category"`   // games / documents / generic
	Confidence float64 `json:"confidence"`
}

// RouteInvestigationIntent 根据用户查询判断是否需要调查
func RouteInvestigationIntent(userMsg string) InvestigationIntent {
	lower := strings.ToLower(userMsg)

	// 游戏相关
	gameWords := []string{"游戏", "玩什么", "steam", "epic", "网游", "单机", "手游", "电竞", "rpg", "fps"}
	for _, w := range gameWords {
		if strings.Contains(lower, w) {
			return InvestigationIntent{Type: "filesystem_inventory", Category: "games", Confidence: 0.9}
		}
	}

	// 文档相关
	docWords := []string{"文档", "文件", "简历", "日记", "周报", "报告", "ppt", "pdf", "word", "excel"}
	for _, w := range docWords {
		if strings.Contains(lower, w) {
			return InvestigationIntent{Type: "filesystem_search", Category: "documents", Confidence: 0.8}
		}
	}

	// 通用目录查询
	dirWords := []string{"有什么", "安装了", "电脑上", "帮我找", "帮我查", "看看", "扫描"}
	for _, w := range dirWords {
		if strings.Contains(lower, w) {
			return InvestigationIntent{Type: "filesystem_inventory", Category: "generic", Confidence: 0.5}
		}
	}

	return InvestigationIntent{Type: "none", Confidence: 0}
}

// ShouldSkipInventoryRouting 过滤不应触发的查询
func ShouldSkipInventoryRouting(userMsg string) bool {
	skipWords := []string{"天气", "几点了", "你好", "再见", "晚安", "早安", "谢谢", "哈哈", "嗯"}
	lower := strings.ToLower(userMsg)
	for _, w := range skipWords {
		if lower == w || strings.HasPrefix(lower, w) {
			return true
		}
	}
	return false
}

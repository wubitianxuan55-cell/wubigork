// Package whisper — desktop_capability_routing.go
// 100% 对齐 ackem desktop-agent/routing/resolveCapability.ts
// 增强版桌面助手能力路由：双阶段关键词+正则 → 置信度评分 → 权限过滤
//
// 路由策略（两级降级）：
//   1. 关键词评分路由：exampleQueries → substring 匹配 → 累加分数
//   2. 正则 fallback：清理意图/调查意图/能力帮助 → 结构化匹配

package whisper

import (
	"strings"
)

// ─── 能力定义 ──────────────────────────────────────────────────

// DesktopCapabilityDef 桌面助手能力定义（含 exampleQueries）
type DesktopCapabilityDef struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	Handler       string   `json:"handler"`
	ExampleQueries []string `json:"exampleQueries"`
	Enabled       bool     `json:"enabled"`
}

// ListRoutableCapabilities 可路由能力清单
// 100% 对齐 ackem shared/desktopAgentCapabilities.ts
func ListRoutableCapabilities() []DesktopCapabilityDef {
	return []DesktopCapabilityDef{
		{
			ID: "list_folder", Label: "浏览目录", Handler: "list_folder", Enabled: true,
			ExampleQueries: []string{
				"列出桌面文件", "看看C盘有什么", "浏览下载目录",
				"显示文件夹内容", "查看目录", "桌面有什么",
				"list desktop files", "show folder contents",
			},
		},
		{
			ID: "search_files", Label: "搜索文件", Handler: "search_files", Enabled: true,
			ExampleQueries: []string{
				"搜索文件", "找一下PDF", "查找文档", "帮我找",
				"搜索电脑上的", "找文件", "find files",
			},
		},
		{
			ID: "read_text", Label: "读取文件", Handler: "read_text", Enabled: true,
			ExampleQueries: []string{
				"读取文件", "打开文本", "查看内容", "显示文件",
				"read file", "show file content",
			},
		},
		{
			ID: "open_app", Label: "打开应用", Handler: "open_app", Enabled: true,
			ExampleQueries: []string{
				"打开浏览器", "启动微信", "运行程序", "打开应用",
				"open app", "launch program", "启动",
			},
		},
		{
			ID: "close_app", Label: "关闭应用", Handler: "close_app", Enabled: true,
			ExampleQueries: []string{
				"关闭浏览器", "退出程序", "结束进程", "关掉",
				"close app", "quit program", "关闭",
			},
		},
		{
			ID: "investigate_games", Label: "游戏清单", Handler: "investigate_games", Enabled: true,
			ExampleQueries: []string{
				"我电脑有什么游戏", "装了哪些游戏", "看看游戏",
				"有哪些游戏", "游戏列表", "steam游戏",
				"电脑游戏", "what games", "list games",
				"帮我查一下游戏", "扫一下游戏",
			},
		},
		{
			ID: "investigate_documents", Label: "文件搜索", Handler: "investigate_documents", Enabled: true,
			ExampleQueries: []string{
				"我有什么文档", "找一下文件", "搜索PDF",
				"桌面有什么文档", "文档列表", "有哪些文件",
				"find documents", "list files",
			},
		},
		{
			ID: "capability_help", Label: "能力说明", Handler: "capability_help", Enabled: true,
			ExampleQueries: []string{
				"你能做什么", "有什么功能", "能帮我干嘛",
				"你会什么", "电脑助手功能", "怎么用",
				"what can you do", "help",
			},
		},
		{
			ID: "organize_files", Label: "整理文件", Handler: "organize_files", Enabled: true,
			ExampleQueries: []string{
				"整理桌面", "收拾文件", "清理文件夹",
				"归类文件", "organize files",
			},
		},
	}
}

// ─── 路由结果 ──────────────────────────────────────────────────

// DesktopCapabilityMatch 能力匹配结果
type DesktopCapabilityMatch struct {
	CapabilityID string  `json:"capabilityId"`
	Label        string  `json:"label"`
	Handler      string  `json:"handler"`
	Score        float64 `json:"score"`
	Source       string  `json:"source"` // "keyword" | "regex_fallback"
}

// ─── 主路由入口 ────────────────────────────────────────────────

// ResolveDesktopCapabilityEnhanced 增强版桌面能力路由
// 100% 对齐 ackem resolveCapability.ts resolveDesktopAgentCapability
func ResolveDesktopCapabilityEnhanced(userText string) *DesktopCapabilityMatch {
	msg := strings.TrimSpace(userText)
	if msg == "" {
		return nil
	}

	// 阶段1：关键词评分路由
	if match := matchByExampleQueries(msg); match != nil {
		// 特殊覆盖：清理意图覆盖 investigate_*
		if cleanupIntents[strings.ToLower(msg)] || isCleanupIntent(msg) {
			if match.Handler == "investigate_games" || match.Handler == "investigate_documents" {
				return &DesktopCapabilityMatch{
					CapabilityID: "organize_files",
					Label:        "整理文件",
					Handler:      "organize_files",
					Score:        0.55,
					Source:       "regex_fallback",
				}
			}
		}
		return match
	}

	// 阶段2：正则 fallback
	if match := matchByRegexFallback(msg); match != nil {
		return match
	}

	return nil
}

// ─── 关键词评分路由 ────────────────────────────────────────────

const midConfidenceThreshold = 0.4

// matchByExampleQueries 关键词评分匹配
func matchByExampleQueries(msg string) *DesktopCapabilityMatch {
	lower := strings.ToLower(msg)
	caps := ListRoutableCapabilities()

	var bestMatch *DesktopCapabilityMatch
	bestScore := 0.0

	for _, cap := range caps {
		if !cap.Enabled {
			continue
		}
		score := scoreByExamples(lower, cap.ExampleQueries)
		if score > bestScore && score >= midConfidenceThreshold {
			bestScore = score
			bestMatch = &DesktopCapabilityMatch{
				CapabilityID: cap.ID,
				Label:        cap.Label,
				Handler:      cap.Handler,
				Score:        score,
				Source:       "keyword",
			}
		}
	}

	return bestMatch
}

// scoreByExamples 计算用户输入与 exampleQueries 的匹配分数
func scoreByExamples(msg string, examples []string) float64 {
	score := 0.0
	for _, ex := range examples {
		lowerEx := strings.ToLower(ex)
		if strings.Contains(msg, lowerEx) {
			// 精确匹配得满分
			return 1.0
		}
		// 部分匹配（至少2个连续汉字匹配）
		if partialMatch(msg, lowerEx) {
			score += 0.3
		}
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// partialMatch 部分匹配：至少2个连续字符共同
func partialMatch(a, b string) bool {
	runesA := []rune(a)
	for i := 0; i < len(runesA)-1; i++ {
		bigram := string(runesA[i : i+2])
		if len([]rune(bigram)) >= 2 && strings.Contains(b, bigram) {
			return true
		}
	}
	return false
}

// ─── 正则 Fallback ─────────────────────────────────────────────

// cleanupIntents 清理意图关键词
var cleanupIntents = map[string]bool{
	"整理桌面": true, "收拾文件": true, "清理": true,
	"归类": true, "分类": true, "整理": true,
}

// isCleanupIntent 检测清理意图
func isCleanupIntent(msg string) bool {
	lower := strings.ToLower(msg)
	patterns := []string{"整理", "收拾", "清理", "归类", "分类", "organize", "clean"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// matchByRegexFallback 正则/关键词 fallback 路由
func matchByRegexFallback(msg string) *DesktopCapabilityMatch {
	lower := strings.ToLower(msg)

	// 1. 清理意图
	if isCleanupIntent(msg) {
		return &DesktopCapabilityMatch{
			CapabilityID: "organize_files",
			Label:        "整理文件",
			Handler:      "organize_files",
			Score:        0.55,
			Source:       "regex_fallback",
		}
	}

	// 2. 游戏调查意图
	gamePatterns := []string{"游戏", "玩什么", "装了哪些游戏", "steam", "epic", "有什么游戏", "list games"}
	for _, p := range gamePatterns {
		if strings.Contains(lower, p) {
			return &DesktopCapabilityMatch{
				CapabilityID: "investigate_games",
				Label:        "游戏清单",
				Handler:      "investigate_games",
				Score:        0.5,
				Source:       "regex_fallback",
			}
		}
	}

	// 3. 文档调查意图
	docPatterns := []string{"文档", "文件", "pdf", "word", "excel", "ppt", "有什么文件", "list files"}
	for _, p := range docPatterns {
		if strings.Contains(lower, p) {
			return &DesktopCapabilityMatch{
				CapabilityID: "investigate_documents",
				Label:        "文件搜索",
				Handler:      "investigate_documents",
				Score:        0.5,
				Source:       "regex_fallback",
			}
		}
	}

	// 4. 能力帮助
	helpPatterns := []string{"能做什么", "你会什么", "有什么功能", "电脑助手.*功能", "怎么用", "what can you do"}
	for _, p := range helpPatterns {
		if strings.Contains(lower, p) {
			return &DesktopCapabilityMatch{
				CapabilityID: "capability_help",
				Label:        "能力说明",
				Handler:      "capability_help",
				Score:        0.5,
				Source:       "regex_fallback",
			}
		}
	}

	return nil
}

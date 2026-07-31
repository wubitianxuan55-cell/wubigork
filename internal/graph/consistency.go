package graph

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/project"
)

// ── 一致性守护 ───────────────────────────────────────────────

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	Severity    string `json:"severity"` // error / warning / info
	Category    string `json:"category"` // attribute / timeline / status / relationship
	EntityName  string `json:"entity_name"`
	Description string `json:"description"`
	Location    string `json:"location"`    // 发现问题的章节
	Evidence    string `json:"evidence"`    // 证据
	Suggestion  string `json:"suggestion"`  // 修复建议
}

// ConsistencyReport 一致性检查报告
type ConsistencyReport struct {
	Issues      []ConsistencyIssue `json:"issues"`
	TotalIssues int                `json:"total_issues"`
	Summary     string             `json:"summary"`
}

// CheckConsistency 运行所有一致性检查
func CheckConsistency(pm *project.Manager) (*ConsistencyReport, error) {
	report := &ConsistencyReport{}

	// 加载实体数据库
	db, err := LoadEntityDB(pm.Dir)
	if err != nil {
		return nil, err
	}
	if err := db.SyncFromProject(pm); err != nil {
		return nil, err
	}

	// 1. 角色属性冲突检测
	checkCharacterAttributes(pm, db, report)

	// 2. 角色状态变化检测
	checkCharacterStatus(pm, db, report)

	// 3. 时间线一致性（概述级别）
	checkTimeline(pm, report)

	report.TotalIssues = len(report.Issues)

	if report.TotalIssues == 0 {
		report.Summary = "✅ 未发现一致性问题"
	} else {
		errors := 0
		warnings := 0
		for _, issue := range report.Issues {
			if issue.Severity == "error" {
				errors++
			} else if issue.Severity == "warning" {
				warnings++
			}
		}
		report.Summary = fmt.Sprintf("发现 %d 个问题（%d 错误, %d 警告, %d 提示）",
			report.TotalIssues, errors, warnings, report.TotalIssues-errors-warnings)
	}

	return report, nil
}

// checkCharacterAttributes 检测角色属性在章节间的冲突
func checkCharacterAttributes(pm *project.Manager, db *EntityDB, report *ConsistencyReport) {
	characters := db.Query(EntityCharacter)
	if len(characters) == 0 {
		return
	}

	// 逐章扫描角色提及，检测属性变化
	var prevChapterContent string
	for chapterNum := 1; ; chapterNum++ {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			break
		}
		summary, _ := pm.ReadChapterSummary(chapterNum)

		// 检测角色状态跳跃（如 Alive → Dead → Alive）
		if summary != nil {
			for _, chName := range summary.CharactersAppeared {
				entity := db.GetByName(chName)
				if entity == nil {
					continue
				}
				prevStatus := entity.Properties["status"]
				// 检查摘要中是否有状态变化暗示
				lowerContent := strings.ToLower(content)
				if strings.Contains(lowerContent, "死") || strings.Contains(lowerContent, "去世") ||
					strings.Contains(lowerContent, "牺牲") || strings.Contains(lowerContent, "陨落") {
					if prevStatus == "Alive" {
						report.Issues = append(report.Issues, ConsistencyIssue{
							Severity:    "warning",
							Category:    "status",
							EntityName:  chName,
							Description: fmt.Sprintf("%s 在第%d章似乎死亡，但当前状态仍为 Alive", chName, chapterNum),
							Location:    fmt.Sprintf("第%d章", chapterNum),
							Evidence:    extractEvidence(content, []string{"死", "去世", "牺牲", "陨落"}),
							Suggestion:  "确认角色状态是否需要在角色页面更新为 Dead",
						})
					}
				}
			}
		}

		// 检测属性关键词冲突（简单版：跨章检测矛盾描述）
		if prevChapterContent != "" && chapterNum > 1 {
			detectAttributeConflicts(db, prevChapterContent, content, chapterNum, report)
		}

		prevChapterContent = content
	}
}

// detectAttributeConflicts 检测两章之间的属性描述冲突
func detectAttributeConflicts(db *EntityDB, prevContent, currContent string, chapterNum int, report *ConsistencyReport) {
	characters := db.Query(EntityCharacter)
	for _, ch := range characters {
		prevEye := extractProperty(prevContent, ch.Name, []string{"眼睛", "眼眸", "瞳孔", "双眼"})
		currEye := extractProperty(currContent, ch.Name, []string{"眼睛", "眼眸", "瞳孔", "双眼"})
		if prevEye != "" && currEye != "" && prevEye != currEye {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "error",
				Category:    "attribute",
				EntityName:  ch.Name,
				Description: fmt.Sprintf("%s 的眼睛颜色不一致：前一章为「%s」，第%d章为「%s」", ch.Name, prevEye, chapterNum, currEye),
				Location:    fmt.Sprintf("第%d章", chapterNum),
				Evidence:    fmt.Sprintf("前: %s / 当前: %s", prevEye, currEye),
				Suggestion:  "统一角色外貌描述，或确认是否有合理原因（如魔法/伪装）",
			})
		}

		prevHair := extractProperty(prevContent, ch.Name, []string{"头发", "发色", "长发", "短发"})
		currHair := extractProperty(currContent, ch.Name, []string{"头发", "发色", "长发", "短发"})
		if prevHair != "" && currHair != "" && prevHair != currHair {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "warning",
				Category:    "attribute",
				EntityName:  ch.Name,
				Description: fmt.Sprintf("%s 的发色/发型描述可能不一致：前一章「%s」，第%d章「%s」", ch.Name, prevHair, chapterNum, currHair),
				Location:    fmt.Sprintf("第%d章", chapterNum),
				Evidence:    fmt.Sprintf("前: %s / 当前: %s", prevHair, currHair),
				Suggestion:  "检查是否为合理变化或需要统一描述",
			})
		}
	}
}

// extractProperty 从文本中提取角色的某个属性描述（Unicode 安全）
func extractProperty(content, charName string, keywords []string) string {
	runes := []rune(content)
	charNameRunes := []rune(charName)

	// 找到角色名在 rune 序列中的位置
	charIdx := indexRunes(runes, charNameRunes)
	if charIdx < 0 {
		return ""
	}

	start := charIdx - 100
	if start < 0 {
		start = 0
	}
	end := charIdx + 200
	if end > len(runes) {
		end = len(runes)
	}
	context := string(runes[start:end])

	for _, kw := range keywords {
		kwIdx := strings.Index(context, kw)
		if kwIdx >= 0 {
			// 提取关键词周围文本（Unicode 安全：在 rune 层面操作）
			ctxRunes := []rune(context)
			kwRuneIdx := len([]rune(context[:kwIdx]))
			ks := kwRuneIdx - 5
			if ks < 0 {
				ks = 0
			}
			ke := kwRuneIdx + len([]rune(kw)) + 15
			if ke > len(ctxRunes) {
				ke = len(ctxRunes)
			}
			return strings.TrimSpace(string(ctxRunes[ks:ke]))
		}
	}
	return ""
}

// indexRunes 在 rune 切片中查找子序列，返回起始索引
func indexRunes(s, sub []rune) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// extractEvidence 提取包含关键词的句子作为证据（Unicode 安全）
func extractEvidence(content string, keywords []string) string {
	runes := []rune(content)
	for _, kw := range keywords {
		idx := strings.Index(content, kw)
		if idx >= 0 {
			runeIdx := len([]rune(content[:idx]))
			start := runeIdx - 20
			if start < 0 {
				start = 0
			}
			end := runeIdx + len([]rune(kw)) + 30
			if end > len(runes) {
				end = len(runes)
			}
			return "..." + strings.TrimSpace(string(runes[start:end])) + "..."
		}
	}
	return ""
}

// checkCharacterStatus 检测角色状态异常
func checkCharacterStatus(pm *project.Manager, db *EntityDB, report *ConsistencyReport) {
	chars, err := pm.ReadCharacters()
	if err != nil || chars == nil {
		return
	}

	// 收集所有章节摘要中的出场角色
	appearances := make(map[string][]int)
	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		for _, name := range summary.CharactersAppeared {
			appearances[name] = append(appearances[name], chapterNum)
		}
	}

	// 检测 Dead 角色在后续章节中出场
	for _, ch := range chars.Characters {
		if ch.Status != "Dead" && ch.Status != "Missing" {
			continue
		}

		chaps, ok := appearances[ch.Name]
		if !ok {
			continue
		}

		// 找到角色状态标记为 Dead/Missing 的章节（从摘要推断）
		// 简化：如果 Dead/Missing 角色在后期章节仍出场，告警
		if len(chaps) > 1 && ch.Status == "Dead" {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "error",
				Category:    "status",
				EntityName:  ch.Name,
				Description: fmt.Sprintf("%s 状态为 Dead，但在多个章节中出场：%v", ch.Name, chaps),
				Location:    fmt.Sprintf("第%d-%d章", chaps[0], chaps[len(chaps)-1]),
				Evidence:    fmt.Sprintf("角色状态: %s, 出场章节: %v", ch.Status, chaps),
				Suggestion:  "更新角色状态或在出场章节中说明原因（回忆/幻觉/复活）",
			})
		}
	}
}

// checkTimeline 时间线一致性检测（概述级别）
func checkTimeline(pm *project.Manager, report *ConsistencyReport) {
	// 收集各章节摘要中的关键事件
	type event struct {
		chapter int
		summary string
	}
	var events []event

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		if summary != nil {
			events = append(events, event{chapter: chapterNum, summary: summary.Summary})
		}
	}

	// 简单检测：如果某章摘要提到"之后"/"第二天"/"几周后"，检查是否与前后章一致
	for i := 1; i < len(events); i++ {
		curr := events[i].summary
		prev := events[i-1].summary

		// 如果当前章说"同时"但前一章是不同地点 → 可能时间线混乱
		if strings.Contains(curr, "同时") && strings.Contains(prev, "同时") {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "info",
				Category:    "timeline",
				EntityName:  "",
				Description: fmt.Sprintf("第%d章和第%d章都使用了「同时」描述，建议确认时间线连续性", events[i-1].chapter, events[i].chapter),
				Location:    fmt.Sprintf("第%d-%d章", events[i-1].chapter, events[i].chapter),
				Evidence:    fmt.Sprintf("前: %s / 当前: %s", prev, curr),
				Suggestion:  "考虑使用具体时间标记（如「三小时后」「翌日」）替代模糊的「同时」",
			})
		}
	}
}

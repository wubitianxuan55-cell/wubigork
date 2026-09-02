package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
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
	Branch      string `json:"branch"`      // 分支标记：""=主线章节，"a"/"b"/"c"=分支章节
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

	// 枚举实际存在的章节文件（主线 NNN.md + 分支 NNNa.md）：
	// 章节断档（如 1,2,5 缺 3、4）不再中断扫描，分支章节一并纳入
	chapters := listChapterFiles(pm)

	// 1. 角色属性冲突检测
	checkCharacterAttributes(pm, db, report, chapters)

	// 2. 角色状态变化检测
	checkCharacterStatus(pm, report, chapters)

	// 3. 时间线一致性（概述级别）
	checkTimeline(pm, report, chapters)

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

// ── 章节枚举（主线 + 分支）──────────────────────────────────

// chapterFileRe 匹配主线/分支章节文件名 NNN.md / NNNa.md（与 stats.Collect 一致）
var chapterFileRe = regexp.MustCompile(`^([0-9]{3})([a-z]?)\.md$`)

// chapterFile 是目录扫描枚举到的章节文件信息
type chapterFile struct {
	num    int    // 章节号
	branch string // ""=主线，"a"/"b"/"c"=分支
}

// Place 返回告警中的位置标签：主线「第3章」，分支「第3章分支a」
func (cf chapterFile) Place() string {
	if cf.branch == "" {
		return fmt.Sprintf("第%d章", cf.num)
	}
	return fmt.Sprintf("第%d章分支%s", cf.num, cf.branch)
}

// listChapterFiles 单次 ReadDir 枚举 chapters/ 下实际存在的章节文件（主线 + 分支，
// 按章节号、分支字母排序），替代逐个文件探测：章节断档（中间缺章）不会中断扫描，
// 分支章节 NNNa.md 也纳入扫描。无 chapters 目录返回空列表。
func listChapterFiles(pm *project.Manager) []chapterFile {
	entries, err := os.ReadDir(filepath.Join(pm.Dir, "chapters"))
	if err != nil {
		return nil
	}
	var out []chapterFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := chapterFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		out = append(out, chapterFile{num: num, branch: m[2]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].num != out[j].num {
			return out[i].num < out[j].num
		}
		return out[i].branch < out[j].branch
	})
	return out
}

// readChapterContent 读章节正文：主线 chapters/NNN.md，分支 chapters/NNNa.md
func readChapterContent(pm *project.Manager, cf chapterFile) (string, error) {
	if cf.branch == "" {
		return pm.ReadChapter(cf.num)
	}
	return pm.ReadChapterBranch(cf.num, cf.branch)
}

// readChapterSummaryOf 读章节摘要：主线 chapters/NNN-summary.json，分支 chapters/NNNa-summary.json
func readChapterSummaryOf(pm *project.Manager, cf chapterFile) (*types.ChapterSummary, error) {
	if cf.branch == "" {
		return pm.ReadChapterSummary(cf.num)
	}
	return pm.ReadChapterBranchSummary(cf.num, cf.branch)
}

// checkCharacterAttributes 检测角色属性在章节间的冲突
// 主线与各分支各自成链扫描：分支章节产生带分支标记的告警，分支不与主线混判。
func checkCharacterAttributes(pm *project.Manager, db *EntityDB, report *ConsistencyReport, chapters []chapterFile) {
	characters := db.Query(EntityCharacter)
	if len(characters) == 0 {
		return
	}

	// 逐章扫描角色提及，检测属性变化（枚举实际存在章节，断档跳过继续）
	prevChapterContent := make(map[string]string) // 分支标记 -> 该线上一章内容（""=主线）
	for _, cf := range chapters {
		content, err := readChapterContent(pm, cf)
		if err != nil {
			continue
		}
		summary, _ := readChapterSummaryOf(pm, cf)

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
							Description: fmt.Sprintf("%s 在%s似乎死亡，但当前状态仍为 Alive", chName, cf.Place()),
							Location:    cf.Place(),
							Evidence:    extractEvidence(content, []string{"死", "去世", "牺牲", "陨落"}),
							Suggestion:  "确认角色状态是否需要在角色页面更新为 Dead",
							Branch:      cf.branch,
						})
					}
				}
			}
		}

		// 检测属性关键词冲突（同一故事线内跨章比较，分支出场不作为主线依据）
		if prev := prevChapterContent[cf.branch]; prev != "" {
			detectAttributeConflicts(db, prev, content, cf, report)
		}
		prevChapterContent[cf.branch] = content
	}
}

// detectAttributeConflicts 检测两章之间的属性描述冲突（同一故事线内）
func detectAttributeConflicts(db *EntityDB, prevContent, currContent string, cf chapterFile, report *ConsistencyReport) {
	characters := db.Query(EntityCharacter)
	for _, ch := range characters {
		prevEye := extractProperty(prevContent, ch.Name, []string{"眼睛", "眼眸", "瞳孔", "双眼"})
		currEye := extractProperty(currContent, ch.Name, []string{"眼睛", "眼眸", "瞳孔", "双眼"})
		if prevEye != "" && currEye != "" && prevEye != currEye {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "error",
				Category:    "attribute",
				EntityName:  ch.Name,
				Description: fmt.Sprintf("%s 的眼睛颜色不一致：前一章为「%s」，%s为「%s」", ch.Name, prevEye, cf.Place(), currEye),
				Location:    cf.Place(),
				Evidence:    fmt.Sprintf("前: %s / 当前: %s", prevEye, currEye),
				Suggestion:  "统一角色外貌描述，或确认是否有合理原因（如魔法/伪装）",
				Branch:      cf.branch,
			})
		}

		prevHair := extractProperty(prevContent, ch.Name, []string{"头发", "发色", "长发", "短发"})
		currHair := extractProperty(currContent, ch.Name, []string{"头发", "发色", "长发", "短发"})
		if prevHair != "" && currHair != "" && prevHair != currHair {
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "warning",
				Category:    "attribute",
				EntityName:  ch.Name,
				Description: fmt.Sprintf("%s 的发色/发型描述可能不一致：前一章「%s」，%s「%s」", ch.Name, prevHair, cf.Place(), currHair),
				Location:    cf.Place(),
				Evidence:    fmt.Sprintf("前: %s / 当前: %s", prevHair, currHair),
				Suggestion:  "检查是否为合理变化或需要统一描述",
				Branch:      cf.branch,
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
// 主线与各分支分别判断出场：分支章节的出场地不作为主线跨章结论的依据。
func checkCharacterStatus(pm *project.Manager, report *ConsistencyReport, chapters []chapterFile) {
	chars, err := pm.ReadCharacters()
	if err != nil || chars == nil {
		return
	}

	// 按故事线（主线/各分支）收集所有章节摘要中的出场角色
	appearances := make(map[string]map[string][]int) // 角色名 -> 分支标记 -> 出场章节号
	for _, cf := range chapters {
		summary, err := readChapterSummaryOf(pm, cf)
		if err != nil || summary == nil {
			continue
		}
		for _, name := range summary.CharactersAppeared {
			if appearances[name] == nil {
				appearances[name] = make(map[string][]int)
			}
			appearances[name][cf.branch] = append(appearances[name][cf.branch], cf.num)
		}
	}

	// 检测 Dead 角色在后续章节中出场
	for _, ch := range chars.Characters {
		if ch.Status != "Dead" && ch.Status != "Missing" {
			continue
		}

		byBranch, ok := appearances[ch.Name]
		if !ok {
			continue
		}

		// 找到角色状态标记为 Dead/Missing 的章节（从摘要推断）
		// 简化：如果 Dead/Missing 角色在同一条故事线的后期章节仍出场，告警（分支单独成线判断）
		branches := make([]string, 0, len(byBranch))
		for branch := range byBranch {
			branches = append(branches, branch)
		}
		sort.Strings(branches)
		for _, branch := range branches {
			chaps := byBranch[branch]
			if len(chaps) < 2 || ch.Status != "Dead" {
				continue
			}
			location := fmt.Sprintf("第%d-%d章", chaps[0], chaps[len(chaps)-1])
			description := fmt.Sprintf("%s 状态为 Dead，但在多个章节中出场：%v", ch.Name, chaps)
			evidence := fmt.Sprintf("角色状态: %s, 出场章节: %v", ch.Status, chaps)
			if branch != "" {
				location = fmt.Sprintf("第%d-%d章（分支%s）", chaps[0], chaps[len(chaps)-1], branch)
				description = fmt.Sprintf("%s 状态为 Dead，但在分支%s的多个章节中出场：%v", ch.Name, branch, chaps)
				evidence = fmt.Sprintf("角色状态: %s, 出场章节: %v, 分支: %s", ch.Status, chaps, branch)
			}
			report.Issues = append(report.Issues, ConsistencyIssue{
				Severity:    "error",
				Category:    "status",
				EntityName:  ch.Name,
				Description: description,
				Location:    location,
				Evidence:    evidence,
				Suggestion:  "更新角色状态或在出场章节中说明原因（回忆/幻觉/复活）",
				Branch:      branch,
			})
		}
	}
}

// checkTimeline 时间线一致性检测（概述级别）
// 主线与各分支分别成链比较：分支章节产生带分支标记的告警，不与主线混判。
func checkTimeline(pm *project.Manager, report *ConsistencyReport, chapters []chapterFile) {
	// 收集各章节摘要中的关键事件（按主线/分支分线）
	type event struct {
		chapter int
		summary string
	}
	eventsByBranch := make(map[string][]event)
	for _, cf := range chapters {
		summary, err := readChapterSummaryOf(pm, cf)
		if err != nil || summary == nil {
			continue
		}
		eventsByBranch[cf.branch] = append(eventsByBranch[cf.branch], event{chapter: cf.num, summary: summary.Summary})
	}

	branches := make([]string, 0, len(eventsByBranch))
	for branch := range eventsByBranch {
		branches = append(branches, branch)
	}
	sort.Strings(branches)

	// 简单检测：如果某章摘要提到"之后"/"第二天"/"几周后"，检查是否与前后章一致
	for _, branch := range branches {
		events := eventsByBranch[branch]
		for i := 1; i < len(events); i++ {
			curr := events[i].summary
			prev := events[i-1].summary

			// 如果当前章说"同时"但前一章是不同地点 → 可能时间线混乱
			if strings.Contains(curr, "同时") && strings.Contains(prev, "同时") {
				location := fmt.Sprintf("第%d-%d章", events[i-1].chapter, events[i].chapter)
				description := fmt.Sprintf("第%d章和第%d章都使用了「同时」描述，建议确认时间线连续性", events[i-1].chapter, events[i].chapter)
				if branch != "" {
					location = fmt.Sprintf("第%d-%d章（分支%s）", events[i-1].chapter, events[i].chapter, branch)
					description = fmt.Sprintf("分支%s的第%d章和第%d章都使用了「同时」描述，建议确认时间线连续性", branch, events[i-1].chapter, events[i].chapter)
				}
				report.Issues = append(report.Issues, ConsistencyIssue{
					Severity:    "info",
					Category:    "timeline",
					EntityName:  "",
					Description: description,
					Location:    location,
					Evidence:    fmt.Sprintf("前: %s / 当前: %s", prev, curr),
					Suggestion:  "考虑使用具体时间标记（如「三小时后」「翌日」）替代模糊的「同时」",
					Branch:      branch,
				})
			}
		}
	}
}

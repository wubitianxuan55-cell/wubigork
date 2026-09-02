package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/graph"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 一致性 AI 深检 v0（Continuity Linter）─────────────────────
//
// 逐章调用 AI 提取「实体状态卡」（出场角色的生死/位置/关键物品 + 时间标记 +
// 场景要点），本地跨章比对出矛盾（死亡角色再出场、位置瞬移、物品凭空消失/
// 无中生有、时间倒流），与既有规则层（graph.CheckConsistency）合并展示。
//
// 降级原则（诚实降级，绝不编造 AI 结果）：
//   - client nil / 无可用模型 / 全部章节 AI 提取失败 → 只返回规则层结果，
//     ai_available=false，ai_note 说明原因；
//   - 单章 AI 失败 → 跳过该章不中断整体（chapters_failed 计数）。

// deepStateCard 单章实体状态卡（AI 提取，结构稳定可解析）
type deepStateCard struct {
	Chapter       int                  `json:"chapter"`
	Branch        string               `json:"branch"`
	TimeMark      string               `json:"time_mark"`      // 本章故事内时间标记（如「第三日夜」）
	TimeRelation  string               `json:"time_relation"`  // later / same / earlier / unknown（相对上一章）
	SceneNotes    []string             `json:"scene_notes"`    // 场景要点
	TravelNotes   []string             `json:"travel_notes"`   // 章间移动/赶路交代
	Characters    []deepCharacterState `json:"characters"`     // 出场角色章末状态
	ItemsLost     []string             `json:"items_lost"`     // 本章内损毁/遗失/交出的关键物品
	ItemsRegained []string             `json:"items_regained"` // 本章内重新获得此前失去的关键物品
}

// deepCharacterState 单个角色的章末状态
type deepCharacterState struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`   // alive / dead / missing / unknown
	Location string   `json:"location"` // 章末所在地（未知为空串）
	Items    []string `json:"items"`    // 章末仍持有的关键物品
}

// CheckConsistencyDeep 一致性 AI 深检：规则层 + AI 状态卡跨章比对合并结果。
// maxChapters 为 AI 深检窗口（最近 N 章），≤0 或 >50 时夹到 50。
func (a *writingState) CheckConsistencyDeep(maxChapters int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	maxChapters = deepClampMaxChapters(maxChapters)

	// 规则层永远先跑（AI 不可用时它是唯一结果来源）
	var ruleIssues []graph.ConsistencyIssue
	var notes []string
	report, ruleErr := graph.CheckConsistency(pm)
	if ruleErr != nil {
		notes = append(notes, fmt.Sprintf("规则检查失败: %v", ruleErr))
	} else {
		ruleIssues = report.Issues
	}

	// AI 深检（client nil / 无模型 / 提取全失败时诚实降级为规则层）
	cards, failed, aiNotes := a.deepExtractCards(pm, maxChapters)
	notes = append(notes, aiNotes...)

	var aiIssues []graph.ConsistencyIssue
	for _, cardsByBranch := range deepGroupByBranch(cards) {
		aiIssues = append(aiIssues, deepCompareStateLine(cardsByBranch)...)
	}

	// 合并：规则层在前（source=rule），AI 在后（source=ai）
	merged := make([]map[string]interface{}, 0, len(ruleIssues)+len(aiIssues))
	for _, iss := range ruleIssues {
		merged = append(merged, deepIssueToMap(iss, "rule"))
	}
	for _, iss := range aiIssues {
		merged = append(merged, deepIssueToMap(iss, "ai"))
	}

	total := len(merged)
	errCount, warnCount := 0, 0
	for _, iss := range merged {
		switch iss["severity"] {
		case "error":
			errCount++
		case "warning":
			warnCount++
		}
	}
	var summary string
	if total == 0 {
		summary = "✅ 未发现一致性问题"
	} else {
		summary = fmt.Sprintf("发现 %d 个问题（%d 错误, %d 警告, %d 提示）",
			total, errCount, warnCount, total-errCount-warnCount)
	}
	if len(cards) > 0 {
		summary += fmt.Sprintf("；AI 深检已扫描 %d 章", len(cards))
	}

	aiAvailable := len(cards) > 0
	return map[string]interface{}{
		"issues":           merged,
		"total_issues":     total,
		"summary":          summary,
		"chapters_scanned": len(cards),
		"chapters_failed":  failed,
		"ai_available":     aiAvailable,
		"ai_note":          strings.Join(notes, "；"),
	}, nil
}

// deepIssueToMap 状态卡比对产出 → 与规则层同构的 issue map（附 source 字段）
func deepIssueToMap(iss graph.ConsistencyIssue, source string) map[string]interface{} {
	return map[string]interface{}{
		"severity":    iss.Severity,
		"category":    iss.Category,
		"entity_name": iss.EntityName,
		"description": iss.Description,
		"location":    iss.Location,
		"evidence":    iss.Evidence,
		"suggestion":  iss.Suggestion,
		"branch":      iss.Branch,
		"source":      source,
	}
}

// ── 章节枚举与窗口 ───────────────────────────────────────────

// deepChapterRef app 层章节引用（与 graph.chapterFile 同构，避免依赖包内私有类型）
type deepChapterRef struct {
	num    int    // 章节号
	branch string // ""=主线，"a"/"b"/"c"=分支
}

// deepPlace 告警位置标签：主线「第3章」，分支「第3章分支a」
func deepPlace(num int, branch string) string {
	if branch == "" {
		return fmt.Sprintf("第%d章", num)
	}
	return fmt.Sprintf("第%d章分支%s", num, branch)
}

var deepChapterRe = regexp.MustCompile(`^([0-9]{3})([a-z]?)\.md$`)

// deepListChapters 枚举 chapters/ 下实际存在的章节文件（主线 + 分支，按章节号、分支排序）
func deepListChapters(pm *project.Manager) []deepChapterRef {
	entries, err := os.ReadDir(filepath.Join(pm.Dir, "chapters"))
	if err != nil {
		return nil
	}
	var out []deepChapterRef
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := deepChapterRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		out = append(out, deepChapterRef{num: num, branch: m[2]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].num != out[j].num {
			return out[i].num < out[j].num
		}
		return out[i].branch < out[j].branch
	})
	return out
}

// deepClampMaxChapters 深检窗口夹取：≤0 或 >50 → 50
func deepClampMaxChapters(n int) int {
	if n <= 0 || n > 50 {
		return 50
	}
	return n
}

// deepReadChapter 读章节正文：主线 chapters/NNN.md，分支 chapters/NNNa.md
func deepReadChapter(pm *project.Manager, ref deepChapterRef) (string, error) {
	if ref.branch == "" {
		return pm.ReadChapter(ref.num)
	}
	return pm.ReadChapterBranch(ref.num, ref.branch)
}

const (
	deepHeadRunes = 6000 // 正文头部保留 rune 数
	deepTailRunes = 2000 // 正文尾部保留 rune 数
)

// deepTruncateChapter 章正文截断：头部 + 尾部拼接，控制 token
func deepTruncateChapter(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= deepHeadRunes+deepTailRunes {
		return string(r)
	}
	return string(r[:deepHeadRunes]) + "\n……（中段过长已省略）……\n" + string(r[len(r)-deepTailRunes:])
}

// ── AI 逐章提取状态卡 ────────────────────────────────────────

// deepExtractCards 串行逐章调用 AI 提取状态卡。
// 返回成功提取的卡片（按扫描顺序）、失败章数与降级说明（空=正常）。
// 单章失败跳过不中断整体；全部失败时返回空卡片 + 失败原因。
func (a *writingState) deepExtractCards(pm *project.Manager, maxChapters int) ([]*deepStateCard, int, []string) {
	var notes []string
	if a.client == nil {
		return nil, 0, []string{"AI 客户端未初始化，仅显示规则检查结果"}
	}

	chapters := deepListChapters(pm)
	if len(chapters) == 0 {
		return nil, 0, []string{"项目中没有可扫描的章节"}
	}
	// 取最近 maxChapters 章（全局按章节号排序后的尾部）
	if len(chapters) > maxChapters {
		chapters = chapters[len(chapters)-maxChapters:]
	}

	eng, model, _ := a.routeModel("novel")
	if eng == "" || model == "" {
		return nil, 0, []string{"未找到可用模型（可能处于离线模式），仅显示规则检查结果"}
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, model, sys, usr, ai.ChatSimpleOptions{
			EngineID:    eng,
			Temperature: 0.2, // 提取任务：低温度保稳定
			MaxTokens:   2048,
		})
	}
	system, _ := deepExtractPrompts()

	prevTimeMark := map[string]string{} // 分支标记 -> 该线上一成功章的时间标记
	var cards []*deepCardResult
	for _, ref := range chapters {
		body, err := deepReadChapter(pm, ref)
		if err != nil {
			cards = append(cards, &deepCardResult{ref: ref, err: fmt.Errorf("读章节失败: %w", err)})
			continue
		}
		_, user := deepExtractPrompts()
		user = fmt.Sprintf("%s\n\n章节：%s\n上一章时间标记：%s\n\n【正文】\n%s",
			user, deepPlace(ref.num, ref.branch), prevTimeMark[ref.branch], deepTruncateChapter(body))

		jsonStr, err := util.RetryJSON(ctx, caller, system, user, 2)
		if err != nil {
			cards = append(cards, &deepCardResult{ref: ref, err: fmt.Errorf("AI 提取失败: %w", err)})
			continue
		}
		card, err := deepParseCard(jsonStr)
		if err != nil {
			cards = append(cards, &deepCardResult{ref: ref, err: err})
			continue
		}
		// 章节号/分支以本地枚举为准（不信任 AI 输出）
		card.Chapter = ref.num
		card.Branch = ref.branch
		prevTimeMark[ref.branch] = card.TimeMark
		cards = append(cards, &deepCardResult{ref: ref, card: card})
	}

	var okCards []*deepStateCard
	failed := 0
	var lastErr string
	for _, cr := range cards {
		if cr.err != nil {
			failed++
			lastErr = cr.err.Error()
			continue
		}
		okCards = append(okCards, cr.card)
	}
	if len(okCards) == 0 && len(chapters) > 0 {
		notes = append(notes, fmt.Sprintf("AI 逐章提取全部失败（%d 章，最后错误: %s），仅显示规则检查结果", failed, util.Truncate(lastErr, 160)))
	} else if failed > 0 {
		notes = append(notes, fmt.Sprintf("%d 章 AI 提取失败已跳过", failed))
	}
	return okCards, failed, notes
}

// deepCardResult 单章提取结果（成功持 card，失败持 err）
type deepCardResult struct {
	ref  deepChapterRef
	card *deepStateCard
	err  error
}

// deepParseCard 解析 AI 回复中的状态卡 JSON
func deepParseCard(reply string) (*deepStateCard, error) {
	jsonStr := util.ExtractJSON(reply)
	if jsonStr == "" {
		return nil, fmt.Errorf("AI 回复中没有 JSON")
	}
	var card deepStateCard
	if err := json.Unmarshal([]byte(jsonStr), &card); err != nil {
		return nil, fmt.Errorf("解析状态卡 JSON 失败: %w", err)
	}
	return &card, nil
}

// deepExtractPrompts 状态卡提取的 system/user prompt 模板
func deepExtractPrompts() (string, string) {
	system := "你是小说一致性审校助手，负责从章节正文中提取实体状态卡。" +
		"只依据正文事实提取，不要编造正文未提及的信息；正文未提及的字段用空串或空数组。" +
		"严格输出 JSON，不要输出任何其它文字。"
	user := `请输出本章状态卡，严格按以下 JSON 结构（字段名不变，输出单个 JSON 对象）：
{"chapter": 0, "branch": "", "time_mark": "", "time_relation": "later|same|earlier|unknown", "scene_notes": [], "travel_notes": [], "characters": [{"name": "", "status": "alive|dead|missing|unknown", "location": "", "items": []}], "items_lost": [], "items_regained": []}
字段说明：
- time_mark：本章故事内时间标记（如「第三日夜」「翌日清晨」），无法判断用空串。
- time_relation：本章时间相对上一章的先后关系；上一章无时间标记或无法判断填 unknown。
- characters：有名字的出场角色；status 为该角色章末状态（alive=存活 / dead=死亡 / missing=失踪 / unknown=未知）；location 为章末所在地；items 为章末仍持有的剧情关键物品（普通消耗品忽略）。
- items_lost：本章内损毁/遗失/交出的关键物品名；items_regained：本章内重新获得此前失去的关键物品名；无则空数组。
- travel_notes：正文明确交代的跨区域移动、赶路、长途旅行；无则空数组。
- scene_notes：本章场景要点，每条不超过 40 字。`
	return system, user
}

// ── 本地跨章比对（纯函数，可测）─────────────────────────────

// deepGroupByBranch 把成功的状态卡按故事线（主线/分支）分组，各线内按章节号排序
func deepGroupByBranch(cards []*deepStateCard) map[string][]*deepStateCard {
	byBranch := map[string][]*deepStateCard{}
	for _, c := range cards {
		byBranch[c.Branch] = append(byBranch[c.Branch], c)
	}
	for branch := range byBranch {
		line := byBranch[branch]
		sort.Slice(line, func(i, j int) bool { return line[i].Chapter < line[j].Chapter })
	}
	return byBranch
}

// deepNormStatus 归一化 AI 给出的角色状态
func deepNormStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.Contains(s, "dead") || strings.Contains(s, "死"):
		return "dead"
	case strings.Contains(s, "missing") || strings.Contains(s, "失踪"):
		return "missing"
	case strings.Contains(s, "alive") || strings.Contains(s, "活"):
		return "alive"
	default:
		return "unknown"
	}
}

func deepHasItem(list []string, item string) bool {
	item = strings.TrimSpace(item)
	for _, it := range list {
		if strings.TrimSpace(it) == item {
			return true
		}
	}
	return false
}

// deepHeldItems 收集一章状态卡中「物品名 -> 持有者」（同名物品首个持有者生效）
func deepHeldItems(card *deepStateCard) map[string]string {
	held := map[string]string{}
	for _, c := range card.Characters {
		for _, it := range c.Items {
			it = strings.TrimSpace(it)
			if it == "" {
				continue
			}
			if _, ok := held[it]; !ok {
				held[it] = c.Name
			}
		}
	}
	return held
}

// deepCompareStateLine 单条故事线内的跨章比对（纯函数）：cards 须同分支且按章节号升序。
// 检出矛盾类型：
//  1. 时间倒流（time_relation=earlier）
//  2. 死亡角色后期以存活状态再出场
//  3. 相邻章位置瞬移（所在地变化且无 travel_notes 交代）
//  4. 关键物品凭空消失（上一章末仍持有，本章无人持有且无失去交代）
//  5. 已失去/损毁的关键物品无中生有（无重新获得交代）
func deepCompareStateLine(cards []*deepStateCard) []graph.ConsistencyIssue {
	var issues []graph.ConsistencyIssue
	deadAt := map[string]int{} // 角色名 -> 标记死亡的章节号
	lostAt := map[string]int{} // 物品名 -> 标记失去/损毁的章节号
	prevHeld := map[string]string{}
	var prev *deepStateCard

	for _, curr := range cards {
		place := deepPlace(curr.Chapter, curr.Branch)

		// 1. 时间倒流
		if prev != nil && strings.EqualFold(strings.TrimSpace(curr.TimeRelation), "earlier") {
			issues = append(issues, graph.ConsistencyIssue{
				Severity:    "error",
				Category:    "timeline",
				Description: fmt.Sprintf("%s 的时间标记（%s）早于上一章（%s），疑似时间倒流", place, curr.TimeMark, prev.TimeMark),
				Location:    place,
				Evidence:    fmt.Sprintf("上一章 time_mark=%s，本章 time_mark=%s / time_relation=earlier", prev.TimeMark, curr.TimeMark),
				Suggestion:  "核对两章时间标记与叙事顺序，调整时间线或在文中说明（闪回需有明确标记）",
				Branch:      curr.Branch,
			})
		}

		prevLoc := map[string]string{}
		if prev != nil {
			for _, c := range prev.Characters {
				if loc := strings.TrimSpace(c.Location); loc != "" {
					prevLoc[c.Name] = loc
				}
			}
		}
		noTravel := len(curr.TravelNotes) == 0

		for _, c := range curr.Characters {
			name := strings.TrimSpace(c.Name)
			if name == "" {
				continue
			}
			status := deepNormStatus(c.Status)

			// 2. 死亡角色再出场（复活仅报一次：报后清除死亡标记）
			if at, ok := deadAt[name]; ok && status == "alive" {
				issues = append(issues, graph.ConsistencyIssue{
					Severity:    "error",
					Category:    "status",
					EntityName:  name,
					Description: fmt.Sprintf("%s 在第%d章已死亡，但%s仍以存活状态出场", name, at, place),
					Location:    place,
					Evidence:    fmt.Sprintf("第%d章状态卡 status=dead；%s状态卡 status=alive", at, place),
					Suggestion:  "确认是否为复活/回忆/幻象并在文中明确交代，否则修正角色生死状态",
					Branch:      curr.Branch,
				})
				delete(deadAt, name)
				continue
			}

			// 3. 位置瞬移（相邻章所在地变化且无移动交代）
			if prev != nil && status != "dead" {
				if prevL, ok := prevLoc[name]; ok {
					if loc := strings.TrimSpace(c.Location); loc != "" && loc != prevL && noTravel {
						issues = append(issues, graph.ConsistencyIssue{
							Severity:    "warning",
							Category:    "status",
							EntityName:  name,
							Description: fmt.Sprintf("%s 所在地从「%s」变为「%s」，但本章无移动/赶路交代，疑似位置瞬移", name, prevL, loc),
							Location:    place,
							Evidence:    fmt.Sprintf("上一章 location=%s；%slocation=%s，travel_notes 为空", prevL, place, loc),
							Suggestion:  "补充角色移动过程（赶路/传送/时间跳跃），或修正两地之一的位置描述",
							Branch:      curr.Branch,
						})
					}
				}
			}

			if status == "dead" {
				deadAt[name] = curr.Chapter
			}
		}

		// 4/5. 关键物品比对（相邻章 + 失踪物品台账）
		held := deepHeldItems(curr)
		if prev != nil {
			for _, item := range deepSortedKeys(prevHeld) {
				if _, stillHeld := held[item]; stillHeld {
					continue
				}
				if deepHasItem(curr.ItemsLost, item) || deepHasItem(prev.ItemsLost, item) {
					continue
				}
				// 无交代消失视同失去：记入台账，后续无交代地再次出现将按「无中生有」报错
				lostAt[item] = curr.Chapter
				issues = append(issues, graph.ConsistencyIssue{
					Severity:    "warning",
					Category:    "item",
					EntityName:  item,
					Description: fmt.Sprintf("关键物品「%s」凭空消失：上一章由 %s 持有，%s已无人持有且无失去交代", item, prevHeld[item], place),
					Location:    place,
					Evidence:    fmt.Sprintf("上一章状态卡 items 含「%s」（持有者 %s）；%s状态卡无人持有且 items_lost 未提及", item, prevHeld[item], place),
					Suggestion:  "补写物品去向（遗失/被夺/存放），或从角色物品清单中移除",
					Branch:      curr.Branch,
				})
			}
			for _, item := range deepSortedKeys(held) {
				if at, ok := lostAt[item]; ok && !deepHasItem(curr.ItemsRegained, item) {
					issues = append(issues, graph.ConsistencyIssue{
						Severity:    "error",
						Category:    "item",
						EntityName:  item,
						Description: fmt.Sprintf("已失去/损毁的关键物品「%s」无中生有：第%d章已标记失去，%s由 %s 再次持有且无重新获得交代", item, at, place, held[item]),
						Location:    place,
						Evidence:    fmt.Sprintf("第%d章状态卡 items_lost 含「%s」；%s状态卡 items 含「%s」且 items_regained 未提及", at, item, place, item),
						Suggestion:  "补写物品失而复得的过程（寻回/修复/复制），或修正前后章的物品状态",
						Branch:      curr.Branch,
					})
				}
			}
		}
		for _, it := range curr.ItemsLost {
			if it = strings.TrimSpace(it); it != "" {
				lostAt[it] = curr.Chapter
			}
		}
		for _, it := range curr.ItemsRegained {
			if it = strings.TrimSpace(it); it != "" {
				delete(lostAt, it)
			}
		}

		prev = curr
		prevHeld = held
	}
	return issues
}

// deepSortedKeys map key 排序（保证告警顺序稳定，可测）
func deepSortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

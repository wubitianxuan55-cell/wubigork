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
	"unicode"

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
//
// 误报缓解（v4.101，宁多提示勿漏报：降级项仍产出告警，只调级别并标注原因）：
//   - 文本归一化后再比对（deepNormalizeText：全半角/空白/包裹标点/大小写），
//     消除「玄铁剑」vs「那柄玄铁剑」类措辞/格式差异；
//   - 位置比对先做归一化包含判定，同区域表述差异（「青云宗」vs「青云宗后山」）
//     降级为 info 提示（reason=wording），不再按瞬移报 warning；
//   - 角色名按项目人物名单归一（deepAliasResolver：项目数据无别名字段，用
//     名单+包含+称谓剥离的保守归一），别名参与的比对标 reason=alias 降置信度；
//   - 时间倒流：任一方时间标记为空/粗粒度（「三年后」「翌日」类）→ 降为
//     warning + reason=granularity（疑似闪回/省略）；
//   - 无中生有：仅因「无交代消失」入账的物品 → 降为 warning +
//     reason=unexplained（可能后期才寻回），有 items_lost 明确交代的仍报 error。
//   - 每条 AI 告警带 confidence（high/medium/low）与 reason 分类，前端分级展示。

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

	// 误报缓解：按项目人物名单归一角色名（characters.json + lorebook 人物词条）
	res := newDeepAliasResolver(deepCanonicalPeople(pm))

	var aiIssues []deepIssue
	for _, cardsByBranch := range deepGroupByBranch(cards) {
		aiIssues = append(aiIssues, deepCompareStateLine(cardsByBranch, res)...)
	}

	// 合并：规则层在前（source=rule），AI 在后（source=ai）
	merged := make([]map[string]interface{}, 0, len(ruleIssues)+len(aiIssues))
	for _, iss := range ruleIssues {
		merged = append(merged, deepIssueToMap(deepIssue{ConsistencyIssue: iss, Confidence: "high"}, "rule"))
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

// deepIssue AI 比对产出：与规则层同构的 issue + 误报缓解标注（嵌入保持字段直读）。
// Confidence：high（确定冲突）/ medium（疑似）/ low（提示级）；Reason 为原因分类
// （wording=措辞差异 / granularity=时间粒度差异 / alias=称谓别名差异 /
// unexplained=缺少明确交代），空串=常规判定。规则层告警 Confidence 恒为 high。
type deepIssue struct {
	graph.ConsistencyIssue
	Confidence string `json:"confidence,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// deepIssueToMap 状态卡比对产出 → 与规则层同构的 issue map（附 source/confidence/reason）
func deepIssueToMap(iss deepIssue, source string) map[string]interface{} {
	confidence := iss.Confidence
	if confidence == "" {
		confidence = "high"
	}
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
		"confidence":  confidence,
		"reason":      iss.Reason,
	}
}

// ── 误报缓解：文本归一化 + 项目人物名单归一 + 时间粒度判定 ──

// deepWrapPunct 比对时剥离的包裹符号（引号/书名号/括号等；全角括号经半角折叠
// 后也落入 ASCII 项，故两套都列）
const deepWrapPunct = "「」『』【】〔〕（）()《》〈〉“”‘’\"'`()[]{}"

// deepEdgePunct 归一化结果首尾可剥离的标点
const deepEdgePunct = "。，、；：！？!?,.:;·…—－–~～＿_-"

// deepNormalizeText 实体名/地名/物品名比对前的文本归一化（缓解误报①措辞差异
// ④数字/单位格式差异）：
//   - 全角 ASCII（U+FF01–U+FF5E，含全角字母数字）折叠为半角、全角空格折叠为空格；
//   - 大写拉丁折叠为小写；
//   - 剥离全部空白与包裹符号（引号/书名号/括号），去掉首尾标点。
//
// 仅用作比对键，不用于正文展示。
func deepNormalizeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			r -= 0xFEE0
		case r == 0x3000:
			r = ' '
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		} else if unicode.IsSpace(r) {
			continue
		}
		if strings.ContainsRune(deepWrapPunct, r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), deepEdgePunct)
}

// deepNormContains 归一化包含判定：b（针）归一化后是 a（堆）归一化结果的子串
// 或与之相等。针的有效长度 <2 rune 时只允许相等命中，避免「剑」「王」等单字
// 造成误合并。
func deepNormContains(a, b string) bool {
	na, nb := deepNormalizeText(a), deepNormalizeText(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	if len([]rune(nb)) < 2 {
		return false
	}
	return strings.Contains(na, nb)
}

// deepCoarseTimeRe 粗粒度时间标记：间隔表达（数/几/余/半/中文数字 + 时间单位，
// 如「三年」「数月」「半月前」；阿拉伯数字须带后/前缀才是间隔，如「1247年后」，
// 「1247年」视为绝对时间保持精确）。锚定词首尾，避免「第三日」「1247年春」误中
var deepCoarseTimeRe = regexp.MustCompile(`^((数|几|余|半|[一二两三四五六七八九十百千万]+)(年|载|月|日|天|夜|晨)(之?后|之?前)?|[0-9]+(年|载|月|日|天|夜|晨)(之?后|之?前))$`)

// deepCoarseTimeWords 粗粒度/相对时间关键词（翌日、次日、当年、回忆插叙类）
var deepCoarseTimeWords = []string{
	"翌", "次日", "次晨", "次夜", "当年", "当初", "昔日", "曾经",
	"从前", "此前", "其后", "此后", "后来", "回忆", "闪回", "梦境", "梦中", "梦里",
}

// deepCoarseTimeMark 时间标记是否为空或粗粒度/相对表述（缓解误报③时间粒度差异）。
// 粗粒度标记下 AI 给出的 time_relation=earlier 不可靠（「三年后」被误判为倒流、
// 闪回被误判为时间线冲突），此时时间倒流只按「疑似」降级处理。
func deepCoarseTimeMark(s string) bool {
	n := deepNormalizeText(s)
	if n == "" {
		return true
	}
	for _, w := range deepCoarseTimeWords {
		if strings.Contains(n, w) {
			return true
		}
	}
	return deepCoarseTimeRe.MatchString(n)
}

// deepAffixes 称谓/绰号前后缀剥离表（缓解误报②角色别名/称谓）。项目数据
// （characters.json / lorebook）没有显式别名字段，只能对名单做保守的
// 「精确命中 → 唯一包含 → 剥一层称谓后重试」归一。
var deepAffixes = []string{
	"师祖", "师尊", "师父", "师傅", "师姐", "师妹", "师兄", "师弟",
	"前辈", "晚辈", "大人", "殿下", "陛下", "姑娘", "公子", "小姐",
	"道友", "掌门", "长老", "宗主", "阁下", "少侠", "将军", "老祖",
	"仙子", "圣主", "儿", "阿", "小", "老",
}

// deepAliasResolver 把状态卡中的角色名对齐到项目人物名单（canonical names）。
// nil 接收者安全（直接返回归一化原名，便于纯函数测试传 nil）。
// 归一是保守的：任一步候选不唯一即放弃映射（宁可对不齐，也不把两个角色错并）。
type deepAliasResolver struct {
	canon []string // 归一化后的既有人名
}

// newDeepAliasResolver 用既有人名建归一器（自动去重、归一化、去空）
func newDeepAliasResolver(names []string) *deepAliasResolver {
	seen := map[string]bool{}
	var canon []string
	for _, n := range names {
		n = deepNormalizeText(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		canon = append(canon, n)
	}
	sort.Strings(canon)
	if len(canon) == 0 {
		return nil
	}
	return &deepAliasResolver{canon: canon}
}

// candidates 与 n（已归一）相等或互为包含的 canonical 名单
func (r *deepAliasResolver) candidates(n string) []string {
	if r == nil {
		return nil
	}
	var hits []string
	for _, c := range r.canon {
		if c == n || strings.Contains(c, n) || strings.Contains(n, c) {
			hits = append(hits, c)
		}
	}
	return hits
}

// Resolve 归一化角色名：精确命中 → 唯一包含命中（「林晚师姐」→「林晚」）→
// 剥一层称谓前后缀后重试（「晚儿」→「晚」→「林晚」）。剥称谓后的单字允许
// 唯一包含命中（候选不唯一即放弃）。未命中返回归一化原名。
func (r *deepAliasResolver) Resolve(name string) string {
	n := deepNormalizeText(name)
	if n == "" || r == nil {
		return n
	}
	for _, c := range r.canon {
		if c == n {
			return c
		}
	}
	if len([]rune(n)) >= 2 {
		if hits := r.candidates(n); len(hits) == 1 {
			return hits[0]
		}
	}
	for _, aff := range deepAffixes {
		trimmed := strings.TrimPrefix(n, aff)
		if trimmed == n {
			trimmed = strings.TrimSuffix(n, aff)
		}
		if trimmed == "" || trimmed == n {
			continue
		}
		for _, c := range r.canon {
			if c == trimmed {
				return c
			}
		}
		if hits := r.candidates(trimmed); len(hits) == 1 {
			return hits[0]
		}
	}
	return n
}

// deepCanonicalPeople 项目人物名单（characters.json 全部角色 + lorebook 人物词条）
func deepCanonicalPeople(pm *project.Manager) []string {
	var names []string
	if chars, err := pm.ReadCharacters(); err == nil && chars != nil {
		for _, ch := range chars.Characters {
			if n := strings.TrimSpace(ch.Name); n != "" {
				names = append(names, n)
			}
		}
	}
	if lb, err := pm.ReadLorebook(); err == nil && lb != nil {
		for _, e := range lb.Entries {
			if e.Category == "character" {
				if n := strings.TrimSpace(e.Key); n != "" {
					names = append(names, n)
				}
			}
		}
	}
	return names
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

// deepMentionsList 列表中是否提及该物品（归一化包含匹配，缓解措辞/格式差异误报：
// 「玄铁剑」vs「那柄玄铁剑」vs「玄铁剑（残）」视为同一物品）
func deepMentionsList(list []string, item string) bool {
	for _, it := range list {
		if deepNormContains(it, item) || deepNormContains(item, it) {
			return true
		}
	}
	return false
}

// deepHeldContains held 表中是否持有该物品（归一化包含匹配，任一方向命中即可）
func deepHeldContains(held map[string]string, item string) bool {
	for name := range held {
		if deepNormContains(name, item) || deepNormContains(item, name) {
			return true
		}
	}
	return false
}

// deepLostLookup 在失去台账（key 为归一化物品名）中查找与 item 匹配的条目：
// 归一化相等优先，再唯一包含命中；多个候选不命中（避免误并不同物品）。
// 返回（标记失去的章节号, 台账 key, 是否命中）。
func deepLostLookup(lostAt map[string]int, norm string) (int, string, bool) {
	if at, ok := lostAt[norm]; ok {
		return at, norm, true
	}
	if len([]rune(norm)) < 2 {
		return 0, "", false
	}
	hit := ""
	for k := range lostAt {
		if strings.Contains(k, norm) || strings.Contains(norm, k) {
			if hit != "" {
				return 0, "", false
			}
			hit = k
		}
	}
	if hit == "" {
		return 0, "", false
	}
	return lostAt[hit], hit, true
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

// deepCompareStateLine 单条故事线内的跨章比对（纯函数）：cards 须同分支且按章节号升序；
// res 为角色名归一器（可为 nil）。检出矛盾类型与误报缓解分级：
//  1. 时间倒流：双方时间标记均精确 → error；任一方为空/粗粒度（「三年后」「翌日」类）
//     → warning + reason=granularity（疑似闪回/省略，非确定冲突）
//  2. 死亡角色后期以存活状态再出场：error；别名归一参与链接两章 → confidence=medium
//     + reason=alias
//  3. 位置变化无移动交代：同地异写（归一化相等）不告警；同区域表述差异（归一化后
//     互为包含）→ info + reason=wording；真跨区域 → warning（别名归一参与时
//     confidence=medium + reason=alias）
//  4. 关键物品凭空消失：warning + reason=unexplained（物品名归一化包含匹配，
//     「玄铁剑」vs「那柄玄铁剑」类措辞差异不再整条误报）
//  5. 失去物品无中生有：items_lost 明确交代过失去 → error；仅因「无交代消失」入账
//     → warning + reason=unexplained（可能后期才寻回/刻意留白），报后清除台账只报一次
func deepCompareStateLine(cards []*deepStateCard, res *deepAliasResolver) []deepIssue {
	var issues []deepIssue
	deadAt := map[string]int{}       // 归一化角色名 -> 标记死亡的章节号
	deadByAlias := map[string]bool{} // 死亡标记是否依赖别名归一
	lostAt := map[string]int{}       // 归一化物品名 -> 标记失去/损毁的章节号
	lostExplicit := map[string]bool{}
	prevHeld := map[string]string{}
	var prev *deepStateCard

	for _, curr := range cards {
		place := deepPlace(curr.Chapter, curr.Branch)

		// 1. 时间倒流
		if prev != nil && strings.EqualFold(strings.TrimSpace(curr.TimeRelation), "earlier") {
			iss := deepIssue{Confidence: "high"}
			iss.Category = "timeline"
			iss.Location = place
			iss.Branch = curr.Branch
			iss.Evidence = fmt.Sprintf("上一章 time_mark=%s，本章 time_mark=%s / time_relation=earlier", prev.TimeMark, curr.TimeMark)
			iss.Suggestion = "核对两章时间标记与叙事顺序，调整时间线或在文中说明（闪回需有明确标记）"
			if deepCoarseTimeMark(prev.TimeMark) || deepCoarseTimeMark(curr.TimeMark) {
				iss.Severity = "warning"
				iss.Confidence = "medium"
				iss.Reason = "granularity"
				iss.Description = fmt.Sprintf("%s 的时间标记（%s）早于上一章（%s），疑似时间倒流（时间表述存在粒度差异，可能为闪回/省略，非确定冲突）", place, curr.TimeMark, prev.TimeMark)
			} else {
				iss.Severity = "error"
				iss.Description = fmt.Sprintf("%s 的时间标记（%s）早于上一章（%s），疑似时间倒流", place, curr.TimeMark, prev.TimeMark)
			}
			issues = append(issues, iss)
		}

		prevLoc := map[string]string{}
		if prev != nil {
			for _, c := range prev.Characters {
				if loc := strings.TrimSpace(c.Location); loc != "" {
					prevLoc[res.Resolve(c.Name)] = loc
				}
			}
		}
		noTravel := len(curr.TravelNotes) == 0

		for _, c := range curr.Characters {
			rawName := strings.TrimSpace(c.Name)
			if rawName == "" {
				continue
			}
			name := res.Resolve(rawName)
			aliasResolved := name != deepNormalizeText(rawName)
			status := deepNormStatus(c.Status)

			// 2. 死亡角色再出场（复活仅报一次：报后清除死亡标记）
			if at, ok := deadAt[name]; ok && status == "alive" {
				iss := deepIssue{Confidence: "high"}
				if aliasResolved || deadByAlias[name] {
					iss.Confidence = "medium"
					iss.Reason = "alias"
				}
				iss.Severity = "error"
				iss.Category = "status"
				iss.EntityName = name
				iss.Description = fmt.Sprintf("%s 在第%d章已死亡，但%s仍以存活状态出场", name, at, place)
				iss.Location = place
				iss.Evidence = fmt.Sprintf("第%d章状态卡 status=dead；%s状态卡 status=alive", at, place)
				if aliasResolved || deadByAlias[name] {
					iss.Evidence += fmt.Sprintf("（「%s」按人物名单归一为「%s」，如为称谓误并请核对）", rawName, name)
				}
				iss.Suggestion = "确认是否为复活/回忆/幻象并在文中明确交代，否则修正角色生死状态"
				iss.Branch = curr.Branch
				issues = append(issues, iss)
				delete(deadAt, name)
				delete(deadByAlias, name)
				continue
			}

			// 3. 位置比对（归一化包含 → 同区域表述差异降级为提示）
			if prev != nil && status != "dead" {
				if prevL, ok := prevLoc[name]; ok {
					if loc := strings.TrimSpace(c.Location); loc != "" {
						switch {
						case deepNormalizeText(prevL) == deepNormalizeText(loc):
							// 同地异写（全半角/空白/标点差异），不告警
						case deepNormContains(prevL, loc) || deepNormContains(loc, prevL):
							iss := deepIssue{Confidence: "low", Reason: "wording"}
							iss.Severity = "info"
							iss.Category = "status"
							iss.EntityName = name
							iss.Description = fmt.Sprintf("%s 所在地从「%s」变为「%s」，为同区域表述差异，提示核对移动描写（非冲突）", name, prevL, loc)
							iss.Location = place
							iss.Evidence = fmt.Sprintf("上一章 location=%s；%slocation=%s（归一化后互为包含）", prevL, place, loc)
							iss.Suggestion = "如为跨区域移动请补充赶路/传送交代；同区域内活动可忽略此提示"
							iss.Branch = curr.Branch
							issues = append(issues, iss)
						case noTravel:
							iss := deepIssue{Confidence: "high"}
							if aliasResolved {
								iss.Confidence = "medium"
								iss.Reason = "alias"
							}
							iss.Severity = "warning"
							iss.Category = "status"
							iss.EntityName = name
							iss.Description = fmt.Sprintf("%s 所在地从「%s」变为「%s」，但本章无移动/赶路交代，疑似位置瞬移", name, prevL, loc)
							iss.Location = place
							iss.Evidence = fmt.Sprintf("上一章 location=%s；%slocation=%s，travel_notes 为空", prevL, place, loc)
							iss.Suggestion = "补充角色移动过程（赶路/传送/时间跳跃），或修正两地之一的位置描述"
							iss.Branch = curr.Branch
							issues = append(issues, iss)
						}
					}
				}
			}

			if status == "dead" {
				deadAt[name] = curr.Chapter
				deadByAlias[name] = aliasResolved
			}
		}

		// 4/5. 关键物品比对（相邻章 + 失踪物品台账）
		held := deepHeldItems(curr)
		if prev != nil {
			for _, item := range deepSortedKeys(prevHeld) {
				if deepHeldContains(held, item) {
					continue
				}
				if deepMentionsList(curr.ItemsLost, item) || deepMentionsList(prev.ItemsLost, item) {
					continue
				}
				// 无交代消失视同失去：记入台账，后续无交代地再次出现按「无中生有」处理
				lostAt[deepNormalizeText(item)] = curr.Chapter
				lostExplicit[deepNormalizeText(item)] = false
				iss := deepIssue{Confidence: "medium", Reason: "unexplained"}
				iss.Severity = "warning"
				iss.Category = "item"
				iss.EntityName = item
				iss.Description = fmt.Sprintf("关键物品「%s」凭空消失：上一章由 %s 持有，%s已无人持有且无失去交代", item, prevHeld[item], place)
				iss.Location = place
				iss.Evidence = fmt.Sprintf("上一章状态卡 items 含「%s」（持有者 %s）；%s状态卡无人持有且 items_lost 未提及", item, prevHeld[item], place)
				iss.Suggestion = "补写物品去向（遗失/被夺/存放），或从角色物品清单中移除"
				iss.Branch = curr.Branch
				issues = append(issues, iss)
			}
			for _, item := range deepSortedKeys(held) {
				at, key, ok := deepLostLookup(lostAt, deepNormalizeText(item))
				if !ok || deepMentionsList(curr.ItemsRegained, item) {
					continue
				}
				iss := deepIssue{Confidence: "high"}
				iss.Category = "item"
				iss.EntityName = item
				iss.Location = place
				iss.Branch = curr.Branch
				iss.Evidence = fmt.Sprintf("第%d章状态卡 items_lost 含「%s」；%s状态卡 items 含「%s」且 items_regained 未提及", at, item, place, item)
				iss.Suggestion = "补写物品失而复得的过程（寻回/修复/复制），或修正前后章的物品状态"
				if lostExplicit[key] {
					iss.Severity = "error"
					iss.Description = fmt.Sprintf("已失去/损毁的关键物品「%s」无中生有：第%d章已标记失去，%s由 %s 再次持有且无重新获得交代", item, at, place, held[item])
				} else {
					// 仅因「无交代消失」入账：可能刻意留白/后期才寻回/提取遗漏，降为疑似
					iss.Severity = "warning"
					iss.Confidence = "medium"
					iss.Reason = "unexplained"
					iss.Description = fmt.Sprintf("关键物品「%s」在第%d章无交代消失后，%s由 %s 再次持有且无重新获得交代（疑似同一物品去留不明，非确定冲突）", item, at, place, held[item])
					// 报后清除台账：再出现即已知，不逐章重复
					delete(lostAt, key)
					delete(lostExplicit, key)
				}
				issues = append(issues, iss)
			}
		}
		for _, it := range curr.ItemsLost {
			if it = strings.TrimSpace(it); it != "" {
				lostAt[deepNormalizeText(it)] = curr.Chapter
				lostExplicit[deepNormalizeText(it)] = true
			}
		}
		for _, it := range curr.ItemsRegained {
			if it = strings.TrimSpace(it); it != "" {
				if _, key, ok := deepLostLookup(lostAt, deepNormalizeText(it)); ok {
					delete(lostAt, key)
					delete(lostExplicit, key)
				}
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

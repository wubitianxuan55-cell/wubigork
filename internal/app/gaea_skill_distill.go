package app

// 阶段七 7.2-2 journal 历史蒸馏（docs/gaea-stage7-plan-2026-09.md §2）：把
// 证据链 Journal 的跨会话沉淀变成程序性记忆的矿床——确定性挖掘（internal/
// skilldistill 纯函数，线 A）跨会话重复 ≥3 次的工具应用模式 → 建议卡只提示 →
// 用户点「结晶为技能」→ LLM 从多次执行的回放蒸馏草稿（复用 7.2-1 审阅管道：
// 编辑→GaeaSkillDraftSave 落盘）→ 决定与结晶全程落审计状态。与 7.2-1 的关系：
// 会话录制=当场演示单次过程；本通道=历史证据链的多次执行共性回放。
// 红线：纯同步按需（无常驻）；work 空间一期（journal 本身恒 work）；只提示
// 不打扰（用户 ignore 的模式静默不再出现）；审计链完整（哪次历史、谁确认、
// 结晶成哪个技能）。本文件只做数据组装、状态持久化与 LLM 编排——挖掘与
// 归一化全在 skilldistill（零 IO 表驱动）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/skilldistill"
	"github.com/gaea/gaea/internal/util"
)

// SkillDistillCandidateView 一条重复模式候选（Wails 绑定，MemoryPanel
// 「流程蒸馏」分区消费）。Pattern 为 PatternLine 行（"edit_file .md"）。
type SkillDistillCandidateView struct {
	ID       string   `json:"id"`
	Pattern  []string `json:"pattern"`
	Repeat   int      `json:"repeat"`
	Sessions []string `json:"sessions"`
	Evidence []string `json:"evidence"`
	LastAt   int64    `json:"lastAt"`
}

// SkillDistillView 候选总览：现算只读、零 LLM。Available=journal 目录可读
// 且读到 ≥1 张证据卡（无候选与不可用两态分开，前端空态两分）。
type SkillDistillView struct {
	Candidates  []SkillDistillCandidateView `json:"candidates"`
	Available   bool                        `json:"available"`
	GeneratedAt string                      `json:"generatedAt"`
}

// ── 决定状态持久化（<DataRoot>/skill_distill_state.json）──────────────────
// 结构 {"version":1,"ignored":{"<id>":{"decidedAt"}},"crystallized":[…]}；
// load 容错（缺失/损坏回空——候选可重算，状态丢得起）、save 原子
// （temp+rename，route_suggestions 同款手法）。

// skillDistillDecision 一条「不再提示」决定（ID 确定性→重算幂等静默）。
type skillDistillDecision struct {
	DecidedAt string `json:"decidedAt"` // RFC3339
}

// skillDistillCrystallized 结晶审计条目：哪个模式、何时、由哪些会话的证据、
// 结晶成哪个技能（判据③：审计链完整）。
type skillDistillCrystallized struct {
	ID        string   `json:"id"`
	Skill     string   `json:"skill"`
	Sessions  []string `json:"sessions"`
	DecidedAt string   `json:"decidedAt"`
}

type skillDistillStateFile struct {
	Version      int                             `json:"version"`
	Ignored      map[string]skillDistillDecision `json:"ignored"`
	Crystallized []skillDistillCrystallized      `json:"crystallized"`
}

func skillDistillStatePath(dataRoot string) string {
	return filepath.Join(dataRoot, "skill_distill_state.json")
}

// loadSkillDistillState 读蒸馏决定状态；文件缺失/损坏一律回空（Ignored/Crystallized
// 恒非 nil，调用方零判空）。
func loadSkillDistillState(dataRoot string) skillDistillStateFile {
	empty := skillDistillStateFile{Version: 1, Ignored: map[string]skillDistillDecision{}, Crystallized: []skillDistillCrystallized{}}
	b, err := os.ReadFile(skillDistillStatePath(dataRoot))
	if err != nil {
		return empty
	}
	var f skillDistillStateFile
	if err := json.Unmarshal(b, &f); err != nil || f.Ignored == nil {
		return empty
	}
	if f.Crystallized == nil {
		f.Crystallized = []skillDistillCrystallized{}
	}
	if f.Version == 0 {
		f.Version = 1
	}
	return f
}

// saveSkillDistillState 整写状态（temp+rename 原子落盘，route_suggestions 手法）。
func saveSkillDistillState(dataRoot string, f skillDistillStateFile) error {
	if f.Ignored == nil {
		f.Ignored = map[string]skillDistillDecision{}
	}
	if f.Crystallized == nil {
		f.Crystallized = []skillDistillCrystallized{}
	}
	f.Version = 1
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	p := skillDistillStatePath(dataRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "skill-distill-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := fileutil.RenameWithRetry(tmpName, p); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// validSkillDistillID 候选 ID 形状防御（jd-+8 小写十六进制）：直接复用
// skilldistill.ParsePatternID——ID 生成与校验同一口径，单一来源不另养正则
// （先例 ParseSuggestionID 防御：ID 只在落盘咽喉二次校验）。
func validSkillDistillID(id string) bool { return skilldistill.ParsePatternID(id) }

// ── 多次执行回放构建（Draft 专用，纯函数）─────────────────────────────────

const (
	journalReplayMaxSessions = 3   // 参与回放的最近会话数上限（模式证据 Sessions 最近优先）
	journalReplaySummaryMax  = 200 // 单条 AfterSummary rune 截断
	journalReplaySegMax      = 600 // 单段（一次执行）整体 rune 截断防护
)

// journalStep 一次工具应用的完整证据——比 skilldistill.Record 多带 AfterSummary
// 与 At（回放排序/段头日期原料）。本地中间结构，不进 skilldistill（防契约漂移）。
type journalStep struct {
	tool   string
	target string
	after  string
	at     int64
}

// journalEvidence 一个会话的证据链（steps 排序前可乱序；lastAt=末步时间）。
type journalEvidence struct {
	sessionID string
	steps     []journalStep
	lastAt    int64
}

// buildJournalReplay 把候选证据压成「多次执行」回放文本：每会话一段
// （段头「── 第 k 次执行（会话 {sid}，{YYYY-MM-DD}）──」，日期取末步 At），
// 段内逐步「【工具】{tool} {target}」，其后跟 AfterSummary 前 200 rune 一行
// （有才写，换行压平保单行）；最近 ≤3 个会话，展示按时间正序（第 1 次=最早）。
// 纯函数、确定性（喂给 LLM 的与返回前端的回放完全一致）。
func buildJournalReplay(sessions []journalEvidence) string {
	sorted := make([]journalEvidence, len(sessions))
	copy(sorted, sessions)
	// 最近优先入选（不信任入参顺序——候选 Sessions 口径最近优先，这里仍自证）
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].lastAt > sorted[j].lastAt })
	if len(sorted) > journalReplayMaxSessions {
		sorted = sorted[:journalReplayMaxSessions]
	}
	// 展示按时间正序：第 1 次执行=最早（叙事口径）
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].lastAt < sorted[j].lastAt })
	var segs []string
	for k, ev := range sorted {
		steps := make([]journalStep, len(ev.steps))
		copy(steps, ev.steps)
		sort.SliceStable(steps, func(i, j int) bool {
			if steps[i].at != steps[j].at {
				return steps[i].at < steps[j].at
			}
			if steps[i].tool != steps[j].tool {
				return steps[i].tool < steps[j].tool
			}
			return steps[i].target < steps[j].target
		})
		var b strings.Builder
		fmt.Fprintf(&b, "── 第 %d 次执行（会话 %s，%s）──\n", k+1, ev.sessionID, time.UnixMilli(ev.lastAt).Format("2006-01-02"))
		for _, st := range steps {
			line := strings.TrimSpace(st.tool + " " + st.target)
			if line == "" {
				continue
			}
			b.WriteString("【工具】" + truncRunes(line, skillReplayToolMax) + "\n")
			if s := strings.TrimSpace(st.after); s != "" {
				b.WriteString("　　→ " + truncRunes(strings.ReplaceAll(s, "\n", " "), journalReplaySummaryMax) + "\n")
			}
		}
		segs = append(segs, truncRunes(strings.TrimRight(b.String(), "\n"), journalReplaySegMax))
	}
	return strings.Join(segs, "\n\n")
}

// ── journal 读取与挖掘管道（Candidates/Draft/Decide 共用）──────────────────

// loadJournalEvidence 读 journal 全量证据卡（GaeaJournalList 同款：ReadDir +
// OpenJournal + 逐文件 List），双投影：skilldistill.Record（挖掘用）+
// journalEvidence（Draft 回放用——ChangeRecord.AfterSummary 在投影时一并带出，
// 不塞进 skilldistill.Record 防契约漂移）。目录不可读 → available=false
// （只读兼容：旧工作区无 journal 不报错）。available=读到 ≥1 张证据卡。
func loadJournalEvidence() (records []skilldistill.Record, evs map[string]journalEvidence, available bool) {
	dir := filepath.Join(gaeaCwd(), ".gaea", "work", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, false
	}
	st, err := evidence.OpenJournal(dir)
	if err != nil {
		return nil, nil, false
	}
	evs = map[string]journalEvidence{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		recs, err := st.List(e.Name()[:len(e.Name())-len(".jsonl")])
		if err != nil {
			continue
		}
		for _, r := range recs {
			available = true
			if r.SessionID == "" || r.Tool == "" {
				continue // MineFlows 同口径：Tool/SessionID 空不参评
			}
			records = append(records, skilldistill.Record{SessionID: r.SessionID, Tool: r.Tool, Target: r.Target, At: r.At})
			ev := evs[r.SessionID]
			ev.sessionID = r.SessionID
			ev.steps = append(ev.steps, journalStep{tool: r.Tool, target: r.Target, after: r.AfterSummary, at: r.At})
			if r.At > ev.lastAt {
				ev.lastAt = r.At
			}
			evs[r.SessionID] = ev
		}
	}
	return records, evs, available
}

// mineSkillDistill 全管道：journal 读取 → MineFlows → Distill。withSilenced=true
// 时 ignored ∪ crystallized 的 ID 集双过滤（判据①：用户拒绝的模式静默）；
// false 供 Decide 追溯审计 Sessions（已被决定过的模式证据仍可查）。
func mineSkillDistill(dataRoot string, withSilenced bool) ([]skilldistill.Candidate, map[string]journalEvidence, bool) {
	records, evs, available := loadJournalEvidence()
	if !available {
		return []skilldistill.Candidate{}, evs, false
	}
	silenced := map[string]struct{}{}
	if withSilenced {
		st := loadSkillDistillState(dataRoot)
		for id := range st.Ignored {
			silenced[id] = struct{}{}
		}
		for _, c := range st.Crystallized {
			silenced[c.ID] = struct{}{}
		}
	}
	return skilldistill.Distill(skilldistill.MineFlows(records), silenced), evs, true
}

// skillDistillCandidateToView 候选 → 绑定视图（Pattern=PatternLine 行；切片
// 恒非 nil，前端零判空）。
func skillDistillCandidateToView(c skilldistill.Candidate) SkillDistillCandidateView {
	v := SkillDistillCandidateView{
		ID:       c.ID,
		Pattern:  make([]string, 0, len(c.Pattern)),
		Repeat:   c.Repeat,
		Sessions: nonNilStrings(c.Sessions),
		Evidence: nonNilStrings(c.Evidence),
		LastAt:   c.LastAt,
	}
	for _, p := range c.Pattern {
		v.Pattern = append(v.Pattern, skilldistill.PatternLine(p))
	}
	return v
}

// GaeaSkillDistillCandidates 现算候选（只读、零 LLM、确定性——同 journal 与
// 状态文件重算幂等）。journal 目录不可读 → Available=false 空表不报错。
func (a *App) GaeaSkillDistillCandidates() SkillDistillView {
	cands, _, available := mineSkillDistill(config.DataRoot(), true)
	view := SkillDistillView{
		Candidates:  make([]SkillDistillCandidateView, 0, len(cands)),
		Available:   available,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for _, c := range cands {
		view.Candidates = append(view.Candidates, skillDistillCandidateToView(c))
	}
	slog.Info("journal 蒸馏候选已计算", "candidates", len(view.Candidates), "available", available)
	return view
}

// GaeaSkillDistillDraft 从多次执行的证据回放蒸馏技能草稿（LLM 结构化 +
// RetryJSON 格式重试；办公功能级路由，本地优先）。草稿只回不落盘——落盘
// 必经 GaeaSkillDraftSave（7.2-1 审阅管道，用户是作者）。
func (a *App) GaeaSkillDistillDraft(patternID string) (SkillRecordResult, error) {
	if gaeaCtrl() == nil {
		return SkillRecordResult{}, fmt.Errorf("办公引擎未初始化")
	}
	// 重算候选按 ID 找（同 Candidates 管道——建议卡与草稿证据同源）
	cands, evs, _ := mineSkillDistill(config.DataRoot(), true)
	var cand *skilldistill.Candidate
	for i := range cands {
		if cands[i].ID == patternID {
			cand = &cands[i]
			break
		}
	}
	if cand == nil {
		return SkillRecordResult{}, fmt.Errorf("未找到该模式（可能已被处理）")
	}
	// 回放：候选 Sessions（最近优先）映射为完整证据（含 AfterSummary）
	var sessions []journalEvidence
	for _, sid := range cand.Sessions {
		if ev, ok := evs[sid]; ok {
			sessions = append(sessions, ev)
		}
	}
	replay := buildJournalReplay(sessions)
	if a.eng == nil {
		return SkillRecordResult{}, fmt.Errorf("提示词引擎未初始化")
	}
	tmpl := a.eng.Get("skill-from-journal")
	if tmpl == nil {
		return SkillRecordResult{}, fmt.Errorf("缺少 skill-from-journal 模板文件")
	}
	if a.client == nil {
		return SkillRecordResult{}, fmt.Errorf("ai client unavailable")
	}
	engID, model, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, model, sys, usr,
			ai.ChatSimpleOptions{EngineID: engID, Feature: "office", Temperature: 0.2, MaxTokens: 1500, TimeoutMinutes: 3})
	}
	patternLines := make([]string, 0, len(cand.Pattern))
	for _, p := range cand.Pattern {
		patternLines = append(patternLines, skilldistill.PatternLine(p))
	}
	system := tmpl.BuildSystemPrompt("")
	user := tmpl.BuildUserPrompt(map[string]string{
		"replay":  replay,
		"pattern": strings.Join(patternLines, "\n"),
		"repeat":  strconv.Itoa(cand.Repeat),
	})
	jsonStr, err := util.RetryJSON(ctx, caller, system, user, 2)
	if err != nil {
		return SkillRecordResult{}, fmt.Errorf("蒸馏技能草稿失败: %w", err)
	}
	var draft SkillDraft
	if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
		return SkillRecordResult{}, fmt.Errorf("解析技能草稿失败: %w", err)
	}
	if err := parseSkillDraft(&draft); err != nil {
		return SkillRecordResult{}, err
	}
	slog.Info("journal 蒸馏草稿就绪", "pattern", patternID, "repeat", cand.Repeat, "sessions", len(sessions))
	return SkillRecordResult{
		Draft:   draft,
		Replay:  replay,
		Preview: skill.RenderSkillFile(draft.Name, draft.Description, renderSkillDraftBody(draft)),
	}, nil
}

// GaeaSkillDistillDecide 落决定与审计：decision ∈ {"ignore","crystallized"}；
// ignore=不再提示（重算静默）；crystallized=已结晶（必须带技能名，审计条目
// 记录哪些会话、何时、结晶成哪个技能）。写状态文件成功即返回 nil。
func (a *App) GaeaSkillDistillDecide(patternID, decision, skillName string) error {
	if !validSkillDistillID(patternID) {
		return &appError{"模式 ID 不合法: " + patternID}
	}
	if decision != "ignore" && decision != "crystallized" {
		return &appError{"未知决定: " + decision + "（需 ignore 或 crystallized）"}
	}
	skillName = strings.TrimSpace(skillName)
	if decision == "crystallized" && skillName == "" {
		return &appError{"结晶为技能必须提供技能名"}
	}
	// 审计要带证据会话：重算原始候选（不过滤已决定——同一模式重复决定仍可
	// 追溯）取该模式的 Sessions。
	dataRoot := config.DataRoot()
	cands, _, _ := mineSkillDistill(dataRoot, false)
	var sessions []string
	found := false
	for _, c := range cands {
		if c.ID == patternID {
			sessions = c.Sessions
			found = true
			break
		}
	}
	if !found {
		return &appError{"未找到该模式（可能已被处理）"}
	}
	st := loadSkillDistillState(dataRoot)
	now := time.Now().Format(time.RFC3339)
	if decision == "ignore" {
		st.Ignored[patternID] = skillDistillDecision{DecidedAt: now}
	} else {
		st.Crystallized = append(st.Crystallized, skillDistillCrystallized{
			ID: patternID, Skill: skillName, Sessions: nonNilStrings(sessions), DecidedAt: now,
		})
	}
	if err := saveSkillDistillState(dataRoot, st); err != nil {
		return fmt.Errorf("写入蒸馏决定状态失败: %w", err)
	}
	slog.Info("journal 蒸馏决定已落审计", "pattern", patternID, "decision", decision, "skill", skillName)
	return nil
}

package analysis

import (
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// ── 记忆提取规则表（t3-P1 记忆生产者）────────────────────────
//
// 口径来源：docs/distill/03-long-range-consistency.md §2.3 / §12.4-4
// （对齐 MuMu plot_analyzer.py:305-486 的入库门槛规格）。纯函数零 IO：
// 从 V2 分析载荷确定性抽取本章故事记忆，入库门槛与重要性赋值全在常量表。

// 入库门槛与固定重要性（MuMu :329-479 同参；改门槛=改这里，勿散落魔数）。
const (
	// MemoryHookThreshold hook.strength ≥ 6 才入库（plot_analyzer.py:362）。
	MemoryHookThreshold = 6.0
	// MemoryPlotPointThreshold plot_point.importance ≥ 0.6 才入库（:415）。
	MemoryPlotPointThreshold = 0.6
	// MemoryConflictThreshold conflict.level ≥ 7 才入库（:456，记为 plot_point 类）。
	MemoryConflictThreshold = 7.0
	// MemorySummaryImportance chapter_summary 固定重要性 0.6（:329）。
	MemorySummaryImportance = 0.6
	// MemoryCharEventImportance character_event 固定重要性 0.7（:439）。
	MemoryCharEventImportance = 0.7
	// memorySummaryFallbackLen 章节内容兜底链末级的截断（:359 content[:300]）。
	memorySummaryFallbackLen = 300
)

// ExtractStoryMemories 从 V2 分析载荷抽取本章故事记忆（纯函数，确定性：
// 同输入同输出，ID 按类型内序号稳定编排）。
//
// 六类规则（spec §2.3）：
//  1. chapter_summary 必有一条：summary → 前 3 条推进点拼接 → 章节正文前 300 字
//     （三级回退链，全空则跳过）；
//  2. hook：strength ≥ 6 入库，importance = min(strength/10, 1)；
//  3. foreshadow：全部入库（is_foreshadow：planted=1 / resolved=2），importance 同上
//     （strength 缺省语义 5，与 t1-P2 埋入同步同口径）；
//  4. plot_point：importance ≥ 0.6 入库，importance 原样；
//  5. character_event：每个角色状态差分一条，固定 0.7，related_characters=[角色名]；
//  6. conflict：level ≥ 7 记为 plot_point 类，importance = min(level/10, 1)。
func ExtractStoryMemories(chapterNum int, chapterFile string, v2 *types.AnalysisResultV2, chapterContent string) []types.StoryMemory {
	if v2 == nil {
		return nil
	}
	out := make([]types.StoryMemory, 0, 8)
	ordinal := map[string]int{}
	add := func(memType string, m types.StoryMemory) {
		ordinal[memType]++
		m.ID = types.StoryMemoryID(chapterNum, memType, ordinal[memType])
		m.ChapterNum = chapterNum
		m.ChapterFile = chapterFile
		m.Type = memType
		m.TextLen = len([]rune(m.Content))
		out = append(out, m)
	}
	trimClamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	// ① chapter_summary：三级回退链（MuMu :329-359）
	summary := strings.TrimSpace(v2.Summary)
	if summary == "" && len(v2.PlotPoints) > 0 {
		pp := make([]string, 0, 3)
		for i, p := range v2.PlotPoints {
			if i >= 3 {
				break
			}
			if c := strings.TrimSpace(p.Content); c != "" {
				pp = append(pp, c)
			}
		}
		summary = strings.Join(pp, "；")
	}
	if summary == "" && chapterContent != "" {
		// 纯截断不带省略号（MuMu content[:300] 是存储截断，展示省略号不入记忆正文）
		r := []rune(strings.TrimSpace(chapterContent))
		if len(r) > memorySummaryFallbackLen {
			r = r[:memorySummaryFallbackLen]
		}
		summary = string(r)
	}
	if summary != "" {
		add(types.MemoryTypeChapterSummary, types.StoryMemory{
			Content:    summary,
			Importance: MemorySummaryImportance,
		})
	}

	// ② hook：门槛 strength ≥ 6（:362-385）
	for _, h := range v2.Hooks {
		if float64(h.Strength) < MemoryHookThreshold {
			continue
		}
		c := strings.TrimSpace(h.Content)
		if c == "" {
			continue
		}
		add(types.MemoryTypeHook, types.StoryMemory{
			Title:      strings.TrimSpace(h.Type),
			Content:    c,
			Importance: trimClamp(float64(h.Strength) / 10.0),
		})
	}

	// ③ foreshadow：全部入库（:388-412）；is_foreshadow：planted=1 / resolved=2
	for _, f := range v2.Foreshadows {
		c := strings.TrimSpace(f.Content)
		if c == "" {
			continue
		}
		strength := f.Strength
		if strength <= 0 {
			strength = 5 // 缺省语义 5，与 t1-P2 埋入同步同口径
		}
		flag := types.ForeshadowFlagPlanted
		if strings.EqualFold(strings.TrimSpace(f.Type), "resolved") {
			flag = types.ForeshadowFlagResolved
		}
		add(types.MemoryTypeForeshadow, types.StoryMemory{
			Title:             strings.TrimSpace(f.Title),
			Content:           c,
			Importance:        trimClamp(float64(strength) / 10.0),
			IsForeshadow:      flag,
			RelatedCharacters: f.RelatedChars,
		})
	}

	// ④ plot_point：门槛 importance ≥ 0.6（:415-436）
	for _, p := range v2.PlotPoints {
		if p.Importance < MemoryPlotPointThreshold {
			continue
		}
		c := strings.TrimSpace(p.Content)
		if c == "" {
			continue
		}
		add(types.MemoryTypePlotPoint, types.StoryMemory{
			Content:    c,
			Importance: trimClamp(p.Importance),
		})
	}

	// ⑤ character_event：每个角色状态差分一条，固定 0.7（:439-453）
	for _, cs := range v2.CharacterStates {
		name := strings.TrimSpace(cs.Name)
		if name == "" {
			continue
		}
		seg := make([]string, 0, 4)
		if old, new := strings.TrimSpace(cs.OldState), strings.TrimSpace(cs.NewState); old != "" || new != "" {
			seg = append(seg, name+"："+old+" → "+new)
		}
		if s := strings.TrimSpace(cs.PsychologicalChange); s != "" {
			seg = append(seg, "心理："+s)
		}
		if s := strings.TrimSpace(cs.KeyEvent); s != "" {
			seg = append(seg, "事件："+s)
		}
		if len(seg) == 0 {
			continue
		}
		add(types.MemoryTypeCharacterEvent, types.StoryMemory{
			Title:             name,
			Content:           strings.Join(seg, "；"),
			Importance:        MemoryCharEventImportance,
			RelatedCharacters: []string{name},
		})
	}

	// ⑥ conflict：门槛 level ≥ 7，记为 plot_point 类（:456-479）
	if v2.Conflict.Level >= MemoryConflictThreshold {
		c := strings.TrimSpace(v2.Conflict.Description)
		if c != "" {
			add(types.MemoryTypePlotPoint, types.StoryMemory{
				Title:      "冲突",
				Content:    c,
				Importance: trimClamp(float64(v2.Conflict.Level) / 10.0),
			})
		}
	}

	return out
}

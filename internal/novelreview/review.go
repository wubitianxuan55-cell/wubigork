package novelreview

// 确定性质量评审引擎（零 LLM / 零网络 / 零 IO）。
//
// 口径重推导自 oh-story-claudecode（MIT）：references/quality-rubric.md 的
// 维度 × PASS/WARN/FAIL 与 S1~S4 分级、references/rubrics/{fanqie,qidian,zhihu}.md
// 的平台差异（章节长度/段落节奏/人称/篇幅带）、story-long-write 的「黄金三问」。
// 判定词表与阈值全部来自 rubric.json 数据资产；本文件只留算法。
//
// 设计取舍（有意与上游不同，写在码里）：
//   - 上游是「给 LLM 的评审提纲」，本引擎是**确定性可单测的体检**：凡需语义
//     判断的维度（核心卖点是否成立、伏笔是否回收）不在此下结论，只做可核对的
//     结构性检查；外部数据缺失时**显式 skip 并说明原因**，不静默给 PASS。
//   - 每条 finding 一律附原文证据（rune 区间 + 段落号），对应黄金三问第三问。

import (
	"fmt"
)

// Options 评审上下文（外部数据由 app 层注入；缺省即跳过对应维度）。
type Options struct {
	ChapterNum       int
	ProtagonistNames []string
}

// EvidenceSpan 命中位置：rune 偏移 + 段落号（1-based；0=无法定位到段落）。
type EvidenceSpan struct {
	Start     int `json:"start"`
	End       int `json:"end"`
	Paragraph int `json:"paragraph"`
}

// Dimension 单维度评审结果（verdict: pass / warn / fail / skip）。
type Dimension struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Verdict  string         `json:"verdict"`
	Severity string         `json:"severity,omitempty"` // S1~S4（pass/skip 缺省）
	Detail   string         `json:"detail"`
	Advice   string         `json:"advice,omitempty"`
	Evidence []EvidenceSpan `json:"evidence,omitempty"`
}

// Report 一次章节评审的完整报告。
type Report struct {
	Platform      string         `json:"platform"`
	PlatformLabel string         `json:"platformLabel"`
	ChapterNum    int            `json:"chapterNum"`
	Words         int            `json:"words"`
	Verdict       string         `json:"verdict"` // APPROVE / CONCERNS / REJECT
	Counts        map[string]int `json:"counts"`
	Dimensions    []Dimension    `json:"dimensions"`
	Advisories    []string       `json:"advisories,omitempty"`
}

// 判定结论（与上游 quality-rubric.md 发布建议门槛同口径）。
const (
	VerdictApprove  = "APPROVE"
	VerdictConcerns = "CONCERNS"
	VerdictReject   = "REJECT"
)

// 维度结论。
const (
	verdictPass = "pass"
	verdictWarn = "warn"
	verdictFail = "fail"
	verdictSkip = "skip"
)

// dimensionLabels 维度中文名（面板直显；数据资产只给阈值与严重度）。
var dimensionLabels = map[string]string{
	"length_band":             "字数区间",
	"opening_freshness":       "开篇钩子",
	"hook_expectation":        "章尾钩子",
	"trailer_ending":          "预告式收尾",
	"emotion_curve":           "情绪节点密度",
	"plot_loop":               "爽点/升级密度",
	"format_readability":      "段落节奏",
	"dialogue_ratio":          "对话占比",
	"punctuation_rhythm":      "标点节奏",
	"dash_usage":              "破折号",
	"format_hygiene":          "格式合规",
	"perspective_consistency": "人称视角",
	"protagonist_presence":    "主角存在感",
	"lever_mention":           "金手指提及",
	"stated_length_accuracy":  "字数表述核对",
}

// paragraph 段落（1-based 序号 + rune 区间 [start,end)）。
type paragraph struct {
	idx        int
	start, end int
	text       string
}

// Review 对一段正文做确定性质量评审；platformID 为空或未知时回落 general 档。
func Review(text, platformID string, opts Options) (*Report, error) {
	runes := []rune(text)
	if countNonSpaceRunes(runes) == 0 {
		return nil, fmt.Errorf("文本为空，无可评审内容")
	}
	p, err := PlatformByID(platformID)
	if err != nil {
		return nil, err
	}
	r := currentRubric()
	paras := splitParagraphs(runes)
	words := countNonSpaceRunes(runes)

	dims := make([]Dimension, 0, 16)
	dims = append(dims, dimLength(p, words))
	dims = append(dims, dimOpeningHook(p, r, paras))
	dims = append(dims, dimEndingHook(p, r, paras))
	dims = append(dims, dimTrailer(p, r, paras))
	dims = append(dims, dimEmotionDensity(p, r, runes, paras, words))
	if p.PowerPer3k > 0 {
		dims = append(dims, dimPowerDensity(p, r, runes, words))
	}
	dims = append(dims, dimParagraphPace(p, paras))
	dims = append(dims, dimDialogRatio(p, runes, words))
	dims = append(dims, dimPunctuation(p, runes, paras, words))
	dims = append(dims, dimDash(p, runes, paras))
	dims = append(dims, dimFormat(p, text))
	if p.FirstPerson {
		dims = append(dims, dimPerspective(p, runes, paras))
	}
	dims = append(dims, dimProtagonist(p, runes, paras, opts.ProtagonistNames))
	if p.LeverRequired {
		dims = append(dims, dimLever(p, r, runes, words))
	}
	dims = append(dims, dimWordcountExpr(p, runes, paras))

	counts := map[string]int{"S1": 0, "S2": 0, "S3": 0, "S4": 0}
	for _, d := range dims {
		if d.Verdict == verdictFail || d.Verdict == verdictWarn {
			if _, ok := counts[d.Severity]; ok {
				counts[d.Severity]++
			}
		}
	}
	verdict := VerdictApprove
	if counts["S1"] > 0 {
		verdict = VerdictReject
	} else if counts["S2"] > 0 || counts["S3"] >= 3 {
		verdict = VerdictConcerns
	}

	return &Report{
		Platform:      p.ID,
		PlatformLabel: p.Label,
		ChapterNum:    opts.ChapterNum,
		Words:         words,
		Verdict:       verdict,
		Counts:        counts,
		Dimensions:    dims,
		Advisories:    r.Advisories,
	}, nil
}

// severityOf 取平台档位里该维度的严重度（数据里没写就按 S3 局部问题）。
func severityOf(p Platform, dim string) string {
	if s, ok := p.Severity[dim]; ok && severityValues[s] {
		return s
	}
	return "S3"
}

// labelOf 维度中文名。
func labelOf(dim string) string {
	if l, ok := dimensionLabels[dim]; ok {
		return l
	}
	return dim
}

// ── 维度实现（一）：篇幅 / 钩子 / 收尾 / 情绪 / 爽点 / 段落 ──────────────

// dimLength 字数区间：form=chapter 按章带，form=story 按整篇带（本章仅提示口径）。
func dimLength(p Platform, words int) Dimension {
	d := Dimension{ID: "length_band", Label: labelOf("length_band"), Severity: severityOf(p, "length_band")}
	if p.Form == "chapter" {
		switch {
		case words < p.ChapterWordsMin/2 || words > p.ChapterWordsMax*2:
			d.Verdict = verdictFail
			d.Detail = fmt.Sprintf("本章 %d 字，严重偏离平台常见区间 %d~%d 字", words, p.ChapterWordsMin, p.ChapterWordsMax)
			d.Advice = "按平台章节长度收束：过短撑不住一章信息量，过长拉低移动端完读率。"
		case words < p.ChapterWordsMin || words > p.ChapterWordsMax:
			d.Verdict = verdictWarn
			d.Detail = fmt.Sprintf("本章 %d 字，平台常见区间 %d~%d 字", words, p.ChapterWordsMin, p.ChapterWordsMax)
			d.Advice = "章节字数贴近平台区间，断章点更自由。"
		default:
			d.Verdict = verdictPass
			d.Detail = fmt.Sprintf("本章 %d 字，在平台区间 %d~%d 字内", words, p.ChapterWordsMin, p.ChapterWordsMax)
		}
		return d
	}
	if words < p.WordsMin {
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("本章 %d 字；%s 按整篇口径要求 %d~%d 字（本章按章评审，仅作进度参考）", words, p.Label, p.WordsMin, p.WordsMax)
		d.Advice = "整篇口径：写完再看总字数是否落在盐言区间；单章长度按叙事单元自然断。"
		return d
	}
	d.Verdict = verdictPass
	d.Detail = fmt.Sprintf("本章 %d 字（整篇口径 %d~%d 字，本章已达标）", words, p.WordsMin, p.WordsMax)
	return d
}

// dimOpeningHook 开篇钩子：前 N 段是否给出冲突/悬念/台词（纯描写开场=FAIL 口径）。
func dimOpeningHook(p Platform, r *Rubric, paras []paragraph) Dimension {
	d := Dimension{ID: "opening_freshness", Label: labelOf("opening_freshness"), Severity: severityOf(p, "opening_freshness")}
	if len(paras) == 0 {
		d.Verdict = verdictSkip
		d.Detail = "无段落，跳过"
		return d
	}
	n := p.OpeningHookParagraphs
	if n > len(paras) {
		n = len(paras)
	}
	head := paras[:n]
	hasDialog, hasConflict, hasSuspense := false, false, false
	for _, pa := range head {
		rs := []rune(pa.text)
		if containsQuote(pa.text) {
			hasDialog = true
		}
		if len(findMarkerStarts(rs, r.Markers.Conflict)) > 0 {
			hasConflict = true
		}
		if len(findMarkerStarts(rs, r.Markers.Suspense)) > 0 {
			hasSuspense = true
		}
	}
	d.Evidence = []EvidenceSpan{{Start: head[0].start, End: head[0].end, Paragraph: head[0].idx}}
	switch {
	case hasConflict || hasDialog:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("前 %d 段已出现%s", n, openingSignals(hasConflict, hasDialog, hasSuspense))
	case hasSuspense:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("前 %d 段只有悬念铺垫，没有冲突或台词落地", n)
		d.Advice = "开篇把悬念变成当场发生的事：一句对话、一次阻拦、一个具体麻烦。"
	default:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("前 %d 段是纯描写/背景介绍，没有冲突、悬念或台词", n)
		d.Advice = "开篇段落进具体人物与处境（动作/对话/麻烦），别用环境与背景起手。"
	}
	return d
}

// openingSignals 开篇信号描述。
func openingSignals(conflict, dialog, suspense bool) string {
	var parts []string
	if conflict {
		parts = append(parts, "冲突")
	}
	if dialog {
		parts = append(parts, "台词")
	}
	if suspense {
		parts = append(parts, "悬念")
	}
	return joinCN(parts)
}

// dimEndingHook 章尾钩子：结尾是否给出翻页动力（悬念/反转/新信息/台词/未完成动作）。
func dimEndingHook(p Platform, r *Rubric, paras []paragraph) Dimension {
	d := Dimension{ID: "hook_expectation", Label: labelOf("hook_expectation"), Severity: severityOf(p, "hook_expectation")}
	if len(paras) == 0 {
		d.Verdict = verdictSkip
		d.Detail = "无段落，跳过"
		return d
	}
	last := paras[len(paras)-1]
	text := last.text
	var signals []string
	if containsAny(text, "？?") {
		signals = append(signals, "疑问")
	}
	if containsAny(text, "……", "...") {
		signals = append(signals, "留白")
	}
	if containsQuote(text) {
		signals = append(signals, "台词")
	}
	rs := []rune(text)
	if len(findMarkerStarts(rs, r.Markers.Suspense)) > 0 {
		signals = append(signals, "悬念词")
	}
	if len(findMarkerStarts(rs, r.Markers.Conflict)) > 0 {
		signals = append(signals, "冲突")
	}
	d.Evidence = []EvidenceSpan{{Start: last.start, End: last.end, Paragraph: last.idx}}
	if len(signals) > 0 {
		d.Verdict = verdictPass
		d.Detail = "章尾有翻页动力（" + joinCN(signals) + "）"
		return d
	}
	if len([]rune(text)) <= 12 {
		d.Verdict = verdictPass
		d.Detail = "章尾收在短句上（短促收束，按翻页动力算通过）"
		return d
	}
	d.Verdict = verdictWarn
	d.Detail = "章尾是平铺叙述，没有悬念、疑问、台词或未完成的动作"
	d.Advice = "结尾停在具体动作、画面或一句台词上，让事件自己挂住读者。"
	return d
}

// dimTrailer 预告式收尾（上游 blocking 级）：章尾总结/预告腔。
func dimTrailer(p Platform, r *Rubric, paras []paragraph) Dimension {
	d := Dimension{ID: "trailer_ending", Label: labelOf("trailer_ending"), Severity: severityOf(p, "trailer_ending")}
	if len(paras) == 0 {
		d.Verdict = verdictSkip
		d.Detail = "无段落，跳过"
		return d
	}
	tail := paras
	if len(tail) > 2 {
		tail = tail[len(tail)-2:]
	}
	markers := append(append([]string{}, r.Markers.TrailerEnding...), r.Markers.TrailerSummary...)
	var hits []string
	var ev []EvidenceSpan
	seen := map[string]bool{}
	for _, pa := range tail {
		rs := []rune(pa.text)
		for _, m := range markers {
			if seen[m] {
				continue
			}
			starts := findAllRunes(rs, []rune(m))
			if len(starts) == 0 {
				continue
			}
			seen[m] = true
			hits = append(hits, m)
			ev = append(ev, EvidenceSpan{Start: pa.start + starts[0], End: pa.start + starts[0] + len([]rune(m)), Paragraph: pa.idx})
		}
	}
	if len(hits) == 0 {
		d.Verdict = verdictPass
		d.Detail = "章尾无预告式总结腔"
		return d
	}
	sortSpans(ev)
	if len(ev) > 3 {
		ev = ev[:3]
	}
	d.Verdict = verdictFail
	d.Evidence = ev
	d.Detail = "章尾出现预告/总结腔：" + joinCN(hits)
	d.Advice = "删掉「才刚刚开始 / 没人知道 / 命运的齿轮」式收束，把悬念落到具体动作或物件上。"
	return d
}

// dimEmotionDensity 情绪节点密度 + 最长平直段（上游：连续 2000 字无起伏=FAIL）。
func dimEmotionDensity(p Platform, r *Rubric, runes []rune, paras []paragraph, words int) Dimension {
	d := Dimension{ID: "emotion_curve", Label: labelOf("emotion_curve"), Severity: severityOf(p, "emotion_curve")}
	markers := append(append([]string{}, r.Markers.Emotion...), r.Markers.Conflict...)
	hits := findMarkerStarts(runes, markers)
	density := float64(len(hits)) * 1000 / float64(words)
	gap, gapStart := longestGap(runes, hits)
	switch {
	case gap >= p.EmotionGapFail:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("连续 %d 字没有情绪或冲突节点（平台上限 %d 字）；全章密度 %.2f/千字", gap, p.EmotionGapFail, density)
		d.Advice = "长平直段要么压短，要么塞进选择、代价或一句带情绪的话。"
		d.Evidence = append(d.Evidence, spanAt(runes, paras, gapStart))
	case density < p.EmotionPerKilo:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("情绪节点密度 %.2f/千字，低于平台线 %.2f/千字（最长平直 %d 字）", density, p.EmotionPerKilo, gap)
		d.Advice = "把情绪落到动作、台词、物件或后果上，别靠形容词堆叠。"
		if gap >= p.EmotionGapWarn {
			d.Evidence = append(d.Evidence, spanAt(runes, paras, gapStart))
		}
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("情绪节点密度 %.2f/千字（平台线 %.2f），最长平直 %d 字", density, p.EmotionPerKilo, gap)
	}
	return d
}

// dimPowerDensity 爽点/升级密度（起点口径：每 3000 字 ≥1 个情绪节点）。
func dimPowerDensity(p Platform, r *Rubric, runes []rune, words int) Dimension {
	d := Dimension{ID: "plot_loop", Label: labelOf("plot_loop"), Severity: severityOf(p, "plot_loop")}
	hits := findMarkerStarts(runes, r.Markers.Power)
	per3k := float64(len(hits)) * 3000 / float64(words)
	switch {
	case per3k < p.PowerPer3k/2:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("爽点/升级节点 %.2f 个/3000 字，远低于平台线 %.2f", per3k, p.PowerPer3k)
		d.Advice = "本章至少给一个可兑现的进展：升级、打脸、成交、翻身，别整章铺垫。"
	case per3k < p.PowerPer3k:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("爽点/升级节点 %.2f 个/3000 字，略低于平台线 %.2f", per3k, p.PowerPer3k)
		d.Advice = "补一个可量化的进展或一次漂亮的回击。"
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("爽点/升级节点 %.2f 个/3000 字（平台线 %.2f）", per3k, p.PowerPer3k)
	}
	return d
}

// dimParagraphPace 段落节奏：长段占比 + 段落匀称度 + 电报体（碎句）。
func dimParagraphPace(p Platform, paras []paragraph) Dimension {
	d := Dimension{ID: "format_readability", Label: labelOf("format_readability"), Severity: severityOf(p, "format_readability")}
	if len(paras) < 3 {
		d.Verdict = verdictSkip
		d.Detail = "段落过少，节奏无法判定"
		return d
	}
	longCount, longest, sum := 0, 0, 0
	longestIdx := 0
	for i, pa := range paras {
		l := len([]rune(pa.text))
		sum += l
		if l > longest {
			longest, longestIdx = l, i
		}
		if l > p.LongParagraphChars {
			longCount++
		}
	}
	avg := float64(sum) / float64(len(paras))
	longRatio := float64(longCount) / float64(len(paras))
	uniform := false
	if len(paras) >= 5 && avg > 0 {
		within := 0
		for _, pa := range paras {
			l := float64(len([]rune(pa.text)))
			if absFloat(l-avg) <= avg*0.2 {
				within++
			}
		}
		uniform = float64(within)/float64(len(paras)) >= p.ParaUniformRatioWarn
	}
	telegraph := len(paras) >= 12 && avg <= 12

	d.Evidence = []EvidenceSpan{{Start: paras[longestIdx].start, End: paras[longestIdx].end, Paragraph: paras[longestIdx].idx}}
	switch {
	case longRatio > 0.5:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("%d/%d 段超过 %d 字（最长 %d 字），大段堆叠", longCount, len(paras), p.LongParagraphChars, longest)
		d.Advice = "按镜头/新动作/新线索断段，长段拆开再补节奏。"
	case longRatio > 0.2 || uniform || telegraph:
		d.Verdict = verdictWarn
		var why []string
		if longRatio > 0.2 {
			why = append(why, fmt.Sprintf("%d 段超过 %d 字", longCount, p.LongParagraphChars))
		}
		if uniform {
			why = append(why, "段落长度过于匀称")
		}
		if telegraph {
			why = append(why, fmt.Sprintf("平均段长仅 %.1f 字，偏电报体", avg))
		}
		d.Detail = fmt.Sprintf("段落节奏：%s（平均 %.1f 字，最长 %d 字）", joinCN(why), avg, longest)
		d.Advice = "长短交错：冲突处短段推进，沉淀处可长段，别通篇同长。"
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("段落长短交错正常（平均 %.1f 字，最长 %d 字）", avg, longest)
	}
	return d
}

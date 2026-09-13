package novelreview

// 维度实现（二）：对话 / 标点 / 破折号 / 格式 / 人称 / 主角 / 金手指 / 字数表述。
// 全部为纯函数，输入只有文本结构与数据资产，便于表驱动单测。

import (
	"fmt"
	"strings"
)

// dimDialogRatio 对话占比（平台区间；番茄偏多、起点可少）。
func dimDialogRatio(p Platform, runes []rune, words int) Dimension {
	d := Dimension{ID: "dialogue_ratio", Label: labelOf("dialogue_ratio"), Severity: severityOf(p, "dialogue_ratio")}
	ratio := float64(countQuotedRunes(runes)) / float64(words)
	switch {
	case ratio < p.DialogRatioMin/2:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("对话占比 %.0f%%，远低于平台区间 %.0f%%~%.0f%%", ratio*100, p.DialogRatioMin*100, p.DialogRatioMax*100)
		d.Advice = "把信息交代改成角色对话或动作，别用旁白讲完。"
	case ratio < p.DialogRatioMin:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("对话占比 %.0f%%，低于平台区间 %.0f%%~%.0f%%", ratio*100, p.DialogRatioMin*100, p.DialogRatioMax*100)
		d.Advice = "适当补对话推进（但不为凑比例硬塞台词）。"
	case ratio > p.DialogRatioMax:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("对话占比 %.0f%%，高于平台区间 %.0f%%~%.0f%%", ratio*100, p.DialogRatioMin*100, p.DialogRatioMax*100)
		d.Advice = "补叙述与动作承载信息，避免通篇对话像剧本。"
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("对话占比 %.0f%%（平台区间 %.0f%%~%.0f%%）", ratio*100, p.DialogRatioMin*100, p.DialogRatioMax*100)
	}
	return d
}

// dimPunctuation 标点节奏：省略号密度 + 通篇句号化（标点跟语气走）。
func dimPunctuation(p Platform, runes []rune, paras []paragraph, words int) Dimension {
	d := Dimension{ID: "punctuation_rhythm", Label: labelOf("punctuation_rhythm"), Severity: severityOf(p, "punctuation_rhythm")}
	ell := countOccurrences(runes, "……") + countOccurrences(runes, "...")
	perKilo := float64(ell) * 1000 / float64(words)
	q := countRune(runes, '？') + countRune(runes, '?')
	e := countRune(runes, '！') + countRune(runes, '!')
	switch {
	case perKilo > p.EllipsisPerKiloWarn*2:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("省略号 %.1f 次/千字，远超平台线 %.1f（共 %d 次）", perKilo, p.EllipsisPerKiloWarn, ell)
		d.Advice = "删掉硬造停顿的省略号，用短句、动作或逗号收束。"
		if spans := markerSpans(runes, paras, "……", 3); len(spans) > 0 {
			d.Evidence = spans
		}
	case perKilo > p.EllipsisPerKiloWarn:
		d.Verdict = verdictWarn
		d.Detail = fmt.Sprintf("省略号 %.1f 次/千字，高于平台线 %.1f（共 %d 次）", perKilo, p.EllipsisPerKiloWarn, ell)
		d.Advice = "保留有功能的留白，其余改成句子收束。"
	case q == 0 && e == 0 && len(paras) >= 10:
		d.Verdict = verdictWarn
		d.Detail = "全章零问号零感叹号，通篇句号化，语气没有起伏"
		d.Advice = "质问留问号、爆发峰值留少量感叹；不要一律改成句号。"
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("标点节奏正常（省略号 %.1f 次/千字、问号 %d、感叹号 %d）", perKilo, q, e)
	}
	return d
}

// dimDash 破折号（上游 blocking）：按功能改写，不一律换句号。
func dimDash(p Platform, runes []rune, paras []paragraph) Dimension {
	d := Dimension{ID: "dash_usage", Label: labelOf("dash_usage"), Severity: severityOf(p, "dash_usage")}
	spans := dashSpans(runes, paras)
	if len(spans) == 0 {
		d.Verdict = verdictPass
		d.Detail = "正文无破折号"
		return d
	}
	d.Verdict = verdictFail
	d.Detail = fmt.Sprintf("正文出现 %d 处破折号（上游按 blocking 处理）", len(spans))
	d.Advice = "破折号按功能改写：打断→动作或短句，拖长音→省略或动作，插入说明→逗号/冒号。"
	if len(spans) > 3 {
		spans = spans[:3]
	}
	d.Evidence = spans
	return d
}

// dimFormat 格式合规：连续空行、段首缩进（盐言档位禁用空行分段）。
func dimFormat(p Platform, text string) Dimension {
	d := Dimension{ID: "format_hygiene", Label: labelOf("format_hygiene"), Severity: severityOf(p, "format_hygiene")}
	var issues []string
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if strings.Contains(normalized, "\n\n\n") {
		if p.NoBlankLines {
			issues = append(issues, "正文出现连续空行（本平台禁用空行分段）")
		} else {
			issues = append(issues, "正文出现连续空行")
		}
	}
	indent := 0
	for _, line := range strings.Split(normalized, "\n") {
		if strings.HasPrefix(line, "　") || strings.HasPrefix(line, "  ") {
			indent++
		}
	}
	if indent > 0 {
		issues = append(issues, fmt.Sprintf("%d 行段首带缩进/全角空格", indent))
	}
	if len(issues) == 0 {
		d.Verdict = verdictPass
		d.Detail = "格式合规：无连续空行、无段首缩进"
		return d
	}
	d.Verdict = verdictWarn
	d.Detail = joinCN(issues)
	d.Advice = "按平台排版惯例：段间单换行、段首不留缩进。"
	return d
}

// dimPerspective 人称视角（盐言档位要求第一人称）。
func dimPerspective(p Platform, runes []rune, paras []paragraph) Dimension {
	d := Dimension{ID: "perspective_consistency", Label: labelOf("perspective_consistency"), Severity: severityOf(p, "perspective_consistency")}
	me := countRune(runes, '我')
	third := countRune(runes, '他') + countRune(runes, '她')
	if len(paras) > 0 {
		d.Evidence = []EvidenceSpan{{Start: paras[0].start, End: paras[0].end, Paragraph: paras[0].idx}}
	}
	switch {
	case me == 0:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("通篇没有「我」（第三人称计数 %d）", third)
		d.Advice = "本平台偏第一人称代入，改由「我」的视角叙述。"
	case third > me*2 && third >= 6:
		d.Verdict = verdictFail
		d.Detail = fmt.Sprintf("第三人称 %d 次 vs 第一人称 %d 次，视角偏第三人称", third, me)
		d.Advice = "除非多视角设计，本平台保持「我」的叙述视角。"
	default:
		d.Verdict = verdictPass
		d.Detail = fmt.Sprintf("第一人称 %d 次 / 第三人称 %d 次，以「我」为主", me, third)
	}
	return d
}

// dimProtagonist 主角存在感（起点 rubric：连续 2 章主角不出场=FAIL；单章只看本章）。
func dimProtagonist(p Platform, runes []rune, paras []paragraph, names []string) Dimension {
	d := Dimension{ID: "protagonist_presence", Label: labelOf("protagonist_presence"), Severity: severityOf(p, "protagonist_presence")}
	if len(names) == 0 {
		d.Verdict = verdictSkip
		d.Detail = "角色库未标注主角，跳过（在角色库把主要角色标为「主角」后本维度生效）"
		return d
	}
	total := 0
	var hits []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		c := len(findAllRunes(runes, []rune(n)))
		if c > 0 {
			hits = append(hits, n)
			total += c
		}
	}
	if total == 0 {
		d.Verdict = verdictFail
		d.Detail = "本章没有出现主角（" + joinCN(names) + "）"
		d.Advice = "起点口径：主角连续 2 章不出场或无推进即掉追读——让主角在本章做一次选择。"
		if len(paras) > 0 {
			d.Evidence = []EvidenceSpan{{Start: paras[0].start, End: paras[0].end, Paragraph: paras[0].idx}}
		}
		return d
	}
	d.Verdict = verdictPass
	d.Detail = fmt.Sprintf("主角出场 %d 次（%s）", total, joinCN(hits))
	return d
}

// dimLever 金手指提及（起点 rubric：连续 5 章未提及=FAIL；单章只看本章）。
func dimLever(p Platform, r *Rubric, runes []rune, words int) Dimension {
	d := Dimension{ID: "lever_mention", Label: labelOf("lever_mention"), Severity: severityOf(p, "lever_mention")}
	hits := findMarkerStarts(runes, r.Markers.Lever)
	if len(hits) == 0 {
		d.Verdict = verdictWarn
		d.Detail = "本章未提及金手指/系统类设定（起点口径：连续 5 章未提及才算问题，需跨章看）"
		d.Advice = "若本书有金手指，让它在本章关键处露一次面；没有就不必硬塞。"
		return d
	}
	d.Verdict = verdictPass
	d.Detail = fmt.Sprintf("金手指/系统提及 %d 次（%.1f 次/千字）", len(hits), float64(len(hits))*1000/float64(words))
	return d
}

// dimWordcountExpr 具体字数表述核对（「这五个字」类表达的字数是否属实）。
func dimWordcountExpr(p Platform, runes []rune, paras []paragraph) Dimension {
	d := Dimension{ID: "stated_length_accuracy", Label: labelOf("stated_length_accuracy"), Severity: severityOf(p, "stated_length_accuracy")}
	digits := map[rune]int{
		'一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
	}
	var bad []EvidenceSpan
	var details []string
	for i := 0; i+3 < len(runes); i++ {
		if runes[i] != '这' {
			continue
		}
		n, ok := digits[runes[i+1]]
		if !ok || runes[i+2] != '个' || runes[i+3] != '字' {
			continue
		}
		// 先往后 20 rune 找引号实体，找不到再向前找（「…」这三个字 两种语序都覆盖）。
		quoted, qlen, _, qend := quotedSpanAfter(runes, i+4, 20)
		if quoted == "" {
			quoted, qlen, _, _ = quotedSpanBefore(runes, i, 20)
		}
		if quoted == "" || qlen == n {
			continue
		}
		bad = append(bad, EvidenceSpan{Start: i, End: qend, Paragraph: paragraphIndexOf(paras, i)})
		details = append(details, fmt.Sprintf("「这%s个字」实际 %d 字（%s）", string(runes[i+1]), qlen, quoted))
	}
	if len(bad) == 0 {
		d.Verdict = verdictPass
		d.Detail = "未发现字数表述与实际不符"
		return d
	}
	d.Verdict = verdictWarn
	d.Detail = "字数表述与实际不符：" + joinCN(details)
	d.Advice = "把具体字数改成「那几个字」，或让引号内实体真的对上字数。"
	if len(bad) > 3 {
		bad = bad[:3]
	}
	d.Evidence = bad
	return d
}

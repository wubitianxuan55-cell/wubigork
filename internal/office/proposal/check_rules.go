// Package proposal — 结构化校验规则（确定性，无需 LLM）
package proposal

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var specCodeRe = regexp.MustCompile(`(?:GB|HJ)\s?\d{2,5}(?:\.\d+)?(?:-\d{4})?`)

// locateNeedle 在方案章节中定位 needle（忽略空白差异），返回最多 3 处原文定位。
func locateNeedle(sections []ProposalSection, needle string) []CheckLocation {
	if needle == "" {
		return nil
	}
	var out []CheckLocation
	for _, sec := range flattenSections(sections) {
		if sec.Content == "" {
			continue
		}
		start, end, ok := LocateQuote(sec.Content, needle)
		if !ok {
			continue
		}
		out = append(out, CheckLocation{
			SectionID: sec.ID,
			Excerpt:   excerptAround(sec.Content, start, end, 36),
			Offset:    start,
		})
		if len(out) >= 3 {
			break
		}
	}
	return out
}

// excerptAround 取 [start,end) 前后 radius 个 rune 的摘录，避免切坏 UTF-8。
func excerptAround(s string, start, end, radius int) string {
	rs := []rune(s)
	sIdx := runeIndexAt(rs, start)
	eIdx := runeIndexAt(rs, end)
	lo := sIdx - radius
	if lo < 0 {
		lo = 0
	}
	hi := eIdx + radius
	if hi > len(rs) {
		hi = len(rs)
	}
	pre, post := "", ""
	if lo > 0 {
		pre = "…"
	}
	if hi < len(rs) {
		post = "…"
	}
	return pre + string(rs[lo:hi]) + post
}

// runeIndexAt 把字节偏移映射到 rune 下标（偏移位于字符边界时）。
func runeIndexAt(rs []rune, byteOff int) int {
	pos := 0
	for i, r := range rs {
		if pos >= byteOff {
			return i
		}
		pos += len(string(r))
	}
	return len(rs)
}

// structuredRules 返回确定性规则集
func (s *Service) structuredRules() []CheckRule {
	return []CheckRule{
		ruleFunc{name: "废标条款响应", severity: "critical", fn: s.runRedLineRule},
		ruleFunc{name: "数据一致性检查", severity: "critical", fn: s.runConsistencyRule},
		ruleFunc{name: "重复率检测", severity: "warning", fn: s.runDuplicateRule},
		ruleFunc{name: "暗标格式检查", severity: "critical", fn: s.runDarkFormatRule},
		ruleFunc{name: "规范引用检查", severity: "warning", fn: s.runSpecRefRule},
	}
}

// runRedLineRule 废标条款响应检查：方案内容未明确回应则 warn
func (s *Service) runRedLineRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	if p.BidSummary == nil {
		return nil, nil
	}
	content := Assemble(p)
	norm := strings.Join(strings.Fields(content), "")
	var out []CheckItem
	add := func(title, clause string, ok bool) {
		status := "pass"
		msg := "已明确响应：" + title
		if !ok {
			status = "warn"
			msg = "方案未明确响应废标条款「" + clause + "」，需人工复核"
		}
		it := CheckItem{Status: status, Message: msg, Evidence: clause}
		if !ok {
			it.Suggestion = "在方案中补充对该废标条款的实质性响应并引用条款原文，避免因未响应而废标。"
			it.Locations = locateNeedle(p.Sections, clause)
		}
		out = append(out, it)
	}
	items := p.BidSummary.RedLineItems
	if len(items) == 0 {
		for _, r := range p.BidSummary.RedLines {
			items = append(items, BidItem{Name: "废标条款", Content: r})
		}
	}
	for _, it := range items {
		key := it.Content
		if key == "" {
			key = it.Name
		}
		keyNorm := strings.Join(strings.Fields(key), "")
		add(it.Name, key, strings.Contains(norm, keyNorm))
	}
	return out, nil
}

// runConsistencyRule 项目事实基线一致性：未体现 warn，工期/单位冲突 fail
func (s *Service) runConsistencyRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	if p.ProjectID == "" {
		return nil, nil
	}
	facts, err := s.store.GetProjectFacts(p.ProjectID)
	if err != nil || len(facts) == 0 {
		return nil, nil
	}
	content := Assemble(p)
	norm := strings.Join(strings.Fields(content), "")
	var out []CheckItem
	for k, v := range facts {
		if strings.TrimSpace(v) == "" {
			continue
		}
		vNorm := strings.Join(strings.Fields(v), "")
		if !strings.Contains(norm, vNorm) {
			out = append(out, CheckItem{
				Status: "warn", Message: "项目事实「" + k + "」未在方案中体现：" + v,
				Suggestion: "在方案中补充「" + k + "」的表述：" + v + "，并与项目事实基线保持一致。",
			})
		}
	}
	if d := facts["工期"]; d != "" {
		vals := distinctDurations(norm)
		if len(vals) > 1 {
			sort.Strings(vals)
			out = append(out, CheckItem{
				Status: "fail", Message: "工期前后不一致：" + strings.Join(vals, " vs "), Evidence: d,
				Suggestion: "统一全篇工期表述，以项目事实基线为准：" + d,
				Locations:  locateNeedle(p.Sections, vals[0]),
			})
		}
	}
	if u := facts["业主单位"]; u != "" && p.BidSummary != nil && hasDarkNoUnit(p) {
		uNorm := strings.Join(strings.Fields(u), "")
		if strings.Contains(norm, uNorm) {
			out = append(out, CheckItem{
				Status: "fail", Message: "暗标场景下出现单位名称：" + u,
				Suggestion: "移除单位名称，改为匿名表述（如「我公司」→「投标人」），避免暗标废标。",
				Locations:  locateNeedle(p.Sections, u),
			})
		}
	}
	return out, nil
}

var durationRe = regexp.MustCompile(`\d+\s*(?:日历天|天)`)

func distinctDurations(norm string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range durationRe.FindAllString(norm, -1) {
		key := strings.Join(strings.Fields(m), "")
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// runDuplicateRule 跨章节重复：20 字 n-gram 集合交并比
func (s *Service) runDuplicateRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	type secInfo struct {
		id    string
		title string
		grams map[string]bool
	}
	var secs []secInfo
	for _, sec := range flattenSections(p.Sections) {
		content := strings.Join(strings.Fields(sec.Content), "")
		if len([]rune(content)) < 40 {
			continue
		}
		secs = append(secs, secInfo{id: sec.ID, title: sec.Title, grams: gramSet(content, 20)})
	}
	var out []CheckItem
	for i := 0; i < len(secs); i++ {
		for j := i + 1; j < len(secs); j++ {
			common := 0
			for g := range secs[i].grams {
				if secs[j].grams[g] {
					common++
				}
			}
			den := len(secs[i].grams)
			if len(secs[j].grams) < den {
				den = len(secs[j].grams)
			}
			if den == 0 {
				continue
			}
			ratio := float64(common) / float64(den)
			if ratio > 0.5 {
				out = append(out, CheckItem{Status: "fail", SectionID: secs[j].id,
					Message: "与章节「" + secs[i].title + "」重复率高（" + pct(ratio) + "）", Evidence: secs[i].title,
					Suggestion: "合并重复内容：仅保留信息更完整的一处，其余章节改为简要引用，避免评分扣分。",
					Locations:  []CheckLocation{{SectionID: secs[i].id}, {SectionID: secs[j].id}},
				})
			} else if ratio > 0.3 {
				out = append(out, CheckItem{Status: "warn", SectionID: secs[j].id,
					Message: "与章节「" + secs[i].title + "」存在较多重复（" + pct(ratio) + "）", Evidence: secs[i].title,
					Suggestion: "梳理两章职责边界，删除重复论述，仅保留差异化内容。",
					Locations:  []CheckLocation{{SectionID: secs[i].id}, {SectionID: secs[j].id}},
				})
			}
		}
	}
	return out, nil
}

func gramSet(s string, n int) map[string]bool {
	runes := []rune(s)
	out := make(map[string]bool)
	for i := 0; i+n <= len(runes); i++ {
		out[string(runes[i:i+n])] = true
	}
	return out
}

func pct(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}

// runDarkFormatRule 暗标格式：加粗/斜体/删除线/emoji/单位名称
func (s *Service) runDarkFormatRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	if p.BidSummary == nil || len(p.BidSummary.DarkRules) == 0 {
		return nil, nil
	}
	content := Assemble(p)
	var out []CheckItem
	if strings.Contains(content, "**") || strings.Contains(content, "__") {
		it := CheckItem{Status: "fail", Message: "检测到加粗标记（暗标禁止加粗）", Evidence: "**",
			Suggestion: "移除加粗标记，恢复普通正文格式（暗标不允许突出样式）。"}
		it.Locations = locateNeedle(p.Sections, "**")
		out = append(out, it)
	}
	if strings.Contains(content, "~~") {
		it := CheckItem{Status: "warn", Message: "检测到删除线标记（暗标建议移除）", Evidence: "~~",
			Suggestion: "删除删除线内容或改为正文表述。"}
		it.Locations = locateNeedle(p.Sections, "~~")
		out = append(out, it)
	}
	for _, r := range content {
		if r >= 0x1F300 && r <= 0x1FAFF {
			out = append(out, CheckItem{Status: "warn", Message: "检测到 emoji 符号（暗标禁止特殊符号）",
				Suggestion: "删除 emoji 符号，改用纯文字表述。"})
			break
		}
	}
	return out, nil
}

func hasDarkNoUnit(p *Proposal) bool {
	for _, it := range p.BidSummary.DarkRules {
		if strings.Contains(it.Content, "单位名称") || strings.Contains(it.Content, "公司") {
			return true
		}
	}
	return false
}

// runSpecRefRule 规范引用：技术方案必须引用规范编号；引用编号应存在于知识库
func (s *Service) runSpecRefRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	content := Assemble(p)
	codes := specCodeRe.FindAllString(content, -1)
	var out []CheckItem
	if len(codes) == 0 {
		hasContent := false
		for _, sec := range flattenSections(p.Sections) {
			if sec.Content != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			it := CheckItem{Status: "warn", Message: "技术方案未引用任何规范标准编号（如 GB 36600、HJ 25.4）",
				Suggestion: "在技术方案中补充所依据的规范标准编号（如 GB 36600-2018、HJ 25.4）及关键限值要求，提升专业可信度。"}
			for _, sec := range flattenSections(p.Sections) {
				if sec.Content != "" {
					it.Locations = []CheckLocation{{SectionID: sec.ID}}
					break
				}
			}
			out = append(out, it)
		}
		return out, nil
	}
	if st := s.knowledgeStore(); st != nil {
		known := map[string]bool{}
		for _, e := range st.ReadAll() {
			known[strings.ToLower(strings.TrimSpace(e.Source))] = true
		}
		for _, c := range codes {
			if !known[strings.ToLower(c)] {
				it := CheckItem{Status: "warn", Message: "引用的规范「" + c + "」不在知识库中，请核实编号",
					Suggestion: "核对规范「" + c + "」的准确编号与现行有效版本，确认后补充规范全称。"}
				it.Locations = locateNeedle(p.Sections, c)
				out = append(out, it)
			}
		}
	}
	return out, nil
}

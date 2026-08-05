// Package proposal — 招标解析管线 v2：逐文件提取字段 + 来源定位
package proposal

import (
	"context"
	"encoding/json"
	"fmt"
)

const parseChunkSize = 12000

// parseItem AI 返回的带摘录要求项
type parseItem struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Quote   string `json:"quote"`
}

// parseScoring AI 返回的评分项
type parseScoring struct {
	Name        string `json:"name"`
	MaxScore    string `json:"maxScore"`
	Requirement string `json:"requirement"`
	Quote       string `json:"quote"`
}

// parseFileResult AI 单次返回的解析结果（含原文摘录 quote）
type parseFileResult struct {
	Overview        string         `json:"overview"`
	OverviewQuote   string         `json:"overviewQuote"`
	Duration        string         `json:"duration"`
	DurationQuote   string         `json:"durationQuote"`
	Qualification   []parseItem    `json:"qualification"`
	TechScoring     []parseScoring `json:"techScoring"`
	KeyRequirements []string       `json:"keyRequirements"`
	RedLines        []parseItem    `json:"redLines"`
	Format          []parseItem    `json:"format"`
	DarkRules       []parseItem    `json:"darkRules"`
}

const parseSystemPrompt = `你是一位专业的招投标专家。基于给定招标文件（Markdown 格式），提取所有对投标方案编写有影响的关键信息。

返回纯 JSON，格式：
{
  "overview": "项目概况（150 字以内）",
  "overviewQuote": "项目概况在原文中的一字不差摘录（尽量完整的一句）",
  "duration": "工期要求",
  "durationQuote": "工期要求原文摘录",
  "qualification": [{"name":"资质项名称","content":"具体资质要求","quote":"原文摘录"}],
  "techScoring": [{"name":"评分项","maxScore":"分值","requirement":"具体要求","quote":"原文摘录"}],
  "keyRequirements": ["核心要求"],
  "redLines": [{"name":"废标条款名称","content":"废标条款内容","quote":"原文摘录"}],
  "format": [{"name":"格式要求名称","content":"格式要求内容","quote":"原文摘录"}],
  "darkRules": [{"name":"暗标要求名称","content":"暗标要求内容","quote":"原文摘录"}]
}

要求：
- quote 必须是文档中出现的原文片段（可含标点，尽量短但足以定位）
- 不存在的类别填空数组或空字符串
- 不要遗漏影响投标的关键信息`

// ParseBidFileV2 执行完整解析管线：逐文件分块提取 → 来源定位 → 落库
func (s *Service) ParseBidFileV2(ctx context.Context, proposalID string) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if p.BidSummary == nil || len(p.BidSummary.RawFiles) == 0 {
		return nil, fmt.Errorf("请先导入招标文件并转换 Markdown")
	}

	merged := parseFileResult{}
	var resultItems []ParseResultItem
	partial := false
	for i, f := range p.BidSummary.RawFiles {
		if f.Markdown == "" {
			continue
		}
		fileID := f.FileID
		if fileID == "" {
			fileID = fmt.Sprintf("file-%d", i)
		}
		pages, _ := ExtractPageText(f.Path)
		if len(pages) == 0 {
			pages = []PageText{{Page: 0, Text: f.Markdown}}
		}
		res, ok := s.parseFileChunks(ctx, f.Markdown)
		if !ok {
			partial = true
			continue
		}
		merged = mergeParseResults(merged, res)
		resultItems = append(resultItems, resolveParseItems(fileID, f.Name, pages, f.Markdown, res)...)
	}
	if len(resultItems) == 0 {
		return nil, fmt.Errorf("AI 解析失败，未提取到任何字段")
	}
	if err := s.store.SaveParseResults(proposalID, resultItems); err != nil {
		return nil, err
	}
	if p.BidSummary == nil {
		p.BidSummary = &BidSummary{}
	}
	applyParseResult(p.BidSummary, merged)
	if partial {
		p.BidSummary.ParseStatus = "partial"
	} else {
		p.BidSummary.ParseStatus = "done"
	}
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// parseFileChunks 对单个文件分块调用 AI 并合并
func (s *Service) parseFileChunks(ctx context.Context, markdown string) (parseFileResult, bool) {
	runes := []rune(markdown)
	if len(runes) <= parseChunkSize {
		reply, err := s.ai.ChatSimpleStream(ctx, "", parseSystemPrompt, "请解析以下招标文件：\n\n"+markdown)
		if err != nil {
			return parseFileResult{}, false
		}
		return decodeParseResult(reply), true
	}
	merged := parseFileResult{}
	any := false
	for start := 0; start < len(runes); start += parseChunkSize {
		end := start + parseChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[start:end])
		reply, err := s.ai.ChatSimpleStream(ctx, "", parseSystemPrompt,
			fmt.Sprintf("请解析以下招标文件片段（第 %d-%d 字，共 %d 字）：\n\n%s", start+1, end, len(runes), chunk))
		if err != nil {
			continue
		}
		merged = mergeParseResults(merged, decodeParseResult(reply))
		any = true
	}
	return merged, any
}

func decodeParseResult(reply string) parseFileResult {
	reply = extractJSON(reply)
	var res parseFileResult
	if err := json.Unmarshal([]byte(reply), &res); err != nil {
		return parseFileResult{}
	}
	return res
}

func mergeParseResults(a, b parseFileResult) parseFileResult {
	if a.Overview == "" {
		a.Overview = b.Overview
		a.OverviewQuote = b.OverviewQuote
	}
	if a.Duration == "" {
		a.Duration = b.Duration
		a.DurationQuote = b.DurationQuote
	}
	a.Qualification = appendUniqueParseItems(a.Qualification, b.Qualification)
	a.RedLines = appendUniqueParseItems(a.RedLines, b.RedLines)
	a.Format = appendUniqueParseItems(a.Format, b.Format)
	a.DarkRules = appendUniqueParseItems(a.DarkRules, b.DarkRules)
	for _, sc := range b.TechScoring {
		if !containsParseScoring(a.TechScoring, sc.Name) {
			a.TechScoring = append(a.TechScoring, sc)
		}
	}
	for _, kr := range b.KeyRequirements {
		if !containsString(a.KeyRequirements, kr) {
			a.KeyRequirements = append(a.KeyRequirements, kr)
		}
	}
	return a
}

func appendUniqueParseItems(dst, src []parseItem) []parseItem {
	for _, it := range src {
		if it.Name == "" && it.Content == "" {
			continue
		}
		if !containsParseItem(dst, it.Name, it.Content) {
			dst = append(dst, it)
		}
	}
	return dst
}

func containsParseItem(items []parseItem, name, content string) bool {
	for _, it := range items {
		if it.Name == name && it.Content == content {
			return true
		}
	}
	return false
}

func containsParseScoring(items []parseScoring, name string) bool {
	for _, it := range items {
		if it.Name == name {
			return true
		}
	}
	return false
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// resolveParseItems 把摘录定位到文件 Markdown 偏移与页码，生成落库行
func resolveParseItems(fileID, fileName string, pages []PageText, markdown string, res parseFileResult) []ParseResultItem {
	var out []ParseResultItem
	add := func(field, value, quote string) {
		item := ParseResultItem{FileID: fileID, Field: field, Value: value}
		if quote != "" {
			start, end, ok := LocateQuote(markdown, quote)
			item.Snippet = quote
			switch {
			case ok && markdown[start:end] == quote:
				item.Confidence = 1
			case ok:
				item.Confidence = 0.8
			default:
				item.Confidence = 0.3
			}
			item.Start, item.End = start, end
			item.Page = LocatePage(pages, quote)
		}
		out = append(out, item)
	}
	add("overview", res.Overview, res.OverviewQuote)
	add("duration", res.Duration, res.DurationQuote)
	for _, it := range res.Qualification {
		add("qualification", it.Name+"："+it.Content, it.Quote)
	}
	for _, it := range res.TechScoring {
		add("techScoring", it.Name+"（"+it.MaxScore+"分）："+it.Requirement, it.Quote)
	}
	for _, kr := range res.KeyRequirements {
		add("keyRequirements", kr, "")
	}
	for _, it := range res.RedLines {
		add("redLines", it.Name+"："+it.Content, it.Quote)
	}
	for _, it := range res.Format {
		add("format", it.Name+"："+it.Content, it.Quote)
	}
	for _, it := range res.DarkRules {
		add("darkRules", it.Name+"："+it.Content, it.Quote)
	}
	return out
}

// applyParseResult 把解析结果映射到 BidSummary（保持旧字段兼容）
func applyParseResult(bs *BidSummary, res parseFileResult) {
	bs.Overview = res.Overview
	bs.Duration = res.Duration
	bs.Qualification = toBidItems(res.Qualification)
	bs.Format = toBidItems(res.Format)
	bs.DarkRules = toBidItems(res.DarkRules)
	bs.RedLineItems = toBidItems(res.RedLines)
	bs.RedLines = nil
	for _, it := range res.RedLines {
		bs.RedLines = append(bs.RedLines, it.Content)
	}
	bs.KeyRequirements = res.KeyRequirements
	bs.TechScoring = nil
	for _, sc := range res.TechScoring {
		bs.TechScoring = append(bs.TechScoring, ScoringItem{
			Name: sc.Name, MaxScore: sc.MaxScore, Requirement: sc.Requirement,
			Sources: refsFromQuote(sc.Quote),
		})
	}
	bs.OverviewSources = refsFromQuote(res.OverviewQuote)
	bs.DurationSources = refsFromQuote(res.DurationQuote)
}

func toBidItems(items []parseItem) []BidItem {
	out := make([]BidItem, 0, len(items))
	for _, it := range items {
		out = append(out, BidItem{Name: it.Name, Content: it.Content, Sources: refsFromQuote(it.Quote)})
	}
	return out
}

// refsFromQuote 精简来源（snippet+置信度占位）；完整页码/偏移以 parse_results 表为准。
func refsFromQuote(quote string) []SourceRef {
	if quote == "" {
		return nil
	}
	return []SourceRef{{Snippet: quote, Confidence: 0.8}}
}

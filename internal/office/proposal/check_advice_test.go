package proposal

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestStructuredRulesCarryAdviceAndLocations 校验结果应携带整改建议与原文定位。
func TestStructuredRulesCarryAdviceAndLocations(t *testing.T) {
	st := newTestKnowledgeStore(t)
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	s.SetKnowledgeStoreForTest(st)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "soil-remediation-bid", "", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章 修复技术方案", Level: 1, Content: "修复目标为砷 ≤ 60mg/kg，采用固化稳定化工艺，配置药剂添加系统。本方案依据现行环保标准编制。"},
		{Title: "第二章 实施计划", Level: 1, Content: "修复目标为砷 ≤ 60mg/kg，采用固化稳定化工艺，配置药剂添加系统。本方案依据现行环保标准编制。**加粗**"},
	})
	p.BidSummary = &BidSummary{
		RedLines:  []string{"投标文件未按要求签字盖章作废标处理"},
		DarkRules: []BidItem{{Name: "暗标", Content: "不得出现单位名称、不得加粗"}},
	}
	_ = s.store.SaveProjectFacts(proj.ID, map[string]string{"工期": "90 日历天"})
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	items := RunChecks(context.Background(), p, s.structuredRules())
	if len(items) == 0 {
		t.Fatal("no check items produced")
	}
	byRule := map[string][]CheckItem{}
	for _, it := range items {
		byRule[it.Rule] = append(byRule[it.Rule], it)
	}
	// 废标条款未响应：必须给出整改建议。
	if red := byRule["废标条款响应"]; len(red) > 0 {
		if red[0].Status != "warn" || red[0].Suggestion == "" {
			t.Fatalf("废标条款建议缺失: %+v", red[0])
		}
	}
	// 暗标加粗：整改建议 + 定位到含 ** 的章节摘录。
	var dark *CheckItem
	for i := range items {
		if items[i].Rule == "暗标格式检查" && strings.Contains(items[i].Message, "加粗") {
			dark = &items[i]
			break
		}
	}
	if dark == nil {
		t.Fatal("未触发暗标加粗检查")
	}
	if dark.Suggestion == "" {
		t.Fatal("暗标加粗缺少整改建议")
	}
	if len(dark.Locations) == 0 || dark.Locations[0].SectionID == "" || dark.Locations[0].Excerpt == "" {
		t.Fatalf("暗标加粗缺少原文定位: %+v", dark.Locations)
	}
	if !strings.Contains(dark.Locations[0].Excerpt, "**") {
		t.Fatalf("定位摘录未包含加粗标记: %s", dark.Locations[0].Excerpt)
	}
	// 跨章节重复：给出合并建议并列出两处章节定位。
	dup := byRule["重复率检测"]
	if len(dup) == 0 {
		t.Fatal("未触发重复率检测")
	}
	if dup[0].Suggestion == "" || len(dup[0].Locations) != 2 {
		t.Fatalf("重复检测缺少建议或双章节定位: %+v", dup[0])
	}
}

// TestCoverageRuleMapsSuggestion AI 覆盖检查的 suggestion 应透传到整改建议。
func TestCoverageRuleMapsSuggestion(t *testing.T) {
	ai := &mockAI{def: `[{"name":"人员配置","maxScore":"10","covered":"none","suggestion":"补充项目负责人与持证人员配置表"}]`}
	s := newServiceAt(t, t.TempDir(), ai)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "soil-remediation-bid", "", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, Content: "修复方案正文。"},
	})
	p.BidSummary = &BidSummary{TechScoring: []ScoringItem{{Name: "人员配置", MaxScore: "10", Requirement: "提供人员表"}}}
	_ = s.store.Update(p)
	items := RunChecks(context.Background(), p, []CheckRule{
		ruleFunc{name: "评分覆盖检查", severity: "warning", fn: s.runCoverageRule},
	})
	if len(items) != 1 || items[0].Status != "fail" {
		t.Fatalf("覆盖检查异常: %+v", items)
	}
	if items[0].Suggestion != "补充项目负责人与持证人员配置表" {
		t.Fatalf("suggestion 未透传: %q", items[0].Suggestion)
	}
}

// TestExcerptAroundKeepsUTF8 摘录切分不能破坏多字节字符。
func TestExcerptAroundKeepsUTF8(t *testing.T) {
	src := "根据招标文件要求，修复目标为砷 ≤ 60mg/kg，采用固化稳定化工艺，配置药剂添加系统。"
	start, end, ok := LocateQuote(src, "砷 ≤ 60mg/kg")
	if !ok {
		t.Fatal("quote not found")
	}
	ex := excerptAround(src, start, end, 5)
	if !utf8.ValidString(ex) || !strings.Contains(ex, "砷 ≤ 60mg/kg") {
		t.Fatalf("excerpt broken: %q", ex)
	}
	if !strings.HasPrefix(ex, "…") || !strings.HasSuffix(ex, "…") {
		t.Fatalf("excerpt should be elided on both sides: %q", ex)
	}
}

package novelgate

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// completePlan 齐备的七字段计划（六项判据全满足）。
func completePlan() *types.ChapterPlan {
	return &types.ChapterPlan{
		SubIndex:       5,
		Title:          "第五章 破庙",
		PlotSummary:    "林晚夜入破庙，撞见内鬼交易。",
		KeyEvents:      []string{"雨夜夺符", "发现内鬼"},
		CharacterFocus: []string{"林晚"},
		EmotionalTone:  "压抑递进",
		NarrativeGoal:  "拿到残符并暴露内鬼",
		ConflictType:   "人际冲突",
		EndingType:     "悬念",
		EstimatedWords: 5000,
	}
}

// planCodes 计划问题码 → 严重度（与既有 codes 同形，避免与 Issue 版重名）。
func planCodes(problems []types.PlanProblem) map[string]string {
	m := map[string]string{}
	for _, p := range problems {
		m[p.Code] = p.Severity
	}
	return m
}

func TestPlanContractIssues_CompletePlanClean(t *testing.T) {
	got := PlanContractIssues(5, completePlan(), nil)
	if len(got) != 0 {
		t.Fatalf("齐备计划不应报问题: %+v", got)
	}
	// 其它章有各自不同的事件时同样零问题。
	others := []types.ChapterPlan{{SubIndex: 4, KeyEvents: []string{"初入宗门", "拜师被拒"}}}
	if got := PlanContractIssues(5, completePlan(), others); len(got) != 0 {
		t.Fatalf("事件不重复时不应报问题: %+v", got)
	}
}

func TestPlanContractIssues_MissingFieldsEach(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(p *types.ChapterPlan)
		wantCode string
		wantSev  string
	}{
		{"缺叙事目标", func(p *types.ChapterPlan) { p.NarrativeGoal = "  " }, types.PlanProblemMissingGoal, "S2"},
		{"缺关键事件", func(p *types.ChapterPlan) { p.KeyEvents = nil }, types.PlanProblemMissingKeyEvents, "S2"},
		{"缺冲突类型", func(p *types.ChapterPlan) { p.ConflictType = "" }, types.PlanProblemMissingConflict, "S2"},
		{"缺结尾类型", func(p *types.ChapterPlan) { p.EndingType = "" }, types.PlanProblemMissingEnding, "S2"},
		{"缺情绪基调", func(p *types.ChapterPlan) { p.EmotionalTone = "\n" }, types.PlanProblemMissingEmotion, "S3"},
		{"缺角色焦点", func(p *types.ChapterPlan) { p.CharacterFocus = []string{"", "  "} }, types.PlanProblemMissingCharacters, "S3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := completePlan()
			tc.mutate(p)
			got := PlanContractIssues(5, p, nil)
			if len(got) != 1 {
				t.Fatalf("应恰好报 1 条问题, got %d: %+v", len(got), got)
			}
			if got[0].Code != tc.wantCode || got[0].Severity != tc.wantSev {
				t.Fatalf("问题码/严重度不对: got %s/%s, want %s/%s",
					got[0].Code, got[0].Severity, tc.wantCode, tc.wantSev)
			}
			if strings.TrimSpace(got[0].Message) == "" {
				t.Fatalf("问题消息不得为空（需中文可执行提示）: %+v", got[0])
			}
		})
	}
}

func TestPlanContractIssues_KeyEventsInsufficient(t *testing.T) {
	for _, events := range [][]string{nil, {}, {"雨夜夺符"}, {"雨夜夺符", "  "}} {
		p := completePlan()
		p.KeyEvents = events
		got := planCodes(PlanContractIssues(5, p, nil))
		if got[types.PlanProblemMissingKeyEvents] != "S2" {
			t.Fatalf("关键事件 %v 应报不足（S2）: %+v", events, got)
		}
	}
	// 恰好 2 条（含空白噪声）视为齐备。
	p := completePlan()
	p.KeyEvents = []string{"雨夜夺符", "", "发现内鬼"}
	if got := planCodes(PlanContractIssues(5, p, nil)); got[types.PlanProblemMissingKeyEvents] != "" {
		t.Fatalf("2 条有效事件不应报不足: %+v", got)
	}
}

func TestPlanContractIssues_DuplicateAcrossChapters(t *testing.T) {
	p := completePlan()
	// 全半角差异（！U+FF01 vs !）与空白差异（"雨夜 夺符"）归一化后同一条，仍须判重；
	// 重复章号由其它章计划的 SubIndex 提供。
	p.KeyEvents = []string{"雨夜夺符！", "发现内鬼"}
	others := []types.ChapterPlan{{SubIndex: 3, KeyEvents: []string{" 雨夜 夺符! ", "无关事件"}}}
	got := PlanContractIssues(5, p, others)
	if len(got) != 1 {
		t.Fatalf("应恰好报 1 条跨章重复: %+v", got)
	}
	if got[0].Code != types.PlanProblemEventDuplicated || got[0].Severity != "S1" {
		t.Fatalf("跨章重复应为 S1 plan_event_duplicated: %+v", got[0])
	}
	if !strings.Contains(got[0].Message, "与第3章重复：雨夜夺符！") {
		t.Fatalf("消息应点名重复章号与事件: %q", got[0].Message)
	}
	if got[0].Evidence != "雨夜夺符！" {
		t.Fatalf("Evidence 应为本章事件原文: %q", got[0].Evidence)
	}

	// 多章重复同一条只点名一次（不刷屏），且不重复的本章事件不报。
	others2 := []types.ChapterPlan{
		{SubIndex: 2, KeyEvents: []string{"雨夜夺符!"}},
		{SubIndex: 4, KeyEvents: []string{"雨 夜 夺 符！"}},
	}
	if got := PlanContractIssues(5, p, others2); len(got) != 1 {
		t.Fatalf("多章重复同一条应只报一次: %+v", got)
	}

	// 无 SubIndex（0）时不编造章号，仍判重。
	noIdx := []types.ChapterPlan{{KeyEvents: []string{"雨夜夺符!"}}}
	gotNoIdx := PlanContractIssues(5, p, noIdx)
	if len(gotNoIdx) != 1 || !strings.Contains(gotNoIdx[0].Message, "与其它章重复：雨夜夺符！") {
		t.Fatalf("无章号时应退化为「其它章」仍判重: %+v", gotNoIdx)
	}
}

func TestPlanContractIssues_NilPlan(t *testing.T) {
	got := PlanContractIssues(7, nil, nil)
	if len(got) != 1 {
		t.Fatalf("无计划应返回单条问题: %+v", got)
	}
	if got[0].Code != PlanMissingCode || got[0].Severity != "S1" {
		t.Fatalf("无计划应为 S1 %s: %+v", PlanMissingCode, got[0])
	}
	if !strings.Contains(got[0].Message, "本章尚未制定计划") {
		t.Fatalf("消息应含「本章尚未制定计划」: %q", got[0].Message)
	}
}

// 既有判据行为不受影响（回归护栏）：OutlineContractIssues 与本文件判据互不干扰。
func TestPlanContractIssues_DoesNotTouchExistingGates(t *testing.T) {
	if got := OutlineContractIssues(types.OutlineNode{Title: "第一章", Summary: "谁在哪做什么", KeyPoints: []string{"a"}, Emotion: "紧张"}); len(got) != 0 {
		t.Fatalf("既有大纲判据行为被改: %+v", got)
	}
}

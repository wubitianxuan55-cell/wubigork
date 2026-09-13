package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newReviewTestApp 构造带临时小说项目的测试 App（零 LLM 依赖）。
func newReviewTestApp(t *testing.T) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "评审测试", "都市", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	return a
}

// reviewDim 取维度视图（缺维度即失败）。
func reviewDim(t *testing.T, payload ChapterReviewPayload, id string) ChapterReviewDimensionView {
	t.Helper()
	for _, d := range payload.Dimensions {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("维度 %s 缺失（%+v）", id, payload.Dimensions)
	return ChapterReviewDimensionView{}
}

func TestNovelReviewPlatforms_Listed(t *testing.T) {
	a := newReviewTestApp(t)
	ps := a.NovelReviewPlatforms()
	got := map[string]string{}
	for _, p := range ps {
		got[p.ID] = p.Label
	}
	if len(ps) < 4 || got["fanqie"] == "" || got["zhihu"] == "" {
		t.Fatalf("档位列表不完整: %+v", ps)
	}
}

func TestNovelChapterReview_PayloadEvidenceAndVerdict(t *testing.T) {
	a := newReviewTestApp(t)
	body := strings.Repeat("他把单据按顺序理好，放进文件袋。\n", 60)
	chapter := body + "他停下——没有回头。\n属于他的反击，才刚刚开始。"
	if err := a.getPM().WriteChapter(1, chapter); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	payload, err := a.NovelChapterReview(1, "fanqie")
	if err != nil {
		t.Fatalf("评审失败: %v", err)
	}
	if payload.Platform != "fanqie" || payload.PlatformLabel == "" || payload.ChapterNum != 1 || payload.Words == 0 {
		t.Fatalf("载荷头不完整: %+v", payload)
	}
	if payload.Verdict != "REJECT" {
		t.Fatalf("含破折号+预告收尾应判 REJECT，实际 %s（%v）", payload.Verdict, payload.Counts)
	}
	if payload.Counts["S1"] == 0 {
		t.Fatalf("应至少有一条 S1：%v", payload.Counts)
	}
	dash := reviewDim(t, payload, "dash_usage")
	if dash.Verdict != "fail" || len(dash.Evidence) == 0 || dash.Evidence[0].Excerpt == "" {
		t.Fatalf("破折号维度应带原文摘录: %+v", dash)
	}
	trailer := reviewDim(t, payload, "trailer_ending")
	if trailer.Verdict != "fail" || len(trailer.Evidence) == 0 {
		t.Fatalf("预告式收尾应带证据: %+v", trailer)
	}
	if len(payload.Advisories) != 3 {
		t.Fatalf("黄金三问应随报告返回: %v", payload.Advisories)
	}
}

func TestNovelChapterReview_ErrorsAndFallback(t *testing.T) {
	a := newReviewTestApp(t)
	if _, err := a.NovelChapterReview(0, "general"); err == nil {
		t.Fatal("章节号非法应报错")
	}
	if _, err := a.NovelChapterReview(1, "general"); err == nil {
		t.Fatal("空章节应报错")
	}
	if err := a.getPM().WriteChapter(1, "他抬头看了一眼。\n「走吧。」他说。\n门外是谁在等他？"); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	payload, err := a.NovelChapterReview(1, "不存在的档位")
	if err != nil {
		t.Fatalf("未知档位应回落 general: %v", err)
	}
	if payload.Platform != "general" {
		t.Fatalf("未知档位应回落 general，实际 %s", payload.Platform)
	}
}

func TestNovelChapterReview_ProtagonistDimension(t *testing.T) {
	a := newReviewTestApp(t)
	cf := &types.CharacterFile{Characters: []types.Character{
		{ID: "c1", Name: "林深", RoleType: "protagonist", Status: "Alive"},
		{ID: "c2", Name: "老陈", RoleType: "supporting", Status: "Alive"},
	}}
	if err := a.getPM().WriteCharacters(cf); err != nil {
		t.Fatalf("写角色库: %v", err)
	}
	if err := a.getPM().WriteChapter(1, strings.Repeat("老陈把账本合上，叹了口气。\n", 40)); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	payload, err := a.NovelChapterReview(1, "qidian")
	if err != nil {
		t.Fatalf("评审失败: %v", err)
	}
	d := reviewDim(t, payload, "protagonist_presence")
	if d.Verdict != "fail" || !strings.Contains(d.Detail, "林深") {
		t.Fatalf("主角缺席应 fail 且点名主角: %+v", d)
	}
	if err := a.getPM().WriteChapter(1, strings.Repeat("林深把账本合上，叹了口气。\n", 40)); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	payload, err = a.NovelChapterReview(1, "qidian")
	if err != nil {
		t.Fatalf("评审失败: %v", err)
	}
	if got := reviewDim(t, payload, "protagonist_presence"); got.Verdict != "pass" {
		t.Fatalf("主角出场应 pass: %+v", got)
	}
}

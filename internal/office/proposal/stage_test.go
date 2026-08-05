package proposal

import (
	"context"
	"testing"
)

func TestStageGate_BlocksGenerateBeforeParse(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{Name: "招标.txt", Markdown: "内容"}}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GenerateOutline(context.Background(), p.ID, "需求", OutlineStrategyReference, 0); err == nil {
		t.Fatal("未解析不应允许生成大纲")
	}
	if _, err := s.GenerateSection(context.Background(), p.ID, "x", ""); err == nil {
		t.Fatal("未解析不应允许生成章节")
	}
}

func TestStageAdvance_AfterParseAndOutline(t *testing.T) {
	ai := &mockAI{replies: map[string]string{
		"请解析以下招标文件": `{"overview":"概况","overviewQuote":"概况"}`,
		"总字数目标":     `{"title":"方案","sections":[{"title":"第一章","level":1}]}`,
	}}
	s := newServiceAt(t, t.TempDir(), ai)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{Name: "t.txt", Markdown: "概况"}}}
	_ = s.store.Update(p)
	parsed, err := s.ParseBidFile(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Stage != StageParse {
		t.Fatalf("解析后 stage = %q, want parse", parsed.Stage)
	}
	outlined, err := s.GenerateOutline(context.Background(), p.ID, "需求", OutlineStrategyReference, 0)
	if err != nil {
		t.Fatalf("GenerateOutline: %v", err)
	}
	if outlined.Stage != StageGenerate {
		t.Fatalf("大纲后 stage = %q, want generate", outlined.Stage)
	}
}

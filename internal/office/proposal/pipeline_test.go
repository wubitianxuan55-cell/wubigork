package proposal

import (
	"context"
	"testing"
)

func TestRunPipeline_AdvancesStages(t *testing.T) {
	ai := &mockAI{
		replies: map[string]string{
			"请解析以下招标文件": `{"overview":"概况","overviewQuote":"概况"}`,
			"总字数目标":     `{"title":"方案","sections":[{"title":"第一章","level":1}]}`,
		},
		def: "章节内容：这是一段足够长的章节正文内容用于流水线测试，包含工艺与参数说明。",
	}
	s := newServiceAt(t, t.TempDir(), ai)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{Name: "t.txt", Markdown: "概况"}}}
	_ = s.store.Update(p)
	var events []string
	_, _, err := s.RunPipeline(context.Background(), p.ID, func(step, status, detail string) {
		events = append(events, step+":"+status)
	})
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}
	final, _ := s.store.Get(p.ID)
	if final.Stage != StageCheck {
		t.Fatalf("流水线后 stage = %q, want check", final.Stage)
	}
	if len(events) == 0 {
		t.Fatal("无流水线进度事件")
	}
}

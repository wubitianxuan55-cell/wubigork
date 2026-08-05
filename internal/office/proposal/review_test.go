package proposal

import (
	"context"
	"testing"
)

func TestCheckSummaryAndReviewChecklist(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, nil)
	_ = s.store.SaveProjectFacts(proj.ID, map[string]string{"工期": "90 天"})
	_, items, err := s.CheckAll(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := s.store.Get(p.ID)
	if got.CheckSummary == nil || got.CheckSummary.Total != len(items) {
		t.Fatalf("CheckSummary 异常: %+v", got.CheckSummary)
	}
	if len(got.ReviewChecklist) == 0 {
		t.Fatal("复核清单为空")
	}
	got.ReviewChecklist[0].Done = true
	if err := s.store.Update(got); err != nil {
		t.Fatal(err)
	}
	again, _ := s.store.Get(p.ID)
	if !again.ReviewChecklist[0].Done {
		t.Fatal("复核清单未持久化")
	}
}

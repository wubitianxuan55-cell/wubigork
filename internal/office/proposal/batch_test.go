package proposal

import (
	"context"
	"testing"
)

func TestBuildBatchUnits_LeafOrder(t *testing.T) {
	sections := []ProposalSection{
		{Title: "第一章", Level: 1, Children: []ProposalSection{
			{ID: "a", Title: "1.1", Level: 2},
			{ID: "b", Title: "1.2", Level: 2},
		}},
		{ID: "c", Title: "第二章", Level: 1},
	}
	units := BuildBatchUnits(sections)
	if len(units) != 3 {
		t.Fatalf("units = %d, want 3", len(units))
	}
	if units[0].SectionID != "a" || units[1].SectionID != "b" || units[2].SectionID != "c" {
		t.Fatalf("顺序异常: %+v", units)
	}
}

func TestRunBatch_MarksCompletedAndResumes(t *testing.T) {
	ai := &mockAI{def: "生成内容：测试章节正文内容"}
	s := newServiceAt(t, t.TempDir(), ai)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, Index: 0, Children: []ProposalSection{
			{ID: "a", Title: "1.1", Level: 2, Index: 0, WordTarget: 800},
			{ID: "b", Title: "1.2", Level: 2, Index: 1, WordTarget: 800},
		}},
	})
	var progress []string
	err := s.RunBatch(context.Background(), p.ID, func(cur, total int, sid, status string, words int) {
		progress = append(progress, sid+":"+status)
	})
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}
	got, _ := s.store.Get(p.ID)
	if got.Sections[0].Children[0].Status != "completed" || got.Sections[0].Children[1].Status != "completed" {
		t.Fatalf("批量后状态异常: %+v", got.Sections)
	}
	if len(progress) != 2 {
		t.Fatalf("进度事件 = %d, want 2: %+v", len(progress), progress)
	}
	// 断点续写：只生成未完成单元
	got.Sections[0].Children[1].Status = "pending"
	got.Sections[0].Children[1].Content = ""
	_ = s.store.Update(got)
	var progress2 []string
	if err := s.RunBatch(context.Background(), p.ID, func(cur, total int, sid, status string, words int) {
		progress2 = append(progress2, sid+":"+status)
	}); err != nil {
		t.Fatal(err)
	}
	if len(progress2) != 2 || progress2[0] != "a:skipped" || progress2[1] != "b:done" {
		t.Fatalf("续写进度异常: %+v", progress2)
	}
}

func TestAssemble_Numbering(t *testing.T) {
	p := &Proposal{
		Title: "测试方案",
		Sections: []ProposalSection{
			{Title: "第一章", Level: 1, Children: []ProposalSection{
				{Title: "1.1", Level: 2, Content: "内容A"},
			}},
			{Title: "第二章", Level: 1, Content: "内容B"},
		},
	}
	md := Assemble(p)
	for _, want := range []string{"第1章", "第2章", "1.1"} {
		if !containsAny(md, want) {
			t.Errorf("装配结果缺少 %q:\n%s", want, md)
		}
	}
}

package tasks

// S1 双空间列落库测试（docs/gaea-space-dimension-design.md §1/§7 S1）：
// Space 字段写读往返、缺省 work、列表空间过滤（空=全量）。

import (
	"testing"
	"time"
)

func TestSubmitSpaceRoundtrip(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})

	// 旧入口 Submit：缺省归 "work"
	tkWork, err := m.Submit(KindFileIndex, "缺省空间任务", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if tkWork.Space != defaultTaskSpace {
		t.Fatalf("Submit 返回视图 Space = %q, want %q", tkWork.Space, defaultTaskSpace)
	}
	// 新入口 SubmitSpace：显式 play
	tkPlay, err := m.SubmitSpace(KindFileIndex, "play 任务", map[string]any{"k": 1}, "play")
	if err != nil {
		t.Fatalf("submit space: %v", err)
	}
	if tkPlay.Space != "play" {
		t.Fatalf("SubmitSpace 返回视图 Space = %q, want play", tkPlay.Space)
	}

	// Get 往返：列值原样回带
	gotWork, err := m.Get(tkWork.ID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if gotWork.Space != defaultTaskSpace {
		t.Errorf("Get(work) Space = %q, want %q", gotWork.Space, defaultTaskSpace)
	}
	gotPlay, err := m.Get(tkPlay.ID)
	if err != nil {
		t.Fatalf("get play: %v", err)
	}
	if gotPlay.Space != "play" {
		t.Errorf("Get(play) Space = %q, want play", gotPlay.Space)
	}
}

func TestListInSpaceFilter(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})

	w1, err := m.SubmitSpace(KindFileIndex, "w1", nil, "")
	if err != nil {
		t.Fatalf("submit w1: %v", err)
	}
	w2, err := m.Submit(KindFileIndex, "w2", nil)
	if err != nil {
		t.Fatalf("submit w2: %v", err)
	}
	p1, err := m.SubmitSpace(KindFileIndex, "p1", nil, "play")
	if err != nil {
		t.Fatalf("submit p1: %v", err)
	}
	// w1/w2 同毫秒提交时 created_at 可能并列，错开保证顺序可断言
	time.Sleep(2 * time.Millisecond)

	// 空 space = 不过滤（旧行为恒真）：全量 3 条
	all, err := m.ListInSpace(0, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListInSpace(\"\") = %d 条, want 3", len(all))
	}
	// work = 2 条（w1/w2）
	works, err := m.ListInSpace(10, defaultTaskSpace)
	if err != nil {
		t.Fatalf("list work: %v", err)
	}
	if len(works) != 2 {
		t.Fatalf("ListInSpace(work) = %d 条, want 2", len(works))
	}
	ids := map[string]bool{}
	for _, tk := range works {
		ids[tk.ID] = true
	}
	if !ids[w1.ID] || !ids[w2.ID] {
		t.Errorf("ListInSpace(work) 缺任务: %v", works)
	}
	// play = 1 条
	plays, err := m.ListInSpace(10, "play")
	if err != nil {
		t.Fatalf("list play: %v", err)
	}
	if len(plays) != 1 || plays[0].ID != p1.ID || plays[0].Space != "play" {
		t.Fatalf("ListInSpace(play) = %+v, want [p1]", plays)
	}
	// 未知空间 = 空集
	none, err := m.ListInSpace(10, "none")
	if err != nil {
		t.Fatalf("list none: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("ListInSpace(none) = %d 条, want 0", len(none))
	}
	// 旧入口 List 保持跨空间全量
	legacy, err := m.List(50)
	if err != nil {
		t.Fatalf("list legacy: %v", err)
	}
	if len(legacy) != 3 {
		t.Fatalf("List() = %d 条, want 3（旧行为零变化）", len(legacy))
	}
}

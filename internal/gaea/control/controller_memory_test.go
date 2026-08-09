package control

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestSaveDreamFactsDedupesByName(t *testing.T) {
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: t.TempDir(),
		DB:      nil,
	})

	n, err := c.SaveDreamFacts([]memory.Memory{
		{Name: "user-unit", Type: memory.TypeUser, Kind: memory.KindSemantic, Description: "单位", Body: "XX 公司"},
		{Name: "project-budget", Type: memory.TypeProject, Kind: memory.KindSemantic, Description: "预算口径", Body: "税前"},
	})
	if err != nil {
		t.Fatalf("SaveDreamFacts: %v", err)
	}
	if n != 2 {
		t.Fatalf("saved = %d, want 2", n)
	}

	// 同名再存 → 更新而非新增（按 name 去重）
	n, err = c.SaveDreamFacts([]memory.Memory{
		{Name: "user-unit", Type: memory.TypeUser, Kind: memory.KindSemantic, Description: "单位（更新）", Body: "YY 公司"},
	})
	if err != nil {
		t.Fatalf("SaveDreamFacts update: %v", err)
	}
	if n != 1 {
		t.Fatalf("update saved = %d, want 1", n)
	}
	list := c.Memory().Store.List()
	if len(list) != 2 {
		t.Fatalf("facts = %d, want 2 (dedupe)", len(list))
	}
	found := false
	for _, m := range list {
		if m.Name == "user-unit" && strings.Contains(m.Description, "更新") {
			found = true
		}
	}
	if !found {
		t.Fatal("user-unit fact should be updated")
	}

	// 空 name / 无内容事实跳过
	n, err = c.SaveDreamFacts([]memory.Memory{
		{Name: "", Description: "空名字"},
		{Name: "empty-body", Description: "", Body: ""},
	})
	if err != nil || n != 0 {
		t.Fatalf("empty facts: n=%d err=%v, want 0/nil", n, err)
	}
}

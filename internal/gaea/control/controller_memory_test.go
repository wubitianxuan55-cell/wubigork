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

	n, err := c.SaveDreamFacts("test", []memory.Memory{
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
	n, err = c.SaveDreamFacts("test", []memory.Memory{
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
	n, err = c.SaveDreamFacts("test", []memory.Memory{
		{Name: "", Description: "空名字"},
		{Name: "empty-body", Description: "", Body: ""},
	})
	if err != nil || n != 0 {
		t.Fatalf("empty facts: n=%d err=%v, want 0/nil", n, err)
	}
}

// T6-8.1 dream 写入审计：自动路径（auto_dream）每次实际写入都落一行审计日志。
func TestSaveDreamFactsAuditAutoPath(t *testing.T) {
	userDir := t.TempDir()
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: userDir,
		DB:      nil,
	})

	n, err := c.SaveDreamFacts("auto_dream", []memory.Memory{
		{Name: "user-unit", Type: memory.TypeUser, Kind: memory.KindSemantic, Description: "单位", Body: "XX 公司"},
		{Name: "project-budget", Type: memory.TypeProject, Kind: memory.KindSemantic, Description: "预算口径", Body: "税前"},
	})
	if err != nil || n != 2 {
		t.Fatalf("SaveDreamFacts: n=%d err=%v, want 2/nil", n, err)
	}
	entries := DreamAuditEntries(userDir, 10)
	if len(entries) != 1 {
		t.Fatalf("审计行数 = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Source != "auto_dream" {
		t.Fatalf("source = %q, want auto_dream", e.Source)
	}
	if e.Saved != 2 {
		t.Fatalf("saved = %d, want 2", e.Saved)
	}
	if len(e.Names) != 2 || e.Names[0] != "user-unit" || e.Names[1] != "project-budget" {
		t.Fatalf("names = %v", e.Names)
	}
	if e.TS == "" {
		t.Fatal("审计行缺少时间戳")
	}
}

// T6-8.1 dream 写入审计：显式路径（explicit，用户接受记忆建议）同样落审计。
func TestSaveDreamFactsAuditExplicitPath(t *testing.T) {
	userDir := t.TempDir()
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: userDir,
		DB:      nil,
	})

	if _, err := c.SaveDreamFacts("explicit", []memory.Memory{
		{Name: "workflow-cost", Type: memory.TypeProject, Kind: memory.KindProcedural, Description: "成本测算工作流", Body: "步骤 1-2-3"},
	}); err != nil {
		t.Fatalf("SaveDreamFacts: %v", err)
	}
	entries := DreamAuditEntries(userDir, 10)
	if len(entries) != 1 {
		t.Fatalf("审计行数 = %d, want 1", len(entries))
	}
	if entries[0].Source != "explicit" || entries[0].Saved != 1 {
		t.Fatalf("审计行 = %+v, want source=explicit saved=1", entries[0])
	}

	// 未实际写入（空事实）不落审计
	if n, err := c.SaveDreamFacts("explicit", []memory.Memory{{Name: "", Description: "空"}}); err != nil || n != 0 {
		t.Fatalf("empty: n=%d err=%v", n, err)
	}
	if entries := DreamAuditEntries(userDir, 10); len(entries) != 1 {
		t.Fatalf("空写入不应新增审计行, got %d", len(entries))
	}
}

// T6-8.1 记忆未配置（mem == nil / UserDir 空）时审计安全跳过。
func TestSaveDreamFactsAuditDisabledStore(t *testing.T) {
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	if n, err := c.SaveDreamFacts("auto_dream", []memory.Memory{{Name: "x", Description: "y"}}); err != nil || n != 0 {
		t.Fatalf("mem nil: n=%d err=%v, want 0/nil", n, err)
	}
}

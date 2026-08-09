package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/factbase"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// TestFactToolsMeta 校验事实底座工具元信息与 Schema。
func TestFactToolsMeta(t *testing.T) {
	tools := []interface {
		Name() string
		Description() string
		Schema() json.RawMessage
		ReadOnly() bool
	}{
		factAddTool{}, factListTool{}, factClearTool{},
	}
	for _, tt := range tools {
		if strings.TrimSpace(tt.Name()) == "" || strings.TrimSpace(tt.Description()) == "" {
			t.Fatalf("tool %q has empty metadata", tt.Name())
		}
		if !json.Valid(tt.Schema()) {
			t.Fatalf("tool %q has invalid schema", tt.Name())
		}
	}
	if (factAddTool{}).ReadOnly() || (factClearTool{}).ReadOnly() {
		t.Fatal("fact_add/fact_clear must be writers")
	}
	if !(factListTool{}).ReadOnly() {
		t.Fatal("fact_list must be read-only")
	}
}

// TestFactBaseSnapshotWithoutEngine 引擎未初始化时快照应为空而非 panic。
func TestFactBaseSnapshotWithoutEngine(t *testing.T) {
	view := factBaseSnapshot()
	if view.Count != 0 || len(view.Facts) != 0 || view.Markdown != "" {
		t.Fatalf("expected empty view without engine, got %+v", view)
	}
}

// TestDeleteSessionFileRemovesFacts 删除会话时顺带清理事实底座文件。
func TestDeleteSessionFileRemovesFacts(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "20260809-123.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp := factbase.PathFor(sessionPath)
	st := factbase.NewStore(fp)
	if err := st.Add("工期", "90 日历天", "招标文件", "project"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fp); err != nil {
		t.Fatalf("facts file should exist: %v", err)
	}
	if err := deleteSessionFile(dir, sessionPath); err != nil {
		t.Fatalf("deleteSessionFile: %v", err)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatal("session file should be gone")
	}
	if _, err := os.Stat(fp); !os.IsNotExist(err) {
		t.Fatal("facts file should be cleaned up with the session")
	}
}

// TestGaeaFactBaseViewNoEngine 直接调用绑定在未初始化时返回空视图。
func TestGaeaFactBaseViewNoEngine(t *testing.T) {
	a := &App{}
	view := a.GaeaFactBase()
	if view.Count != 0 {
		t.Fatalf("expected empty view, got %+v", view)
	}
	if err := a.GaeaFactBaseClear(); err != nil {
		t.Fatalf("clear without engine should be a no-op: %v", err)
	}
}

// TestPromoteFactsToMemory 事实底座一键沉淀：写入长期记忆并按名去重。
func TestPromoteFactsToMemory(t *testing.T) {
	store := memory.Store{Dir: t.TempDir(), GlobalDir: t.TempDir()}
	now := time.Now()
	facts := []factbase.Fact{
		{Key: "construction-period", Value: "90 日历天", Source: "招标文件 P3", Category: "project", UpdatedAt: now},
		{Key: "修复目标", Value: "砷 ≤ 60 mg/kg", Source: "招标文件 P5", Category: "data", UpdatedAt: now},
		{Key: "汇报风格", Value: "先结论后过程，一页一要点", Category: "preference", UpdatedAt: now},
	}
	n, err := promoteFactsToMemory(store, facts)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if n != 3 {
		t.Fatalf("want 3 promoted, got %d", n)
	}
	all := store.List()
	if len(all) != 3 {
		t.Fatalf("want 3 memories on disk, got %d", len(all))
	}
	// 按名去重：再次沉淀同 key 只更新不新增。
	facts[0].Value = "120 日历天（答疑纪要修正）"
	if _, err := promoteFactsToMemory(store, facts); err != nil {
		t.Fatalf("re-promote: %v", err)
	}
	if got := len(store.List()); got != 3 {
		t.Fatalf("re-promote should update in place, got %d memories", got)
	}
	byName := map[string]memory.Memory{}
	for _, m := range store.List() {
		byName[m.Name] = m
	}
	period, ok := byName[factMemoryName("construction-period")]
	if !ok {
		t.Fatalf("construction-period memory missing: %+v", byName)
	}
	if !strings.Contains(period.Body, "120 日历天") {
		t.Fatalf("upserted body missing new value: %s", period.Body)
	}
	if !strings.Contains(period.Body, "来源：招标文件 P3") {
		t.Fatalf("body missing source: %s", period.Body)
	}
	pref, ok := byName[factMemoryName("汇报风格")]
	if !ok {
		t.Fatalf("preference memory missing")
	}
	if pref.Type != memory.TypeUser {
		t.Fatalf("preference category should map to user type, got %s", pref.Type)
	}
}

// TestFactMemoryName CJK key 回退到稳定 hash 名，ASCII key 保留 kebab。
func TestFactMemoryName(t *testing.T) {
	if got := factMemoryName("construction-period"); got != "construction-period" {
		t.Fatalf("ascii slug mismatch: %s", got)
	}
	a := factMemoryName("工期")
	b := factMemoryName("工期")
	if a == "" || a != b {
		t.Fatalf("cjk slug should be stable and non-empty: %q %q", a, b)
	}
	if strings.ContainsAny(a, "/\\:*?\"<>| ") {
		t.Fatalf("slug has unsafe chars: %q", a)
	}
	if got := factMemoryName(""); got == "" {
		t.Fatal("empty key should still produce a fallback name")
	}
}

// TestOneLineSummary 摘要折叠空白并限长，避免破坏单行索引。
func TestOneLineSummary(t *testing.T) {
	long := strings.Repeat("字", 200)
	got := oneLineSummary("  90   日历天  \n\n 含验收 ")
	if got != "90 日历天 含验收" {
		t.Fatalf("whitespace collapse failed: %q", got)
	}
	if len([]rune(oneLineSummary(long))) > 81 {
		t.Fatalf("summary should be capped")
	}
}

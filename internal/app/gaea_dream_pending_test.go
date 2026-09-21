package app

import (
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// v4.377 建议制：待确认队列 FIFO 上限、出队语义、空间过滤。
func TestDreamPendingAppendListTake(t *testing.T) {
	dir := t.TempDir()
	items := []dreamPendingItem{
		{ID: newDreamPendingID(), Kind: "fact", Name: "user-unit", Type: "user", Description: "单位", Body: "XX 公司", Space: "work"},
		{ID: newDreamPendingID(), Kind: "note", NoteScope: "local", Note: "口径：税前", Space: "work"},
	}
	if n, err := dreamPendingAppend(dir, items); err != nil || n != 2 {
		t.Fatalf("append = %d, %v; want 2, nil", n, err)
	}
	got := dreamPendingList(dir, "work")
	if len(got) != 2 {
		t.Fatalf("list = %d, want 2", len(got))
	}
	// 空间过滤：work 队列在 play 不可见
	if got := dreamPendingList(dir, "play"); len(got) != 0 {
		t.Fatalf("play list = %d, want 0（空间隔离）", len(got))
	}

	// take 中间一条：返回该项且队列少一
	it, ok, err := dreamPendingTake(dir, got[0].ID)
	if err != nil || !ok {
		t.Fatalf("take: ok=%v err=%v", ok, err)
	}
	if it.Kind != "fact" || it.Name != "user-unit" {
		t.Fatalf("took %+v, want fact user-unit", it)
	}
	if rest := dreamPendingList(dir, "work"); len(rest) != 1 {
		t.Fatalf("after take = %d, want 1", len(rest))
	}
	// take 不存在：ok=false、err=nil（调用方据此走旧路径或报「不存在」）
	if _, ok, err := dreamPendingTake(dir, "no-such-id"); ok || err != nil {
		t.Fatalf("take missing: ok=%v err=%v, want false/nil", ok, err)
	}
	// 空目录安全
	if got := dreamPendingList(t.TempDir(), "work"); len(got) != 0 {
		t.Fatalf("empty dir list = %d, want 0", len(got))
	}
}

// FIFO 截断：超过 dreamPendingMax 丢最旧。
func TestDreamPendingFIFOCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < dreamPendingMax+5; i++ {
		items := []dreamPendingItem{{
			ID: newDreamPendingID(), Kind: "note", Note: "note-" + strings.Repeat("x", i+1), Space: "",
		}}
		if n, err := dreamPendingAppend(dir, items); err != nil || n != 1 {
			t.Fatalf("append %d: n=%d err=%v", i, n, err)
		}
	}
	got := dreamPendingList(dir, "")
	if len(got) != dreamPendingMax {
		t.Fatalf("queue = %d, want cap %d", len(got), dreamPendingMax)
	}
	// 最旧的 5 条被丢：队首应是第 6 次写入（note 长度 6）
	if got[0].Note != "note-"+strings.Repeat("x", 6) {
		t.Fatalf("FIFO dropped wrong end: %q", got[0].Note)
	}
}

// suggest 入队分流：work 空间 facts+notes 全入队；play 空间 notes 不入队
// （接受路径 QuickAdd 只写 work 项目文档，与旧直写行为的 play 丢弃纪律一致）。
func TestDreamEnqueueSuggestions(t *testing.T) {
	dir := t.TempDir()
	res := dreamResult{
		Facts: []dreamFact{{Name: "user-unit", Type: "user", Kind: "semantic", Description: "单位", Body: "XX 公司"}},
		Notes: []dreamNote{{Scope: "local", Note: "口径：税前"}},
	}
	n, err := dreamEnqueueSuggestions(dir, "work", res)
	if err != nil || n != 2 {
		t.Fatalf("work enqueue = %d, %v; want 2, nil", n, err)
	}
	n, err = dreamEnqueueSuggestions(dir, "play", res)
	if err != nil || n != 1 {
		t.Fatalf("play enqueue = %d, %v; want 1（notes 不入队）, nil", n, err)
	}
	play := dreamPendingList(dir, "play")
	if len(play) != 1 || play[0].Kind != "fact" {
		t.Fatalf("play queue = %+v, want 1 fact", play)
	}
}

// 待确认建议 → 面板视图转换：fact 型带 name/type，note 型 Type="note"。
func TestDreamPendingViews(t *testing.T) {
	dir := t.TempDir()
	_, _ = dreamEnqueueSuggestions(dir, "work", dreamResult{
		Facts: []dreamFact{{Name: "user-unit", Type: "user", Kind: "semantic", Description: "单位", Body: "XX 公司"}},
		Notes: []dreamNote{{Scope: "project", Note: "隧道衬砌用 C35"}},
	})
	views := dreamPendingViews(dir, "work")
	if len(views) != 2 {
		t.Fatalf("views = %d, want 2", len(views))
	}
	if views[0].Type != "user" || views[0].Name != "user-unit" || views[0].ID == "" {
		t.Fatalf("fact view = %+v", views[0])
	}
	if views[1].Type != "note" || !strings.Contains(views[1].Body, "C35") || !strings.Contains(views[1].Description, "project") {
		t.Fatalf("note view = %+v", views[1])
	}
	// 空间过滤（play 看不到 work 建议）
	if views := dreamPendingViews(dir, "play"); len(views) != 0 {
		t.Fatalf("play views = %d, want 0", len(views))
	}
}

// gaeaDreamMode：ga.cfg 生效值优先；off/suggest 归一见 config 包单测。
func TestGaeaDreamModeFromGlobalCfg(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ga.cfg = &gaeaConfig.Config{Dream: gaeaConfig.DreamConfig{Mode: "auto"}}
	if m := gaeaDreamMode(); m != "auto" {
		t.Fatalf("gaeaDreamMode(auto) = %q", m)
	}
	ga.cfg = &gaeaConfig.Config{}
	if m := gaeaDreamMode(); m != "suggest" {
		t.Fatalf("gaeaDreamMode(空) = %q, want suggest", m)
	}
}

// 接受待确认建议（fact 型）：按队列项内容入库并出队（source=explicit）；
// 忽略：出队不写库。
func TestAcceptAndDismissPendingSuggestion(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	userDir := t.TempDir()
	mem := memory.Load(memory.Options{CWD: t.TempDir(), UserDir: userDir, DB: nil})
	ctrl := control.New(control.Options{
		Sink:   event.FuncSink(func(event.Event) {}),
		Memory: mem,
		Space:  "work",
	})
	ga.cfg = &gaeaConfig.Config{}
	ga.ctrl = ctrl

	dir2 := userDir // 队列与记忆同目录
	n, err := dreamEnqueueSuggestions(dir2, "work", dreamResult{
		Facts: []dreamFact{{Name: "user-unit", Type: "user", Kind: "semantic", Description: "单位", Body: "XX 公司"}},
	})
	if err != nil || n != 1 {
		t.Fatalf("enqueue = %d, %v", n, err)
	}
	view := dreamPendingViews(dir2, "work")[0]

	a := &App{}
	if _, err := a.GaeaAcceptMemorySuggestion(view); err != nil {
		t.Fatalf("accept: %v", err)
	}
	// 入库 1 条 + 队列清空
	facts := ctrl.Memory().Store.List()
	if len(facts) != 1 || facts[0].Name != "user-unit" {
		t.Fatalf("facts after accept = %+v", facts)
	}
	if rest := dreamPendingList(dir2, "work"); len(rest) != 0 {
		t.Fatalf("queue after accept = %d, want 0", len(rest))
	}
	// 再接受同一条（已出队但带 name）→ 走旧路径按 name 幂等更新，不产生重复
	if _, err := a.GaeaAcceptMemorySuggestion(view); err != nil {
		t.Fatalf("accept twice (legacy path): %v", err)
	}
	if facts := ctrl.Memory().Store.List(); len(facts) != 1 {
		t.Fatalf("facts after double accept = %d, want 1（同名去重）", len(facts))
	}

	// dismiss：出队不写库
	_, _ = dreamEnqueueSuggestions(dir2, "work", dreamResult{
		Facts: []dreamFact{{Name: "tmp-fact", Type: "project", Description: "临时", Body: "内容"}},
	})
	v2 := dreamPendingViews(dir2, "work")[0]
	if err := a.GaeaDismissMemorySuggestion(v2.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	if rest := dreamPendingList(dir2, "work"); len(rest) != 0 {
		t.Fatalf("queue after dismiss = %d, want 0", len(rest))
	}
	if facts := ctrl.Memory().Store.List(); len(facts) != 1 {
		t.Fatalf("facts after dismiss = %d, want 1（dismiss 不写库）", len(facts))
	}
}

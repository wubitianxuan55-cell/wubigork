package memory

// mode_test.go — 记忆运行模式维度的行为契约（2026-11-25 办公工具梳理批），
// 外加记忆工具合并（forget / promote_session_facts / memory_get 并入
// remember 与 memory_search）后迁来的用例。全部走临时库，不碰用户正式库。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// fakeQueue records the turn-tail notes the remember tool queues.
type fakeQueue struct{ notes []string }

func (f *fakeQueue) QueueMemory(note string) { f.notes = append(f.notes, note) }

// fakePromoter 让 remember(action=promote) 走通 ctx 钩子。
type fakePromoter struct{ n int }

func (f *fakePromoter) PromoteSessionFacts() (int, error) { return f.n, nil }

// modeStore 建一个临时 SQLite 后端的未收窄存储（模式维度关闭）。
func modeStore(t *testing.T) Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	return SQLiteStoreFor(gdb, dir, "/Users/me/proj")
}

func modeNamesOf(ms []Memory) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

func TestModeVisibleRules(t *testing.T) {
	cases := []struct {
		session string
		mode    string
		want    bool
	}{
		{"", "", true}, {"", "cost", true}, {"", "proposal", true}, // 维度关闭：全可见
		{"standard", "", true}, {"standard", "cost", false}, // 标准档只看通用
		{"cost", "", true}, {"cost", "cost", true}, {"cost", "proposal", false},
		{"proposal", "cost", false}, {"proposal", "proposal", true},
	}
	for _, c := range cases {
		if got := modeVisible(c.session, Memory{Mode: c.mode}); got != c.want {
			t.Errorf("modeVisible(%q, mode=%q) = %v, want %v", c.session, c.mode, got, c.want)
		}
	}
}

func TestModeViewIsolation(t *testing.T) {
	base := modeStore(t)
	for _, m := range []Memory{
		{Name: "general-fact", Description: "g", Body: "b"},
		{Name: "cost-fact", Description: "c", Body: "b", Mode: "cost"},
		{Name: "proposal-fact", Description: "p", Body: "b", Mode: "proposal"},
	} {
		if _, err := base.Save(m); err != nil {
			t.Fatal(err)
		}
	}
	check := func(session string, want ...string) {
		t.Helper()
		got := strings.Join(modeNamesOf(base.InMode(session).List()), ",")
		if got != strings.Join(want, ",") {
			t.Errorf("InMode(%q).List() = %q, want %q", session, got, strings.Join(want, ","))
		}
	}
	check("", "cost-fact", "general-fact", "proposal-fact")
	check("cost", "cost-fact", "general-fact")
	check("proposal", "general-fact", "proposal-fact")
	check("standard", "general-fact")

	cost := base.InMode("cost")
	if _, ok := cost.Get("proposal-fact"); ok {
		t.Error("造价档不应读到方案档记忆")
	}
	// 按名写先过可见性：造价档删不掉方案档的记忆，也不报「删成功」。
	if _, err := cost.Archive("proposal-fact"); err != nil {
		t.Fatalf("不可见记忆的归档应静默 no-op，得到 %v", err)
	}
	if _, ok := base.Get("proposal-fact"); !ok {
		t.Error("造价档视图不该删掉方案档记忆")
	}
	if err := cost.ChangeType("proposal-fact", TypeFeedback); err == nil {
		t.Error("造价档视图不该改得了方案档记忆")
	}
	if err := cost.Delete("cost-fact"); err != nil {
		t.Fatalf("删除本模式记忆: %v", err)
	}
	if _, ok := base.Get("cost-fact"); ok {
		t.Error("本模式记忆应被删除（归档）")
	}

	// 索引：模式专属条目带模式名，通用条目不带；维度关闭时无标签（逐字节同旧）。
	// cost-fact 已在上一步删除，重建一条用于标签断言。
	if _, err := base.Save(Memory{Name: "cost-fact", Description: "c", Body: "b", Mode: "cost"}); err != nil {
		t.Fatal(err)
	}
	idx := base.InMode("cost").Index()
	if !strings.Contains(idx, "general-fact.md)") || !strings.Contains(idx, "cost-fact.md)") {
		t.Fatalf("索引缺条目：%s", idx)
	}
	if !strings.Contains(idx, "cost-fact.md)〔造价〕") || strings.Contains(idx, "general-fact.md)〔") {
		t.Errorf("模式标签不对：%s", idx)
	}
	if strings.Contains(base.Index(), "〔") {
		t.Errorf("维度关闭的索引不该有模式标签：%s", base.Index())
	}
	if strings.Contains(base.InMode("proposal").Index(), "cost-fact") {
		t.Error("方案档索引不该列造价档记忆")
	}
}

func rememberExec(t *testing.T, store Store, args string) (string, error) {
	t.Helper()
	return NewRememberTool(store, nil).Execute(context.Background(), json.RawMessage(args))
}

func TestRememberDefaultModeStamping(t *testing.T) {
	base := modeStore(t)
	cost := base.InMode("cost")

	// 默认规则：项目/参考资料/过程规矩归当前模式；user/feedback 通用。
	for _, tc := range []struct {
		typ  string
		kind string
		want string
	}{
		{"project", "semantic", "cost"}, {"reference", "semantic", "cost"},
		{"project", "procedural", "cost"}, {"feedback", "semantic", ""},
		{"user", "semantic", ""}, {"", "", "cost"},
	} {
		name := "auto-" + tc.typ + "-" + tc.kind
		if _, err := rememberExec(t, cost, `{"name":"`+name+`","description":"d","body":"b","type":"`+tc.typ+`","kind":"`+tc.kind+`"}`); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		m, ok := base.Get(name)
		if !ok || m.Mode != tc.want {
			t.Errorf("%s 归属 = %q, want %q", name, m.Mode, tc.want)
		}
	}

	// 显式 scope。
	if _, err := rememberExec(t, cost, `{"name":"scoped-general","description":"d","body":"b","type":"project","scope":"general"}`); err != nil {
		t.Fatal(err)
	}
	if m, _ := base.Get("scoped-general"); m.Mode != "" {
		t.Errorf("scope=general 应落通用，得到 %q", m.Mode)
	}
	if _, err := rememberExec(t, cost, `{"name":"scoped-mode","description":"d","body":"b","type":"feedback","scope":"mode"}`); err != nil {
		t.Fatal(err)
	}
	if m, _ := base.Get("scoped-mode"); m.Mode != "cost" {
		t.Errorf("scope=model 应归当前模式，得到 %q", m.Mode)
	}

	// 按名修订已有记忆：不给 scope 不改归属（通用仍是通用，造价仍是造价）。
	if _, err := rememberExec(t, cost, `{"name":"scoped-general","description":"d2","body":"b2","type":"project"}`); err != nil {
		t.Fatal(err)
	}
	if m, _ := base.Get("scoped-general"); m.Mode != "" || m.Body != "b2" {
		t.Errorf("修订通用记忆应保持通用，得到 mode=%q body=%q", m.Mode, m.Body)
	}

	// 标准档与未启用维度：新记忆一律通用。
	for _, s := range []Store{base.InMode("standard"), base} {
		if _, err := rememberExec(t, s, `{"name":"in-`+s.SessionMode()+`-x","description":"d","body":"b","type":"project"}`); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"in-standard-x", "in--x"} {
		if m, _ := base.Get(n); m.Mode != "" {
			t.Errorf("%s 应为通用，得到 %q", n, m.Mode)
		}
	}

	// 回执说明归属。
	out, err := rememberExec(t, cost, `{"name":"receipt-mode","description":"d","body":"b","type":"project"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "造价") {
		t.Errorf("回执应说明只在造价模式加载：%s", out)
	}

	// 造价档看不见的记忆，按名「修订」等于新建一条造价的（不覆盖方案档那条）。
	if _, err := base.Save(Memory{Name: "proposal-only", Description: "d", Body: "orig", Mode: "proposal"}); err != nil {
		t.Fatal(err)
	}
	if _, err := rememberExec(t, cost, `{"name":"proposal-only","description":"d","body":"from-cost","type":"project"}`); err == nil {
		t.Fatal("同名记忆被别的模式占用时应被拒绝，而不是悄悄覆盖")
	}
}

func TestRememberActions(t *testing.T) {
	store := modeStore(t)
	tl := NewRememberTool(store, nil)
	if tl.Name() != "remember" || tl.ReadOnly() || !json.Valid(tl.Schema()) {
		t.Fatalf("unexpected tool identity/schema: name=%q readonly=%v", tl.Name(), tl.ReadOnly())
	}
	if _, err := store.Save(Memory{Name: "stale-fact", Description: "d", Type: TypeProject, Body: "b"}); err != nil {
		t.Fatal(err)
	}

	// forget：删掉并排一条回合尾注（索引行本会话仍在缓存里，提醒模型别再信）。
	q := &fakeQueue{}
	out, err := tl.Execute(WithQueue(context.Background(), q), json.RawMessage(`{"action":"forget","name":"stale-fact"}`))
	if err != nil {
		t.Fatalf("forget: %v", err)
	}
	if !strings.Contains(out, "stale-fact") || len(store.List()) != 0 {
		t.Fatalf("forget 没删掉记忆：%q / %v", out, store.List())
	}
	if len(q.notes) != 1 || !strings.Contains(q.notes[0], "stale-fact") {
		t.Fatalf("应排一条点名被删记忆的尾注，得到 %v", q.notes)
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"forget"}`)); err == nil {
		t.Error("forget 缺 name 应报错")
	}

	// promote：经 ctx 钩子转正会话事实。
	out, err = tl.Execute(WithPromoter(context.Background(), &fakePromoter{n: 2}), json.RawMessage(`{"action":"promote"}`))
	if err != nil || !strings.Contains(out, "2") {
		t.Fatalf("promote: out=%q err=%v", out, err)
	}
	if out, _ = tl.Execute(WithPromoter(context.Background(), &fakePromoter{}), json.RawMessage(`{"action":"promote"}`)); !strings.Contains(out, "没有") {
		t.Errorf("无临时记忆时应如实说明：%q", out)
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"promote"}`)); err == nil {
		t.Error("没有 promoter 钩子时应报不可用")
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"explode"}`)); err == nil {
		t.Error("未知 action 应报错")
	}

	// 缺参骨架：动作不同骨架不同。
	if h := (rememberTool{store: store}).UsageHint(json.RawMessage(`{"action":"forget"}`)); !strings.Contains(h, `"action":"forget"`) {
		t.Errorf("forget 骨架不对：%s", h)
	}
	if h := (rememberTool{store: store}).UsageHint(json.RawMessage(`{"body":"x"}`)); !strings.Contains(h, "description") {
		t.Errorf("save 缺 description 的骨架应点名 description：%s", h)
	}
}

func TestRenderMemory(t *testing.T) {
	out := RenderMemory(Memory{Name: "prefers-tabs", Title: "Prefers tabs", Type: TypeUser, Kind: KindEpisodic,
		Tags: []string{"tabs"}, Body: "Use tabs.", Mode: "cost"})
	for _, want := range []string{"# Prefers tabs", "prefers-tabs", "Use tabs.", "episodic", "mode: 造价"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered memory missing %q: %s", want, out)
		}
	}
}

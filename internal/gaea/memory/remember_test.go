package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestRememberToolSaves drives the tool the way the agent does — raw JSON args —
// and verifies the fact lands in the store and the index.
func TestRememberToolSaves(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	tl := NewRememberTool(store, nil)

	if tl.Name() != "remember" || tl.ReadOnly() {
		t.Fatalf("unexpected tool identity: name=%q readonly=%v", tl.Name(), tl.ReadOnly())
	}
	// Schema must be valid JSON the provider can forward.
	if !json.Valid(tl.Schema()) {
		t.Fatal("remember schema is not valid JSON")
	}

	args := []byte(`{"name":"likes-go","title":"Likes Go","description":"User likes Go","type":"user","body":"Default to Go for backend work."}`)
	out, err := tl.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "已存记忆") {
		t.Fatalf("unexpected tool output: %q", out)
	}

	list := store.List()
	if len(list) != 1 || list[0].Name != "likes-go" || list[0].Type != TypeUser {
		t.Fatalf("memory not saved correctly: %+v", list)
	}
	if list[0].Title != "Likes Go" {
		t.Fatalf("title not persisted through the tool: %q", list[0].Title)
	}
}

// TestRememberToolValidates rejects calls missing required fields rather than
// writing an empty memory.
func TestRememberToolValidates(t *testing.T) {
	tl := NewRememberTool(Store{Dir: t.TempDir()}, nil)
	if _, err := tl.Execute(context.Background(), []byte(`{"description":"d"}`)); err == nil {
		t.Fatal("expected error when body is missing")
	}
	if _, err := tl.Execute(context.Background(), []byte(`{"body":"b"}`)); err == nil {
		t.Fatal("expected error when description is missing")
	}
}

// TestRememberToolQueuesNote verifies a save injects a turn-tail note so the
// fact applies this session, not only the next.
func TestRememberToolQueuesNote(t *testing.T) {
	q := &fakeQueue{}
	ctx := WithQueue(context.Background(), q)
	tl := NewRememberTool(Store{Dir: t.TempDir()}, nil)
	if _, err := tl.Execute(ctx, []byte(`{"name":"uses-rmb","description":"balance is RMB","type":"user","body":"b"}`)); err != nil {
		t.Fatal(err)
	}
	if len(q.notes) != 1 || !strings.Contains(q.notes[0], "uses-rmb") {
		t.Fatalf("expected one queued note naming the saved memory, got %v", q.notes)
	}
}

// TestRememberDescriptionConsentGate 确认制措辞（2026-11-18 众神殿余项；2026-11-25
// 办公工具梳理批描述改中文，锚点同步换）：描述必须同时写明「值得记就主动记」
// 与「确认卡=用户确认才落盘」，并排除瞬时状态与用户明令不存的内容——红线
// 「未经确认不写持久化数据」靠确认卡机制与这句措辞共同成立。
func TestRememberDescriptionConsentGate(t *testing.T) {
	desc := rememberTool{}.Description()
	for _, sub := range []string{
		"值得长期保留的才记",
		"用户明令不存的不要记",
		"弹确认卡，用户确认才落盘",
	} {
		if !strings.Contains(desc, sub) {
			t.Errorf("remember 描述缺确认制锚 %q", sub)
		}
	}
}

// TestRememberMissingFieldEchoesKeys（铜陵反馈 #19）：缺参报错回显本次收到的
// 键面并点名缺的是哪个，UsageHint 给正确骨架——模型不再对着「required」盲撞
// （实测连发 4 次「只有 body 没有 description」）。
func TestRememberMissingFieldEchoesKeys(t *testing.T) {
	tl := NewRememberTool(Store{Dir: t.TempDir()}, nil)
	_, err := tl.Execute(context.Background(), []byte(`{"body":"b","name":"n"}`))
	if err == nil {
		t.Fatal("expected error when description is missing")
	}
	for _, want := range []string{"description", "本次收到键：body, name"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("错误应含 %q，得到：%v", want, err)
		}
	}
	hint := (rememberTool{}).UsageHint(json.RawMessage(`{"body":"b","name":"n"}`))
	if !strings.Contains(hint, "description") || !strings.Contains(hint, "骨架") {
		t.Fatalf("UsageHint 应给骨架并点名 description：%s", hint)
	}
}

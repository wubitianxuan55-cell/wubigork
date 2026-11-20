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
	if !strings.Contains(out, "Saved memory") {
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

// TestRememberDescriptionConsentGate 确认制措辞（2026-11-18 众神殿余项）：
// 全局工具描述与办公/文学身份稿及执行纪律统一为确认制——只读描述的模型
// 不再被引导主动保存（红线「未经确认不写持久化数据」；确认卡机制本就在）。
func TestRememberDescriptionConsentGate(t *testing.T) {
	desc := rememberTool{}.Description()
	for _, sub := range []string{
		"When you learn something worth keeping long-term, call this proactively",
		"Do not log transient state",
	} {
		if !strings.Contains(desc, sub) {
			t.Errorf("remember 描述缺确认制锚 %q", sub)
		}
	}
}

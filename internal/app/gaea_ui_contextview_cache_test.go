package app

// 刀E（v4.249）会话条目缓存：事件日志 append-only，size+mtime 未变 ⇒ 内容
// 未变；追加后必须立即可见（失效语义）。

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestReadEntriesForCacheInvalidation(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := session.LogPathFor(sessionPath)

	appendEvent := func(kind, text string) {
		t.Helper()
		w, err := session.OpenLog(logPath, "", "work")
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(map[string]string{"text": text})
		if _, err := w.AppendRaw(kind, payload); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	appendEvent("turn_started", "")
	appendEvent("user_message", "第一轮")

	e1, err := readEntriesForCached(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	e2, err := readEntriesForCached(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(e1) != 2 || len(e2) != 2 {
		t.Fatalf("首读/二读 = %d/%d, want 2/2", len(e1), len(e2))
	}

	// 追加一回合（size 增长）⇒ 缓存失效，新事件立即可见。
	appendEvent("user_message", "第二轮")
	e3, err := readEntriesForCached(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(e3) != 3 {
		t.Fatalf("追加后 = %d, want 3（缓存失效语义破坏）", len(e3))
	}
}

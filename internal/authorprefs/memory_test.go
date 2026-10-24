package authorprefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openMemory(t *testing.T) (*Memory, string, string) {
	t.Helper()
	ws := t.TempDir()
	book := t.TempDir()
	m, err := Open(ws, book)
	if err != nil {
		t.Fatal(err)
	}
	return m, ws, book
}

func remember(t *testing.T, m *Memory, id, scope, key, kind, text string) Preference {
	t.Helper()
	p, changed, err := m.Apply(Event{
		ID: id, Action: "remember",
		Preference: Preference{
			Key: key, Kind: kind, Scope: scope, Text: text,
			Source: SourceExplicitUser, Importance: 70,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("remember should change the store")
	}
	return p
}

func TestTwoLevelScopesAndBookOverride(t *testing.T) {
	m, ws, book := openMemory(t)
	ap := remember(t, m, "e1", "global", "voice", "voice", "偏爱短句但要有呼吸")
	bp := remember(t, m, "e2", "book", "voice", "voice", "本书用疏朗长句")
	if !strings.HasPrefix(ap.ID, "AP-") || !strings.HasPrefix(bp.ID, "BP-") {
		t.Fatalf("ids = %q / %q", ap.ID, bp.ID)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "author-memory", "memory.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(book, ".gaea", "author-memory", "memory.json")); err != nil {
		t.Fatal(err)
	}
	q, err := m.Query("voice", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 1 || q.Items[0].ID != bp.ID || !strings.Contains(q.Text, "本书用疏朗长句") {
		t.Fatalf("book override query = %+v", q)
	}
	global, _ := m.Query("voice", "global")
	if len(global.Items) != 1 || global.Items[0].ID != ap.ID {
		t.Fatalf("global query = %+v", global)
	}
}

func TestEventReplayIsIdempotent(t *testing.T) {
	m, _, _ := openMemory(t)
	p := remember(t, m, "same-event", "global", "pacing", "pacing", "避免每段同速")
	before, err := m.AP.load()
	if err != nil {
		t.Fatal(err)
	}
	got, changed, err := m.Apply(Event{
		ID: "same-event", Action: "remember",
		Preference: Preference{Kind: "pacing", Scope: "global", Text: "duplicate", Source: SourceExplicitUser},
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed || got.ID != p.ID {
		t.Fatalf("replay changed=%v pref=%+v", changed, got)
	}
	after, _ := m.AP.load()
	if after.Revision != before.Revision || len(after.Preferences) != len(before.Preferences) {
		t.Fatalf("replay mutated snapshot: before=%+v after=%+v", before, after)
	}
}

func TestConflictRequiresDecision(t *testing.T) {
	m, _, _ := openMemory(t)
	first := remember(t, m, "c1", "book", "tone", "tone", "冷峻克制")
	second := remember(t, m, "c2", "book", "tone", "tone", "热烈张扬")
	if second.Status != StatusConflict {
		t.Fatalf("second status = %q, want conflict", second.Status)
	}
	q, _ := m.Query("tone", "")
	if len(q.Items) != 1 || q.Items[0].ID != first.ID {
		t.Fatalf("conflict preferences must not become active: %+v", q)
	}
	accepted, changed, err := m.Apply(Event{ID: "d1", Action: "decide", TargetID: second.ID, Accept: true})
	if err != nil || !changed || accepted.Status != StatusAccepted {
		t.Fatalf("decide = %+v changed=%v err=%v", accepted, changed, err)
	}
	q, _ = m.Query("tone", "")
	if len(q.Items) != 1 || q.Items[0].ID != second.ID {
		t.Fatalf("accepted conflict should supersede old preference: %+v", q)
	}
}

func TestReplaceAndForget(t *testing.T) {
	m, _, _ := openMemory(t)
	p := remember(t, m, "r1", "book", "theme", "theme", "复仇")
	replaced, changed, err := m.Apply(Event{ID: "r2", Action: "replace", TargetID: p.ID, Text: "复仇与救赎"})
	if err != nil || !changed || replaced.Text != "复仇与救赎" || replaced.Revision <= p.Revision {
		t.Fatalf("replace = %+v changed=%v err=%v", replaced, changed, err)
	}
	if _, changed, err := m.Apply(Event{ID: "r3", Action: "forget", TargetID: p.ID}); err != nil || !changed {
		t.Fatalf("forget changed=%v err=%v", changed, err)
	}
	q, _ := m.Query("theme", "")
	if len(q.Items) != 0 {
		t.Fatalf("forgotten preference remained: %+v", q)
	}
}

func TestQueryHardLimitAndOmittedIDs(t *testing.T) {
	m, _, _ := openMemory(t)
	for i := 0; i < 20; i++ {
		remember(t, m, "q-"+string(rune('a'+i)), "global", "", "voice",
			strings.Repeat(string(rune('甲'+i)), 180))
	}
	q, err := m.Query("voice", "")
	if err != nil {
		t.Fatal(err)
	}
	if len([]byte(q.Text)) > QueryByteLimit {
		t.Fatalf("query text = %d bytes, want <= %d", len([]byte(q.Text)), QueryByteLimit)
	}
	if len(q.Omitted) == 0 {
		t.Fatal("query should report omitted IDs")
	}
}

func TestRejectsAutomaticSourceAndMigratesVersionZero(t *testing.T) {
	m, _, _ := openMemory(t)
	if _, _, err := m.Apply(Event{
		ID: "bad", Action: "remember",
		Preference: Preference{Kind: "voice", Scope: "global", Text: "自动推断", Source: "auto"},
	}); err == nil {
		t.Fatal("automatic preference inference should be rejected")
	}
	legacy := `{"version":0,"revision":3,"preferences":[{"id":"AP-old","kind":"voice","scope":"global","text":"旧档偏好","source":"manual","status":"accepted","revision":3}]}`
	if err := os.WriteFile(m.AP.file(), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	remember(t, m, "post-migration", "global", "", "voice", "迁移后仍可写")
	snap, err := m.AP.load()
	if err != nil {
		t.Fatal(err)
	}
	if snap.Version != storeVersion || snap.Revision != 4 || len(snap.Preferences) != 2 {
		t.Fatalf("migrated snapshot = %+v", snap)
	}
}

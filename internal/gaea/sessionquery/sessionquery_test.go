package sessionquery

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

// ─── session_search：过往会话全文检索（v4.384，dsh ⑧ 蒸馏首刀）──────────────

type logLine struct {
	Seq     int64           `json:"seq"`
	Ts      int64           `json:"ts"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func writeSessionLog(t *testing.T, dir, id string, lines []logLine) string {
	t.Helper()
	var b strings.Builder
	for _, l := range lines {
		raw, err := json.Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	path := filepath.Join(dir, id+".gaea-log.jsonl")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func userLine(seq, ts int64, content string) logLine {
	return logLine{Seq: seq, Ts: ts, Kind: "user_message", Payload: mustJSON(map[string]string{"content": content})}
}

func assistantLine(seq, ts int64, text string) logLine {
	return logLine{Seq: seq, Ts: ts, Kind: "assistant_message", Payload: mustJSON(map[string]string{"text": text})}
}

// dbForTest 打开带全量迁移（含 SchemaV24）的独立测试库。
func dbForTest(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return gdb
}

// FTS 路径：ASCII 词命中，命中带 session id/role/snippet；非消息 kind 不入索引。
func TestSearchFindsPastSessions(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "sess-a", []logLine{
		userLine(1, 100, "how do we run the release pipeline?"),
		assistantLine(2, 200, "use the release script with tag vN"),
		{Seq: 3, Ts: 300, Kind: "text", Payload: mustJSON(map[string]string{"text": "streaming noise not indexed"})},
	})
	writeSessionLog(t, dir, "sess-b", []logLine{
		userLine(1, 400, "unrelated chat about coffee"),
	})
	s := NewStore(dbForTest(t), dir)

	hits, err := s.Search("release", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits for 'release', got %d: %+v", len(hits), hits)
	}
	if hits[0].Role != "assistant" || hits[0].SessionID != "sess-a" {
		t.Fatalf("newest-first ordering broken: %+v", hits[0])
	}
	if !strings.Contains(hits[0].Snippet, "release") {
		t.Fatalf("snippet must carry the match, got %q", hits[0].Snippet)
	}
	joined := joinSnippets(hits)
	if strings.Contains(joined, "streaming noise") {
		t.Fatal("non-message kinds must not be indexed")
	}

	hits, err = s.Search("coffee", 0)
	if err != nil || len(hits) != 1 {
		t.Fatalf("coffee should hit only sess-b: %d hits / %v", len(hits), err)
	}
}

// LIKE 中文降级：CJK 子串（长 CJK 连串的中间片段）FTS MATCH 不中、LIKE 命中。
func TestSearchCJKSubstringFallback(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "sess-c", []logLine{
		userLine(1, 100, "帮我把交付物清单整理成工作区文档"),
	})
	s := NewStore(dbForTest(t), dir)

	// 「作区」是「工作区」的中间子串——FTS5 unicode61 把整连串当一个 token，
	// MATCH 不中；LIKE 子串扫描命中。
	hits, err := s.Search("作区", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || !strings.Contains(hits[0].Snippet, "作区") {
		t.Fatalf("CJK substring fallback must match via LIKE, got %+v", hits)
	}
}

// 增量新鲜度：文件追加新内容（size/mtime 变化）后再次 Refresh 可搜到。
func TestRefreshIndexesAppends(t *testing.T) {
	dir := t.TempDir()
	path := writeSessionLog(t, dir, "sess-d", []logLine{
		userLine(1, 100, "first topic alpha"),
	})
	s := NewStore(dbForTest(t), dir)
	if _, err := s.Search("alpha", 0); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	extra := mustJSON(logLine{Seq: 2, Ts: 200, Kind: "user_message", Payload: mustJSON(map[string]string{"content": "second topic beta"})})
	// 与真实 LogWriter 同纪律：每行以换行收尾（torn-tail 修复才会保留本行）。
	extra = append(extra, '\n')
	if _, err := f.Write(append([]byte("\n"), extra...)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	// mtime 精度兜底：显式 bump 2 秒。
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search("beta", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("appended entry must become searchable, got %d hits", len(hits))
	}
}

// 目录隔离：绑定不同会话目录的 Store 互相看不见（空间隔离由目录构造）。
func TestStoresAreIsolatedByDir(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	writeSessionLog(t, dirA, "sess-a", []logLine{
		userLine(1, 100, "secret of workspace A"),
	})
	sa, sb := NewStore(dbForTest(t), dirA), NewStore(dbForTest(t), dirB)
	if hits, err := sa.Search("workspace A", 0); err != nil || len(hits) != 1 {
		t.Fatalf("own dir must hit: %d hits / %v", len(hits), err)
	}
	if hits, err := sb.Search("workspace A", 0); err != nil || len(hits) != 0 {
		t.Fatalf("other dir must not see foreign sessions: %d hits / %v", len(hits), err)
	}
}

// 工具面：结果格式化、空查询拒绝、零命中诚实空消息。
func TestSessionSearchTool(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "sess-e", []logLine{
		assistantLine(1, 100, "the migration plan is documented"),
	})
	tl := NewSearchTool(dbForTest(t), dir)
	if !tl.ReadOnly() {
		t.Fatal("session_search must be read-only")
	}
	out, err := tl.Execute(context.Background(), json.RawMessage(`{"query":"migration"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sess-e") || !strings.Contains(out, "migration plan") {
		t.Fatalf("tool must format session id + snippet, got:\n%s", out)
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"query":"  "}`)); err == nil {
		t.Fatal("empty query must error")
	}
	out, err = tl.Execute(context.Background(), json.RawMessage(`{"query":"zzz-nonexistent"}`))
	if err != nil || !strings.Contains(out, "No past session matches") {
		t.Fatalf("zero hits must degrade to an honest empty message, got %q / %v", out, err)
	}
}

// ── 助手 ─────────────────────────────────────────────────────────────────

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func joinSnippets(hits []Hit) string {
	var b strings.Builder
	for _, h := range hits {
		b.WriteString(h.Snippet)
		b.WriteString("\n")
	}
	return b.String()
}

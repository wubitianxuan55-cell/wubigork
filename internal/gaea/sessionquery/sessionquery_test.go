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
	// 2026-11-27 AI 反馈 #1：query 留空不再是错误——列出最近的会话（标题 / 起止 / 任务）。
	out, err = tl.Execute(context.Background(), json.RawMessage(`{"query":"  "}`))
	if err != nil || !strings.Contains(out, "最近的 1 场会话") || !strings.Contains(out, "sess-e") {
		t.Fatalf("空 query 应列出最近会话，got %q / %v", out, err)
	}
	out, err = tl.Execute(context.Background(), json.RawMessage(`{"query":"zzz-nonexistent"}`))
	if err != nil || !strings.Contains(out, "No past session matches") || !strings.Contains(out, "已索引本工作区 1 个会话") {
		t.Fatalf("零命中必须诚实报空并说明索引规模，got %q / %v", out, err)
	}
}

// liveAssistantLine 实时落盘的助手成稿（event.Message → kind=message）。
func liveAssistantLine(seq, ts int64, text string) logLine {
	return logLine{Seq: seq, Ts: ts, Kind: "message", Payload: mustJSON(map[string]string{"text": text, "reasoning": ""})}
}

// 2026-11-27 AI 反馈 #1 根修：实时会话里助手的回复落盘 kind 是 message——此前只
// 索引 assistant_message，等于只搜得到用户说过的话。
func TestSearchIndexesLiveAssistantMessages(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "20261127-123417.443964200-gaea", []logLine{
		userLine(1, 100, "把包2的测算书做一下"),
		{Seq: 2, Ts: 150, Kind: "reasoning", Payload: mustJSON(map[string]string{"text": "思考碎片不入索引"})},
		{Seq: 3, Ts: 160, Kind: "text", Payload: mustJSON(map[string]string{"text": "流式增量不入索引"})},
		liveAssistantLine(4, 200, "包2测算书已按造价库导入，合计 675 万"),
	})
	s := NewStore(dbForTest(t), dir)
	for _, q := range []string{"造价库导入", "测算书已按"} {
		hits, err := s.Search(q, 0)
		if err != nil || len(hits) != 1 || hits[0].Role != "assistant" {
			t.Fatalf("助手的实时回复应可按 %q 检索到：%d hits / %v", q, len(hits), err)
		}
	}
	// 流式增量与推理不入索引。
	if hits, _ := s.Search("流式增量不入索引", 0); len(hits) != 0 {
		t.Fatal("流式增量 text 不应入索引")
	}
}

// 多关键词三级放宽：同一条消息里各词都出现 → 同一会话的不同消息里分别出现。
func TestSearchMultipleKeywords(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "sess-m1", []logLine{
		userLine(1, 100, "沿江包2的造价库导入怎么做"),
		liveAssistantLine(2, 200, "先取测算样表"),
	})
	writeSessionLog(t, dir, "sess-m2", []logLine{
		userLine(1, 300, "帮我看看沿江包2的模板"),
		liveAssistantLine(2, 400, "模板在 .gaea/exports 下"),
		userLine(3, 500, "造价库导入一直失败"),
		liveAssistantLine(4, 600, "改用载荷文件再导"),
	})
	writeSessionLog(t, dir, "sess-m3", []logLine{
		userLine(1, 700, "只有造价库这一个词"),
	})
	s := NewStore(dbForTest(t), dir)
	// 同一条消息：只有 sess-m1。
	hits, err := s.Search("沿江包2 造价库导入", 0)
	if err != nil || len(hits) != 1 || hits[0].SessionID != "sess-m1" {
		t.Fatalf("同一条消息里各词都出现应最先命中： %+v / %v", hits, err)
	}
	// 不同消息里分别出现：sess-m2 的两条消息各含一部分；sess-m3 缺词不算。
	hits, err = s.Search("沿江包2 载荷文件", 0)
	if err != nil || len(hits) != 2 {
		t.Fatalf("同一会话不同消息里分别出现应命中两条：%+v / %v", hits, err)
	}
	for _, h := range hits {
		if h.SessionID != "sess-m2" || !h.Across {
			t.Fatalf("放宽路应只命中 sess-m2 并带 Across 标记：%+v", h)
		}
	}
	if hits, _ := s.Search("沿江包2 不存在的词", 0); len(hits) != 0 {
		t.Fatal("有词在任何消息里都没出现时不应命中")
	}
}

// 坏行、空行与没写完的半行都不拖垮检索，且不在盘上截 torn tail（正在写的日志
// 被截会损坏写入）。
func TestRefreshToleratesBadLinesAndTornTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sess-t.gaea-log.jsonl")
	good := mustJSON(userLine(1, 100, "完整的一行 alpha"))
	blank := []byte("\n")
	bad := []byte("{not json at all\n")
	later := mustJSON(userLine(2, 200, "坏行之后的内容 beta"))
	torn := []byte(`{"seq":3,"ts":300,"kind":"user_message","payload":{"content":"半截gamma`) // 没有换行收尾
	data := append([]byte{}, good...)
	data = append(data, '\n')
	data = append(data, blank...)
	data = append(data, bad...)
	data = append(data, later...)
	data = append(data, '\n')
	data = append(data, torn...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(dbForTest(t), dir)
	for _, q := range []string{"alpha", "beta"} {
		if hits, err := s.Search(q, 0); err != nil || len(hits) != 1 {
			t.Fatalf("坏行/半行不应拖垮检索（%q）：%d hits / %v", q, len(hits), err)
		}
	}
	if info, err := os.Stat(path); err != nil || info.Size() != int64(len(data)) {
		t.Fatalf("索引不得在盘上截 torn tail：size=%v want %d (%v)", info.Size(), len(data), err)
	}
	if n, _ := s.Skipped(); n != 1 {
		t.Fatalf("有坏行的日志应在结果里注明（skipped=%d）", n)
	}
	out := s.FormatHits("alpha", nil)
	if !strings.Contains(out, "1 份会话日志没能完整读出") {
		t.Fatalf("结果尾部应注明有日志没完整读出：%s", out)
	}
	// 半行补完后再搜得到（增量刷新）。
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`"}}` + "\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	if hits, err := s.Search("gamma", 0); err != nil || len(hits) != 1 {
		t.Fatalf("补完的半行应可检索：%d hits / %v", len(hits), err)
	}
}

// 最近会话清单：标题、起止、首条任务与消息条数，按最后活动时间倒序。
func TestRecentListsSessions(t *testing.T) {
	dir := t.TempDir()
	writeSessionLog(t, dir, "20261126-080000.000000000-gaea", []logLine{
		userLine(1, 1_000, "昨天的第一件事\n换行也算一句"),
		liveAssistantLine(2, 1_100, "好的"),
	})
	writeSessionLog(t, dir, "20261127-210000.000000000-gaea", []logLine{
		userLine(1, 2_000, "刚结束的会话：整理\n  使用情况"),
		liveAssistantLine(2, 2_100, "整理好了"),
		userLine(3, 2_200, "再回顾一下"),
	})
	if err := os.WriteFile(filepath.Join(dir, ".titles.json"), []byte(`{"20261127-210000.000000000-gaea.jsonl":"包2测算复盘"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(dbForTest(t), dir)
	list, err := s.Recent(0)
	if err != nil || len(list) != 2 {
		t.Fatalf("应列出两场会话：%+v / %v", list, err)
	}
	newest, older := list[0], list[1]
	if newest.ID != "20261127-210000.000000000-gaea" || newest.Title != "包2测算复盘" ||
		newest.UserMsgs != 2 || newest.AsstMsgs != 1 || newest.FirstPrompt != "刚结束的会话：整理 使用情况" {
		t.Fatalf("最新会话的概要不对：%+v", newest)
	}
	if older.Title != "" || older.FirstPrompt != "昨天的第一件事 换行也算一句" {
		t.Fatalf("旧会话的概要不对：%+v", older)
	}
	if newest.Started.Format("2006-01-02 15:04") != "2026-11-27 21:00" {
		t.Fatalf("开始时间应取会话 id 的时间戳：%v", newest.Started)
	}
	out := FormatRecent(list)
	for _, want := range []string{"最近的 2 场会话", "包2测算复盘", "用户 2 条 / 助手 1 条", "任务：刚结束的会话：整理 使用情况", "（无标题）"} {
		if !strings.Contains(out, want) {
			t.Errorf("清单应含 %q：\n%s", want, out)
		}
	}
	if _, err := s.Search("", 0); err == nil {
		t.Fatal("Store.Search 仍要求 query 非空（清单走 Recent）")
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

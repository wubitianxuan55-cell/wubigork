package memory

// S1 双空间列落库测试（docs/gaea-space-dimension-design.md §1/§7 S1）：
// facts 写带 space（缺省 work）、按 space 读过滤、迁移前旧行默认 work 可查。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func namesOf(ms []Memory) map[string]bool {
	out := map[string]bool{}
	for _, m := range ms {
		out[m.Name] = true
	}
	return out
}

func TestSQLiteStoreSpaceFilter(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")

	// 缺省写入：Space 零值 → space_id='work'
	if _, err := s.Save(Memory{Name: "fact-work", Description: "d", Body: "b"}); err != nil {
		t.Fatalf("save work: %v", err)
	}
	// 显式 play 写入
	if _, err := s.Save(Memory{Name: "fact-play", Space: "play", Description: "d", Body: "b"}); err != nil {
		t.Fatalf("save play: %v", err)
	}
	// 模拟迁移前旧行：绕过 Save 直接 SQL 插入（不含 space_id 列值 → 列默认 'work'）
	project := slugify(absOf("/Users/me/proj"))
	if _, err := gdb.Exec(`INSERT INTO facts(project, name, title, description, type, kind, tags, body, archived, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		project, "fact-legacy", "旧事实", "d", "project", "semantic", "[]", "b"); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// 不过滤（空 space = 旧行为恒真）：3 条全量
	if got := s.List(); len(got) != 3 {
		t.Fatalf("List() = %d 条, want 3", len(got))
	}
	if got := s.ListInSpace(""); len(got) != 3 {
		t.Fatalf("ListInSpace(\"\") = %d 条, want 3", len(got))
	}
	// work：缺省行 + 旧行
	work := namesOf(s.ListInSpace("work"))
	if len(work) != 2 || !work["fact-work"] || !work["fact-legacy"] {
		t.Fatalf("ListInSpace(work) = %v, want [fact-work fact-legacy]", work)
	}
	// play：仅显式 play 行
	play := namesOf(s.ListInSpace("play"))
	if len(play) != 1 || !play["fact-play"] {
		t.Fatalf("ListInSpace(play) = %v, want [fact-play]", play)
	}
	// 未知空间：空集
	if got := s.ListInSpace("none"); len(got) != 0 {
		t.Fatalf("ListInSpace(none) = %d 条, want 0", len(got))
	}
}

func TestSQLiteStoreSpaceUpsertRehomes(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")

	if _, err := s.Save(Memory{Name: "fact-a", Description: "v1", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(s.ListInSpace("work")); len(got) != 1 || !got["fact-a"] {
		t.Fatalf("first save should land in work: %v", got)
	}
	// 显式 play 重新保存同一 (project, name)：UPSERT 更新内容并改归属空间
	//（唯一键不动，跨空间同名冲突策略留 S1.2）
	if _, err := s.Save(Memory{Name: "fact-a", Space: "play", Description: "v2", Body: "b2"}); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(s.ListInSpace("work")); len(got) != 0 {
		t.Fatalf("upsert to play should remove from work: %v", got)
	}
	play := namesOf(s.ListInSpace("play"))
	if len(play) != 1 || !play["fact-a"] {
		t.Fatalf("upsert to play should land in play: %v", play)
	}
	// 不过滤视图仍恰好 1 条（未新增行）
	if got := s.List(); len(got) != 1 || got[0].Description != "v2" {
		t.Fatalf("List() after upsert = %+v, want single v2 row", got)
	}
}

package app

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestLoadHerdsmanDigitalLife(t *testing.T) {
	path := filepath.Join(t.TempDir(), "life.sqlite3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE characters (id TEXT PRIMARY KEY, data TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE relationships (character_id TEXT, user_id TEXT, data TEXT, updated_at TEXT)`,
		`CREATE TABLE memory_summaries (character_id TEXT, user_id TEXT, data TEXT, updated_at TEXT)`,
		`CREATE TABLE life_timeline_events (id TEXT PRIMARY KEY, character_id TEXT, category TEXT, source TEXT, ref_type TEXT, ref_id TEXT, data TEXT, occurred_at TEXT, created_at TEXT)`,
		`CREATE TABLE world_events (id TEXT PRIMARY KEY, character_id TEXT, type TEXT, data TEXT, created_at TEXT)`,
		`CREATE TABLE life_state_commits (id TEXT)`,
		`CREATE TABLE memory_events (id TEXT)`,
		`CREATE TABLE turn_traces (id TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("建表失败 %q: %v", s, err)
		}
	}
	_, _ = db.Exec(`INSERT INTO characters VALUES ('c1', '{"name":"林晚","gender":"女","identity":"品牌设计师","worldview":"现实都市","model_selection":{"text":"Qwen3.5-35B-A3B-MTP"}}', 'x', '2026-08-13T00:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO relationships VALUES ('c1', 'u1', '{"intimacy":89,"trust":62,"safety":77,"conflict":0,"last_interacted_at":"2026-08-09T00:17:07+08:00"}', 'x')`)
	_, _ = db.Exec(`INSERT INTO memory_summaries VALUES ('c1', 'u1', '{"summary":"关系画像: 稳定互动。","highlights":["h1","h2","h3","h4","h5","h6"],"reinforcement":32,"event_count":12}', '2026-08-13T00:36:05Z')`)
	_, _ = db.Exec(`INSERT INTO life_timeline_events VALUES ('t1','c1','system','character','character','c1','{"title":"character created","summary":"林晚"}','2026-07-30T21:48:04+08:00','x')`)
	_, _ = db.Exec(`INSERT INTO world_events VALUES ('w1','c1','npc_actor','{"title":"妈妈问起近况","detail":"给了她一点现实里的稳定感。"}','2026-07-30T21:51:52+08:00')`)
	_, _ = db.Exec(`INSERT INTO life_state_commits VALUES ('s1')`)
	_, _ = db.Exec(`INSERT INTO memory_events VALUES ('m1')`)
	_, _ = db.Exec(`INSERT INTO turn_traces VALUES ('tt1')`)

	out, err := loadHerdsmanDigitalLife(path)
	if err != nil {
		t.Fatalf("loadHerdsmanDigitalLife: %v", err)
	}
	if !out.Available || out.CharacterCount != 1 || out.TimelineEvents != 1 || out.WorldEvents != 1 ||
		out.StateCommits != 1 || out.MemoryEvents != 1 || out.MemorySummaries != 1 || out.Relationships != 1 || out.TurnTraces != 1 {
		t.Fatalf("计数异常: %+v", out)
	}
	if len(out.Characters) != 1 {
		t.Fatalf("characters = %d", len(out.Characters))
	}
	c := out.Characters[0]
	if c.Name != "林晚" || c.Identity != "品牌设计师" || c.TextModel != "Qwen3.5-35B-A3B-MTP" {
		t.Errorf("角色解析异常: %+v", c)
	}
	if c.Intimacy != 89 || c.Trust != 62 || c.Safety != 77 {
		t.Errorf("关系解析异常: %+v", c)
	}
	if !strings.Contains(c.MemorySummary, "稳定互动") || c.Reinforcement != 32 || c.MemoryEventCount != 12 {
		t.Errorf("记忆摘要解析异常: %+v", c)
	}
	if len(c.Highlights) != 5 {
		t.Errorf("Highlights 应截断为 5 条, got %d", len(c.Highlights))
	}
	if len(out.RecentTimeline) != 1 || out.RecentTimeline[0].Title != "character created" {
		t.Errorf("时间线异常: %+v", out.RecentTimeline)
	}
	if len(out.RecentWorld) != 1 || out.RecentWorld[0].Title != "妈妈问起近况" {
		t.Errorf("世界事件异常: %+v", out.RecentWorld)
	}
	// 审计 P0 AP9-01：全部读取成功时 Warnings 必须为空（契约：空切片或 nil）。
	if len(out.Warnings) != 0 {
		t.Errorf("正常库不应有降级说明: %v", out.Warnings)
	}
}

// newDigitalLifeFixture 造一个「一切正常」的数字生命库（1 角色 + 1 关系 +
// 1 摘要 + 1 时间线 + 1 世界事件），返回库文件路径。AP9-01 的降级用例在此
// 基础上注入故障。
func newDigitalLifeFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "life.sqlite3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE characters (id TEXT PRIMARY KEY, data TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE relationships (character_id TEXT, user_id TEXT, data TEXT, updated_at TEXT)`,
		`CREATE TABLE memory_summaries (character_id TEXT, user_id TEXT, data TEXT, updated_at TEXT)`,
		`CREATE TABLE life_timeline_events (id TEXT PRIMARY KEY, character_id TEXT, category TEXT, source TEXT, ref_type TEXT, ref_id TEXT, data TEXT, occurred_at TEXT, created_at TEXT)`,
		`CREATE TABLE world_events (id TEXT PRIMARY KEY, character_id TEXT, type TEXT, data TEXT, created_at TEXT)`,
		`CREATE TABLE life_state_commits (id TEXT)`,
		`CREATE TABLE memory_events (id TEXT)`,
		`CREATE TABLE turn_traces (id TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("建表失败 %q: %v", s, err)
		}
	}
	exec := func(q string) {
		t.Helper()
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("插入失败 %q: %v", q, err)
		}
	}
	exec(`INSERT INTO characters VALUES ('c1', '{"name":"林晚","gender":"女","identity":"品牌设计师","model_selection":{"text":"m1"}}', 'x', '2026-08-13T00:00:00Z')`)
	exec(`INSERT INTO relationships VALUES ('c1', 'u1', '{"intimacy":89,"trust":62,"safety":77,"conflict":0,"last_interacted_at":"2026-08-09T00:17:07+08:00"}', 'x')`)
	exec(`INSERT INTO memory_summaries VALUES ('c1', 'u1', '{"summary":"关系画像: 稳定互动。","highlights":["h1"],"reinforcement":32,"event_count":12}', '2026-08-13T00:36:05Z')`)
	exec(`INSERT INTO life_timeline_events VALUES ('t1','c1','system','character','character','c1','{"title":"character created","summary":"林晚"}','2026-07-30T21:48:04+08:00','x')`)
	exec(`INSERT INTO world_events VALUES ('w1','c1','npc_actor','{"title":"妈妈问起近况","detail":"给了她一点现实里的稳定感。"}','2026-07-30T21:51:52+08:00')`)
	return path
}

// TestLoadHerdsmanDigitalLife_TableRenamedWarns：表被改名（Herdsman 升级后表名
// 变化）时不得静默显示 0——降级说明必须点出表名与底层错误，其余表照常读出。
func TestLoadHerdsmanDigitalLife_TableRenamedWarns(t *testing.T) {
	path := newDigitalLifeFixture(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE characters RENAME TO characters_v2`); err != nil {
		t.Fatalf("改表名: %v", err)
	}
	_ = db.Close()

	out, err := loadHerdsmanDigitalLife(path)
	if err != nil {
		t.Fatalf("单表缺失应降级而非致命: %v", err)
	}
	if len(out.Warnings) == 0 {
		t.Fatal("表改名必须进 Warnings（此前静默显示「角色 0 个」）")
	}
	w := strings.Join(out.Warnings, " | ")
	if !strings.Contains(w, "读取 characters 表失败") || !strings.Contains(w, "no such table") {
		t.Fatalf("降级说明不含糊（须含表名与底层错误，且来自计数读取而非仅明细）: %s", w)
	}
	if out.CharacterCount != 0 || len(out.Characters) != 0 {
		t.Errorf("characters 读不到时计数/明细应为 0，got count=%d chars=%d", out.CharacterCount, len(out.Characters))
	}
	// 降级而非整体失败：其余表照常读出。
	if out.Relationships != 1 || out.MemorySummaries != 1 || out.TimelineEvents != 1 || out.WorldEvents != 1 {
		t.Errorf("其余表应照常读出: %+v", out)
	}
	if len(out.RecentTimeline) != 1 || len(out.RecentWorld) != 1 {
		t.Errorf("时间线/世界事件应保留: %+v", out)
	}
}

// TestLoadHerdsmanDigitalLife_BadRowJSONWarns：单行 data 是非法 JSON 时该行不再
// 凭空消失——好行保留、坏行进 Warnings（含表名与条数），并回落 slog。
func TestLoadHerdsmanDigitalLife_BadRowJSONWarns(t *testing.T) {
	path := newDigitalLifeFixture(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO characters VALUES ('c2', '{这不是合法 JSON', 'x', '2026-08-14T00:00:00Z')`); err != nil {
		t.Fatalf("插入坏行: %v", err)
	}
	_ = db.Close()

	out, err := loadHerdsmanDigitalLife(path)
	if err != nil {
		t.Fatalf("单行坏数据应降级而非致命: %v", err)
	}
	if out.CharacterCount != 2 {
		t.Errorf("计数是原始 COUNT(*)，应为 2，got %d", out.CharacterCount)
	}
	if len(out.Characters) != 1 || out.Characters[0].Name != "林晚" {
		t.Fatalf("坏行应被跳过、好行保留: %+v", out.Characters)
	}
	w := strings.Join(out.Warnings, " | ")
	if !strings.Contains(w, "characters") || !strings.Contains(w, "1 条记录") || !strings.Contains(w, "JSON") {
		t.Fatalf("单行坏 JSON 的降级说明须含表名/条数/原因: %s", w)
	}
}

// TestLoadHerdsmanDigitalLife_CorruptDB：库文件损坏（连表都读不出来）时返回致命
// error，且 out.Warnings 仍带齐逐表原因（调用方与日志都拿得到现场）。
func TestLoadHerdsmanDigitalLife_CorruptDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "life.sqlite3")
	if err := os.WriteFile(path, []byte("这不是一个 sqlite 数据库，只是一段垃圾文本 garbage garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := loadHerdsmanDigitalLife(path)
	if err == nil {
		t.Fatal("库文件损坏应返回致命错误（不可静默显示全 0）")
	}
	if len(out.Warnings) == 0 {
		t.Fatal("损坏库同样要留下逐表降级说明")
	}
	w := strings.Join(out.Warnings, " | ")
	if !strings.Contains(w, "characters") {
		t.Fatalf("逐表说明须点出核心表 characters: %s", w)
	}
	if !strings.Contains(err.Error(), "不可读") {
		t.Errorf("致命错误文案应说明库不可读: %v", err)
	}
}

// TestDigitalCountReturnsError：单点钉死 digitalCount 不再吞错。
func TestDigitalCountReturnsError(t *testing.T) {
	path := newDigitalLifeFixture(t)
	db, err := openDigitalLifeDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if n, err := digitalCount(db, "characters"); err != nil || n != 1 {
		t.Fatalf("digitalCount(characters) = (%d,%v)，期望 (1,nil)", n, err)
	}
	if _, err := digitalCount(db, "no_such_table"); err == nil {
		t.Fatal("表不存在时 digitalCount 必须返回错误")
	}
}

func TestHerdsmanDigitalLife_Missing(t *testing.T) {
	t.Setenv("HERDSMAN_DATA_DIR", t.TempDir())
	a := &App{}
	out, err := a.HerdsmanDigitalLife()
	if err == nil || out.Available || !strings.Contains(out.Error, "数字生命库不存在") {
		t.Fatalf("缺失应报错: out=%+v err=%v", out, err)
	}
}

const operationsFixture = `[
  {"id":"b2","kind":"image_generate","model":"zimage-turbo","status":"completed","stage":"completed","progress":100,"artifacts":[{"name":"a.png"}],"created_at":"2026-08-13T21:09:23+08:00","completed_at":"2026-08-13T21:11:00+08:00"},
  {"id":"a1","kind":"model_start","model":"qwen3-embedding-4b","status":"completed","stage":"running","progress":100,"artifacts":[],"created_at":"2026-08-13T22:22:35+08:00","completed_at":"2026-08-13T22:22:38+08:00"},
  {"id":"c3","kind":"tts","model":"voxcpm2","status":"running","stage":"running","progress":40,"artifacts":[],"created_at":"2026-08-13T22:30:00+08:00"}
]`

func TestParseHerdsmanOperations(t *testing.T) {
	items, err := parseHerdsmanOperations([]byte(operationsFixture))
	if err != nil {
		t.Fatalf("parseHerdsmanOperations: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d", len(items))
	}
	// 按 created_at 倒序：c3 最新在前。
	if items[0].ID != "c3" || items[1].ID != "a1" || items[2].ID != "b2" {
		t.Fatalf("排序错误: %+v", items)
	}
	if items[0].Kind != "tts" || items[0].Progress != 40 || items[2].Artifacts != 1 {
		t.Fatalf("字段解析异常: %+v", items)
	}
	if _, err := parseHerdsmanOperations([]byte("nope")); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestHerdsmanOperations_FromDisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skill-operations.json"), []byte(operationsFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDSMAN_DATA_DIR", dir)
	a := &App{}
	out, err := a.HerdsmanOperations()
	if err != nil || out.Total != 3 || len(out.Items) != 3 {
		t.Fatalf("HerdsmanOperations: %+v err=%v", out, err)
	}
}

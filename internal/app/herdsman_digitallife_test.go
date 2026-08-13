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

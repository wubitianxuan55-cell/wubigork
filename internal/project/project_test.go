package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateV4Project(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-v4-create")
	defer os.RemoveAll(dir)

	pm, err := Create(dir, "Test Novel", "Fantasy", "Default", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !pm.IsV4() {
		t.Fatal("new project should be v4")
	}

	// 验证 .wubigork/v4 标记文件存在
	if _, err := os.Stat(filepath.Join(dir, ".wubigork", "v4")); os.IsNotExist(err) {
		t.Fatal("v4 marker not created")
	}

	// 验证场景管理器可工作
	sm := pm.SceneManager(1)
	if sm == nil {
		t.Fatal("SceneManager returned nil")
	}

	// 验证快照存储可工作
	ss := pm.SnapshotStore(1)
	if ss == nil {
		t.Fatal("SnapshotStore returned nil")
	}
}

func TestMigrateV3ToV4(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-v3-migrate")
	defer os.RemoveAll(dir)

	// 模拟 v3 项目结构
	chaptersDir := filepath.Join(dir, "chapters")
	os.MkdirAll(chaptersDir, 0755)

	os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{
		"schema_version": 1,
		"title": "Migration Test",
		"genre": "Fantasy",
		"style": "Default",
		"created_at": "2026-01-01T00:00:00Z",
		"last_opened_at": "2026-01-01T00:00:00Z"
	}`), 0644)

	os.WriteFile(filepath.Join(chaptersDir, "001.md"), []byte("# Chapter One\n\nOnce upon a time..."), 0644)
	os.WriteFile(filepath.Join(chaptersDir, "001-summary.json"), []byte(`{"title":"Chapter One","summary":"A beginning"}`), 0644)

	// 打开
	pm, err := Open(dir)
	if err != nil {
		t.Fatalf("Open v3 project failed: %v", err)
	}

	if pm.IsV4() {
		t.Fatal("v3 project should not be v4 before migration")
	}

	// 迁移
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("MigrateV3ToV4 failed: %v", err)
	}

	// 验证 v4 标记
	if !pm.IsV4() {
		t.Fatal("should be v4 after migration")
	}

	// 验证备份
	if _, err := os.Stat(filepath.Join(dir, "_v3_backup", "001.md")); os.IsNotExist(err) {
		t.Fatal("v3 backup not created")
	}

	// 验证场景可读
	content, err := pm.ReadChapterAsStitch(1)
	if err != nil {
		t.Fatalf("ReadChapterAsStitch failed: %v", err)
	}
	if content != "# Chapter One\n\nOnce upon a time..." {
		t.Fatalf("content mismatch: %q", content)
	}

	// 验证旧 API 也工作（fallback）
	oldContent, err := pm.ReadChapter(1)
	if err != nil {
		t.Fatalf("ReadChapter (old API) failed: %v", err)
	}
	if oldContent != "# Chapter One\n\nOnce upon a time..." {
		t.Fatalf("old API content mismatch: %q", oldContent)
	}
}

func TestReadChapterAsStitch_V3Fallback(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-v3-fallback")
	defer os.RemoveAll(dir)

	chaptersDir := filepath.Join(dir, "chapters")
	os.MkdirAll(chaptersDir, 0755)

	os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{
		"schema_version": 1,
		"title": "Fallback Test",
		"genre": "Fantasy",
		"style": "Default",
		"created_at": "2026-01-01T00:00:00Z",
		"last_opened_at": "2026-01-01T00:00:00Z"
	}`), 0644)

	os.WriteFile(filepath.Join(chaptersDir, "001.md"), []byte("V3 content here"), 0644)

	pm, err := Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// 不迁移，直接读（应 fallback 到旧 API）
	content, err := pm.ReadChapterAsStitch(1)
	if err != nil {
		t.Fatalf("ReadChapterAsStitch (v3 fallback) failed: %v", err)
	}
	if content != "V3 content here" {
		t.Fatalf("content mismatch: %q", content)
	}
}

// ── LoadContext ───────────────────────────────────────────

func TestLoadContext_FullAssembly(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-context-full")
	defer os.RemoveAll(dir)

	// 创建完整项目结构
	chaptersDir := filepath.Join(dir, "chapters")
	os.MkdirAll(chaptersDir, 0755)

	os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{
		"schema_version": 1,
		"title": "LoadContext Test",
		"genre": "玄幻",
		"style": "Default",
		"created_at": "2026-01-01T00:00:00Z",
		"last_opened_at": "2026-01-01T00:00:00Z"
	}`), 0644)

	os.WriteFile(filepath.Join(dir, "worldview.json"), []byte(`{"sections":[
		{"id":"era","title":"时代背景","content":"上古时代"},
		{"id":"geo","title":"地理环境","content":"九洲大陆"}
	]}`), 0644)

	os.WriteFile(filepath.Join(dir, "characters.json"), []byte(`{
		"characters": [
			{"id":"mc","name":"主角","status":"alive"},
			{"id":"ant","name":"反派","status":"alive"}
		],
		"organizations": [
			{"id":"sect","name":"青云宗"}
		],
		"relationships": [
			{"from_id":"mc","to_id":"ant","relation_type":"敌对"}
		]
	}`), 0644)

	os.WriteFile(filepath.Join(dir, "outline.json"), []byte(`{
		"story_thread": "少年逆天改命",
		"nodes": [
			{"id":"ch1","title":"第一章","summary":"觉醒","order_index":1,"status":"planned"}
		]
	}`), 0644)

	os.WriteFile(filepath.Join(dir, "foreshadows.json"), []byte(`{"items":[
		{"id":"f1","category":"plot","description":"神秘玉佩","planted_in":"001.md","status":"planted"}
	]}`), 0644)

	pm, err := Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	ctx, err := pm.LoadContext("ch1")
	if err != nil {
		t.Fatalf("LoadContext failed: %v", err)
	}

	// 验证项目元信息
	if ctx.Project.Title != "LoadContext Test" {
		t.Fatalf("Project title: %q", ctx.Project.Title)
	}

	// 验证世界观（markdown 格式）
	if ctx.Worldview == "" {
		t.Fatal("Worldview 不应为空")
	}

	// 验证角色
	if len(ctx.Characters) != 2 {
		t.Fatalf("Characters 数量: %d", len(ctx.Characters))
	}
	if ctx.Characters[0].Name != "主角" {
		t.Fatalf("第1个角色: %q", ctx.Characters[0].Name)
	}

	// 验证组织
	if len(ctx.Organizations) != 1 || ctx.Organizations[0].Name != "青云宗" {
		t.Fatalf("Organizations: %+v", ctx.Organizations)
	}

	// 验证关系
	if len(ctx.Relationships) != 1 || ctx.Relationships[0].RelationType != "敌对" {
		t.Fatalf("Relationships: %+v", ctx.Relationships)
	}

	// 验证大纲
	if len(ctx.Outlines) != 1 || ctx.Outlines[0].Title != "第一章" {
		t.Fatalf("Outlines: %+v", ctx.Outlines)
	}
	if ctx.StoryThread != "少年逆天改命" {
		t.Fatalf("StoryThread: %q", ctx.StoryThread)
	}
	if ctx.CurrentOutline == nil || ctx.CurrentOutline.ID != "ch1" {
		t.Fatal("CurrentOutline 应对应 ch1")
	}

	// 验证伏笔
	if len(ctx.Foreshadows) != 1 || ctx.Foreshadows[0].Description != "神秘玉佩" {
		t.Fatalf("Foreshadows: %+v", ctx.Foreshadows)
	}
}

func TestLoadContext_EmptyProject(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-context-empty")
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	os.WriteFile(filepath.Join(dir, "project.json"), []byte(`{
		"schema_version": 1,
		"title": "Empty Project",
		"genre": "",
		"style": "Default",
		"created_at": "2026-01-01T00:00:00Z",
		"last_opened_at": "2026-01-01T00:00:00Z"
	}`), 0644)

	pm, err := Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// 空项目不应 panic
	ctx, err := pm.LoadContext("")
	if err != nil {
		t.Fatalf("空项目 LoadContext 不应报错: %v", err)
	}
	if ctx == nil {
		t.Fatal("ctx 不应为 nil")
	}

	// 基本字段应存在
	if ctx.Project.Title != "Empty Project" {
		t.Fatalf("Project title: %q", ctx.Project.Title)
	}

	// 空大纲时不应 panic
	if ctx.CurrentOutline != nil {
		t.Fatal("空项目 CurrentOutline 应为 nil")
	}
	if len(ctx.Outlines) != 0 {
		t.Fatalf("空项目 Outlines 应为空: %d", len(ctx.Outlines))
	}
	if len(ctx.Characters) != 0 {
		t.Fatalf("空项目 Characters 应为空: %d", len(ctx.Characters))
	}
}

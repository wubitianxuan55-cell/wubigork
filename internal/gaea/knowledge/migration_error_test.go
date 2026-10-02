package knowledge

// 审计 GA3-13：旧 Markdown 知识库迁移的失败必须可见、可区分。
//   ① 迁移标记读取失败（表缺失/库损坏）≠「标记不存在」，不得当作未迁移重跑；
//   ② filepath.Glob 失败不得被 _ 丢弃（否则迁移静默变成「没条目」）；
//   ③ 迁移失败要并入 Service 状态（MigrationState），启动路径与面板可查。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

const migrationTestEntry = "---\nname: soil-guide\ntitle: 土壤修复指南\ncategory: 经验总结\n---\n土壤修复方案编制要点。\n"

// newLegacyDir 造一个含单条旧 Markdown 条目的目录。
func newLegacyDir(t *testing.T, dir string) string {
	t.Helper()
	legacy := filepath.Join(dir, "legacy")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "soil-guide.md"), []byte(migrationTestEntry), 0o644); err != nil {
		t.Fatal(err)
	}
	return legacy
}

// 标记读取失败（profile 表缺失）必须返回 error 且不迁移任何条目。
func TestMigrateMarkerReadFailureIsNotUnmigrated(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()
	legacy := newLegacyDir(t, dir)

	// 注入真实故障（不是「标记不存在」）：删掉 profile 表 → 标记读取失败。
	if _, err := gdb.Exec("DROP TABLE profile"); err != nil {
		t.Fatalf("drop profile: %v", err)
	}

	n, err := MigrateLegacyKnowledge(gdb, legacy)
	if err == nil {
		t.Fatalf("标记读取失败必须返回 error（不得当作未迁移重跑）: n=%d", n)
	}
	if !strings.Contains(err.Error(), "迁移标记") {
		t.Errorf("错误须点名标记读取失败，实际: %v", err)
	}
	if n != 0 {
		t.Errorf("标记读取失败时不应迁移任何条目，实际 n=%d", n)
	}
	var cnt int
	if qerr := gdb.QueryRow("SELECT COUNT(*) FROM knowledge").Scan(&cnt); qerr != nil {
		t.Fatal(qerr)
	}
	if cnt != 0 {
		t.Errorf("标记读取失败时不得写入条目，实际 %d 条", cnt)
	}
}

// Glob 失败（模式非法）必须返回 error，不得被丢弃成「没条目」。
func TestMigrateGlobFailureReported(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()

	// 目录名带未闭合的 '['：filepath.Glob 返回 ErrBadPattern。
	badDir := filepath.Join(dir, "bad[dir")
	if _, err := filepath.Glob(filepath.Join(badDir, "*.md")); err == nil {
		t.Skip("本平台 filepath.Glob 未对未闭合 '[' 报错，无法注入")
	}
	n, err := MigrateLegacyKnowledge(gdb, badDir)
	if err == nil {
		t.Fatalf("Glob 失败必须返回 error（旧实现被 _ 丢弃）: n=%d", n)
	}
	if !strings.Contains(err.Error(), "扫描旧知识库目录") {
		t.Errorf("错误须点名目录扫描失败，实际: %v", err)
	}
}

// 迁移失败必须并入 Service 状态：Store 仍可用，但 MigrationState 要如实报
// 「失败 + 原因」（启动路径/面板据此提示部分迁移）。
func TestServiceMigrationFailureVisible(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()
	legacy := newLegacyDir(t, dir)
	if _, err := gdb.Exec("DROP TABLE profile"); err != nil {
		t.Fatalf("drop profile: %v", err)
	}

	svc := &Service{}
	svc.mu.Lock()
	st, err := svc.openSQLite(gdb, legacy)
	svc.mu.Unlock()
	if err != nil {
		t.Fatalf("迁移失败不应阻断 Store（知识库仍可用）: %v", err)
	}
	if st == nil {
		t.Fatal("Store 应为可用实例")
	}
	state := svc.MigrationState()
	if !state.Ran {
		t.Error("迁移已尝试，MigrationState.Ran 应为 true")
	}
	if !state.Failed {
		t.Fatalf("迁移失败必须可见，实际 %+v", state)
	}
	if !strings.Contains(state.Reason, "迁移标记") {
		t.Errorf("失败原因须可读且指向标记读取，实际: %q", state.Reason)
	}
}

// 对照：迁移成功时 MigrationState 不得报失败（Ran=true / Failed=false）。
func TestServiceMigrationSuccessState(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()
	legacy := newLegacyDir(t, dir)

	svc := &Service{}
	svc.mu.Lock()
	st, err := svc.openSQLite(gdb, legacy)
	svc.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if state := svc.MigrationState(); !state.Ran || state.Failed || state.Reason != "" {
		t.Fatalf("正常迁移不得报失败: %+v", state)
	}
	if got, gerr := st.Get("soil-guide"); gerr != nil || got.Body == "" {
		t.Errorf("迁移条目应可读: %+v err=%v", got, gerr)
	}
}

// 对照：标记不存在（sql.ErrNoRows）是正常路径，必须继续迁移而不是报错。
func TestMigrateMarkerAbsentProceeds(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()
	legacy := newLegacyDir(t, dir)

	n, err := MigrateLegacyKnowledge(gdb, legacy)
	if err != nil {
		t.Fatalf("标记不存在是正常路径，不应报错: %v", err)
	}
	if n != 1 {
		t.Errorf("应迁移 1 条，实际 %d", n)
	}
}

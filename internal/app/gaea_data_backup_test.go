package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGaeaDataBackupAppFlow 用 GAEA_DATA_ROOT 隔离数据根，验证 Info/Create/Restore/Pending/Cancel。
func TestGaeaDataBackupAppFlow(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)

	// 造一点数据：whisper_data/hermes 结构 + Hephaestus.db（真实 db 单例会写入数据根）
	if err := os.MkdirAll(filepath.Join(dataRoot, "whisper_data", "office"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "whisper_data", "assistants.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "whisper_data", "office", "note.md"), []byte("# 笔记"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	info := a.GaeaDataBackupInfo()
	if info["data_root"] != dataRoot {
		t.Fatalf("data_root 异常: %v", info["data_root"])
	}
	if info["pending"] != false {
		t.Fatalf("初始不应有 pending: %v", info["pending"])
	}

	// 创建备份
	destDir := filepath.Join(t.TempDir(), "bk")
	res, err := a.GaeaDataBackupCreate(destDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	zipPath, _ := res["zip_path"].(string)
	if zipPath == "" || !strings.HasSuffix(zipPath, ".zip") {
		t.Fatalf("zip_path 异常: %v", res)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("备份文件不存在: %v", err)
	}

	// 恢复（两阶段）
	restore, err := a.GaeaDataBackupRestore(zipPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restore["restart_required"] != true {
		t.Fatalf("应要求重启: %v", restore)
	}
	// pending 出现
	p := a.GaeaDataBackupPending()
	if p["pending"] != true {
		t.Fatalf("应有 pending: %v", p)
	}
	info2 := a.GaeaDataBackupInfo()
	if info2["pending"] != true {
		t.Fatalf("Info 应显示 pending: %v", info2)
	}

	// 取消
	if err := a.GaeaDataBackupCancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	p2 := a.GaeaDataBackupPending()
	if p2["pending"] != false {
		t.Fatalf("取消后不应有 pending: %v", p2)
	}
}

// TestGaeaDataBackupRejectsForeignZip 非 gaea 备份应被拒绝。
func TestGaeaDataBackupRejectsForeignZip(t *testing.T) {
	t.Setenv("GAEA_DATA_ROOT", t.TempDir())
	// 造一个非 gaea zip
	foreign := filepath.Join(t.TempDir(), "foreign.zip")
	if err := os.WriteFile(foreign, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDataBackupRestore(foreign); err == nil {
		t.Fatal("应拒绝非法 zip")
	}
}

// TestApplyPendingRestoreHook applyPendingRestore 无 pending 时静默。
func TestApplyPendingRestoreHook(t *testing.T) {
	t.Setenv("GAEA_DATA_ROOT", t.TempDir())
	a := &App{}
	a.applyPendingRestore() // 不应 panic
}

// TestGaeaDataBackupRestoreRejectsExistingPending 验证 #5：已有 pending 时拒绝再次恢复。
func TestGaeaDataBackupRestoreRejectsExistingPending(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	a := &App{}
	// 造一个合法备份 zip
	if err := os.MkdirAll(filepath.Join(dataRoot, "whisper_data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "whisper_data", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := a.GaeaDataBackupCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	zipPath, _ := res["zip_path"].(string)
	// 第一次恢复
	if _, err := a.GaeaDataBackupRestore(zipPath); err != nil {
		t.Fatalf("首次恢复: %v", err)
	}
	// 第二次应拒绝
	if _, err := a.GaeaDataBackupRestore(zipPath); err == nil || !strings.Contains(err.Error(), "已有待应用恢复") {
		t.Fatalf("应有 pending 时拒绝再次恢复: %v", err)
	}
	// 清理
	_ = a.GaeaDataBackupCancel()
}

// TestGaeaDataBackupRollback 验证 #7：回滚到恢复前数据。
func TestGaeaDataBackupRollback(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	a := &App{}
	// 无 before 目录时回滚 no-op
	done, err := a.GaeaDataBackupRollback()
	if err != nil || done {
		t.Fatalf("无 before 时应 no-op: done=%v err=%v", done, err)
	}
	// 构造 before 目录 + 数据根被新数据占据
	before := filepath.Join(dataRoot, ".restore-before")
	if err := os.MkdirAll(before, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(before, "note.txt"), []byte("old-note"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "note.txt"), []byte("new-note"), 0o644); err != nil {
		t.Fatal(err)
	}
	done, err = a.GaeaDataBackupRollback()
	if err != nil || !done {
		t.Fatalf("应回滚: done=%v err=%v", done, err)
	}
	data, _ := os.ReadFile(filepath.Join(dataRoot, "note.txt"))
	if string(data) != "old-note" {
		t.Fatalf("回滚后应为 old-note: %q", data)
	}
}


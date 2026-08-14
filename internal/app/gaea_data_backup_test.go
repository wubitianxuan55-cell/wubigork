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

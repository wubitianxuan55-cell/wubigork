package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 构造一个带 SQLite 库的假数据根（WAL 模式写入后打包，验证 VACUUM INTO 快照一致）。
func setupDataRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// whisper_data 目录 + 普通文件
	if err := os.MkdirAll(filepath.Join(root, "whisper_data", "office"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "whisper_data", "assistants.json"), []byte(`{"assistants":["gaea"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "whisper_data", "office", "note.md"), []byte("# 办公笔记"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 应跳过的文件
	if err := os.WriteFile(filepath.Join(root, "whisper_data", "gaea.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "whisper_data", "Hephaestus.db-wal"), []byte("wal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "whisper_data", "Hephaestus.db-shm"), []byte("shm"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 一个 SQLite 库（先建表写数据，保持 WAL 未合并）
	dbPath := filepath.Join(root, "Hephaestus.db")
	if err := writeSQLite(dbPath, "CREATE TABLE t(id INTEGER PRIMARY KEY, v TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := writeSQLiteRows(dbPath, "INSERT INTO t(v) VALUES (?)", "hello", "world"); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCreateAndExtractRoundtrip(t *testing.T) {
	root := setupDataRoot(t)
	plan := NewPlan(root, []Source{
		{ZipRel: "whisper_data", Abs: filepath.Join(root, "whisper_data")},
		{ZipRel: "Hephaestus.db", Abs: filepath.Join(root, "Hephaestus.db")},
	}, []string{"gaea.log", "-wal", "-shm"})

	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	m, err := plan.Create(zipPath, "2.20.0")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.App != "gaea" || m.Version != "2.20.0" {
		t.Fatalf("manifest 异常: %+v", m)
	}
	if m.EntryCount < 3 {
		t.Fatalf("应至少 3 个条目（assistants.json/note.md/Hephaestus.db），实际 %d", m.EntryCount)
	}

	// 读取 zip 内容验证跳过规则
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	zr.Close()
	if names["manifest.json"] != true {
		t.Error("缺少 manifest.json")
	}
	if names["whisper_data/gaea.log"] {
		t.Error("gaea.log 应被跳过")
	}
	if names["whisper_data/Hephaestus.db-wal"] {
		t.Error("Hephaestus.db-wal 应被跳过（后缀规则）")
	}
	if !names["whisper_data/assistants.json"] {
		t.Error("assistants.json 应存在")
	}

	// 解压到新目录并验证内容（含 SQLite 快照数据完整性）
	dest := t.TempDir()
	if _, err := Extract(zipPath, dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "whisper_data", "assistants.json"))
	if err != nil || !strings.Contains(string(data), "gaea") {
		t.Fatalf("assistants.json 内容异常: %v %q", err, data)
	}
	// SQLite 快照应包含全部行（WAL 已合并）
	count := querySQLiteCount(t, filepath.Join(dest, "Hephaestus.db"))
	if count != 2 {
		t.Fatalf("快照应含 2 行，实际 %d", count)
	}
}

func TestReadManifestRejectsForeign(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "foreign.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("manifest.json")
	w.Write([]byte(`{"app":"other","version":"1.0"}`))
	zw.Close()
	f.Close()

	if _, err := ReadManifest(zipPath); err == nil || !strings.Contains(err.Error(), "不是 gaea 备份") {
		t.Fatalf("应拒绝非 gaea 备份，实际: %v", err)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("manifest.json")
	w.Write([]byte(`{"app":"gaea","version":"1.0"}`))
	w2, _ := zw.Create("../../evil.txt")
	w2.Write([]byte("pwned"))
	zw.Close()
	f.Close()

	dest := t.TempDir()
	if _, err := Extract(zipPath, dest); err == nil || !strings.Contains(err.Error(), "非法") {
		t.Fatalf("应拒绝路径穿越，实际: %v", err)
	}
}

func TestPendingApplyRoundtrip(t *testing.T) {
	// 数据根 A（旧数据）→ 备份 → 数据根 B（新数据）→ 应用恢复 → 内容等于 A
	rootA := setupDataRoot(t)
	planA := NewPlan(rootA, []Source{
		{ZipRel: "whisper_data", Abs: filepath.Join(rootA, "whisper_data")},
		{ZipRel: "Hephaestus.db", Abs: filepath.Join(rootA, "Hephaestus.db")},
	}, []string{"gaea.log", "-wal", "-shm"})
	zipPath := filepath.Join(t.TempDir(), "a.zip")
	if _, err := planA.Create(zipPath, "2.20.0"); err != nil {
		t.Fatal(err)
	}

	// 数据根 B：模拟恢复目标（已有不同数据）
	rootB := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootB, "whisper_data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "whisper_data", "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// staging：解压备份到 staging 目录
	stageDir := filepath.Join(rootB, ".restore-stage-test")
	if _, err := Extract(zipPath, stageDir); err != nil {
		t.Fatal(err)
	}
	// 写 pending
	if err := WritePending(PendingState{
		StageDir:  stageDir,
		ZipName:   "a.zip",
		CreatedAt: time.Now(),
		DataRoot:  rootB,
		HomeDir:   t.TempDir(),
	}); err != nil {
		t.Fatal(err)
	}

	// 应用
	res, err := ApplyPending(rootB, t.TempDir())
	if err != nil {
		t.Fatalf("ApplyPending: %v", err)
	}
	if !res.Applied {
		t.Fatal("应标记 applied")
	}
	if res.BeforeDir == "" {
		t.Fatal("应有恢复前备份目录")
	}
	// 数据根 B 现在等于 A 的内容
	data, err := os.ReadFile(filepath.Join(rootB, "whisper_data", "assistants.json"))
	if err != nil || !strings.Contains(string(data), "gaea") {
		t.Fatalf("恢复后 assistants.json 缺失: %v", err)
	}
	count := querySQLiteCount(t, filepath.Join(rootB, "Hephaestus.db"))
	if count != 2 {
		t.Fatalf("恢复后 SQLite 应 2 行，实际 %d", count)
	}
	// pending 已清
	st, err := ReadPending(rootB)
	if err != nil || st != nil {
		t.Fatalf("pending 应已清理: %v %v", st, err)
	}
	// 旧数据已备份到 before 目录
	if _, err := os.Stat(filepath.Join(res.BeforeDir, "whisper_data", "old.txt")); err != nil {
		t.Fatalf("旧数据应保留在 before 目录: %v", err)
	}
}

func TestApplyPendingNoopWhenNone(t *testing.T) {
	root := t.TempDir()
	res, err := ApplyPending(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("无 pending 时不应 applied")
	}
}

func TestSHA256Stable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte("gaea"), 0o644); err != nil {
		t.Fatal(err)
	}
	s1, err := SHA256(p)
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := SHA256(p)
	if s1 != s2 || len(s1) != 64 {
		t.Fatalf("SHA256 不稳定: %s %s", s1, s2)
	}
}


// TestApplyPendingRetryIdempotent 验证 #1：部分失败后重试可成功且不破坏数据（幂等）。
func TestApplyPendingRetryIdempotent(t *testing.T) {
	rootA := setupDataRoot(t)
	planA := NewPlan(rootA, []Source{
		{ZipRel: "whisper_data", Abs: filepath.Join(rootA, "whisper_data")},
		{ZipRel: "Hephaestus.db", Abs: filepath.Join(rootA, "Hephaestus.db")},
	}, []string{"gaea.log", "-wal", "-shm"})
	zipPath := filepath.Join(t.TempDir(), "a.zip")
	if _, err := planA.Create(zipPath, "2.20.0"); err != nil {
		t.Fatal(err)
	}
	rootB := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootB, "whisper_data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "whisper_data", "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "Hephaestus.db"), []byte("old-hep"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageDir := filepath.Join(rootB, ".restore-stage-test")
	if _, err := Extract(zipPath, stageDir); err != nil {
		t.Fatal(err)
	}
	// 模拟 #1 部分失败残留：Hephaestus.db 已应用（staging 中已删），whisper_data 未应用
	if err := os.Remove(filepath.Join(stageDir, "Hephaestus.db")); err != nil {
		t.Fatal(err)
	}
	if err := WritePending(PendingState{
		StageDir:  stageDir,
		ZipName:   "a.zip",
		CreatedAt: time.Now(),
		DataRoot:  rootB,
	}); err != nil {
		t.Fatal(err)
	}
	res1, err := ApplyPending(rootB, "")
	if err != nil {
		t.Fatalf("ApplyPending: %v", err)
	}
	if !res1.Applied {
		t.Fatal("应标记 applied")
	}
	// 重试幂等：Hephaestus.db 应保持目标已有内容（不被重复搬移），whisper_data 应用新数据
	if data, _ := os.ReadFile(filepath.Join(rootB, "Hephaestus.db")); string(data) != "old-hep" {
		t.Fatalf("Hephaestus.db 应保持目标已有内容（重试幂等），实际 %q", data)
	}
	if _, err := os.Stat(filepath.Join(rootB, "whisper_data", "assistants.json")); err != nil {
		t.Fatalf("whisper_data 应从 staging 应用: %v", err)
	}
	if st, _ := ReadPending(rootB); st != nil {
		t.Fatal("pending 应已清理")
	}
}

// TestApplyPendingHomeConfig 验证 #2：home-config 恢复到 homeDir（此前从未恢复）。
func TestApplyPendingHomeConfig(t *testing.T) {
	rootA := t.TempDir()
	homeA := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeA, ".gaea_config.json"), []byte("{\"key\":\"v1\"}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootA, "note.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	planA := NewPlan(rootA, []Source{
		{ZipRel: "note.txt", Abs: filepath.Join(rootA, "note.txt")},
		{ZipRel: HomeConfigRel, Abs: filepath.Join(homeA, ".gaea_config.json")},
	}, nil)
	zipPath := filepath.Join(t.TempDir(), "a.zip")
	if _, err := planA.Create(zipPath, "2.20.0"); err != nil {
		t.Fatal(err)
	}
	rootB := t.TempDir()
	homeB := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeB, ".gaea_config.json"), []byte("{\"key\":\"old\"}"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageDir := filepath.Join(rootB, ".restore-stage-test")
	if _, err := Extract(zipPath, stageDir); err != nil {
		t.Fatal(err)
	}
	if err := WritePending(PendingState{
		StageDir:  stageDir,
		ZipName:   "a.zip",
		CreatedAt: time.Now(),
		DataRoot:  rootB,
		HomeDir:   homeB,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := ApplyPending(rootB, homeB)
	if err != nil {
		t.Fatalf("ApplyPending: %v", err)
	}
	if !res.Applied {
		t.Fatal("应 applied")
	}
	data, err := os.ReadFile(filepath.Join(homeB, ".gaea_config.json"))
	if err != nil || !strings.Contains(string(data), "v1") {
		t.Fatalf("home 配置应恢复到 v1: %v %q", err, data)
	}
	if _, err := os.Stat(filepath.Join(rootB, "home-config")); err == nil {
		t.Fatal("数据根不应残留 home-config 目录")
	}
	if _, err := os.Stat(filepath.Join(res.BeforeDir, "home-config-.gaea_config.json")); err != nil {
		t.Fatalf("旧 home 配置应备份到 before: %v", err)
	}
}

// TestSafeZipRelRejectsDriveLetter 验证 #8：盘符限定名被拒绝。
func TestSafeZipRelRejectsDriveLetter(t *testing.T) {
	for _, name := range []string{"C:/x", "C:\\x", "c:/x"} {
		if _, err := safeZipRel(name); err == nil {
			t.Errorf("%q 应被拒绝", name)
		}
	}
}


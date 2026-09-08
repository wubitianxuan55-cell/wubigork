package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPruneExpiredLogs 过期清理：超保留期的按日日志删除，近期/legacy/异物保留
func TestPruneExpiredLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	mk := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	old := now.AddDate(0, 0, -400).Format("20060102")
	recent := now.AddDate(0, 0, -3).Format("20060102")
	mk("gaea-" + old + ".log")   // 过期 → 删
	mk("gaea-" + recent + ".log") // 近期 → 留
	mk("gaea-legacy.log")         // 名字解析失败 → 留（历史豁免）
	mk("gaea-legacy.log.bak")     // 异物 → 留
	pruneLogs(dir, now, logKeepDays, logDirCapBytes)
	if _, err := os.Stat(filepath.Join(dir, "gaea-"+old+".log")); !os.IsNotExist(err) {
		t.Fatalf("过期日志未被清理")
	}
	for _, keep := range []string{"gaea-" + recent + ".log", "gaea-legacy.log", "gaea-legacy.log.bak"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Fatalf("%s 不应被清理: %v", keep, err)
		}
	}
}

// TestPruneCapTotalSize 总量兜底：超 cap 从最旧删起，直到达标
func TestPruneCapTotalSize(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// 3 个 1MB 文件，cap=2MB → 最旧的 1 个被删
	data := make([]byte, 1<<20)
	for i := 0; i < 3; i++ {
		name := "gaea-" + now.AddDate(0, 0, -10+i).Format("20060102") + ".log"
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
		// mtime 精度可能不足，显式拉开时间序
		mod := now.AddDate(0, 0, -10+i)
		os.Chtimes(filepath.Join(dir, name), mod, mod)
	}
	pruneLogs(dir, now, logKeepDays, 2<<20)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	total := int64(0)
	count := 0
	for _, e := range entries {
		count++
		if info, err := e.Info(); err == nil {
			total += info.Size()
		}
	}
	if total > 2<<20 {
		t.Fatalf("总量未压到 cap 内: %d", total)
	}
	if count != 2 {
		t.Fatalf("应剩 2 个文件（cap 起效且不误删），实际 %d", count)
	}
}

// TestSetupLoggingDailyFile 端到端小验证：目录建立、今日文件创建、写入落到该文件
func TestSetupLoggingDailyFile(t *testing.T) {
	dataRoot := t.TempDir()
	closeFn, err := setupLogging(dataRoot, "test-0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if _, err := os.Stat(filepath.Join(dataRoot, "logs", logFileNameFor(time.Now()))); err != nil {
		t.Fatalf("今日日志文件未创建: %v", err)
	}
	// 旧日志迁移：造一个 legacy 旧文件再跑一次 setup，应搬进 logs/
	if err := os.MkdirAll(filepath.Join(dataRoot, "whisper_data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "whisper_data", "gaea.log"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	// 前面已 setup 过一次，此处直接调迁移逻辑验证
	migrateLegacyLog(filepath.Join(dataRoot, "whisper_data", "gaea.log"), filepath.Join(dataRoot, "logs", "gaea-legacy.log"))
	if _, err := os.Stat(filepath.Join(dataRoot, "logs", "gaea-legacy.log")); err != nil {
		t.Fatalf("legacy 日志未迁移: %v", err)
	}
}

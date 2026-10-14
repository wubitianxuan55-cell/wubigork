package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Windows：目标文件被无 FILE_SHARE_DELETE 的句柄持有时，覆盖 rename 报
// Access is denied（os.ErrPermission）——AV/索引器瞬态持锁是 ci 假红根源。
func TestRenameWithRetry_TransientLockSucceeds(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Access-denied rename 语义仅 Windows")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "target.json")
	os.WriteFile(dst, []byte("old"), 0o644)
	src := filepath.Join(dir, "src.json")
	os.WriteFile(src, []byte("new"), 0o644)

	// 持锁 5ms 后释放；RenameWithRetry 的重试序列（5/10/20ms）应覆盖到释放后。
	h, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(5 * time.Millisecond)
		h.Close()
	}()

	if err := RenameWithRetry(src, dst); err != nil {
		t.Fatalf("瞬态持锁应经重试成功: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new" {
		t.Fatalf("目标内容应为新数据: %q", got)
	}
}

func TestRenameWithRetry_PersistentLockFailsFastClass(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Access-denied rename 语义仅 Windows")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "target.json")
	os.WriteFile(dst, []byte("old"), 0o644)
	src := filepath.Join(dir, "src.json")
	os.WriteFile(src, []byte("new"), 0o644)

	// 全程持锁：重试穷尽后报错（错误可继续用 errors.Is 判别权限类）
	h, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	start := time.Now()
	err = RenameWithRetry(src, dst)
	if err == nil {
		t.Fatal("持续持锁应失败")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("应保留权限类错误: %v", err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatalf("应经历完整退避序列: %v", time.Since(start))
	}
	// 原数据不被破坏
	got, _ := os.ReadFile(dst)
	if string(got) != "old" {
		t.Fatalf("失败后目标应原样: %q", got)
	}
}

func TestAtomicWrite_UsesRetryPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := AtomicWrite(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("首次写入: %v", err)
	}
	if err := AtomicWrite(path, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("覆盖写入: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != `{"a":2}` {
		t.Fatalf("覆盖结果不符: %q", got)
	}
}

// perm 参数生效（审计上报未动池收口）：CreateTemp 恒 0600，此前 rename 后
// 目标权限恒为 0600、调用方传入的 perm 被静默忽略。收口后目标权限=perm，
// 覆盖写同样带新 perm（editfile 工具「保持原权限位」依赖该路径）。
// 断言仅 POSIX 有意义——Windows 的 Mode() 不模拟权限位（只反映只读属性），
// 但 Windows 下其他用例仍会实际执行 chmod 代码路径。
func TestAtomicWrite_PermTakesEffect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不模拟 POSIX 权限位")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := AtomicWrite(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("写入 0600: %v", err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("目标权限应为 0600: mode=%v err=%v", fi.Mode(), err)
	}
	if err := AtomicWrite(path, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("覆盖写入 0644: %v", err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("覆盖后目标权限应为 0644: mode=%v err=%v", fi.Mode(), err)
	}
}

// AtomicWrite 自带 MkdirAll：父目录缺失时自动创建（幂等、多层），写入成功。
// project/narrative 收敛到本实现后全仓都依赖该性质。
func TestAtomicWrite_CreatesMissingParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c.json")
	if err := AtomicWrite(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("缺失父目录应自动创建: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "data" {
		t.Fatalf("写入结果不符: err=%v data=%q", err, got)
	}
	// 失败不留临时文件（<base>.<rand>.tmp）
	entries, rerr := os.ReadDir(filepath.Join(dir, "a", "b"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 1 || entries[0].Name() != "c.json" {
		for _, e := range entries {
			t.Fatalf("目录含意外条目: %s", e.Name())
		}
	}
}

// TestAtomicWriteSyncSmoke Sync 变体烟测（批 54 A7）：fsync 效果本身不可观测，
// 本用例钉「Sync 代码路径全平台可走通 + 落盘内容正确」，权限断言仅 POSIX
// （Windows 不模拟权限位，但 Sync 分支同样被本用例执行覆盖）。
func TestAtomicWriteSyncSmoke(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.md")
	if err := AtomicWriteSync(path, []byte("# 报告"), 0o600); err != nil {
		t.Fatalf("写入: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "# 报告" {
		t.Fatalf("内容不符: %q err=%v", got, err)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("权限应为 0600: mode=%v err=%v", fi.Mode(), err)
		}
	}
}

// BenchmarkAtomicWrite fsync 换装拍板的量化材料（2026-10-06 断电全零事故后续，
// 批 56 遗留待办「fsync 基准」）：三体量 × 三变体，两条差值读法——
//
//	Plain−Raw = 原子性（temp+rename+chmod+重试）的成本；
//	Sync−Plain = 断电级 durability（rename 前 FlushFileBuffers）的成本。
//
// 体量谱取 gaea 真实写入形态：1KiB=config/清单 meta 类（存量事故面）、
// 64KiB=会话侧车/报告类、1MiB=导出/转录大件。同一路径覆盖写代表真实替换
// 语义；每体量内 Raw→Plain→Sync 相邻跑，磁盘状态近似可比。
// 结论口径（见提交/台账）：差值换算到 gaea 写频（人尺度，日千次量级）判断
// 换装默认是否可负担；本基准不测并发（gaea 单文件写入串行）。
// 已知边界：两变体都只 fsync 临时件、不 fsync 父目录——POSIX 上 rename
// 元数据仍有断电空窗（NTFS $Log 元数据日志默认覆盖 rename）；换装若追求
// 完整 POSIX durability 需追加 dir fsync（Windows 需 BACKUP_SEMANTICS 句柄，
// 成本另测）。
func BenchmarkAtomicWrite(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"1KiB", 1 << 10},
		{"64KiB", 64 << 10},
		{"1MiB", 1 << 20},
	}
	// 确定性载荷（避免全零页对压缩/去重敏感盘的失真）
	payload := make([]byte, sizes[len(sizes)-1].n)
	for i := range payload {
		payload[i] = byte(i*31 + i>>8)
	}
	for _, sz := range sizes {
		for _, variant := range []struct {
			name string
			fn   func(string, []byte, os.FileMode) error
		}{
			{"Raw", func(p string, d []byte, m os.FileMode) error { return os.WriteFile(p, d, m) }},
			{"Plain", AtomicWrite},
			{"Sync", AtomicWriteSync},
		} {
			b.Run(sz.name+"/"+variant.name, func(b *testing.B) {
				path := filepath.Join(b.TempDir(), "bench.bin")
				b.SetBytes(int64(sz.n))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := variant.fn(path, payload[:sz.n], 0o600); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

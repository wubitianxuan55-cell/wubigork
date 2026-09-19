package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// T6-8.2 归档清理绑定：超期归档硬删（溯源审计落盘），活跃事实不受影响。
func TestGaeaMemoryCleanupArchived(t *testing.T) {
	// 隔离 APPDATA/XDG：memoryRetentionDays 缺省读 gaeaLoadConfig，避免读真实配置
	restore := workspaceTestIsolate(t)
	defer restore()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for _, n := range []string{"old-fact", "recent-fact"} {
		if _, err := s.Save(memory.Memory{Name: n, Description: n + " desc", Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(n); err != nil {
			t.Fatal(err)
		}
	}
	// 回拨 old-fact 归档时间到 100 天前（超期）
	ts := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := gdb.Exec(`UPDATE facts SET updated_at=? WHERE name=?`, ts, "old-fact"); err != nil {
		t.Fatal(err)
	}

	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()
	t.Setenv("GAEA_DATA_ROOT", dir)

	a := &App{}
	n, err := a.GaeaMemoryCleanupArchived()
	if err != nil {
		t.Fatalf("CleanupArchived: %v", err)
	}
	if n != 1 {
		t.Fatalf("cleaned = %d, want 1", n)
	}
	// 活跃 List 不受影响（本来就没有活跃事实）；归档只剩 recent-fact
	if arch := s.ListArchived(); len(arch) != 1 || arch[0].Name != "recent-fact" {
		t.Fatalf("archived = %+v, want recent-fact only", arch)
	}
	// 溯源审计落盘
	path := filepath.Join(dir, "memory", "purge-audit.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("purge audit missing: %v", err)
	}
	if !strings.Contains(string(b), "old-fact") || !strings.Contains(string(b), "old-fact desc") {
		t.Fatalf("purge audit lacks provenance: %s", b)
	}
	// 幂等：再次清理返回 0
	n2, err := a.GaeaMemoryCleanupArchived()
	if err != nil || n2 != 0 {
		t.Fatalf("second cleanup: n=%d err=%v, want 0/nil", n2, err)
	}
}

// T6-8.2 归档分页绑定：总量 + 分页条目 + 越界空页。
func TestGaeaMemoryArchivedListPaged(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 3; i++ {
		name := "fact-" + string(rune('a'+i))
		if _, err := s.Save(memory.Memory{Name: name, Description: name, Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(name); err != nil {
			t.Fatal(err)
		}
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	page, err := a.GaeaMemoryArchivedList(2, 0)
	if err != nil {
		t.Fatalf("ArchivedList: %v", err)
	}
	if page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("page: %d items, total %d, want 2/3", len(page.Items), page.Total)
	}
	if page.Items[0].Name == "" || page.Items[0].ArchivedAt == "" {
		t.Fatalf("item view incomplete: %+v", page.Items[0])
	}

	page2, err := a.GaeaMemoryArchivedList(2, 4)
	if err != nil || len(page2.Items) != 0 || page2.Total != 3 {
		t.Fatalf("beyond page: %+v err=%v", page2, err)
	}
}

// 记忆统一层第一刀：归档分页绑定携带保留期天数（前端「归档保留 N 天」文案）。
func TestGaeaMemoryArchivedList_RetentionDays(t *testing.T) {
	// 隔离 APPDATA/XDG：断言值 90 依赖缺省配置，避免被真实/污染配置干扰
	restore := workspaceTestIsolate(t)
	defer restore()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	page, err := a.GaeaMemoryArchivedList(10, 0)
	if err != nil {
		t.Fatalf("ArchivedList: %v", err)
	}
	if page.RetentionDays != 90 {
		t.Fatalf("RetentionDays = %d, want 90（与 memory.ArchivedRetention 一致）", page.RetentionDays)
	}
}

// 记忆统一层第一刀：GaeaMemoryUnarchive 恢复归档事实回活跃列表；
// 未归档/不存在返回错误；恢复后归档列表减一。
func TestGaeaMemoryUnarchive(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(memory.Memory{Name: "fact-a", Description: "d", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Archive("fact-a"); err != nil {
		t.Fatal(err)
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	if err := a.GaeaMemoryUnarchive("fact-a"); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if list := s.List(); len(list) != 1 || list[0].Name != "fact-a" {
		t.Fatalf("恢复后活跃列表: %+v", list)
	}
	if arch := s.ListArchived(); len(arch) != 0 {
		t.Fatalf("恢复后归档应为空: %+v", arch)
	}

	// 未归档（活跃事实）恢复报错
	if err := a.GaeaMemoryUnarchive("fact-a"); err == nil {
		t.Fatal("活跃事实的恢复应报错")
	}
	// 不存在的事实恢复报错
	if err := a.GaeaMemoryUnarchive("nope"); err == nil {
		t.Fatal("不存在事实的恢复应报错")
	}
}

// 记忆统一层第二刀：GaeaMemoryUnarchiveBatch 批量恢复——全部成功返回计数；
// 部分失败跳过并聚合错误，成功数正确。
func TestGaeaMemoryUnarchiveBatch(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for _, n := range []string{"fact-a", "fact-b", "fact-c"} {
		if _, err := s.Save(memory.Memory{Name: n, Description: n, Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(n); err != nil {
			t.Fatal(err)
		}
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	// 全部成功
	n, err := a.GaeaMemoryUnarchiveBatch([]string{"fact-a", "fact-b", "fact-c"})
	if err != nil {
		t.Fatalf("全成功批量恢复不应报错: %v", err)
	}
	if n != 3 {
		t.Fatalf("恢复计数 = %d, want 3", n)
	}
	if list := s.List(); len(list) != 3 {
		t.Fatalf("恢复后活跃列表 = %d, want 3", len(list))
	}

	// 部分失败：2 个活跃（再恢复会错）+ 1 个已删（不存在）→ 全部失败
	n2, err2 := a.GaeaMemoryUnarchiveBatch([]string{"fact-a", "fact-b", "nope"})
	if err2 == nil {
		t.Fatal("部分失败应返回聚合错误")
	}
	if n2 != 0 {
		t.Fatalf("全部失败计数 = %d, want 0", n2)
	}

	// 混合：重新归档 fact-a，与不存在的事实混合 → 成功 1 / 失败 1
	if _, err := s.Archive("fact-a"); err != nil {
		t.Fatal(err)
	}
	n3, err3 := a.GaeaMemoryUnarchiveBatch([]string{"fact-a", "nope"})
	if err3 == nil {
		t.Fatal("混合批量应返回聚合错误")
	}
	if n3 != 1 {
		t.Fatalf("混合批量成功计数 = %d, want 1", n3)
	}

	// 空列表 no-op
	n4, err4 := a.GaeaMemoryUnarchiveBatch(nil)
	if err4 != nil || n4 != 0 {
		t.Fatalf("空列表: n=%d err=%v, want 0/nil", n4, err4)
	}
}

// 记忆统一层第二刀：memoryRetentionDays 读配置、钳制、回退默认。
// 使用 workspaceTestIsolate 隔离 APPDATA/XDG，避免读取真实用户配置。
func TestMemoryRetentionDays(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()

	ga.cfg = nil
	ga.ctrl = nil
	// 缺省（配置未初始化）：回退 90
	if d := memoryRetentionDays(); d != 90 {
		t.Fatalf("缺省保留期 = %d, want 90", d)
	}
	// 配置生效
	ga.cfg = &gaeaConfig.Config{Memory: gaeaConfig.MemoryConfig{ArchivedRetentionDays: 30}}
	if d := memoryRetentionDays(); d != 30 {
		t.Fatalf("配置保留期 = %d, want 30", d)
	}
	// 钳制上限
	ga.cfg.Memory.ArchivedRetentionDays = 9999
	if d := memoryRetentionDays(); d != 730 {
		t.Fatalf("钳制上限 = %d, want 730", d)
	}
	// 钳制下限
	ga.cfg.Memory.ArchivedRetentionDays = 0
	if d := memoryRetentionDays(); d != 90 {
		t.Fatalf("0 值回退 = %d, want 90", d)
	}
}

// 记忆统一层第二刀：GaeaMemorySetRetentionDays 钳制 + 配置写入。
// 使用 workspaceTestIsolate 隔离 APPDATA——gaeaConfig.Save 落盘到临时目录，
// 绝不触碰真实用户配置（此前一版测试直接污染了真实 config.toml，已修复）。
func TestGaeaMemorySetRetentionDays(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()
	ga.cfg = &gaeaConfig.Config{Workspace: t.TempDir()}
	ga.ctrl = nil

	a := &App{}
	// 正常设置
	if err := a.GaeaMemorySetRetentionDays(45); err != nil {
		t.Fatalf("SetRetentionDays(45): %v", err)
	}
	if d := ga.cfg.Memory.ArchivedRetentionDays; d != 45 {
		t.Fatalf("设置后 = %d, want 45", d)
	}
	// 钳制下限（<1 → 1）
	if err := a.GaeaMemorySetRetentionDays(0); err != nil {
		t.Fatalf("SetRetentionDays(0): %v", err)
	}
	if d := ga.cfg.Memory.ArchivedRetentionDays; d != 1 {
		t.Fatalf("0 钳制后 = %d, want 1", d)
	}
	// 钳制上限（>730 → 730）
	if err := a.GaeaMemorySetRetentionDays(9999); err != nil {
		t.Fatalf("SetRetentionDays(9999): %v", err)
	}
	if d := ga.cfg.Memory.ArchivedRetentionDays; d != 730 {
		t.Fatalf("9999 钳制后 = %d, want 730", d)
	}
	// 落盘到隔离 APPDATA：重载（模拟下次启动）应读到 730
	got, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("重载隔离配置: %v", err)
	}
	if got.Memory.ArchivedRetentionDays != 730 {
		t.Fatalf("隔离落盘重载 = %d, want 730", got.Memory.ArchivedRetentionDays)
	}
}

// TestGaeaMemorySetRetentionDaysUninitialized 死锁回归（2026-09-19 后端审计）：
// 引擎未初始化（ga.cfg==nil）时设置保留期，须先在锁外 GaeaInit 再加锁处理——
// 此前持 ga.mu 调 GaeaInit（内部首行再取 ga.mu）sync.Mutex 不可重入自死锁，
// 绑定 goroutine 永久卡死且持有 ga.mu，办公板块全部消费方连锁冻结。超时分支
// 照 gaea_init_deadlock_test 先例直接退出二进制（defer 恢复函数会再次挂起）。
func TestGaeaMemorySetRetentionDaysUninitialized(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	bridge.SetClient(ai.NewClient(config.Load()))

	// 种子配置：GaeaInit 完整成功（引擎拉起后设置才应生效）。
	seed := gaeaConfig.Default()
	seed.DefaultModel = "gaea"
	seed.Workspace = t.TempDir()
	seed.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "gaea",
		Kind:          "wubigrok",
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	seed.Tools.Enabled = nil
	seed.Sandbox.Bash = "off"
	if err := gaeaConfig.Save(seed); err != nil {
		t.Fatalf("种子配置保存失败: %v", err)
	}

	a := &App{core: &core{
		ctx:    context.Background(),
		cfg:    config.Load(),
		client: ai.NewClient(config.Load()),
	}}

	// 快照全局 gaea 运行时（同包测试互不污染）；超时分支不走 defer——
	// 修复前本用例挂在 ga.mu 重入，defer 的 ga.mu.Lock() 会再次挂起进程。
	ga.mu.Lock()
	oldCtrl, oldCfg := ga.ctrl, ga.cfg
	ga.ctrl, ga.cfg = nil, nil
	ga.mu.Unlock()

	done := make(chan error, 1)
	go func() { done <- a.GaeaMemorySetRetentionDays(30) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("未初始化时设置保留期: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Log("SetRetentionDays 超时：持 ga.mu 调 GaeaInit 自死锁（回归）")
		os.Exit(1)
	}
	defer func() {
		ga.mu.Lock()
		defer ga.mu.Unlock()
		if ga.ctrl != nil && ga.ctrl != oldCtrl {
			ga.ctrl.Close()
		}
		ga.ctrl, ga.cfg = oldCtrl, oldCfg
		_ = db.CloseDatabase(gaeaConfig.MemoryUserDir())
	}()

	ga.mu.Lock()
	d := ga.cfg.Memory.ArchivedRetentionDays
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if d != 30 {
		t.Fatalf("设置后保留期 = %d, want 30", d)
	}
	if ctrl == nil {
		t.Fatal("未初始化路径应先在锁外拉起引擎（ctrl 非 nil）")
	}
}

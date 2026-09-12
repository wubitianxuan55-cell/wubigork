package app

import (
	"testing"

	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// 注入体检绑定测试（市场调研候选2）：真实库 + 真实构建器 + 真实门控读取的
// 接线冒烟——全开/总关两态。不变量的逐形态覆盖在 memory 包纯核测试。

// evalTestSetup 隔离临时库 + 覆盖办公记忆 store + 钉死门控（ga.cfg 保存恢复，
// 防泄漏到其他测试）。
func evalTestSetup(t *testing.T) memory.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	store := memory.SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	SetOfficeStoreForTest(store)
	t.Cleanup(ResetOfficeStoreForTest)

	oldCfg := ga.cfg
	ga.mu.Lock()
	ga.cfg = &config.Config{Memory: config.MemoryConfig{Enabled: true}} // 总开关开/空间 on/两注入门开（Get* 缺省 true）
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})
	return store
}

func TestGaeaMemoryEvalRunWiring(t *testing.T) {
	store := evalTestSetup(t)

	// 种库：work 固化一条 + work 普通一条 + play 一条（跨空间不得泄）。
	if _, err := store.Save(memory.Memory{
		Name: "eval-pinned", Title: "固化规范", Description: "综合单价按 2026 定额",
		Type: memory.TypeProject, Kind: memory.KindSemantic, Space: "work",
	}); err != nil {
		t.Fatalf("save pinned: %v", err)
	}
	if err := store.Pin("eval-pinned"); err != nil {
		t.Fatalf("pin: %v", err)
	}
	if _, err := store.Save(memory.Memory{
		Name: "eval-fresh", Title: "新鲜事实", Description: "昨天刚沉淀",
		Type: memory.TypeUser, Kind: memory.KindEpisodic, Space: "work",
	}); err != nil {
		t.Fatalf("save fresh: %v", err)
	}
	if _, err := store.Save(memory.Memory{
		Name: "play-note", Title: "闲庭记忆", Description: "只属于 play",
		Type: memory.TypeUser, Kind: memory.KindEpisodic, Space: "play",
	}); err != nil {
		t.Fatalf("save play: %v", err)
	}

	a := &App{core: &core{cfg: &appconfig.Config{MorningPreload: true, ProjectBrief: true}}}
	rep, err := a.GaeaMemoryEvalRun()
	if err != nil {
		t.Fatalf("GaeaMemoryEvalRun: %v", err)
	}
	if !rep.Passed || len(rep.Violations) != 0 {
		t.Fatalf("接线全合规场景应 Passed，violations=%v", rep.Violations)
	}
	if !rep.MemoryEnabled || !rep.MorningPreload || !rep.ProjectBrief || !rep.SpaceModeOn {
		t.Errorf("门控全开应透出 true，got %+v", rep.MemoryEvalGates)
	}
	if !rep.PreloadPresent || !rep.BriefPresent {
		t.Fatal("两注入块都应 Present")
	}
	// work 视图只含 work 两条；play-note 不得进任何块（跨空间零泄漏的接线实证）。
	if rep.EntryCount != 2 {
		t.Errorf("EntryCount = %d, want 2（仅 work 两条）", rep.EntryCount)
	}
	if rep.PinnedTotal != 1 || rep.PinnedInBrief != 1 {
		t.Errorf("固化覆盖 = (%d,%d), want (1,1)", rep.PinnedTotal, rep.PinnedInBrief)
	}
}

func TestGaeaMemoryEvalRunDisabledNotes(t *testing.T) {
	evalTestSetup(t)
	// 总开关关闭：块恒空、Passed 结构合规、Note 注明注入未发生。
	ga.mu.Lock()
	ga.cfg = &config.Config{Memory: config.MemoryConfig{Enabled: false}}
	ga.mu.Unlock()

	a := &App{core: &core{cfg: &appconfig.Config{MorningPreload: true, ProjectBrief: true}}}
	rep, err := a.GaeaMemoryEvalRun()
	if err != nil {
		t.Fatalf("GaeaMemoryEvalRun: %v", err)
	}
	if !rep.Passed {
		t.Errorf("关态结构体检应 Passed，violations=%v", rep.Violations)
	}
	if rep.MemoryEnabled || rep.PreloadPresent || rep.BriefPresent {
		t.Errorf("关态下 Present/开关应全 false，got %+v", rep)
	}
	if rep.Note == "" {
		t.Fatal("关态必须带 Note 说明注入未发生")
	}
}

package app

// GaeaMemory 管理面直连取数测试（v4.244 观察池清账）：控制器未构建（新起壳
// 形态）时 facts/available 仍从 hubOfficeStore 读出——修复「面板误报不可用而
// 同库生命周期/体检读得到」的口径分裂（v4.243 真机走查实锤）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestGaeaMemoryFactsFromStoreWithoutController(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	store := memory.SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	SetOfficeStoreForTest(store)
	t.Cleanup(ResetOfficeStoreForTest)
	if _, err := store.Save(memory.Memory{
		Name: "panel-fact", Title: "面板事实", Description: "描述",
		Type: memory.TypeUser, Kind: memory.KindSemantic,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 裸 App：ga.ctrl 为 nil=控制器未构建（新起壳形态），旧实现在此返回空视图
	// available=false（面板误报不可用）。
	a := &App{}
	v := a.GaeaMemory()
	if !v.Available {
		t.Fatal("store 可读即应 available=true（不再等控制器）")
	}
	if len(v.Facts) != 1 || v.Facts[0].Name != "panel-fact" || v.Facts[0].Title != "面板事实" {
		t.Fatalf("facts 应从 store 直读: %+v", v.Facts)
	}
	if !v.Enabled {
		t.Fatal("memoryEnabled 缺省应 true")
	}
}

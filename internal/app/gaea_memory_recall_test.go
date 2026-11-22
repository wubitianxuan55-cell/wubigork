package app

import (
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// GaeaMemoryRecall（星云召回）app 级实弹：词法臂命中 + 置顶恒进 + 失效排除
// + 语义臂无 embedder 降级如实标注 + 图谱臂 [[互引]] 邻居扩散。
func TestGaeaMemoryRecallArms(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	// 与 hubOfficeStore 同库（MemoryUserDir）：office 覆盖、事件日志、图物化
	// 三面同句柄，图谱臂才能看见 [[互引]] 邻居。
	gdb := db.GetDatabase(gaeaConfig.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(gaeaConfig.MemoryUserDir())

	s := memory.SQLiteStoreFor(gdb, gaeaConfig.MemoryUserDir(), "/Users/me/proj")
	save := func(m memory.Memory) {
		t.Helper()
		if _, err := s.Save(m); err != nil {
			t.Fatal(err)
		}
	}
	save(memory.Memory{Name: "vulkan-raster-pass", Title: "Vulkan 光栅化崩溃", Description: "Vulkan 光栅化管线崩溃复现与修复步骤", Body: "Vulkan 管线在多纹理提交时崩溃，禁用 fine-grained 锁修复，配套 [[pipeline-lock-policy]]"})
	save(memory.Memory{Name: "pipeline-lock-policy", Title: "并发锁粒度策略", Description: "多纹理并发提交的锁粒度约束", Body: "并发提交走粗粒度锁，禁用 fine-grained 路径"})
	save(memory.Memory{Name: "vulkan-notes", Title: "Vulkan 笔记", Description: "光栅化相关笔记", Body: "详见 [[vulkan-raster-pass]] 的复现步骤"})
	save(memory.Memory{Name: "pinned-policy", Title: "发布纪律", Description: "发版门与冒烟口径", Body: "发布前置冒烟 200 校验"})
	save(memory.Memory{Name: "stale-fact", Title: "已推翻旧口径", Description: "vulkan 旧口径", Body: "vulkan 旧步骤已废弃"})
	if err := s.Invalidate("stale-fact", "被新事实推翻"); err != nil {
		t.Fatal(err)
	}
	if err := s.Pin("pinned-policy"); err != nil {
		t.Fatal(err)
	}

	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	// core 非零值构造（app 测试语义管线地雷：nil core 在 engineMgr 访问即崩）。
	a := &App{core: &core{}}
	view := a.GaeaMemoryRecall("vulkan", 8)
	if view.Query != "vulkan" {
		t.Fatalf("Query=%q", view.Query)
	}
	byName := map[string]MemoryRecallHit{}
	for _, h := range view.Hits {
		byName[h.Name] = h
	}
	// 词法臂：vulkan-raster-pass 命中且带 lexical 归因。
	hit, ok := byName["vulkan-raster-pass"]
	if !ok {
		t.Fatalf("词法臂未命中 vulkan-raster-pass：hits=%v", view.Hits)
	}
	if !hasArm(hit.Arms, "lexical") {
		t.Fatalf("arms=%v 缺 lexical", hit.Arms)
	}
	if hit.Score <= 0 || hit.Score > 1 {
		t.Fatalf("score=%v 越界", hit.Score)
	}
	// 图谱臂（发现语义）：pipeline-lock-policy 本体与 "vulkan" 零词面重叠，
	// 只经 vulkan-raster-pass 的 [[互引]] 扩散可达——命中即图谱臂实证。
	if lock, ok := byName["pipeline-lock-policy"]; ok {
		if !hasArm(lock.Arms, "graph") {
			t.Fatalf("pipeline-lock-policy arms=%v 缺 graph（互引扩散未生效）", lock.Arms)
		}
		if hasArm(lock.Arms, "lexical") {
			t.Fatalf("pipeline-lock-policy 不应有词法命中：arms=%v", lock.Arms)
		}
	} else {
		t.Fatalf("图谱臂未扩散到 pipeline-lock-policy：hits=%v", view.Hits)
	}
	// 置顶臂：pinned-policy 无词法/语义命中也恒进结果。
	pin, ok := byName["pinned-policy"]
	if !ok {
		t.Fatalf("置顶条未恒进结果：hits=%v", view.Hits)
	}
	if !hasArm(pin.Arms, "pinned") || !pin.Pinned {
		t.Fatalf("pinned arms=%v pinned=%v", pin.Arms, pin.Pinned)
	}
	// 失效条目排除（注入面同口径）。
	if _, ok := byName["stale-fact"]; ok {
		t.Fatal("失效条目不应出现在星云召回")
	}
	// 语义臂降级：测试无 embedder，Degraded 如实标注。
	if _, ok := view.Degraded["semantic"]; !ok {
		t.Fatalf("Degraded=%v 缺 semantic 降级标注", view.Degraded)
	}
	for _, h := range view.Hits {
		for _, arm := range h.Arms {
			if arm == "semantic" {
				t.Fatalf("无 embedder 却出现 semantic 臂命中：%s", h.Name)
			}
		}
	}
}

// 空查询/极限截断：不炸、limit 生效。
func TestGaeaMemoryRecallLimits(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	gdb := db.GetDatabase(gaeaConfig.MemoryUserDir())
	defer db.CloseDatabase(gaeaConfig.MemoryUserDir())
	s := memory.SQLiteStoreFor(gdb, gaeaConfig.MemoryUserDir(), "/Users/me/proj")
	for i := 0; i < 12; i++ {
		if _, err := s.Save(memory.Memory{Name: string(rune('a'+i)) + "-fact", Description: "common desc", Body: "common body"}); err != nil {
			t.Fatal(err)
		}
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()
	a := &App{core: &core{}}
	if v := a.GaeaMemoryRecall("", 8); v.Total != 0 {
		t.Fatalf("空查询 Total=%d", v.Total)
	}
	v := a.GaeaMemoryRecall("common", 5)
	if v.Total > 5 || len(v.Hits) > 5 {
		t.Fatalf("limit 未生效：total=%d len=%d", v.Total, len(v.Hits))
	}
	if !strings.Contains(v.Query, "common") {
		t.Fatalf("Query=%q", v.Query)
	}
}

func hasArm(arms []string, want string) bool {
	for _, a := range arms {
		if a == want {
			return true
		}
	}
	return false
}

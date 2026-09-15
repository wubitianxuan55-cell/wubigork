// Engine 覆盖层测试（规格 进度计划/gaea-prompt-workshop-t6-20260916.md §3.4）：
// SetOverride 优先级（覆盖命中 > 内置 / 回调 nil 回落 / 清除回落）、Names 字典序
// 稳定、三级解析 e2e（覆盖 → 磁盘 → embed，embedded fs 用 fstest.MapFS，与既有
// prompt_test.go 的 embed/disk 优先级回归同一手法）。
package prompt

import (
	"strings"
	"testing"
	"testing/fstest"
)

// 覆盖模板构造辅助（三处用例共用最小形状）。
func overrideTemplate(name, system string) *Template {
	return &Template{
		Name:   name,
		System: system,
		Task:   "覆盖任务",
	}
}

// TestSetOverridePriority 覆盖解析优先级三态：
//  1. 回调命中 → 覆盖模板优先于内置（磁盘/embed 加载结果）；
//  2. 同一回调对其他键返回 nil → 回落内置 map；
//  3. SetOverride(nil) 清除 → 完全回落内置。
func TestSetOverridePriority(t *testing.T) {
	// dir 指向不存在目录 + embed 兜底：内置表完全来自 MapFS（不依赖仓库磁盘，
	// 用例自包含；磁盘叠加场景见 TestEngineThreeLevelResolution）。
	embedded := fstest.MapFS{
		"prompts/create-chapter.json": &fstest.MapFile{
			Data: []byte(`{"name":"create-chapter","system":"embedded-sys","task":"embedded-task",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
		"prompts/rewrite-chapter.json": &fstest.MapFile{
			Data: []byte(`{"name":"rewrite-chapter","system":"embedded-rw","task":"embedded-task",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
	}
	eng := NewEngineWithEmbedded("../../prompts-does-not-exist", embedded)

	// 前置：未设覆盖时 Get 走内置表。
	if got := eng.Get("create-chapter"); got == nil || got.System != "embedded-sys" {
		t.Fatalf("未设覆盖应回落内置模板: %+v", got)
	}

	// 1. 覆盖命中 > 内置。
	eng.SetOverride(func(name string) *Template {
		if name == "create-chapter" {
			return overrideTemplate(name, "OVERRIDE-SYS")
		}
		return nil
	})
	got := eng.Get("create-chapter")
	if got == nil || got.System != "OVERRIDE-SYS" || got.Task != "覆盖任务" {
		t.Fatalf("覆盖命中应优先于内置: %+v", got)
	}
	// 2. 回调对未覆盖键返回 nil → 回落内置。
	if got := eng.Get("rewrite-chapter"); got == nil || got.System != "embedded-rw" {
		t.Fatalf("回调返回 nil 应回落内置模板: %+v", got)
	}
	// 3. 清除覆盖 → 完全回落内置。
	eng.SetOverride(nil)
	if got := eng.Get("create-chapter"); got == nil || got.System != "embedded-sys" {
		t.Fatalf("SetOverride(nil) 后应回落内置模板: %+v", got)
	}
}

// TestEngineThreeLevelResolution 三级解析 e2e（规格 §1「全局覆盖 → 磁盘 JSON →
// embed JSON」）：磁盘 > embed 的两级叠加沿用既有行为（prompt_test.go
// TestEngineEmbeddedDiskOverrides 回归），本用例在其上验证覆盖层为第一级——
// 覆盖命中 > 磁盘 > embed，覆盖未命中回落磁盘。
func TestEngineThreeLevelResolution(t *testing.T) {
	embedded := fstest.MapFS{
		"prompts/create-chapter.json": &fstest.MapFile{
			Data: []byte(`{"name":"create-chapter","system":"embedded-sys","task":"t",
				"input_sections":{},"output":{"format":"json","description":"d"},
				"constraints":{"must":[],"forbidden":[]}}`),
		},
	}
	// dir 指向真实仓库 prompts/：磁盘 create-chapter 应叠加覆盖 embed 同名模板
	// （既有两级优先级回归）。
	eng := NewEngineWithEmbedded("../../prompts", embedded)
	if got := eng.Get("create-chapter"); got == nil || strings.Contains(got.System, "embedded-sys") {
		t.Fatalf("磁盘模板应优先于内置模板（既有回归）: %+v", got)
	}
	// 覆盖层命中 → 压过磁盘。
	eng.SetOverride(func(name string) *Template {
		if name == "create-chapter" {
			return overrideTemplate(name, "THREE-LEVEL-WIN")
		}
		return nil
	})
	if got := eng.Get("create-chapter"); got == nil || got.System != "THREE-LEVEL-WIN" {
		t.Fatalf("覆盖层应为第一级优先: %+v", got)
	}
	// 覆盖未命中（回调 nil）→ 回落磁盘模板，其余键不受影响。
	if got := eng.Get("chapter-summary"); got == nil {
		t.Fatal("未覆盖键应回落磁盘模板")
	}
}

// TestNamesSorted Names 返回内置 templates map 全部键、字典序稳定：
// 仓库 20 个模板全量在册、严格升序、与 Get 存在性互证；乱序源（map 遍历）
// 不影响输出序。
func TestNamesSorted(t *testing.T) {
	eng := NewEngine("../../prompts")
	names := eng.Names()
	if len(names) != 20 {
		t.Fatalf("仓库磁盘模板应恰 20 个，得到 %d: %v", len(names), names)
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("Names 应严格字典序升序：第 %d 位 %q >= %q（全序 %v）", i, names[i-1], names[i], names)
		}
	}
	for _, n := range names {
		if eng.Get(n) == nil {
			t.Fatalf("Names 中的键 %q 应可 Get（覆盖未设置时即内置表）", n)
		}
	}
	// 覆盖层不新增键：SetOverride 后 Names 不变（V1 只做既有键整模板覆盖）。
	eng.SetOverride(func(name string) *Template { return overrideTemplate(name, "x") })
	after := eng.Names()
	if len(after) != len(names) {
		t.Fatalf("SetOverride 不应改变 Names: %v vs %v", after, names)
	}
	for i := range names {
		if after[i] != names[i] {
			t.Fatalf("Names 在设置覆盖后应保持不变: %v vs %v", after, names)
		}
	}
}

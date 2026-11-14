package tool

// P0 注册表地图守卫（2026-11-09 众神殿 pan_1794234593182 裁决 action-1 验收线：
// 清单无未知工具；新增工具缺契约即守卫测试失败）。boot 运行时普查守卫在 boot 包
// （registrymap_census_test.go），app ExtraTools 守卫在 app 包
// （registrymap_extra_test.go）。

import (
	"os"
	"strings"
	"testing"
)

// TestRegistryMapCoversAllBuiltins 每个 RegisterBuiltin 注册的工具必须在
// RegistryMap 有契约行（来源/空间/副作用/证据档/验收谓词）。本用例红 = 新工具
// 漏登记，去 registrymap.go 补一行 + 更新 docs/gaea-tool-registry-map-2026-11.md。
func TestRegistryMapCoversAllBuiltins(t *testing.T) {
	var missing []string
	for _, tl := range Builtins() {
		if _, ok := RegistryMap[tl.Name()]; !ok {
			missing = append(missing, tl.Name())
		}
	}
	if len(missing) > 0 {
		t.Fatalf("已注册但缺契约行（P0 工具面地图漂移）: %v", missing)
	}
}

// TestRegistryMapRowsWellFormed 每行契约字段非空且取值在册（机械校验，防占位行）。
func TestRegistryMapRowsWellFormed(t *testing.T) {
	srcs := map[string]bool{
		"builtin-workspace": true, "builtin-domain": true,
		"build-internal": true, "extra-app": true, "plugin-mcp": true,
	}
	effects := map[string]bool{
		"read": true, "write": true, "produce": true, "meta": true, "interactive": true,
		"随工具": true,
	}
	for name, row := range RegistryMap {
		if !srcs[row.Source] {
			t.Errorf("%s: Source=%q 不在册", name, row.Source)
		}
		if row.Spaces == "" || row.Effect == "" || row.Evidence == "" || row.Predicate == "" {
			t.Errorf("%s: 契约字段有空值: %+v", name, row)
		}
		if !effects[row.Effect] {
			t.Errorf("%s: Effect=%q 不在册", name, row.Effect)
		}
	}
}

func TestSourceOfUsesRegistryMapAndMCPNamespace(t *testing.T) {
	cases := map[string]string{
		"read_file":                 "builtin-workspace",
		"task":                      "build-internal",
		"office_generate":           "extra-app",
		"mcp__github__search_code":  "mcp:github",
		"mcp__context7__query-docs": "mcp:context7",
		"legacy_unknown_tool":       "",
	}
	for name, want := range cases {
		if got := SourceOf(name); got != want {
			t.Errorf("SourceOf(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestRegistryMapDocCoversMap 导出文档覆盖本表全部工具名（文档防漂移；P0 交付物
// docs/gaea-tool-registry-map-2026-11.md 与数据同源）。
func TestRegistryMapDocCoversMap(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/gaea-tool-registry-map-2026-11.md")
	if err != nil {
		t.Fatalf("P0 注册表地图文档缺失（须与 RegistryMap 同步导出）: %v", err)
	}
	doc := string(raw)
	var missing []string
	for name := range RegistryMap {
		if !strings.Contains(doc, "`"+name+"`") {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("文档缺工具名（须重新导出）: %v", missing)
	}
	if !strings.Contains(doc, RegistryMapNotePromptBypass) {
		t.Fatal("文档缺 /mcp__ prompt 旁路备注行")
	}
	// P3 四类验收谓词段覆盖（类 id+类名）
	for id, cls := range PredicateClasses {
		if !strings.Contains(doc, id) || !strings.Contains(doc, cls.Name) {
			t.Fatalf("文档缺谓词类 %s（%s）——重新导出", id, cls.Name)
		}
	}
}

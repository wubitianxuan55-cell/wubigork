package app

// P0 注册表地图守卫·app ExtraTools 装配单点（2026-11-09 众神殿裁决 action-1）：
// gaeaExtraTools 装配的每个工具名必须在 RegistryMap 有契约行；其生效空间标签
// （builtin.SpaceTagOf：自声明优先，否则静态表，缺省 shared）必须与契约行
// Spaces 一致（2026-11-10 漂移刀——office 六工具曾因两处皆缺靠 fail-open
// 缺省 shared 混进 play 空间）。boot 运行时普查守卫在 boot 包
// （registrymap_census_test.go）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

func TestRegistryMapCoversAppExtraTools(t *testing.T) {
	var missing []string
	for _, tl := range gaeaExtraTools(&App{}) {
		if _, ok := tool.RegistryMap[tl.Name()]; !ok {
			missing = append(missing, tl.Name())
		}
	}
	if len(missing) > 0 {
		t.Fatalf("gaeaExtraTools 有工具缺 P0 契约行: %v", missing)
	}
}

// TestRegistryMapSpacesMatchExtraTools 空间契约一致性（extra-app 面）：本用例红 =
// 工具未归类（自声明/静态表两处皆缺→fail-open shared）或与契约行漂移。
func TestRegistryMapSpacesMatchExtraTools(t *testing.T) {
	for _, tl := range gaeaExtraTools(&App{}) {
		row, ok := tool.RegistryMap[tl.Name()]
		if !ok {
			continue // 漏契约行由 TestRegistryMapCoversAppExtraTools 钉
		}
		if got := builtin.SpaceTagOf(tl); got != row.Spaces {
			t.Errorf("%s: RegistryMap.Spaces=%q 与生效空间标签 %q 漂移（未归类工具 fail-open shared 会混进 play）",
				tl.Name(), row.Spaces, got)
		}
	}
}

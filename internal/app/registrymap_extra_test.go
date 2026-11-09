package app

// P0 注册表地图守卫·app ExtraTools 装配单点（2026-11-09 众神殿裁决 action-1）：
// gaeaExtraTools 装配的每个工具名必须在 RegistryMap 有契约行。
// boot 运行时普查守卫在 boot 包（registrymap_census_test.go）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
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

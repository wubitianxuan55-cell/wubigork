//go:build ignore

// registry_map_doc — P0 工具面地图导出器（2026-11-09 众神殿裁决 action-1）。
// 从 internal/gaea/tool.RegistryMap 单源生成 docs/gaea-tool-registry-map-2026-11.md，
// 文档与数据同源零手工漂移；守卫（tool/boot/app 三处 registrymap 测试）钉住覆盖面。
//
// 用法：go run scripts/registry_map_doc.go > docs/gaea-tool-registry-map-2026-11.md
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func main() {
	fmt.Println("# gaea 工具面注册表地图（P0）")
	fmt.Println()
	fmt.Println("> 2026-11-09 众神殿 pan_1794234593182 裁决 action-1 交付物。唯一注册表" +
		"（`boot.Build` 单一 `tool.Registry`）全部 reg.Add 行的契约清单：来源 / 空间 / 副作用 / " +
		"证据档 / 验收谓词。本文件由 `go run scripts/registry_map_doc.go` 从 " +
		"`internal/gaea/tool.RegistryMap` 单源生成，勿手改；防漂移守卫见 " +
		"`tool/registrymap_test.go`、`boot/registrymap_census_test.go`、`app/registrymap_extra_test.go`。")
	fmt.Println()
	fmt.Println(tool.RegistryMapNotePromptBypass)
	fmt.Println()
	fmt.Println("| 工具 | 来源 | 空间 | 副作用 | 证据档 | 验收谓词 |")
	fmt.Println("|---|---|---|---|---|---|")
	names := make([]string, 0, len(tool.RegistryMap))
	for n := range tool.RegistryMap {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		r := tool.RegistryMap[n]
		fmt.Printf("| `%s` | %s | %s | %s | %s | %s |\n", n, r.Source, r.Spaces, r.Effect, r.Evidence, r.Predicate)
	}
	fmt.Println()
	fmt.Println("适用条件：办公路径继续走 `boot.Build` 唯一注册表（四外部入口：builtin / ExtraTools / 插件 / Build 内部）；" +
		"若出现第五装配面（如按 `boot.go` 注释绕过 Build 的自定义装配器）则本清单失效，需重新盘点。")
	_ = os.Stdout
}

//go:build ignore

// registry_map_doc — P0 工具面地图导出器（2026-11-09 众神殿裁决 action-1）。
// 从 internal/gaea/tool.RegistryMap 单源生成 docs/gaea-tool-registry-map-2026-11.md，
// 文档与数据同源零手工漂移；守卫（tool/boot/app 三处 registrymap 测试）钉住覆盖面。
//
// 用法：
//
//	go run scripts/registry_map_doc.go            # 生成（重定向到文档）
//	go run scripts/registry_map_doc.go -check    # 漂移检查：文档≠单源即 exit 1
//	                                             # （2026-11-17 审查批3：candidate 两行
//	                                             # 漏导事故后补的发版门缝刀）
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

const registryMapDocPath = "docs/gaea-tool-registry-map-2026-11.md"

func main() {
	content := buildDoc()
	if len(os.Args) > 1 && os.Args[1] == "-check" {
		cur, err := os.ReadFile(registryMapDocPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "registry_map_doc -check: 读文档失败: %v\n", err)
			os.Exit(2)
		}
		if string(cur) != content {
			fmt.Fprintf(os.Stderr, "registry_map_doc -check: 文档漂移——重导：go run scripts/registry_map_doc.go > %s\n", registryMapDocPath)
			os.Exit(1)
		}
		fmt.Println("registry_map_doc -check: OK")
		return
	}
	fmt.Print(content)
}

func buildDoc() string {
	var b strings.Builder
	w := func(s string) { b.WriteString(s + "\n") }
	w("# gaea 工具面注册表地图（P0）")
	w("")
	w("> 2026-11-09 众神殿 pan_1794234593182 裁决 action-1 交付物。唯一注册表" +
		"（`boot.Build` 单一 `tool.Registry`）全部 reg.Add 行的契约清单：来源 / 空间 / 副作用 / " +
		"证据档 / 验收谓词。本文件由 `go run scripts/registry_map_doc.go` 从 " +
		"`internal/gaea/tool.RegistryMap` 单源生成，勿手改；防漂移守卫见 " +
		"`tool/registrymap_test.go`、`boot/registrymap_census_test.go`、`app/registrymap_extra_test.go`、" +
		"`go run scripts/registry_map_doc.go -check`（发版门缝刀 2026-11-17）。")
	w("")
	w(tool.RegistryMapNotePromptBypass)
	w("")
	w("| 工具 | 来源 | 空间 | 副作用 | 证据档 | 验收谓词 |")
	w("|---|---|---|---|---|---|")
	names := make([]string, 0, len(tool.RegistryMap))
	for n := range tool.RegistryMap {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		r := tool.RegistryMap[n]
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n", n, r.Source, r.Spaces, r.Effect, r.Evidence, r.Predicate)
	}
	w("")
	w("## P3 四类验收谓词（非文档工具）")
	w("")
	w("每类至少 1 金标例 + 1 已知缺陷例，harness 见 `builtin/p3_predicates_test.go`（缺类=守卫红）。")
	w("")
	w("| 类 | 统一验收口径 | 代表工具 |")
	w("|---|---|---|---|")
	ids := make([]string, 0, len(tool.PredicateClasses))
	for id := range tool.PredicateClasses {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := tool.PredicateClasses[id]
		fmt.Fprintf(&b, "| %s（%s） | %s | %s |\n", id, c.Name, c.Definition, strings.Join(c.Tools, " / "))
	}
	w("")
	b.WriteString("适用条件：办公路径继续走 `boot.Build` 唯一注册表（四外部入口：builtin / ExtraTools / 插件 / Build 内部）；" +
		"若出现第五装配面（如按 `boot.go` 注释绕过 Build 的自定义装配器）则本清单失效，需重新盘点。\n")
	return b.String()
}

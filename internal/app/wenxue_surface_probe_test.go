package app

// wenxue_surface_probe_test.go — 图文引擎装配面 census 守卫（2026-11-15 众神殿
// 首刀收口批；原一次性打印探针在裁剪拍板落地后转正为断言守卫）。
// 钉死三件事：① wenxue_* 十五件全装配；② fact_*（交付事实库除名）与办公域
// 全族物理不装；③ 通用底座媒体/技能入口在场。防"新增 builtin 静默流进图文"。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// wenxueAssembledFaces 按图文引擎装配谓词（space=work + wenxueToolFilter）分
// 拣 keep/deny 两面（与 boot.Build 装配期过滤同构；wx ExtraTools=wenxueBase
// Tools（媒体根分档+image_gen work 适配）+wenxue 专属两单点）。
func wenxueAssembledFaces() (map[string]bool, []string) {
	all := append(tool.Builtins(), wenxueBaseTools(&App{})...)
	all = append(all, wenxueExtraTools(&App{})...)
	keep := map[string]bool{}
	var denied []string
	for _, tl := range all {
		name := tl.Name()
		if !builtin.AllowsSpace(tl, "work") {
			denied = append(denied, name+"(space)")
			continue
		}
		if !wenxueToolFilter(name) {
			denied = append(denied, name+"(filter)")
			continue
		}
		keep[name] = true
	}
	return keep, denied
}

func TestWenxueSurfaceKeepsExclusiveFamily(t *testing.T) {
	keep, _ := wenxueAssembledFaces()
	for _, n := range []string{
		"wenxue_chapter_list", "wenxue_chapter_read", "wenxue_chapter_write",
		"wenxue_candidate_write", "wenxue_candidate_promote",
		"wenxue_character_list", "wenxue_character_read", "wenxue_character_write",
		"wenxue_character_import",
		"wenxue_timeline_read", "wenxue_timeline_append", "wenxue_stats",
		"wenxue_quality_check", "wenxue_style_report",
		"wenxue_final_list", "wenxue_final_read", "wenxue_final_write",
		"wenxue_chapter_deliver", // 2026-11-23 实测反馈：单章交付
	} {
		if !keep[n] {
			t.Fatalf("图文专属工具 %s 应装配进图文引擎", n)
		}
	}
}

func TestWenxueSurfaceDeniesFactAndOfficeFamilies(t *testing.T) {
	_, denied := wenxueAssembledFaces()
	denySet := map[string]bool{}
	for _, d := range denied {
		denySet[d] = true
	}
	// fact_* 除名（众神殿首刀）：办公交付事实库不进图文会话。
	for _, n := range []string{"fact_add(filter)", "fact_list(filter)", "fact_clear(filter)"} {
		if !denySet[n] {
			t.Fatalf("交付事实库工具 %s 应被图文引擎拒绝装配（census 守卫）", n)
		}
	}
	// 办公域抽样：生成/批处理/COM/CAD/造价/进度/浏览器。（文件流水线 dag_plan
	// 2026-11-25 起不再进办公 ExtraTools，不在本 census 枚举面。）
	for _, n := range []string{
		"office_generate(filter)", "office_batch(filter)", "docx_apply(filter)",
		"xlsx_apply(filter)", "cad_ask(filter)", "cost_library(filter)",
		"schedule_get(filter)", "browser_navigate(filter)",
	} {
		if !denySet[n] {
			t.Fatalf("办公域工具 %s 应被图文引擎拒绝装配（census 守卫）", n)
		}
	}
	// 图文瘦身刀（2026-11-16）装配面抽查：编辑族/大文件件/办公侧栏。
	// （boot 直注册的 summarize_file/子代理管理三件/会话元工具两件不在此
	// census 枚举面——它们由 boot.addGated 过闸，单测见 wenxue_tool_policy_test
	// 与 boot.boot_toolgate_test。）
	for _, n := range []string{
		// （edit_lines/multi_edit/replace 已于 2026-11-25 全局退役，不在装配面。）
		"edit_file(filter)", "move_file(filter)", "read_spill(filter)",
		"sidebar_open(filter)",
	} {
		if !denySet[n] {
			t.Fatalf("瘦身刀工具 %s 应被图文引擎拒绝装配（census 守卫）", n)
		}
	}
}

func TestWenxueSurfaceKeepsGenericBaseAndMedia(t *testing.T) {
	keep, _ := wenxueAssembledFaces()
	// 技能入口（read_skill 实例化替换同名 builtin，装配面同名在场）、媒体底座
	// 与 image_gen（wxWorkSpaceTool work 适配——媒体根分档裁决的生图面）。
	for _, n := range []string{"read_skill", "ocr", "vision", "web_search", "write_file", "image_gen"} {
		if !keep[n] {
			t.Fatalf("通用底座工具 %s 应装配进图文引擎", n)
		}
	}
}

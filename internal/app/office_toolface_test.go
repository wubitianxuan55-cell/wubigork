package app

// office_toolface_test.go — 办公工具面体检（2026-11-25 办公工具梳理批）。
//
// 模型每轮都为全部可见工具的 schema 付上下文，工具面回胖没有守卫就会悄悄
// 发生（2026-11-25 实测：造价档 92 件、4.7 万字符，是身份提示词的 5 倍）。
// 本文件两件事：
//   - TestOfficeToolFaceBudget：三档各一条字符预算，超了即红（防回胖守卫）；
//   - TestOfficeToolFaceDump：设 GAEA_TOOLDUMP_OUT 时导出三档逐工具尺寸
//     （name/来源/描述字符/schema 字符），梳理时盘底对账用，平时跳过。
//
// 装配口径与 gaeaBuildController 同源（ExtraTools=gaeaExtraTools、
// ToolHide=composeToolHide、SkillExtras/ModeLine/SystemPrompt 同批），只把
// provider 换成 mock——量的是注册表里模型实见的那一面（ctrl.Schemas 已应用
// Hide 与 schema 预算截断）。

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/provider"

	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// toolFaceRow 逐工具尺寸（字符按 rune 计；Bytes 是该工具 schema JSON 的
// UTF-8 字节数——provider 桥接按字节计量时看这一列）。
type toolFaceRow struct {
	Name   string `json:"name"`
	Group  string `json:"group,omitempty"`
	Source string `json:"source,omitempty"`
	Desc   int    `json:"descChars"`
	Schema int    `json:"schemaChars"`
	Total  int    `json:"totalChars"`
	Bytes  int    `json:"utf8Bytes"`
}

type toolFaceView struct {
	Mode       string        `json:"mode"`
	Tools      int           `json:"tools"`
	TotalChars int           `json:"totalChars"`
	TotalBytes int           `json:"totalBytes"`
	DescChars  int           `json:"descChars"`
	Rows       []toolFaceRow `json:"rows"`
}

// officeFaceModes 体检覆盖的三档（与 config.OfficeRunMode 归一口径一致）。
var officeFaceModes = []string{OfficeRunModeStandard, OfficeRunModeCost, OfficeRunModeProposal}

// buildOfficeFaceForTest 按生产装配口径构建一档办公引擎（mock provider）。
func buildOfficeFaceForTest(t *testing.T, mode string) (*control.Controller, []string) {
	t.Helper()
	ws := t.TempDir()
	kind := testKind("test-mock-office-face")
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name: "mock", Kind: kind, Model: "m1", ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	t.Cleanup(func() { gaeaConfig.SetLoader(nil) })

	hide := composeToolHide(mode, OfficeToolFaceLean, nil)
	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		Model: "mock", RequireKey: false,
		Sink:         event.FuncSink(func(event.Event) {}),
		Stderr:       io.Discard,
		SessionDir:   filepath.Join(ws, "sessions"),
		Cwd:          ws,
		ExtraTools:   gaeaExtraTools(&App{}),
		ToolHide:     hide,
		ModeLine:     composeModeLine(mode, OfficeToolFaceLean, hide),
		SkillExtras:  officeRunModeSkillExtras(mode),
		SystemPrompt: officeModeSystemPrompt(mode, (*gaeaConfig.Config)(nil)),
	})
	if err != nil {
		t.Fatalf("boot.Build(%s): %v", mode, err)
	}
	t.Cleanup(ctrl.Close)
	return ctrl, hide
}

// measureOfficeFace 量一档模型实见的工具面。
func measureOfficeFace(t *testing.T, mode string) toolFaceView {
	t.Helper()
	ctrl, hide := buildOfficeFaceForTest(t, mode)
	hidden := map[string]bool{}
	for _, n := range hide {
		hidden[n] = true
	}
	view := toolFaceView{Mode: mode, Rows: []toolFaceRow{}}
	for _, s := range ctrl.Schemas() {
		if hidden[s.Name] {
			continue // Hide 面成员不在模型实见面（协议层仍可按名调）
		}
		schema := string(s.Parameters)
		row := toolFaceRow{
			Name: s.Name, Group: string(tool.ToolGroupOf(s.Name)), Source: tool.SourceOf(s.Name),
			Desc: utf8.RuneCountInString(s.Description), Schema: utf8.RuneCountInString(schema),
			Bytes: len(s.Description) + len(schema),
		}
		row.Total = row.Desc + row.Schema
		view.Rows = append(view.Rows, row)
		view.TotalChars += row.Total
		view.TotalBytes += row.Bytes
		view.DescChars += row.Desc
	}
	sort.Slice(view.Rows, func(i, j int) bool { return view.Rows[i].Total > view.Rows[j].Total })
	view.Tools = len(view.Rows)
	return view
}

// officeFaceBudgetEntry 一档的工具面预算。
type officeFaceBudgetEntry struct {
	Tools int // 件数上限：加工具必须有意识地抬这里
	Chars int // 字符上限（描述+schema，rune 计）：留了余量给文案微调
}

// officeFaceBudget 三档预算定数（2026-11-25 办公工具梳理二批收口时实测：
// standard 58 件 / 16404 字符、cost 66 件 / 27824 字符、proposal 70 件 /
// 20863 字符）。余量口径：件数=实测值（加一件就得改这里，改之前先回答「它
// 能不能并进已有工具」）；字符=实测值上取整到 250 的倍数再留一档（文案微调
// 不必动预算，但塞不进一整段新描述）。工具面回胖没有守卫就会悄悄发生——
// 造价档曾到 92 件 / 4.7 万字符。
var officeFaceBudget = map[string]officeFaceBudgetEntry{
	// standard 16750→17000（2026-11-27 方案库导入批）：format_convert 增 to=docx
	// （PDF→Word，录库 PDF 源必经）整段新描述塞不进原余量，实测 16775 后按口径上取整。
	OfficeRunModeStandard: {Tools: 58, Chars: 17000},
	// cost 28250→28500（2026-11-27 造价复盘刀A/B/C）：cost_project 增 offset 翻页、
	// cost_library 增 import_status 与 dims summary、import 同码部分修正契约，
	// 实测 28360 后按口径上取整。
	OfficeRunModeCost:     {Tools: 66, Chars: 28500},
	// proposal 21000→21250（2026-11-27 沿江包1 复盘批）：md_validate 增硬伤/提示
	// 分级与 baseline 基线差集，compact 描述净增 15 字符，实测 21265 后按口径上取整。
	OfficeRunModeProposal: {Tools: 70, Chars: 21500},
}

// officeFaceRetired 已从模型工具面下线、不得回潮的名字（办公工具梳理两批）。
// 回潮途径=有人把旧名字重新注册进 gaeaExtraTools / boot 直注册面。
var officeFaceRetired = []string{
	// 首批（799e9ba8）
	"cost_compose", "multi_edit", "edit_lines", "replace", "dag_plan",
	// 二批：slash_command 并入 run_skill；docx_accept 并入 docx_apply；
	// office_macro 退役（COM 档按本机条件装配，注册面一并钉）。
	"slash_command", "docx_accept", "office_macro",
}

// TestOfficeToolFaceBudget 防回胖守卫：三档模型实见的工具面不得超预算，
// 已下线的工具名不得重新出现在注册面。
func TestOfficeToolFaceBudget(t *testing.T) {
	for _, mode := range officeFaceModes {
		v := measureOfficeFace(t, mode)
		budget, ok := officeFaceBudget[mode]
		if !ok {
			t.Fatalf("%s 没有预算定数", mode)
		}
		t.Logf("%s: %d 件 / %d 字符（描述 %d；预算 %d 件 / %d 字符）",
			mode, v.Tools, v.TotalChars, v.DescChars, budget.Tools, budget.Chars)
		if v.Tools > budget.Tools {
			t.Errorf("%s 工具面 %d 件，超预算 %d 件——新工具先问能不能并进已有工具；确需新增再抬预算并写明理由",
				mode, v.Tools, budget.Tools)
		}
		if v.TotalChars > budget.Chars {
			t.Errorf("%s 工具面 %d 字符，超预算 %d——工具描述是模型每轮都付的常驻面，长说明改随回执发放",
				mode, v.TotalChars, budget.Chars)
		}
		seen := map[string]bool{}
		for _, r := range v.Rows {
			seen[r.Name] = true
		}
		for _, name := range officeFaceRetired {
			if seen[name] {
				t.Errorf("%s：已下线的工具 %s 回到了模型工具面", mode, name)
			}
		}
	}
}

// TestOfficeToolFaceDump 按需导出：GAEA_TOOLDUMP_OUT=<输出 json 路径> 时写三档
// 逐工具尺寸；未设置即跳过（零成本）。
func TestOfficeToolFaceDump(t *testing.T) {
	out := os.Getenv("GAEA_TOOLDUMP_OUT")
	if out == "" {
		t.Skip("GAEA_TOOLDUMP_OUT 未设置：仅按需导出")
	}
	views := make([]toolFaceView, 0, len(officeFaceModes))
	for _, mode := range officeFaceModes {
		v := measureOfficeFace(t, mode)
		t.Logf("%s: %d 个工具，%d 字符 / %d 字节（描述 %d）", mode, v.Tools, v.TotalChars, v.TotalBytes, v.DescChars)
		views = append(views, v)
	}
	b, err := json.MarshalIndent(views, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatalf("写导出文件: %v", err)
	}
}

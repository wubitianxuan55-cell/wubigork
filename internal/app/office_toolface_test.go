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

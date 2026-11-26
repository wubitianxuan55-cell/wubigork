package app

// wenxue_toolface_test.go — 图文工具面体检（2026-11-26 文学同步批；与办公
// office_toolface_test.go 同款守卫，补上「办公有预算守卫、文学只有名单守卫」
// 的缺口）。
//
// 模型每轮都为全部可见工具的 schema 付上下文。census 守卫
// （wenxue_surface_probe_test.go）只钉「谁在/谁不在」，钉不住「在的那批变
// 胖」——共享 builtin 的描述加长、boot 直注册面新增件，都会静默流进文学。
//   - TestWenxueToolFaceBudget：一条字符预算，超了即红（防回胖守卫）；同时
//     断言 wenxueToolFilter 的 deny 名单没有任何一件出现在模型实见面；
//   - TestWenxueToolFaceDump：设 GAEA_TOOLDUMP_OUT 时导出逐工具尺寸，平时跳过。
//
// 装配口径=wenxueBuildController 生产路径（真实 WenxueEngineInit：ToolFilter/
// SkillFilter/SkillExtras/ExecBoard/SpaceAnchor 同批），只把 provider 换成
// 测试桩——量的是注册表里模型实见的那一面（ctrl.Schemas 已应用 schema 预算
// 截断；文学引擎无 ToolHide，Schemas 即模型实见面）。

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// buildWenxueFaceForTest 按生产装配路径构建图文引擎（配置/桥接/脑目录/
// 书根全部隔离，同 TestWenxueChatSessionBindingSmoke）。
func buildWenxueFaceForTest(t *testing.T) *control.Controller {
	t.Helper()
	prevBrain := wenxueBrainDirOverride
	wenxueBrainDirOverride = t.TempDir()
	brainDir := wenxueBrainDirOverride
	t.Cleanup(func() {
		wenxueBrainDirOverride = prevBrain
		db.CloseDatabase(brainDir)
	})

	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{Name: "gaea", Kind: "wubigrok", Model: "", ContextWindow: 1_000_000}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	t.Cleanup(func() { gaeaConfig.SetLoader(nil) })
	bridge.SetClient(ai.NewClient(config.Load()))
	t.Cleanup(func() { bridge.SetClient(nil) })

	a := &App{core: &core{cfg: config.Load()}}
	a.ctx = context.Background()
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	t.Cleanup(func() {
		wx.mu.Lock()
		old := wx.ctrl
		wx.ctrl = nil
		wx.mu.Unlock()
		if old != nil {
			old.Close()
		}
	})
	if err := a.WenxueEngineInit(); err != nil {
		t.Fatalf("图文引擎首建失败: %v", err)
	}
	wx.mu.Lock()
	ctrl := wx.ctrl
	wx.mu.Unlock()
	if ctrl == nil {
		t.Fatal("图文引擎构建后控制器为空")
	}
	return ctrl
}

// measureWenxueFace 量图文引擎模型实见的工具面。
func measureWenxueFace(t *testing.T) toolFaceView {
	t.Helper()
	ctrl := buildWenxueFaceForTest(t)
	view := toolFaceView{Mode: "wenxue", Rows: []toolFaceRow{}}
	for _, s := range ctrl.Schemas() {
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

// wenxueFaceBudget 图文工具面预算（余量口径与办公同款）：件数=实测值（加一件
// 就得改这里，改之前先回答「它能不能并进已有工具」）；字符=实测值上取整到
// 250 的倍数再留一档（文案微调不必动预算，但塞不进一整段新描述）。
//
// 2026-11-26 首测 55 件 / 17682 字符——量出造价/长文方案两组 6 件办公运行模式
// 工具混进图文工具面（约 4400 字符，25%），按组拒绝后复测 49 件 / 13285 字符
// （描述 4971）。对照办公标准档 58 件 / 16404 字符：文学件数与字符都更少，
// 但 wenxue_* 专属 19 件占约 4200 字符，是后续梳理的候选面（本批只立守卫）。
var wenxueFaceBudget = officeFaceBudgetEntry{Tools: 49, Chars: 13750}

// TestWenxueToolFaceBudget 防回胖守卫：模型实见的工具面不得超预算；deny 名单
// 里的名字（含 boot 直注册面）不得出现在模型实见面。
func TestWenxueToolFaceBudget(t *testing.T) {
	v := measureWenxueFace(t)
	t.Logf("wenxue: %d 件 / %d 字符（描述 %d；预算 %d 件 / %d 字符）",
		v.Tools, v.TotalChars, v.DescChars, wenxueFaceBudget.Tools, wenxueFaceBudget.Chars)
	if v.Tools > wenxueFaceBudget.Tools {
		t.Errorf("图文工具面 %d 件，超预算 %d 件——新工具先问能不能并进已有工具；确需新增再抬预算并写明理由",
			v.Tools, wenxueFaceBudget.Tools)
	}
	if v.TotalChars > wenxueFaceBudget.Chars {
		t.Errorf("图文工具面 %d 字符，超预算 %d——工具描述是模型每轮都付的常驻面，长说明改随回执发放",
			v.TotalChars, wenxueFaceBudget.Chars)
	}
	seen := map[string]bool{}
	for _, r := range v.Rows {
		seen[r.Name] = true
	}
	for _, name := range wenxueDeniedTools {
		hit := false
		if name != "" && name[len(name)-1] == '_' {
			for n := range seen {
				if len(n) >= len(name) && n[:len(name)] == name {
					hit = true
				}
			}
		} else {
			hit = seen[name]
		}
		if hit {
			t.Errorf("deny 名单工具 %s 出现在图文模型实见面", name)
		}
	}
}

// TestWenxueToolFaceDump 按需导出：GAEA_TOOLDUMP_OUT=<输出 json 路径> 时写逐
// 工具尺寸；未设置即跳过（零成本）。
func TestWenxueToolFaceDump(t *testing.T) {
	out := os.Getenv("GAEA_TOOLDUMP_OUT")
	if out == "" {
		t.Skip("GAEA_TOOLDUMP_OUT 未设置：仅按需导出")
	}
	v := measureWenxueFace(t)
	t.Logf("wenxue: %d 个工具，%d 字符 / %d 字节（描述 %d）", v.Tools, v.TotalChars, v.TotalBytes, v.DescChars)
	b, err := json.MarshalIndent([]toolFaceView{v}, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatalf("写导出文件: %v", err)
	}
}

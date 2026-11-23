package app

// office_runmode_test.go — 办公运行模式守卫矩阵（2026-11-21 众神殿
// pan_1795202763988 终案台账项④「三档×四表守卫矩阵」）。
//
// 钉死四层一致：
//   ① 组合表自身（hide 面按档正确、extras 五件齐全、模式行三态非空）；
//   ② 组合表 ↔ 注册表对账（hide 名单全部真实注册——名单漂移即红）；
//   ③ 配置归一双口径（config.OfficeRunMode 与 normalizeOfficeRunMode 一致）；
//   ④ 造价/方案工具行为契约（写前同批编码闸、机检硬检/信息项分级、
//      longdoc 机检七类与构建闸）。

import (
	"context"
	"slices"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// TestOfficeRunModeMatrix 三档组合表断言（终案 DSH-r1 表 + Codex-r3 终表）。
func TestOfficeRunModeMatrix(t *testing.T) {
	costAll := append([]string(nil), officeCostToolNames...)
	longdocAll := append([]string(nil), officeLongdocToolNames...)
	cadAll := tool.ToolGroupMembers(tool.ToolGroupCAD)
	schedAll := tool.ToolGroupMembers(tool.ToolGroupSchedule)

	// 2026-11-21 用户拍板改排：CAD 随造价、进度随方案，标准档两族全关。
	stdAll := slices.Concat(costAll, longdocAll, cadAll, schedAll)
	costHide := slices.Concat(longdocAll, schedAll)
	proposalHide := slices.Concat(costAll, cadAll)

	cases := []struct {
		mode    string
		hide    []string
		extras  []string
		lineKey string
	}{
		{OfficeRunModeStandard, stdAll, nil, "标准"},
		{OfficeRunModeCost, costHide, []string{"cost-estimation-playbook", "soil-remediation-costing"}, "造价"},
		{OfficeRunModeProposal, proposalHide, []string{"proposal-longform", "proposal-review", "soil-remediation-proposal"}, "方案"},
	}
	for _, tc := range cases {
		got := officeRunModeHide(tc.mode)
		if len(got) != len(tc.hide) {
			t.Fatalf("模式 %s hide 面数量 %d ≠ 期望 %d（got=%v）", tc.mode, len(got), len(tc.hide), got)
		}
		for _, name := range tc.hide {
			if !ormContains(got, name) {
				t.Fatalf("模式 %s hide 面缺 %s", tc.mode, name)
			}
		}
		extras := officeRunModeSkillExtras(tc.mode)
		if len(extras) != len(tc.extras) {
			t.Fatalf("模式 %s extras 数量 %d ≠ 期望 %d", tc.mode, len(extras), len(tc.extras))
		}
		for i, want := range tc.extras {
			if extras[i].Name != want {
				t.Fatalf("模式 %s extras[%d]=%s ≠ %s", tc.mode, i, extras[i].Name, want)
			}
			if strings.TrimSpace(extras[i].Body) == "" || strings.TrimSpace(extras[i].Description) == "" {
				t.Fatalf("模式 %s 技能 %s 缺正文或描述", tc.mode, want)
			}
		}
		line := officeRunModeLine(tc.mode)
		if !strings.Contains(line, "【运行模式】") || !strings.Contains(line, tc.lineKey) {
			t.Fatalf("模式 %s 模式行缺【运行模式】头或档名 %q", tc.mode, tc.lineKey)
		}
		if !strings.Contains(line, "新会话生效") {
			t.Fatalf("模式 %s 模式行缺切档指引（新会话生效）", tc.mode)
		}
	}
	// 误选恢复断言（DSH-r5 第八条验收）：标准档模式行必须点名切换方向并
	// 交代 CAD/进度归档（2026-11-21 改排：造价带 CAD、方案带进度）。
	std := officeRunModeLine(OfficeRunModeStandard)
	if !strings.Contains(std, "造价") || !strings.Contains(std, "方案") {
		t.Fatalf("标准档模式行缺「切到造价/方案」指引：%s", std)
	}
	if !strings.Contains(std, "CAD") || !strings.Contains(std, "进度计划") {
		t.Fatalf("标准档模式行缺 CAD/进度归档说明：%s", std)
	}
	if !strings.Contains(officeRunModeLine(OfficeRunModeCost), "cad_ask") {
		t.Fatalf("造价档模式行应点名 cad_ask（CAD 并入造价）")
	}
	if !strings.Contains(officeRunModeLine(OfficeRunModeProposal), "schedule_get") {
		t.Fatalf("方案档模式行应点名 schedule_get（进度并入方案）")
	}
}

// TestComposeToolHideMatrix 双轴组合矩阵（2026-11-21 终案+用户改排）：
// run_mode×tool_face×组覆盖的确定性汇总。
func TestComposeToolHideMatrix(t *testing.T) {
	inHide := func(hide []string, name string) bool { return ormContains(hide, name) }
	// 标准运行模式+精简面（新建会话默认）：cost7+longdoc2+cad1+schedule3+browser12+com10 全关；
	// media 四件开。
	lean := composeToolHide(OfficeRunModeStandard, OfficeToolFaceLean, nil)
	for _, name := range append(append(append(append([]string(nil),
		tool.ToolGroupMembers(tool.ToolGroupCost)...),
		tool.ToolGroupMembers(tool.ToolGroupLongdoc)...),
		tool.ToolGroupMembers(tool.ToolGroupCAD)...),
		tool.ToolGroupMembers(tool.ToolGroupSchedule)...) {
		if !inHide(lean, name) {
			t.Fatalf("精简标准档应藏 %s：%v", name, lean)
		}
	}
	for _, name := range append(tool.ToolGroupMembers(tool.ToolGroupBrowser), tool.ToolGroupMembers(tool.ToolGroupCOM)...) {
		if !inHide(lean, name) {
			t.Fatalf("精简档应藏 browser/COM %s", name)
		}
	}
	for _, name := range tool.ToolGroupMembers(tool.ToolGroupMedia) {
		if inHide(lean, name) {
			t.Fatalf("产图两档默认开（用户签注），%s 不应藏", name)
		}
	}
	// 标准面：browser/COM 回；CAD/进度仍随 run_mode 关（标准档）。
	stdFace := composeToolHide(OfficeRunModeStandard, OfficeToolFaceStandard, nil)
	for _, name := range append(tool.ToolGroupMembers(tool.ToolGroupBrowser), tool.ToolGroupMembers(tool.ToolGroupCOM)...) {
		if inHide(stdFace, name) {
			t.Fatalf("标准面应恢复 %s", name)
		}
	}
	if !inHide(stdFace, "cad_ask") || !inHide(stdFace, "schedule_get") {
		t.Fatalf("标准档 CAD/进度应仍关（改排后归造价/方案档）")
	}
	// 造价档：CAD 开；方案档：进度开。
	costFace := composeToolHide(OfficeRunModeCost, OfficeToolFaceStandard, nil)
	if inHide(costFace, "cad_ask") || inHide(costFace, "cost_project") {
		t.Fatalf("造价档 CAD+造价族应开：%v", costFace)
	}
	if !inHide(costFace, "schedule_get") {
		t.Fatalf("造价档进度应关（进度归方案档）")
	}
	proposalFace := composeToolHide(OfficeRunModeProposal, OfficeToolFaceLean, nil)
	if inHide(proposalFace, "schedule_get") || inHide(proposalFace, "longdoc_manifest") {
		t.Fatalf("方案档 进度+方案族应开")
	}
	if !inHide(proposalFace, "browser_click") {
		t.Fatalf("精简面 browser 应关")
	}
	// 用户覆盖：精简档强制开 schedule → schedule-edit 技能门同步恢复。
	openSchedule := composeToolHide(OfficeRunModeStandard, OfficeToolFaceLean, map[string]bool{"schedule": true})
	if inHide(openSchedule, "schedule_get") {
		t.Fatalf("组覆盖 schedule=true 应恢复 schedule_get")
	}
	// 谓词语义：false=被拒（deny-by-absence）、true=放行——判定直接按谓词
	// 返回值写，勿再引入中间变量反义（2026-11-21 实录：曾把放行误当被拒）。
	sf := officeRunModeSkillFilter(openSchedule)
	if sf == nil {
		t.Fatalf("cad/schedule 仍有关组，技能门谓词不应为 nil")
	}
	if !sf("schedule-edit") {
		t.Fatalf("schedule 组开启后 schedule-edit 应放行")
	}
	if sf("cad-takeoff") {
		t.Fatalf("cad 组仍关，cad-takeoff 应拒")
	}
	// 用户覆盖：media=false 强制关产图。
	noMedia := composeToolHide(OfficeRunModeCost, OfficeToolFaceLean, map[string]bool{"media": false})
	if !inHide(noMedia, "image_gen") {
		t.Fatalf("组覆盖 media=false 应关产图")
	}
}

// TestComposeModeLineDeterministic ModeLine golden：相同配置逐字节相同；
// 精简面点名隐藏组、COM 未装机不列名。
func TestComposeModeLineDeterministic(t *testing.T) {
	hide := composeToolHide(OfficeRunModeStandard, OfficeToolFaceLean, nil)
	a := composeModeLine(OfficeRunModeStandard, OfficeToolFaceLean, hide)
	b := composeModeLine(OfficeRunModeStandard, OfficeToolFaceLean, composeToolHide(OfficeRunModeStandard, OfficeToolFaceLean, nil))
	if a != b {
		t.Fatalf("ModeLine 非确定性：a=%q b=%q", a, b)
	}
	if !strings.Contains(a, "【工具面】精简") {
		t.Fatalf("ModeLine 缺工具面段：%s", a)
	}
	if !strings.Contains(a, "进度计划") || !strings.Contains(a, "CAD 图纸") {
		t.Fatalf("ModeLine 应点名隐藏组（CAD/进度）：%s", a)
	}
	// 组覆盖全开 cad/schedule → ModeLine 的【工具面】段不再列它们（组恢复；
	// 基线运行模式行本就点名 CAD/进度归档，断言只看工具面段）。
	openGroups := composeToolHide(OfficeRunModeStandard, OfficeToolFaceLean, map[string]bool{"cad": true, "schedule": true})
	line := composeModeLine(OfficeRunModeStandard, OfficeToolFaceLean, openGroups)
	faceSeg := line[strings.Index(line, "【工具面】"):]
	if strings.Contains(faceSeg, "CAD 图纸") || strings.Contains(faceSeg, "进度计划") {
		t.Fatalf("组覆盖恢复后工具面段不应列名：%s", faceSeg)
	}
	if !strings.Contains(faceSeg, "浏览器自动化") {
		t.Fatalf("精简面工具面段应列浏览器自动化：%s", faceSeg)
	}
	// COM 未装机（测试环境 officeComAvailable 探测失败路径）——ModeLine 不列
	// COM 名：断言走 officeComAvailable()==false 的机器必然成立；装机机器上
	// 本断言跳过（诚实缺省，避免环境假红）。
	if !officeComAvailable() && strings.Contains(a, "COM 实时协同") {
		t.Fatalf("COM 未装机时 ModeLine 不应列 COM 实时协同：%s", a)
	}
}

// TestOfficeRunModeToolFaceAllRegistered 组合表 ↔ 注册表对账：hide 名单里的
// 每个名字都必须真实注册（builtin 注册面 ∪ app 层 ExtraTools 装配面）。名单
// 漂移（改名/漏注册）= standard 档 Hide 落空 + 守卫红。
func TestOfficeRunModeToolFaceAllRegistered(t *testing.T) {
	registered := map[string]bool{}
	for _, tl := range tool.Builtins() {
		registered[tl.Name()] = true
	}
	for _, tl := range extraToolsWithMediaRoot(&App{}, nil) {
		if tl != nil {
			registered[tl.Name()] = true
		}
	}
	for _, name := range append(append([]string(nil), officeCostToolNames...), officeLongdocToolNames...) {
		if !registered[name] {
			t.Fatalf("专用工具 %s 未注册（builtin/ExtraTools 两装配面均无）——ToolHide 会落空", name)
		}
	}
}

// TestOfficeRunModeConfigNormalized 配置归一双口径一致（终案 ZCode-r3：两处
// 口径守卫钉死）。
func TestOfficeRunModeConfigNormalized(t *testing.T) {
	cases := map[string]string{
		"":         "standard",
		"standard": "standard",
		"STANDARD": "standard",
		" cost ":   "cost",
		"Proposal": "proposal",
		"bogus":    "standard",
	}
	for in, want := range cases {
		if got := normalizeOfficeRunMode(in); got != want {
			t.Fatalf("normalizeOfficeRunMode(%q)=%s ≠ %s", in, got, want)
		}
		var c config.Config
		c.Office.RunMode = in
		if got := c.OfficeRunMode(); got != want {
			t.Fatalf("config(%q).OfficeRunMode()=%s ≠ %s", in, got, want)
		}
	}
	if got := (*config.Config)(nil).OfficeRunMode(); got != "standard" {
		t.Fatalf("nil Config.OfficeRunMode()=%s ≠ standard", got)
	}
}

// TestCostProjectApplyBatchCollision 写前同批编码闸（终案 Grok-r3 定罪：库
// UNIQUE 对同码走 UPDATE 会吞后行，必须写前整批拒绝）。
func TestCostProjectApplyBatchCollision(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	if _, err := a.hubWorkcostStore().UpsertBillProject("闸门测试", "", "", "", "", "", nil, false, 0); err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	projects, _ := a.hubWorkcostStore().BillProjects()
	tl := costProjectApplyTool{a: a}
	args := map[string]any{
		"action": "save_items", "project_id": projects[0].ID,
		"items": []map[string]any{
			{"code": "WP01", "title": "行一", "unit": "m³", "quantity": 1},
			{"code": "WP01", "title": "行二", "unit": "m²", "quantity": 2},
		},
	}
	b, _ := json.Marshal(args)
	if _, err := tl.Execute(context.Background(), b); err == nil || !strings.Contains(err.Error(), "同批编码重复") {
		t.Fatalf("同批同码应整批拒绝，got err=%v", err)
	}
	items, _ := a.hubWorkcostStore().BillItems(projects[0].ID)
	if len(items) != 0 {
		t.Fatalf("写前闸必须一行不落（got %d 行）", len(items))
	}
}

// TestCostProjectApplySaveItems 建项目→批量清单（套定额/手填）→定额悬空整批拒绝。
func TestCostProjectApplySaveItems(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, err := s.SaveResource(workcost.Resource{Kind: "人工", Title: "普工", Unit: "工日", BasePrice: 100, CurrentPrice: 120})
	if err != nil {
		t.Fatalf("保存资源失败: %v", err)
	}
	if _, err := s.SaveQuota(workcost.Quota{Code: "SZ-DL-001", Title: "土方开挖", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 0.05}}}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}
	tl := costProjectApplyTool{a: a}
	mustOK := func(args map[string]any, want string) string {
		t.Helper()
		b, _ := json.Marshal(args)
		out, err := tl.Execute(context.Background(), b)
		if err != nil {
			t.Fatalf("Execute(%v) 失败: %v", args["action"], err)
		}
		if !strings.Contains(out, want) {
			t.Fatalf("回执缺 %q：%s", want, out)
		}
		return out
	}
	out := mustOK(map[string]any{"action": "create_project", "name": "测算A", "management_rate": 0.1, "tax_rate": 0.09}, "项目已建档")
	pid := parseID(t, out)
	mustOK(map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "C01", "title": "挖土", "unit": "m³", "quantity": 100, "quota_code": "SZ-DL-001"},
		{"title": "外委检测", "unit": "项", "quantity": 1, "price_override": 500},
	}}, "清单已写入 2/2 行")
	// 悬空定额：整批拒绝（不编造定额号）。
	bad := map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"title": "X", "unit": "m", "quantity": 1, "quota_code": "NOPE-999"},
	}}
	bb, _ := json.Marshal(bad)
	if _, err := tl.Execute(context.Background(), bb); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("悬空定额应整批拒绝，got err=%v", err)
	}
	// 费率 0 = 显式不计取是合法值：update_rates 全零成功。
	mustOK(map[string]any{"action": "update_rates", "project_id": pid, "management_rate": 0, "regulatory_rate": 0, "profit_rate": 0, "tax_rate": 0}, "费率已更新")
}

// TestCostAuditFindings 机检分级：四类硬检 + 费率≤0 不报 + >1 只 info。
func TestCostAuditFindings(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	labor, _ := s.SaveResource(workcost.Resource{Kind: "人工", Title: "技工", Unit: "工日", BasePrice: 150, CurrentPrice: 180})
	if _, err := s.SaveQuota(workcost.Quota{Code: "QK-1", Title: "好定额", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: labor.Code, Quantity: 0.1}}}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}
	// 悬空资源定额：含量行引用不存在的资源码。
	if _, err := s.SaveQuota(workcost.Quota{Code: "QK-BAD", Title: "坏定额", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: "GONE-1", Quantity: 0.2}}}, nil); err != nil {
		t.Fatalf("保存坏定额失败: %v", err)
	}
	pid, err := s.UpsertBillProject("机检项目", "", "", "", "", "",
		&workcost.RateSet{ManagementRate: 0, RegulatoryRate: -0.01, ProfitRate: 1.2, TaxRate: 0.09}, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	seed := []workcost.BillItem{
		{ProjectID: pid, Code: "A1", Title: "正常行", Unit: "m³", Quantity: 10, QuotaCode: "QK-1", QuotaCodes: []string{"QK-1"}},
		{ProjectID: pid, Code: "A2", Title: "缺单位", Quantity: 1, QuotaCodes: []string{}},
		{ProjectID: pid, Code: "A3", Title: "零工程量", Unit: "m", Quantity: 0, PriceOverride: 5, QuotaCodes: []string{}},
		{ProjectID: pid, Code: "A4", Title: "无价路径", Unit: "m", Quantity: 1, QuotaCodes: []string{}},
		{ProjectID: pid, Code: "A5", Title: "悬空定额行", Unit: "m", Quantity: 1, QuotaCode: "NOPE-9", QuotaCodes: []string{"NOPE-9"}},
		{ProjectID: pid, Code: "A6", Title: "坏定额行", Unit: "m³", Quantity: 1, QuotaCode: "QK-BAD", QuotaCodes: []string{"QK-BAD"}},
	}
	for i := range seed {
		if _, err := s.SaveBillItem(seed[i]); err != nil {
			t.Fatalf("播种行 %d 失败: %v", i, err)
		}
	}
	out, err := costAuditTool{a: a}.Execute(context.Background(), ormJSON(t, map[string]any{"project_id": pid}))
	if err != nil {
		t.Fatalf("cost_audit 失败: %v", err)
	}
	for _, want := range []string{
		"HardCount=5",
		"missing_identity", "unpriced_line", "dangling_quota", "dangling_resource",
		"[INFO] rate_over_one", // 利润 1.2>1 → info
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("机检回执缺 %q：\n%s", want, out)
		}
	}
	// 费率 0 与 −0.01 是显式不计取：不得出现「费率异常」类结论（终案 Grok-r3
	// 文案禁令），也不得进硬检。
	if strings.Contains(out, "费率异常") || strings.Contains(out, "[HARD] rate") {
		t.Fatalf("费率≤0 被误报：%s", out)
	}
	if !strings.Contains(out, "信息项") {
		t.Fatalf("回执缺信息项分段：%s", out)
	}
}

// TestLongdocManifestCheckFlow 清单机检七类判定与流转。
func TestLongdocManifestCheckFlow(t *testing.T) {
	// t.Chdir = gaeaCwd() 回落进程 cwd（gaea_export_test 先例）：output_path
	// 声明路径解析落在沙箱 temp 内。
	t.Chdir(t.TempDir())
	manifest := "manifest.json"
	tl := longdocManifestTool{a: &App{}}
	// init → set 带硬伤（缺正文文件 + 悬空来源 + 未映射评分点）。
	ormRun(t, tl, map[string]any{"action": "init", "manifest": manifest, "title": "修复方案"})
	out := ormRun(t, tl, map[string]any{"action": "set", "manifest": manifest, "title": "修复方案",
		"requirements": []map[string]any{{"id": "R1", "text": "响应招标评分点1"}},
		"chapters": []map[string]any{
			{"no": 1, "title": "第一章", "path": "chapters/01.md", "sources": []string{"gone.md"}},
		}})
	for _, want := range []string{"missing_chapter", "dangling_source", "unmapped_requirement"} {
		if !strings.Contains(out, want) {
			t.Fatalf("set 回执缺 %q：%s", want, out)
		}
	}
	// 构建闸：带硬伤拒绝（md 格式免 python 依赖，专测闸与组装）。
	btl := longdocBuildTool{a: &App{}}
	// 正文落盘 + 清单修正 → check 全绿。
	os.MkdirAll("chapters", 0o755)
	os.WriteFile(filepath.Join("chapters", "01.md"), []byte("# 第一章\n\n正文内容。"), 0o644)
	os.WriteFile("src.md", []byte("来源材料"), 0o644)
	ormRun(t, tl, map[string]any{"action": "set", "manifest": manifest, "title": "修复方案",
		"requirements": []map[string]any{{"id": "R1", "text": "评分点1"}},
		"chapters": []map[string]any{
			{"no": 1, "title": "第一章", "path": "chapters/01.md", "sources": []string{"src.md"}, "requirements": []string{"R1"}},
		}})
	out = ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out, "机检全绿") {
		t.Fatalf("修正后应全绿：%s", out)
	}
	// 漂移：改正文标题不回写清单 → drift 硬伤。
	os.WriteFile(filepath.Join("chapters", "01.md"), []byte("# 第一章（改版）\n\n正文。"), 0o644)
	out = ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out, "drift") {
		t.Fatalf("正文改标题不回写清单应报漂移：%s", out)
	}
	// 占位符：未闭合插值 + 占位标记。
	os.WriteFile(filepath.Join("chapters", "01.md"), []byte("# 第一章\n\n数值 {{待}} 填。【待补充】"), 0o644)
	out = ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out, "placeholder_found") {
		t.Fatalf("占位标记应拦截：%s", out)
	}
	os.WriteFile(filepath.Join("chapters", "01.md"), []byte("# 第一章\n\n数值 {{a 与 {{b}} 不平衡。"), 0o644)
	out = ormRun(t, tl, map[string]any{"action": "check", "manifest": manifest})
	if !strings.Contains(out, "unclosed_placeholder") {
		t.Fatalf("插值不平衡应拦截：%s", out)
	}
	// 构建闸：硬伤拒绝组装。
	_, err := btl.Execute(context.Background(), ormJSON(t, map[string]any{
		"manifest": manifest, "format": "md", "output_path": "out.md"}))
	if err == nil || !strings.Contains(err.Error(), "拒绝组装") {
		t.Fatalf("带硬伤构建应拒绝，got err=%v", err)
	}
	// 修复 → build 组装成功（机检 0 硬伤 + 章节序拼接 + md 直写）。
	os.WriteFile(filepath.Join("chapters", "01.md"), []byte("# 第一章\n\n正文内容。"), 0o644)
	ormRun(t, tl, map[string]any{"action": "set", "manifest": manifest, "title": "修复方案",
		"requirements": []map[string]any{{"id": "R1", "text": "评分点1"}},
		"chapters": []map[string]any{
			{"no": 1, "title": "第一章", "path": "chapters/01.md", "sources": []string{"src.md"}, "requirements": []string{"R1"}},
		}})
	out, err = btl.Execute(context.Background(), ormJSON(t, map[string]any{
		"manifest": manifest, "format": "md", "output_path": "out.md"}))
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if !strings.Contains(out, "sha256=") || !strings.Contains(out, "1 章") {
		t.Fatalf("构建回执缺 sha256/章数：%s", out)
	}
	b, _ := os.ReadFile("out.md")
	if !strings.Contains(string(b), "# 修复方案") || !strings.Contains(string(b), "# 第一章") {
		t.Fatalf("整册缺标题/章节：%s", b)
	}
}

// ── 测试小工具 ─────────────────────────────────────────────────────────────

func ormContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func ormJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func ormRun(t *testing.T, tl longdocManifestTool, args map[string]any) string {
	t.Helper()
	out, err := tl.Execute(context.Background(), ormJSON(t, args))
	if err != nil {
		t.Fatalf("longdoc_manifest %v 失败: %v", args["action"], err)
	}
	return out
}

// parseID 从建档回执里抠项目 id（"项目已建档：#3 ..."）。
func parseID(t *testing.T, out string) int64 {
	t.Helper()
	i := strings.Index(out, "#")
	if i < 0 {
		t.Fatalf("回执缺项目 id：%s", out)
	}
	j := i + 1
	for j < len(out) && out[j] >= '0' && out[j] <= '9' {
		j++
	}
	var id int64
	if _, err := fmt.Sscan(out[i+1:j], &id); err != nil {
		t.Fatalf("解析项目 id 失败（%s）: %v", out, err)
	}
	return id
}

// TestOfficeModeSystemPrompt 模式专属身份提示词解析（2026-11-21 用户拍板）：
// standard 空=现状逐字节；cost/proposal 落内置默认；配置键非空优先。
func TestOfficeModeSystemPrompt(t *testing.T) {
	if got := officeModeSystemPrompt(OfficeRunModeStandard, config.Default()); got != "" {
		t.Fatalf("standard 档应返回空（现状逐字节），got %d 字", len(got))
	}
	for _, mode := range []string{OfficeRunModeCost, OfficeRunModeProposal} {
		got := officeModeSystemPrompt(mode, config.Default())
		if got == "" {
			t.Fatalf("%s 档应落内置默认", mode)
		}
	}
	// 内容锚：造价稿点名 workcost/写前闸/编造禁令；方案稿点名清单唯一事实源/
	// 机检/评分点/四查与并行事实卡（2026-11-21 复盘经验沉淀批补锚）。
	cost := officeModeSystemPrompt(OfficeRunModeCost, config.Default())
	for _, want := range []string{"workcost", "同批编码唯一", "绝不编造定额号", "summary", "勾稽", "数据指纹", "规格档"} {
		if !strings.Contains(cost, want) {
			t.Fatalf("造价身份稿缺 %q", want)
		}
	}
	proposal := officeModeSystemPrompt(OfficeRunModeProposal, config.Default())
	for _, want := range []string{"唯一事实源", "清单漂移", "评分点全覆盖", "四查", "共享事实卡"} {
		if !strings.Contains(proposal, want) {
			t.Fatalf("方案身份稿缺 %q", want)
		}
	}
	// 配置键覆盖：非空顶替内置默认。
	cfg := config.Default()
	cfg.Office.SystemPromptCost = "自定义造价身份稿"
	if got := officeModeSystemPrompt(OfficeRunModeCost, cfg); got != "自定义造价身份稿" {
		t.Fatalf("配置键应顶替内置默认，got %q", got)
	}
	// 配置键空白串=未配置（TrimSpace 归一）。
	cfg.Office.SystemPromptCost = "   "
	if got := officeModeSystemPrompt(OfficeRunModeCost, cfg); got != officeCostSystemPromptDefault {
		t.Fatalf("空白配置键应回落内置默认")
	}
}

// TestProposalRunModeLibraryRef 方案档参照方案库接线锚（2026-11-23 知识库退役改接
// 方案库）：ModeLine 与长文技能正文都必须点名 proposal_search 与「章节骨架」
// 参照口径（写方案前按标题层级检索历史方案对照骨架与评分点）；standard/cost 档
// 不得携带该句；知识库旧锚不得回流。
func TestProposalRunModeLibraryRef(t *testing.T) {
	proposalLine := officeRunModeLine(OfficeRunModeProposal)
	for _, want := range []string{"proposal_search", "proposal_read", "proposal_save", "章节骨架"} {
		if !strings.Contains(proposalLine, want) {
			t.Fatalf("方案档 ModeLine 缺参照锚 %q", want)
		}
	}
	if strings.Contains(proposalLine, "knowledge_search") {
		t.Fatal("方案档 ModeLine 不应再引用已退役的 knowledge_search")
	}
	for _, mode := range []string{OfficeRunModeStandard, OfficeRunModeCost} {
		if line := officeRunModeLine(mode); strings.Contains(line, "proposal_search") {
			t.Fatalf("%s 档 ModeLine 不应携带方案库参照句", mode)
		}
	}
	if body := officeProposalLongformBody; !strings.Contains(body, "proposal_search") || !strings.Contains(body, "章节骨架") || strings.Contains(body, "knowledge_search") {
		t.Fatalf("长文技能正文缺「先按标题层级检索方案库参照再落清单」步")
	}
}

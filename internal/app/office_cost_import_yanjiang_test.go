package app

// office_cost_import_yanjiang_test.go —— 真实项目簿端到端：沿江包1 最终成本测算表
// （按 v5.5 样表填出的三级树表簿：33 清单 / 35 定额块 / 68 消耗行 / 41 工料机）走
// import_project 快速通道落库，库内现算与工作簿逐分对账。全走隔离库。
// 夹具与 workcost 包的金样是同一份文件（解析断言在那边，这里管落库与汇总）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

func TestCostImportProjectYanjiangWorkbook(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	cp := costProjectTool{a: a}
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })

	for _, d := range []workcost.DictItem{
		{Kind: workcost.DictSpecialty, Name: "土壤修复"},
		{Kind: workcost.DictRegion, Name: "重庆市/大足区"},
	} {
		if err := s.DictAdd(d); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "gaea", "workcost", "testdata", "沿江包1成本测算表_三级树表.xlsx"))
	if err != nil {
		t.Fatalf("读夹具失败: %v", err)
	}
	path := filepath.Join(ws, "沿江包1成本测算表.xlsx")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	base := map[string]any{"action": "import_project", "path": path,
		"specialty": "土壤修复", "region": "重庆市/大足区", "period": "2026-12"}

	// 演练：规模、取费链核对、没有跳过行。
	dry := toolRun(t, tl, mergeKeys(base, map[string]any{"dry_run": true}))
	t.Logf("演练回执：\n%s", dry)
	for _, want := range []string{
		"**项目**: 沿江包1（鑫乐洋地块）土壤及地下水修复成本测算",
		"工料机 41 条｜清单项 33 条｜消耗行 68 条｜定额块 35 个",
		"**取费**: 企管 5.00%｜规费 不计取｜利润 不计取｜增值税 3.00%",
		"**取费链核对**: 各段基数与模版口径一致",
	} {
		if !strings.Contains(dry, want) {
			t.Errorf("演练回执缺 %q：\n%s", want, dry)
		}
	}
	if strings.Contains(dry, "跳过 ") || strings.Contains(dry, "工程量为 0") || strings.Contains(dry, "将被拒收") {
		t.Errorf("实填表不该有跳过行 / 零工程量 / 拒收行：\n%s", dry)
	}
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatalf("演练不应落库，得到 %d 个项目", len(projects))
	}

	// 正式导入。
	out := toolRun(t, tl, base)
	t.Logf("导入回执：\n%s", out)
	for _, want := range []string{
		"工料机 新建 41/更新 0｜定额 新建 33/更新 2｜消耗行 68｜清单 33 行",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("导入回执缺 %q：\n%s", want, out)
		}
	}
	if i := strings.Index(out, "**未落成"); i >= 0 {
		t.Errorf("不该有未落成的行：\n%s", out[i:])
	}

	// 汇总：库内现算对工作簿（直接费 6,242,754.43、含税 6,751,538.91）。
	projects, _ := s.BillProjects()
	if len(projects) != 1 {
		t.Fatalf("应落 1 个项目，得到 %+v", projects)
	}
	sum := toolRun(t, cp, map[string]any{"action": "summary", "project_id": projects[0].ID})
	t.Logf("summary 回执：\n%s", sum)
	for _, want := range []string{"人材机直接费=6242754.43", "含税合计=6751538.91"} {
		if !strings.Contains(sum, want) {
			t.Errorf("summary 缺 %q：\n%s", want, sum)
		}
	}
	// 33 条清单的现算综合单价与工作簿（导入快照）逐条一致——逐分对账的核心断言。
	if strings.Contains(sum, "与导入快照") {
		t.Errorf("库内现算的综合单价应与工作簿逐条一致，summary 报了不一致：\n%s", sum)
	}
}

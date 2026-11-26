package app

// 工料法绑定面：五表项目工作簿解析/落库的端到端验证。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// workcostTestApp 建隔离的临时库并把工料法存储注入进去。
func workcostTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	SetWorkcostStoreForTest(workcost.Open(gdb))
	return &App{}, func() {
		ResetWorkcostStoreForTest()
		db.CloseDatabase(dir)
	}
}

// TestGaeaWorkcostProjectParseAndApplyBinding 五表工作簿绑定面端到端：
// 解析 → 落库 → 按定额核算综合单价。
func TestGaeaWorkcostProjectParseAndApplyBinding(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	fixture := filepath.Join("..", "gaea", "workcost", "testdata",
		"什邡2个污染地块施工成本测算表_补项.xlsx")

	b, err := a.GaeaWorkcostProjectParse(fixture)
	if err != nil {
		t.Skipf("夹具不可用（%v），跳过", err)
	}
	if len(b.Resources) == 0 || len(b.Lines) == 0 {
		t.Fatalf("解析结果为空：资源 %d / 消耗行 %d", len(b.Resources), len(b.Lines))
	}

	res, err := a.GaeaWorkcostProjectApply(fixture)
	if err != nil {
		t.Fatalf("落库失败: %v", err)
	}
	t.Logf("落库：资源 新增%d / 定额 新增%d / 消耗行 %d", res.ResourceNew, res.QuotaNew, res.Lines)
	if res.ResourceNew == 0 || res.QuotaNew == 0 {
		t.Fatalf("落库应产生资源与定额：%+v", res)
	}

	// 资源库可查（绑定面读路径）。
	list := a.GaeaWorkcostResourceList(workcost.KindMachine, "")
	if len(list) == 0 {
		t.Fatal("资源库应有机械设备")
	}

	// 定额可查 + 可核算。
	q := a.GaeaWorkcostQuotaGet("SF01")
	if q == nil {
		t.Fatal("应能读到定额 SF01")
	}
	c, err := a.GaeaWorkcostQuotaCompose("SF01", nil)
	if err != nil {
		t.Fatalf("核算 SF01 失败: %v", err)
	}
	if c.CompositePrice <= 0 {
		t.Fatalf("SF01 综合单价应为正，得到 %.2f", c.CompositePrice)
	}
	if c.CompositePrice != c.Subtotal {
		t.Error("综合单价必须等于人材机小计（取费不进单价）")
	}
	t.Logf("SF01「%s」综合单价 = %.2f（人工 %.2f / 材料 %.2f / 机械 %.2f）",
		q.Title, c.CompositePrice, c.LaborFee, c.MaterialFee, c.MachineFee)
}

// TestGaeaWorkcostProjectFeesBinding 取费绑定：百分数口径直通内核。
func TestGaeaWorkcostProjectFeesBinding(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	// 什邡口径：管理费 5%、利润 0、税 3%（实测产物封面自述）。
	f := a.GaeaWorkcostProjectFees(10000,
		workcost.RateSet{ManagementRate: 0.05, RegulatoryRate: 0, ProfitRate: 0, TaxRate: 0.03},
		false, 0, 0, 0)
	if diff := f.ManagementFee - 500; diff > 0.02 || diff < -0.02 {
		t.Errorf("企业管理费 = %.2f, want 500", f.ManagementFee)
	}
	if diff := f.PreTaxTotal - 10500; diff > 0.02 || diff < -0.02 {
		t.Errorf("税前合计 = %.2f, want 10500", f.PreTaxTotal)
	}
	if diff := f.Total - 10815; diff > 0.02 || diff < -0.02 {
		t.Errorf("含税总造价 = %.2f, want 10815", f.Total)
	}
}

// TestGaeaWorkcostProjectExportBinding 五表导出绑定：产出可再解析的工作簿。
func TestGaeaWorkcostProjectExportBinding(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	src := filepath.Join("..", "gaea", "workcost", "testdata",
		"旺平矿业地块污染土壤治理与修复_工程费用成本测算表_运费按距离.xlsx")
	if _, err := a.GaeaWorkcostProjectParse(src); err != nil {
		t.Skipf("夹具不可用（%v），跳过", err)
	}
	out := filepath.Join(t.TempDir(), "exported.xlsx")
	got, err := a.GaeaWorkcostProjectExport(src, out, "导出验证项目", "测试地点", "12个月")
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	if got != out {
		t.Errorf("返回路径 = %q, want %q", got, out)
	}
	// 再解析验证结构与内容可往返。
	b, err := a.GaeaWorkcostProjectParse(out)
	if err != nil {
		t.Fatalf("导出结果不可再解析: %v", err)
	}
	if b.Project != "导出验证项目" {
		t.Errorf("项目名 = %q, want 导出验证项目", b.Project)
	}
	if len(b.Resources) == 0 || len(b.Lines) == 0 {
		t.Fatalf("导出结果缺内容：资源 %d / 消耗行 %d", len(b.Resources), len(b.Lines))
	}
}

// TestGaeaWorkcostProjectExportRejectsEmptyArgs 参数缺失必须报错。
func TestGaeaWorkcostProjectExportRejectsEmptyArgs(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	if _, err := a.GaeaWorkcostProjectExport("", "out.xlsx", "", "", ""); err == nil {
		t.Fatal("缺源路径应报错")
	}
	if _, err := a.GaeaWorkcostProjectExport("src.xlsx", "", "", "", ""); err == nil {
		t.Fatal("缺输出路径应报错")
	}
}

// TestGaeaWorkcostProjectExportToWorkspace 导出到工作区 .gaea/exports：
// 前端无文件系统能力，由后端按既有产物分区约定落盘（work → .gaea/exports）。
// ga.cfg 全局态先存后还，避免污染同包其他测试。
func TestGaeaWorkcostProjectExportToWorkspace(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	src := filepath.Join("..", "gaea", "workcost", "testdata",
		"什邡2个污染地块施工成本测算表_补项.xlsx")
	if _, err := a.GaeaWorkcostProjectParse(src); err != nil {
		t.Skipf("夹具不可用（%v），跳过", err)
	}

	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })

	out, err := a.GaeaWorkcostProjectExportToWorkspace(src, "成本测算表")
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	t.Logf("导出路径: %s", out)

	// 必须落在 .gaea/exports 下（与既有导出链同分区约定）。
	if !strings.Contains(filepath.ToSlash(out), "/.gaea/exports/") {
		t.Errorf("导出路径应位于 .gaea/exports 下，得到 %q", out)
	}
	if !strings.HasSuffix(out, ".xlsx") {
		t.Errorf("导出应为 xlsx，得到 %q", out)
	}
	fi, err := os.Stat(filepath.FromSlash(out))
	if err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("导出文件为空")
	}
	t.Logf("导出文件 %d 字节", fi.Size())
}

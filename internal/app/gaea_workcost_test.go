package app

// 工料法绑定面：测算工作簿解析 / 落库的端到端验证。

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// workcostTestApp 建隔离的临时库并把工料法存储注入进去。
func workcostTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	a, _, cleanup := workcostTestAppDB(t)
	return a, cleanup
}

// workcostTestAppDB 同上，另给出底层库句柄（造价库向量索引要开在同一个库上）。
func workcostTestAppDB(t *testing.T) (*App, *sql.DB, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	SetWorkcostStoreForTest(workcost.Open(gdb))
	return &App{}, gdb, func() {
		ResetWorkcostStoreForTest()
		db.CloseDatabase(dir)
	}
}

// TestGaeaWorkcostProjectParseAndApplyBinding 测算簿绑定面端到端：解析 → 落库 →
// 按定额核算综合单价。夹具 = 沿江包1 最终测算表（三级树表：33 清单 / 35 定额块 /
// 68 消耗行 / 41 工料机）。夹具缺失即失败——旧夹具被删后这里曾静默跳过（Skipf），
// 绑定面等于没有验证。
func TestGaeaWorkcostProjectParseAndApplyBinding(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	fixture := filepath.Join("..", "gaea", "workcost", "testdata", "沿江包1成本测算表_三级树表.xlsx")

	b, err := a.GaeaWorkcostProjectParse(fixture)
	if err != nil {
		t.Fatalf("解析失败（夹具 %s）: %v", fixture, err)
	}
	if len(b.Resources) != 41 || len(b.Lines) != 68 || len(b.QuotaBlocks) != 35 || len(b.Items) != 33 {
		t.Fatalf("规模不对：资源 %d / 消耗行 %d / 定额块 %d / 清单 %d，want 41 / 68 / 35 / 33",
			len(b.Resources), len(b.Lines), len(b.QuotaBlocks), len(b.Items))
	}

	res, err := a.GaeaWorkcostProjectApply(fixture)
	if err != nil {
		t.Fatalf("落库失败: %v", err)
	}
	t.Logf("落库：资源 新增%d / 定额 新增%d 更新%d / 消耗行 %d", res.ResourceNew, res.QuotaNew, res.QuotaUpd, res.Lines)
	if len(res.Errors) > 0 || res.ResourceNew != 41 || res.QuotaNew+res.QuotaUpd != 35 || res.Lines != 68 {
		t.Fatalf("落库结果不对：%+v", res)
	}

	// 资源库可查（绑定面读路径）。
	list := a.GaeaWorkcostResourceList(workcost.KindMachine, "")
	if len(list) == 0 {
		t.Fatal("资源库应有机械设备")
	}

	// 定额可查 + 可核算：V-BJ17A 污染土壤分层清挖 = 挖机 21.28 + 普工 15 = 36.28
	// （与工作簿 AC01 的综合单价逐分一致）。
	q := a.GaeaWorkcostQuotaGet("V-BJ17A")
	if q == nil {
		t.Fatal("应能读到定额 V-BJ17A")
	}
	c, err := a.GaeaWorkcostQuotaCompose("V-BJ17A", nil)
	if err != nil {
		t.Fatalf("核算 V-BJ17A 失败: %v", err)
	}
	if diff := c.CompositePrice - 36.28; diff > 0.011 || diff < -0.011 {
		t.Fatalf("V-BJ17A 综合单价 = %.2f, want 36.28", c.CompositePrice)
	}
	if c.CompositePrice != c.Subtotal {
		t.Error("综合单价必须等于人材机小计（取费不进单价）")
	}
	t.Logf("V-BJ17A「%s」综合单价 = %.2f（人工 %.2f / 材料 %.2f / 机械 %.2f）",
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

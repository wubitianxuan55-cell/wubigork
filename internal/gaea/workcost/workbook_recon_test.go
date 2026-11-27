package workcost

// workbook_recon_test.go — 工作簿内对账的行为契约（2026-11-27 AI 反馈 #15/#16）：
// 按金额链逐级复算，断在哪一级就报哪一级；一致时不误报。另钉解析器新读出的
// 缓存值与「明细行溢出覆盖定额头」的识别。

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// reconBundle 一份自洽的测算簿解析结果：Q01 垫层 = D-A1（人工 90 + 材料 438.60
// = 528.60）+ D-A2 × 0.5（机械 0.02 × 1770 = 35.40 → 17.70）= 546.30，工程量 100。
func reconBundle() *ProjectBundle {
	return &ProjectBundle{
		Resources: []ProjectResource{
			{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日", Price: 180},
			{Code: "C20", Kind: "材料", Title: "商品混凝土", Unit: "m³", Price: 430},
			{Code: "EXC", Kind: "机械", Title: "挖掘机台班", Unit: "台班", Price: 1770},
		},
		Items: []ProjectItem{{
			Code: "Q01", Title: "垫层", Unit: "m³", ListPrice: 546.30, Quantity: 100,
			Amount: 546.30, Labor: 90, Material: 438.60, Machine: 17.70, SheetQuantity: 100,
			BillUnitPrice: 546.30, BillAmount: 54630, BillRow: 4,
		}},
		QuotaBlocks: []ProjectQuotaBlock{
			{ItemCode: "Q01", Code: "D-A1", Factor: 1, Row: 5, Amount: 528.60},
			{ItemCode: "Q01", Code: "D-A2", Factor: 0.5, Row: 10, Amount: 35.40},
		},
		Lines: []ProjectConsumption{
			{ItemCode: "Q01", QuotaCode: "D-A1", Kind: "人工", ResourceCode: "L01", Quantity: 0.5, Price: 180, Amount: 90},
			{ItemCode: "Q01", QuotaCode: "D-A1", Kind: "材料", ResourceCode: "C20", Quantity: 1.02, Price: 430, Amount: 438.60},
			{ItemCode: "Q01", QuotaCode: "D-A2", Kind: "机械", ResourceCode: "EXC", Quantity: 0.02, Price: 1770, Amount: 35.40},
		},
		Fee:            ProjectFeeParams{HasAmounts: true, DirectAmount: 54630, TotalAmount: 59547},
		BillSheetTotal: 54630,
		HasFeeSplit:    true,
	}
}

func reconKinds(b *ProjectBundle) []string {
	var out []string
	for _, is := range WorkbookRecon(b) {
		out = append(out, is.Kind)
	}
	return out
}

func hasKind(kinds []string, want string) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

// TestWorkbookReconConsistentIsQuiet 自洽的簿零告警（不误报）。
func TestWorkbookReconConsistentIsQuiet(t *testing.T) {
	if issues := WorkbookRecon(reconBundle()); len(issues) != 0 {
		t.Fatalf("自洽的测算簿不应报对账不平，得到：%+v", issues)
	}
}

// TestWorkbookReconFlagsEachBreak 链条断在哪一级，就报哪一级。
func TestWorkbookReconFlagsEachBreak(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ProjectBundle)
		want string
	}{
		{"新加的定额块没进清单行公式", func(b *ProjectBundle) {
			b.QuotaBlocks = append(b.QuotaBlocks, ProjectQuotaBlock{ItemCode: "Q01", Code: "D-A3", Factor: 1, Row: 20, Amount: 10})
			b.Lines = append(b.Lines, ProjectConsumption{ItemCode: "Q01", QuotaCode: "D-A3", Kind: "材料", ResourceCode: "C20", Quantity: 0.02, Price: 430, Amount: 8.60})
			b.QuotaBlocks[2].Amount = 8.60
		}, "item_range"},
		{"定额行求和区间没覆盖新加的明细行", func(b *ProjectBundle) {
			b.QuotaBlocks[0].Amount = 400
		}, "quota_range"},
		{"清单行金额被手填、与明细对不上", func(b *ProjectBundle) {
			b.Items[0].Amount = 500
			b.Items[0].Labor, b.Items[0].Material, b.Items[0].Machine = 80, 400, 20
		}, "recompute"},
		{"三费公式漏续", func(b *ProjectBundle) {
			b.Items[0].Machine = 0
		}, "fee_split"},
		{"清单工程量列引错清单表行", func(b *ProjectBundle) {
			b.Items[0].SheetQuantity = 120
		}, "qty_ref"},
		{"清单表引用行号错位", func(b *ProjectBundle) {
			b.Items[0].BillUnitPrice = 500
		}, "bill_ref"},
		{"清单表直接费合计区间没改", func(b *ProjectBundle) {
			b.BillSheetTotal = 50000
		}, "bill_total"},
		{"费用汇总直接费区间没改", func(b *ProjectBundle) {
			b.Fee.DirectAmount = 50000
		}, "total"},
		{"工料机有编码没单价", func(b *ProjectBundle) {
			b.Resources = append(b.Resources, ProjectResource{Code: "X1", Kind: "材料", Title: "无价材料", Unit: "kg"})
		}, "empty_price"},
	}
	for _, tc := range cases {
		b := reconBundle()
		tc.mut(b)
		kinds := reconKinds(b)
		if !hasKind(kinds, tc.want) {
			t.Errorf("%s：应报 %s，得到 %v", tc.name, tc.want, kinds)
		}
	}
	// 报出来的那一行要说人话：哪里、表内多少、应为多少。
	b := reconBundle()
	b.QuotaBlocks[0].Amount = 400
	for _, is := range WorkbookRecon(b) {
		if is.Kind == "quota_range" && (!strings.Contains(is.Text(), "Q01（第 5 行）") || !strings.Contains(is.Text(), "表内 400.00 / 应为 528.60")) {
			t.Errorf("对账行应点名清单编码、行号与两个金额：%s", is.Text())
		}
	}
}

// TestWorkbookReconNoCacheIsSilent 没重算的簿（缓存值全 0）不误报成「断链」。
func TestWorkbookReconNoCacheIsSilent(t *testing.T) {
	b := reconBundle()
	b.Items[0].Amount, b.Items[0].Labor, b.Items[0].Material, b.Items[0].Machine = 0, 0, 0, 0
	b.Items[0].BillUnitPrice, b.Items[0].BillAmount, b.Items[0].SheetQuantity = 0, 0, 0
	b.QuotaBlocks[0].Amount, b.QuotaBlocks[1].Amount = 0, 0
	for i := range b.Lines {
		b.Lines[i].Amount = 0
	}
	b.Fee = ProjectFeeParams{}
	b.BillSheetTotal = 0
	if issues := WorkbookRecon(b); len(issues) != 0 {
		t.Fatalf("未重算的簿不该报对账不平（另有「公式无缓存值」告警兜着），得到：%+v", issues)
	}
	// 但取价公式没出数、明细行金额是 0 而清单行有金额：按价格表重算能抓到。
	b2 := reconBundle()
	b2.Items[0].Amount = 100
	b2.Items[0].Labor, b2.Items[0].Material, b2.Items[0].Machine = 30, 60, 10
	if !hasKind(reconKinds(b2), "recompute") {
		t.Fatal("清单行金额与明细行重算不符应报 recompute")
	}
}

// TestParseProjectWorkbookReconFields 解析器读出对账要用的缓存值；金样夹具
// （沿江包1，真实 LibreOffice 重算过）应逐级对得上、零告警。
func TestParseProjectWorkbookReconFields(t *testing.T) {
	fixture := filepath.Join("testdata", "沿江包1成本测算表_三级树表.xlsx")
	b, err := ParseProjectWorkbook(fixture)
	if err != nil {
		t.Fatalf("解析夹具失败: %v", err)
	}
	if !b.HasFeeSplit || b.BillSheetTotal <= 0 || len(b.Overflows) != 0 {
		t.Fatalf("夹具应带三费列与清单合计、没有溢出行：%+v / %v / %v", b.HasFeeSplit, b.BillSheetTotal, b.Overflows)
	}
	var withCache int
	for _, it := range b.Items {
		if it.Amount > 0 && it.BillRow > 0 && it.BillAmount > 0 && it.SheetQuantity > 0 {
			withCache++
		}
	}
	if withCache != len(b.Items) {
		t.Fatalf("每条清单都应读到金额、清单表行号、合价与清单工程量：读到 %d / %d", withCache, len(b.Items))
	}
	if issues := WorkbookRecon(b); len(issues) != 0 {
		t.Fatalf("金样夹具的工作簿内对账应全部一致，得到：%+v", issues)
	}
	for i := range b.Lines {
		if b.Lines[i].ResourceTitle == "" || b.Lines[i].Unit == "" {
			t.Fatalf("明细行应带名称与单位（缓存值或回退价格表）：%+v", b.Lines[i])
		}
	}
}

// TestParseUnitPriceDetectsDetailOverflow 明细行超过 4 条顺着往下写、盖住了下一块
// 的定额头：定额行的编码是本簿工料机价格表里的编码 → 进 Overflows，不落成定额。
func TestParseUnitPriceDetectsDetailOverflow(t *testing.T) {
	src := filepath.Join("..", "..", "gaea", "skill", "assets", "cost-estimate-sample.xlsx")
	f, err := excelize.OpenFile(src)
	if err != nil {
		t.Fatal(err)
	}
	set := func(sheet, cell string, v any) {
		t.Helper()
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			t.Fatal(err)
		}
	}
	set("工料机价格", "A4", "L01")
	set("工料机价格", "C4", "人工")
	set("工料机价格", "D4", "普工")
	set("工料机价格", "F4", "工日")
	set("工料机价格", "G4", 180)
	set("封面", "B2", "溢出测试")
	set("综合单价", "C4", "Q01")
	set("综合单价", "C5", "D-A1")
	for i := 0; i < 4; i++ {
		r := strconv.Itoa(6 + i)
		set("综合单价", "B"+r, "人工")
		set("综合单价", "C"+r, "L01")
		set("综合单价", "F"+r, 0.1)
	}
	// 第 5 条明细写进了第 10 行：B 列仍是样表预置的「定额」。
	set("综合单价", "C10", "L01")
	set("综合单价", "F10", 0.3)
	path := filepath.Join(t.TempDir(), "overflow.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	b, err := ParseProjectWorkbook(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(b.Overflows) != 1 || b.Overflows[0].Row != 10 || !strings.Contains(b.Overflows[0].Reason, "溢出") {
		t.Fatalf("第 10 行应被认成明细溢出：%+v", b.Overflows)
	}
	for _, blk := range b.QuotaBlocks {
		if blk.Code == "L01" {
			t.Fatalf("溢出的工料机编码不应落成定额块：%+v", b.QuotaBlocks)
		}
	}
	if len(b.Lines) != 4 {
		t.Fatalf("溢出行与其后的行都不应挂进消耗行，得到 %d 行", len(b.Lines))
	}
}

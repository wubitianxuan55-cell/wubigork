package app

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// 造价族绑定面 nil 切片契约回归钉：无 error 出口的列表绑定在空库与降级路径
// 恒返回非 nil 空切片。Wails 把 nil 切片序列化成 JSON null，前端裸读
// .length/.map/.slice 即崩（v4.467.1 清单分析弹窗、v4.469.2 知识库面板两次
// 真机发作后，按契约扫对造价族全量收口的钉）。
func TestCostFamilyBindingsNonNilSlices(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	SetCostStoreForTest(cost.Open(gdb))
	SetCostProjectStoreForTest(costproject.Open(gdb))
	SetCostInquiryStoreForTest(costinquiry.Open(gdb))
	SetWorkcostStoreForTest(workcost.Open(gdb))
	t.Cleanup(func() {
		ResetCostStoreForTest()
		ResetCostProjectStoreForTest()
		ResetCostInquiryStoreForTest()
		ResetWorkcostStoreForTest()
	})
	a := &App{}

	// 空库路径：全部无 error 出口的列表绑定。
	links, _ := a.GaeaWorkcostBillQuotaLinks(1)
	checks := map[string]any{
		"GaeaCostInquiryExpiring":    a.GaeaCostInquiryExpiring(30),
		"GaeaCostInquiryAdjust":      a.GaeaCostInquiryAdjust(),
		"GaeaCostInquiryScan":        a.GaeaCostInquiryScan(),
		"GaeaCostProjectList":        a.GaeaCostProjectList(),
		"GaeaCostIndicators":         a.GaeaCostIndicators("title"),
		"GaeaPriceSources":           a.GaeaPriceSources(),
		"GaeaPriceFetches":           a.GaeaPriceFetches(),
		"GaeaPriceHistory":           a.GaeaPriceHistory("不存在"),
		"GaeaWorkcostResourceList":   a.GaeaWorkcostResourceList("", ""),
		"GaeaWorkcostResourcePrices": a.GaeaWorkcostResourcePrices(1),
		"GaeaWorkcostQuotaList":      a.GaeaWorkcostQuotaList("", ""),
		"GaeaWorkcostBillProjects":   a.GaeaWorkcostBillProjects(),
		"GaeaWorkcostBillItems":      a.GaeaWorkcostBillItems(1),
		"GaeaWorkcostBillQuotaLinks": links,
	}
	for name, v := range checks {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Slice || rv.IsNil() {
			t.Errorf("%s = %#v, want 非 nil 空切片（JSON null 崩前端）", name, v)
		}
	}

	// 空定额（无含量行）：QuotaGet.Items 非 nil——正是 v4.467.1 造价弹窗
	// 崩溃的相邻形态。
	if _, err := a.hubWorkcostStore().SaveQuota(workcost.Quota{
		Code: "Q-EMPTY", Title: "空定额", Specialty: "建筑", Unit: "m³", Status: "现行",
	}, nil); err != nil {
		t.Fatal(err)
	}
	q := a.GaeaWorkcostQuotaGet("Q-EMPTY")
	if q == nil {
		t.Fatal("GaeaWorkcostQuotaGet(Q-EMPTY) = nil")
	}
	if q.Items == nil {
		t.Error("QuotaGet.Items = nil, want 非 nil 空切片")
	}

	// 序列化层：抓取记录（含 nil 候选的降级形态）不得出现 candidates:null。
	blob, err := json.Marshal(a.GaeaPriceFetches())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), `"candidates":null`) {
		t.Errorf("序列化出现 candidates:null: %s", blob)
	}
}

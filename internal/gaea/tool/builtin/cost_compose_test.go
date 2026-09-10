package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

// toJSON 测试参数助手（cost_tools_test.go 同款形态）。
func composeToJSON(t *testing.T, v map[string]interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCostComposeBandAndEvidence 组价主链：相似条目 → 价格带+推荐价+证据链，
// 只读（不落库），并引导 cost_save 确认沉淀。
func TestCostComposeBandAndEvidence(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	// 造 4 条同单位相似样本（置信度=中），价格 3000/3200/3400/3600。
	for i, price := range []float64{3000, 3200, 3400, 3600} {
		if err := store.Save(cost.Entry{
			Name:  fmt.Sprintf("hp300-%d", i),
			Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班",
			Price: price, Spec: "300kW", Source: "历史项目", Status: "现行",
			Body: strings.Repeat("x", i+1), // body 微差避免同 name 互相覆盖
		}); err != nil {
			t.Fatal(err)
		}
	}

	cc := costCompose{}
	out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "HP300 高频液压振动锤", "unit": "台班",
	}))
	if err != nil {
		t.Fatalf("cost_compose failed: %v", err)
	}
	for _, want := range []string{
		"AI 组价", "价格带", "推荐价", "证据链", "置信度", "中位数",
		"历史项目", "cost_save",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output 缺 %q: %s", want, out)
		}
	}

	// 只读：库不被写（条目数不变）。
	if got := len(store.List()); got != 4 {
		t.Errorf("cost_compose 应只读，条目数 %d != 4", got)
	}
}

// TestCostComposeEmptyAndEdge 空库引导、description 必填、单位全排除 band=nil。
func TestCostComposeEmptyAndEdge(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := cost.Open(gdb)
	SetCostStoreForTest(store)
	defer SetCostStoreForTest(nil)

	cc := costCompose{}

	// 空库：诚实说无法组价并引导沉淀。
	out, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "挖机土方",
	}))
	if err != nil {
		t.Fatalf("empty store failed: %v", err)
	}
	if !strings.Contains(out, "相似条目") || !strings.Contains(out, "cost_save") {
		t.Errorf("空库应引导沉淀: %s", out)
	}

	// description 必填。
	if _, err := cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{})); err == nil {
		t.Error("缺 description 应报错")
	}

	// 有样本但单位过滤全排除 → band=nil 提示去单位重试。
	if err := store.Save(cost.Entry{Name: "steel-bar", Title: "钢筋", Category: "材料", Unit: "吨", Price: 4200, Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	out, err = cc.Execute(context.Background(), composeToJSON(t, map[string]interface{}{
		"description": "钢筋", "unit": "m³",
	}))
	if err != nil {
		t.Fatalf("unit-filter failed: %v", err)
	}
	if !strings.Contains(out, "去掉 unit") {
		t.Errorf("单位全排除应提示重试: %s", out)
	}
}

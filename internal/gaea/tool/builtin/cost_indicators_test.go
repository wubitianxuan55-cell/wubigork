package builtin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/db"
)

// TestCostIndicatorsTool 造价参考工具：案例（有版本）参与聚合、工作稿不参与、
// 缺单价排除、分类过滤与「样本少」标注。
func TestCostIndicatorsTool(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	store := costproject.Open(gdb)
	SetCostProjectStoreForTest(store)
	defer func() { costProjectStoreOverride = nil }()

	pid, err := store.SaveProject(costproject.Project{Name: "厂房 A"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.SaveItem(costproject.Item{ProjectID: pid, Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 1, Price: 480})
	_, _ = store.SaveItem(costproject.Item{ProjectID: pid, Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 1, Price: 500})
	_, _ = store.SaveItem(costproject.Item{ProjectID: pid, Title: "HRB400 螺纹钢", CategoryPath: "材料/土建材料/钢材", Unit: "t", Quantity: 1, Price: 0}) // 缺单价排除
	if _, err := store.SaveVersion(pid, "V1"); err != nil {
		t.Fatal(err)
	}
	// 工作稿项目（无版本）：不参与
	pid2, _ := store.SaveProject(costproject.Project{Name: "厂房 B（编制中）"})
	_, _ = store.SaveItem(costproject.Item{ProjectID: pid2, Title: "C30 商品混凝土", Unit: "m³", Quantity: 1, Price: 9999})

	// 按科目
	res, err := (costIndicators{}).Execute(t.Context(), json.RawMessage(`{"group":"title"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, "C30 商品混凝土") || strings.Contains(res, "9999") || strings.Contains(res, "螺纹钢") {
		t.Errorf("title 结果异常: %s", res)
	}
	if !strings.Contains(res, "2") || !strings.Contains(res, "490.00") {
		t.Errorf("title 聚合缺失样本/中位数: %s", res)
	}
	// 按分类 + 分类过滤
	res, err = (costIndicators{}).Execute(t.Context(), json.RawMessage(`{"group":"category","category":"材料"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, "材料") || !strings.Contains(res, "样本少") {
		t.Errorf("category 结果异常: %s", res)
	}
	// 空库
	emptyDir := t.TempDir()
	emptyDB := db.GetDatabase(emptyDir)
	SetCostProjectStoreForTest(costproject.Open(emptyDB))
	defer db.CloseDatabase(emptyDir)
	res, err = (costIndicators{}).Execute(t.Context(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, "暂无造价参考指标") {
		t.Errorf("空库应返回提示: %s", res)
	}
}

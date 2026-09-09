package costproject

// 明细行编码持久化测试（v4.178.0）：Save/List 往返 + 归一化 + 版本快照携带。

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestItemCodeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	id, err := s.SaveProject(Project{Name: "编码往返项目"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveItem(Item{ProjectID: id, Name: "dig", Title: "挖沟槽", Unit: "m³", Quantity: 10, Price: 42, Code: " a1－12 "}); err != nil {
		t.Fatal(err)
	}
	items := s.ListItems(id)
	if len(items) != 1 || items[0].Code != "A1-12" {
		t.Fatalf("items = %+v, code want A1-12（保存时归一化）", items)
	}
	// 更新路径：改价不丢编码。
	items[0].Price = 50
	if _, err := s.SaveItem(items[0]); err != nil {
		t.Fatal(err)
	}
	if got := s.ListItems(id)[0]; got.Code != "A1-12" || got.Price != 50 {
		t.Fatalf("更新后 = %+v", got)
	}
	// 版本快照 JSON 携带 code（后续恢复/对比可用）。
	if _, err := s.SaveVersion(id, ""); err != nil {
		t.Fatal(err)
	}
	v := s.ListVersions(id)[0]
	var snap []Item
	if err := json.Unmarshal([]byte(v.Snapshot), &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap) != 1 || snap[0].Code != "A1-12" {
		t.Fatalf("snapshot = %+v", snap)
	}
}

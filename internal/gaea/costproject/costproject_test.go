package costproject

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestCostProjectFlow(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	// 项目 CRUD
	id, err := s.SaveProject(Project{Name: "某厂房土建测算", ProjectType: "房建", Scale: "2 万 m²", Craft: "框架结构"})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("项目 id 为空")
	}
	if _, err := s.SaveProject(Project{Name: ""}); err == nil {
		t.Fatal("空名称应拒绝")
	}
	got, err := s.GetProject(id)
	if err != nil || got.Name != "某厂房土建测算" || got.ProjectType != "房建" {
		t.Fatalf("GetProject = %+v err=%v", got, err)
	}

	// 明细行：金额自动计算
	itemID, err := s.SaveItem(Item{ProjectID: id, Name: "c30", Title: "C30 商品混凝土", CategoryPath: "材料/土建材料/水泥及水泥制品", Unit: "m³", Quantity: 120, Price: 480})
	if err != nil {
		t.Fatal(err)
	}
	if itemID <= 0 {
		t.Fatal("明细 id 无效")
	}
	_, _ = s.SaveItem(Item{ProjectID: id, Name: "rebar", Title: "HRB400 螺纹钢", CategoryPath: "材料/土建材料/钢材", Unit: "t", Quantity: 8, Price: 3900})
	items := s.ListItems(id)
	if len(items) != 2 {
		t.Fatalf("明细行数 = %d, want 2", len(items))
	}
	if items[0].Amount != 120*480 {
		t.Errorf("amount = %v, want %v", items[0].Amount, 120*480)
	}

	// 版本快照：无明细拒绝；有明细生成不可变快照
	pid, _ := s.SaveProject(Project{Name: "空项目"})
	if _, err := s.SaveVersion(pid, ""); err == nil {
		t.Fatal("无明细保存版本应拒绝")
	}
	v, err := s.SaveVersion(id, "初稿")
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 || v.Total != 120*480+8*3900 {
		t.Errorf("version = %+v", v)
	}
	var snap []Item
	if err := json.Unmarshal([]byte(v.Snapshot), &snap); err != nil || len(snap) != 2 {
		t.Fatalf("snapshot 解析失败: %v len=%d", err, len(snap))
	}
	// 版本不可变：改明细后旧快照不变化
	items[0].Price = 500
	_, _ = s.SaveItem(items[0])
	versions := s.ListVersions(id)
	if len(versions) != 1 || versions[0].Total != 120*480+8*3900 {
		t.Errorf("versions = %+v", versions)
	}
	// 版本递增
	if _, err := s.SaveVersion(id, "修订"); err != nil {
		t.Fatal(err)
	}
	if vs := s.ListVersions(id); len(vs) != 2 || vs[0].Version != 2 {
		t.Errorf("第二版 = %+v", vs)
	}

	// 列表摘要 + 级联删除
	sums := s.ListProjects()
	if len(sums) != 2 {
		t.Fatalf("项目数 = %d, want 2", len(sums))
	}
	if sums[0].ItemCount != 2 || sums[0].VersionCount != 2 {
		t.Errorf("summary = %+v", sums[0])
	}
	if err := s.DeleteProject(id); err != nil {
		t.Fatal(err)
	}
	if len(s.ListItems(id)) != 0 || len(s.ListVersions(id)) != 0 {
		t.Fatal("级联删除未生效")
	}
	if _, err := s.GetProject(id); err == nil {
		t.Fatal("删除后应查不到")
	}
}

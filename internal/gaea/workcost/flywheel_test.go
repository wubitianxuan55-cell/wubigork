package workcost

// 企业定额自生成测试：聚类（orig_code 词干）/宁缺勿误四闸（带含量行数、同名同
// 单位、跨源、含量相异）/中位数聚合与覆盖度/ENT-* 落库整组替换。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func flyTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	s := Open(gdb)
	for _, r := range []Resource{
		{Code: "CL0001", Kind: "材料", Title: "P.O 42.5 水泥", Unit: "kg", CurrentPrice: 0.42},
		{Code: "JX0001", Kind: "机械", Title: "挖掘机 220", Unit: "台班", CurrentPrice: 1200},
		{Code: "RG0001", Kind: "人工", Title: "普工", Unit: "工日", CurrentPrice: 150},
		{Code: "CL0002", Kind: "材料", Title: "中砂", Unit: "m3", CurrentPrice: 90},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func item(code, res, kind, title, unit string, qty float64) QuotaItem {
	return QuotaItem{ResourceCode: res, Kind: kind, Title: title, Unit: unit, Quantity: qty}
}

// 三个项目对同一原码 WP-100 各带不同含量（水泥/机械有分歧、人工覆盖不全）。
func seedFlywheelCluster(t *testing.T, s *Store) {
	t.Helper()
	for _, q := range []Quota{
		{Code: "WP-100", OrigCode: "WP-100", Title: "污染土壤开挖", Unit: "m3",
			Source: "项目导入：甲项目.xlsx",
			Items:  []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.10), item("", "JX0001", "机械", "挖掘机 220", "台班", 0.030), item("", "CL0001", "材料", "P.O 42.5 水泥", "kg", 2.0)}},
		{Code: "WP-100-2", OrigCode: "WP-100", Title: "污染土壤开挖", Unit: "m3",
			Source: "项目导入：乙项目.xlsx",
			Items:  []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.20), item("", "JX0001", "机械", "挖掘机 220", "台班", 0.040)}},
		{Code: "WP-100-3", OrigCode: "WP-100", Title: "污染土壤开挖", Unit: "m3",
			Source: "项目导入：丙项目.xlsx",
			Items:  []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.30), item("", "JX0001", "机械", "挖掘机 220", "台班", 0.050), item("", "CL0002", "材料", "中砂", "m3", 0.05)}},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestQuotaClusterKey(t *testing.T) {
	// -N 消歧变体的 orig_code 保留原码（导入链溯源语义）→ 键=orig_code；
	// 原码自身的合法数字尾不受影响（WP-100 不剥成 WP——首版文本剥尾被此处
	// 语义钉死）。
	cases := []struct {
		q    Quota
		want string
	}{
		{Quota{Code: "WP-100", OrigCode: "WP-100"}, "WP-100"},
		{Quota{Code: "WP-100-2", OrigCode: "WP-100"}, "WP-100"},
		{Quota{Code: "WP-100-12", OrigCode: "WP-100"}, "WP-100"},
		{Quota{Code: "ENT-WP-100", OrigCode: "WP-100"}, "WP-100"}, // 键相同但 ENT- 前缀在聚类入口被跳过
		{Quota{Code: "MANUAL-1"}, "MANUAL-1"},                     // orig_code 空退 code（单例）
	}
	for _, c := range cases {
		if got := quotaClusterKey(c.q); got != c.want {
			t.Errorf("quotaClusterKey(%+v)=%q want %q", c.q, got, c.want)
		}
	}
}

func TestBuildQuotaFlywheelPreview_ClusterAndMedian(t *testing.T) {
	s := flyTestStore(t)
	seedFlywheelCluster(t, s)
	quotas, err := s.allQuotasWithItems()
	if err != nil {
		t.Fatal(err)
	}
	prev := BuildQuotaFlywheelPreview(quotas)
	if prev.Clusters < 1 {
		t.Fatalf("簇数 = %d，至少应扫到 WP-100 一簇", prev.Clusters)
	}
	if len(prev.Candidates) != 1 {
		t.Fatalf("候选 = %d，want 1（WP-100）；skipped=%+v", len(prev.Candidates), prev.Skipped)
	}
	c := prev.Candidates[0]
	if c.OrigCode != "WP-100" || c.NewCode != "ENT-WP-100" {
		t.Errorf("候选键/新码 = %s/%s，want WP-100/ENT-WP-100", c.OrigCode, c.NewCode)
	}
	if len(c.Members) != 3 || c.Sources != 3 {
		t.Errorf("成员/源 = %d/%d，want 3/3", len(c.Members), c.Sources)
	}
	// 中位数：人工 0.10/0.20/0.30 → 0.20；机械 0.03/0.04/0.05 → 0.04。
	wantMed := map[string]float64{"RG0001": 0.20, "JX0001": 0.04, "CL0001": 2.0, "CL0002": 0.05}
	if len(c.Rows) != len(wantMed) {
		t.Fatalf("聚合行 = %d，want %d", len(c.Rows), len(wantMed))
	}
	for _, r := range c.Rows {
		if want := wantMed[r.ResourceCode]; r.Quantity != want {
			t.Errorf("%s 中位 = %v，want %v", r.ResourceCode, r.Quantity, want)
		}
	}
	// 覆盖度：人工/机械 3/3 多数派；水泥 1/3 与中砂 1/3 少数派。
	cov := map[string]int{}
	maj := map[string]bool{}
	for _, r := range c.Rows {
		cov[r.ResourceCode] = r.Coverage
		maj[r.ResourceCode] = r.Majority
	}
	if cov["RG0001"] != 3 || cov["CL0001"] != 1 {
		t.Errorf("覆盖度 人工=%d 水泥=%d，want 3/1", cov["RG0001"], cov["CL0001"])
	}
	if !maj["RG0001"] || maj["CL0001"] || maj["CL0002"] {
		t.Errorf("多数派标注错误：人工=%v 水泥=%v 中砂=%v", maj["RG0001"], maj["CL0001"], maj["CL0002"])
	}
	// 极差如实显形。
	for _, r := range c.Rows {
		if r.ResourceCode == "RG0001" && (r.Min != 0.10 || r.Max != 0.30) {
			t.Errorf("人工极差 = %v~%v，want 0.1~0.3", r.Min, r.Max)
		}
	}
}

func TestBuildQuotaFlywheelPreview_Gates(t *testing.T) {
	// 纯函数闸门测试：不落库，直接构造定额切片。
	// 簇1 单变体带含量（无分歧）；簇2 双变体同源；簇3 双变体含量全等；
	// 簇4 双变体标题互异；簇5 单变体无含量行。全部不成立。
	quotas := []Quota{
		{Code: "A-1", OrigCode: "A-1", Title: "只有一条", Unit: "m3", Source: "项目导入：甲.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
		{Code: "B-1", OrigCode: "B-1", Title: "同源变体", Unit: "m3", Source: "项目导入：甲.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
		{Code: "B-1-2", OrigCode: "B-1", Title: "同源变体", Unit: "m3", Source: "项目导入：甲.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.2)}},
		{Code: "C-1", OrigCode: "C-1", Title: "含量全等", Unit: "m3", Source: "项目导入：甲.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
		{Code: "C-1-2", OrigCode: "C-1", Title: "含量全等", Unit: "m3", Source: "项目导入：乙.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
		{Code: "D-1", OrigCode: "D-1", Title: "标题互异甲", Unit: "m3", Source: "项目导入：甲.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
		{Code: "D-1-2", OrigCode: "D-1", Title: "标题互异乙", Unit: "m2", Source: "项目导入：乙.xlsx",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.2)}},
		{Code: "E-1", OrigCode: "E-1", Title: "空定额", Unit: "m3", Source: "项目导入：甲.xlsx"},
		{Code: "E-1-2", OrigCode: "E-1", Title: "空定额", Unit: "m3", Source: "项目导入：乙.xlsx"},
		// 无溯源的手工定额不参与聚合。
		{Code: "F-1", OrigCode: "F-1", Title: "手工定额", Unit: "m3", Source: "手工录入",
			Items: []QuotaItem{item("", "RG0001", "人工", "普工", "工日", 0.1)}},
	}
	prev := BuildQuotaFlywheelPreview(quotas)
	if len(prev.Candidates) != 0 {
		t.Fatalf("候选 = %d，want 0（全部应被闸拒）", len(prev.Candidates))
	}
	// F 簇（手工录入）也成单例簇进跳过报告——参与聚合但闸门如实拒。
	if len(prev.Skipped) != 6 {
		t.Fatalf("跳过 = %d，want 6（A/B/C/D/E/F 六簇）；got %+v", len(prev.Skipped), prev.Skipped)
	}
	reasons := map[string]string{}
	for _, sk := range prev.Skipped {
		reasons[sk.Key] = sk.Reason
	}
	for key, fragment := range map[string]string{
		"A-1": "含量行", "B-1": "同一来源", "C-1": "完全一致", "D-1": "同名同单位", "E-1": "含量行", "F-1": "含量行",
	} {
		if !strings.Contains(reasons[key], fragment) {
			t.Errorf("簇 %s 跳过原因 %q 不含 %q", key, reasons[key], fragment)
		}
	}
}

func TestQuotaFlywheelPreviewAndApply(t *testing.T) {
	s := flyTestStore(t)
	seedFlywheelCluster(t, s)

	prev, err := s.QuotaFlywheelPreview()
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Candidates) != 1 {
		t.Fatalf("候选 = %d，want 1", len(prev.Candidates))
	}

	// 无效词干 → 全部跳过（宁缺勿误；空串在入口被滤掉不算点名）。
	res, err := s.QuotaFlywheelApply([]string{"NOPE-9", "NOPE-10", " "})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Saved) != 0 || len(res.Skipped) != 2 {
		t.Fatalf("无效点名回执 = saved %d / skipped %d，want 0/2", len(res.Saved), len(res.Skipped))
	}

	// 正式点名 → ENT-WP-100 落库，含量=中位数，Source=企业聚合。
	res, err = s.QuotaFlywheelApply([]string{"WP-100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Saved) != 1 || res.Saved[0] != "ENT-WP-100" {
		t.Fatalf("saved = %v，want [ENT-WP-100]", res.Saved)
	}
	ent, err := s.GetQuota("ENT-WP-100")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Source != "企业聚合：3 项目" {
		t.Errorf("Source = %q，want 企业聚合：3 项目", ent.Source)
	}
	if len(ent.Items) != 4 {
		t.Fatalf("含量行 = %d，want 4", len(ent.Items))
	}
	byRes := map[string]float64{}
	for _, it := range ent.Items {
		byRes[it.ResourceCode] = it.Quantity
		// ResourcePrice 不钉值：SaveQuota 按保存时资源价补快照（兜底追溯用），
		// 核算（WorkcostQuotaCompose）永远按现行价重算。
	}
	if byRes["RG0001"] != 0.20 || byRes["JX0001"] != 0.04 || byRes["CL0001"] != 2.0 {
		t.Errorf("落库含量 = %+v，中位数不符", byRes)
	}

	// 重生成（飞轮进化）：再应用同词干 → 同码整组替换，不产生 ENT-WP-100-2。
	res, err = s.QuotaFlywheelApply([]string{"WP-100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Saved) != 1 || res.Saved[0] != "ENT-WP-100" {
		t.Fatalf("重生成 saved = %v，want 仍为 [ENT-WP-100]", res.Saved)
	}
	quotas, err := s.allQuotasWithItems()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, q := range quotas {
		if q.Code == "ENT-WP-100" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("ENT-WP-100 出现 %d 次，want 1（整组替换不新增）", n)
	}
	// 生成的企业定额自身不进聚类（ENT- 前缀跳过）。
	prev2, err := s.QuotaFlywheelPreview()
	if err != nil {
		t.Fatal(err)
	}
	if len(prev2.Candidates) != 1 || prev2.Candidates[0].OrigCode != "WP-100" {
		t.Fatalf("重扫候选 = %+v，want 仍只有 WP-100", prev2.Candidates)
	}
}

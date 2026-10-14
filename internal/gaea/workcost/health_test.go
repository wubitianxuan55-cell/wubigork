package workcost

// 四期 4.1/4.2 测试：五类扫描各一正例（出口判据「五类扫描各有测试」）+ 去重
// 引用重定向正确性（出口判据）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// insertResourceRaw 测试直插资源（绕过 SaveResource 的同身份 upsert——重复组
// 是历史导入产物的形态）。
func insertResourceRaw(s *Store, r Resource) error {
	_, err := s.db.Exec(`
INSERT INTO gf_resources(code, kind, title, spec, unit, base_price, current_price)
VALUES(?,?,?,?,?,?,?)`,
		r.Code, r.Kind, r.Title, r.Spec, r.Unit, r.BasePrice, r.CurrentPrice)
	return err
}

// insertQuotaItemRaw 测试直插含量行（绕过 SaveQuota 的资源索引校验——扫描面
// 要能构造悬挂引用）。
func insertQuotaItemRaw(s *Store, qi QuotaItem) error {
	_, err := s.db.Exec(`
INSERT INTO gf_quota_items(quota_code, resource_code, kind, title, spec, unit, quantity, resource_price)
VALUES(?,?,?,?,?,?,?,?)`,
		qi.QuotaCode, qi.ResourceCode, "材料", qi.Title, qi.Spec, qi.Unit, qi.Quantity, qi.ResourcePrice)
	return err
}

func healthTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	s := Open(gdb)
	// 种子：
	// - 零价资源（CL9001 现行/基准皆 0）
	// - 正常资源（CL9002 有价，被含量行引用）
	// - 重复组（CL9003/CL9004 同身份）
	// - 编码混用组（EXC 挖掘机=助记码 + M9001 挖掘机=前缀序号，同 title 不同 spec → 非重复）
	mustRes := func(r Resource) {
		t.Helper()
		if _, err := s.SaveResource(r); err != nil {
			t.Fatalf("seed %s: %v", r.Title, err)
		}
	}
	mustRes(Resource{Code: "CL9001", Kind: "材料", Title: "无价砂浆", Spec: "", Unit: "m³"})
	mustRes(Resource{Code: "CL9002", Kind: "材料", Title: "有价水泥", Spec: "散装", Unit: "kg", CurrentPrice: 0.42})
	// ④ 重复防护：V25 身份唯一索引已消灭同身份多行（SaveResource 同身份
	// upsert 也会合并）——正常态断言为「索引在位=零发现」。
	mustRes(Resource{Code: "EXC", Kind: "机械", Title: "挖掘机", Spec: "220", Unit: "台班", CurrentPrice: 1200})
	mustRes(Resource{Code: "M9001", Kind: "机械", Title: "挖掘机", Spec: "320", Unit: "台班", CurrentPrice: 1800})

	// 含量行：正常引用（CL9002）+ 缺价行 + 悬挂引用（NOPE）
	for _, qi := range []QuotaItem{
		{QuotaCode: "A1-1", ResourceCode: "CL9002", Title: "有价水泥", Unit: "kg", Quantity: 100, ResourcePrice: 0.42},
		{QuotaCode: "A1-2", ResourceCode: "CL9001", Title: "无价砂浆", Unit: "m³", Quantity: 1, ResourcePrice: 0},
		{QuotaCode: "A1-3", ResourceCode: "NOPE-404", Title: "悬挂子目", Unit: "t", Quantity: 2, ResourcePrice: 5},
	} {
		if err := insertQuotaItemRaw(s, qi); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestHealthScan_FiveCategories(t *testing.T) {
	s := healthTestStore(t)
	rep, err := s.HealthScan()
	if err != nil {
		t.Fatal(err)
	}
	// ① 零价
	foundZero := false
	for _, f := range rep.ZeroPrice {
		if f.Code == "CL9001" {
			foundZero = true
		}
	}
	if !foundZero {
		t.Fatalf("零价资源应检出 CL9001: %+v", rep.ZeroPrice)
	}
	// ② 缺价定额子目
	foundNoPrice := false
	for _, f := range rep.QuotaNoPrice {
		if f.Code == "A1-2" {
			foundNoPrice = true
		}
	}
	if !foundNoPrice {
		t.Fatalf("缺价子目应检出 A1-2: %+v", rep.QuotaNoPrice)
	}
	// ③ 悬挂引用
	foundBroken := false
	for _, f := range rep.BrokenRef {
		if f.Code == "NOPE-404" {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Fatalf("悬挂引用应检出 NOPE-404: %+v", rep.BrokenRef)
	}
	// ④ 重复防护：唯一索引在位=零发现（防护失效才告警）
	if len(rep.Duplicates) != 0 {
		t.Fatalf("唯一索引在位时不得有重复告警: %+v", rep.Duplicates)
	}
	// ⑤ 编码混用（挖掘机：EXC 助记码 vs M9001 前缀序号）
	mixOK := false
	for _, g := range rep.CodeMismatch {
		if g.Title == "挖掘机" {
			mixOK = true
		}
	}
	if !mixOK {
		t.Fatalf("编码混用应检出挖掘机: %+v", rep.CodeMismatch)
	}
	// 零价1+缺价1+悬挂1+编码混用1=4（重复防护正常态零发现）
	if rep.TotalFindings < 4 {
		t.Fatalf("总发现数: %d", rep.TotalFindings)
	}
}

func TestDedupeResources_CrossIdentitySemanticMerge(t *testing.T) {
	s := healthTestStore(t)
	// 编码未统一场景（⑤ 的处置闭环）：「挖掘机 220（EXC 助记码）」与「挖掘机
	// 320（M9001 前缀序号）」实为同一机械——作者显式裁决合并（跨身份）。
	for _, r := range []struct{ code string }{{"EXC"}, {"M9001"}} {
		if err := insertQuotaItemRaw(s, QuotaItem{QuotaCode: "A9-" + r.code, ResourceCode: r.code, Title: "挖掘机", Unit: "台班", Quantity: 2, ResourcePrice: 1500}); err != nil {
			t.Fatal(err)
		}
	}
	var idEXC, idM int64
	var codeEXC string
	if err := s.db.QueryRow(`SELECT id, code FROM gf_resources WHERE code='EXC'`).Scan(&idEXC, &codeEXC); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT id FROM gf_resources WHERE code='M9001'`).Scan(&idM); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateResourcePrice(idM, 1900, "2026年3期", "", "manual", "测试", ""); err != nil {
		t.Fatal(err)
	}

	res, err := s.DedupeResources(idEXC, []int64{idM})
	if err != nil {
		t.Fatalf("跨身份合并: %v", err)
	}
	if res.Merged != 1 || res.Kept != "EXC" {
		t.Fatalf("回执: %+v", res)
	}
	if res.RedirectedItems != 1 || res.RedirectedPrices != 1 {
		t.Fatalf("重定向计数: %+v", res)
	}
	// M9001 的含量引用已重定向到 EXC
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_quota_items WHERE resource_code='EXC'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 { // A9-EXC 原有 1 条 + A9-M9001 重定向 1 条
		t.Fatalf("EXC 应持有 2 条含量引用: %d", n)
	}
	// merged 条目已删
	if _, err := s.GetResource(idM); err == nil {
		t.Fatal("merged 条目应已删除")
	}
	// 不存在的 merge id 报错
	if _, err := s.DedupeResources(idEXC, []int64{999999}); err == nil {
		t.Fatal("不存在的 merge id 应报错")
	}
	// keep/merge 同条目拒并
	if _, err := s.DedupeResources(idEXC, []int64{idEXC}); err == nil {
		t.Fatal("merge 含保留条目本身应拒并")
	}
}

func TestCodeStyle(t *testing.T) {
	cases := map[string]string{"EXC": "mnemonic", "HP300": "mnemonic", "M001": "prefix", "P012": "prefix", "L002": "prefix"}
	for code, want := range cases {
		if got := codeStyle(code); got != want {
			t.Fatalf("codeStyle(%q)=%q want %q", code, got, want)
		}
	}
}

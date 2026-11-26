package workcost

import (
	"strings"
	"testing"
)

// ── 第二跳并入裁决（2026-11-26 造价反馈三批）─────────────────────────────
//
// 铜陵反馈 #18 要的是「并入、别再落重复行」，百锦路反馈 #3 要的是「别的地区的行
// 别被改写地区」——两者对撞。口径收窄为：并入只发生在地区缺失 / 未规范时；两边
// 都是规范地区且不同 = 按地区分行新建，原行不动。

func TestRegionMergeDecision(t *testing.T) {
	cases := []struct {
		name, incoming, existing string
		existingInDict           bool
		wantMerge                bool
		wantRegion               string
	}{
		{"导入没给地区：并入并继承原行地区", "", "皖A", true, true, "皖A"},
		{"原行没标地区：并入并补上导入地区", "皖B", "", false, true, "皖B"},
		{"原行地区不是字典值（旧写法）：并入并规范", "皖B", "安徽-铜陵", false, true, "皖B"},
		{"两边都是规范地区且不同：分行", "皖B", "皖A", true, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merge, region := regionMergeDecision(tc.incoming, tc.existing, tc.existingInDict)
			if merge != tc.wantMerge || region != tc.wantRegion {
				t.Fatalf("got merge=%v region=%q, want merge=%v region=%q", merge, region, tc.wantMerge, tc.wantRegion)
			}
		})
	}
}

func regionStore(t *testing.T, regions ...string) *Store {
	t.Helper()
	s, cleanup := newWorkcostStore(t)
	t.Cleanup(cleanup)
	seedDict(t, s, DictRegion, regions...)
	return s
}

func regionRowsOf(t *testing.T, s *Store) []Resource {
	t.Helper()
	list, err := s.ListResources("人工", "普工")
	if err != nil {
		t.Fatalf("查普工失败: %v", err)
	}
	return list
}

// 两边都是规范地区且不同：按地区分行新建，原行的地区与价都不动；真撞码
// （身份不同）照旧消歧，映射带回执。
func TestApplyProjectRegionSplitKeepsOriginalRow(t *testing.T) {
	s := regionStore(t, "皖A", "皖B")
	orig, err := s.SaveResource(Resource{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 100, Region: "皖A"})
	if err != nil {
		t.Fatalf("建旧行失败: %v", err)
	}
	// 占码者：M1 已是「瓷砖」（身份与将要导入的「水泥」不同）→ 真撞码。
	if _, err := s.SaveResource(Resource{Code: "M1", Kind: KindMaterial, Title: "瓷砖", Unit: "m²", CurrentPrice: 80, Region: "皖A"}); err != nil {
		t.Fatalf("建占码行失败: %v", err)
	}

	res := s.ApplyProjectBundle(&ProjectBundle{
		FileName: "重导入.xlsx", Project: "分行", Region: "皖B", Period: "2026-06",
		Resources: []ProjectResource{
			{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日", Price: 110},
			{Code: "M1", Kind: "材料", Title: "水泥", Unit: "t", Price: 420},
		},
	})
	if len(res.Errors) > 0 {
		t.Fatalf("不应有错误: %v", res.Errors)
	}
	if res.ResourceNew != 2 || res.ResourceUpd != 0 {
		t.Fatalf("资源 新增=%d/更新=%d, want 2/0（普工分行新建、水泥新建）", res.ResourceNew, res.ResourceUpd)
	}
	if len(res.RegionMerged) != 0 {
		t.Fatalf("地区不同的规范行不该并入: %v", res.RegionMerged)
	}
	if len(res.RegionRows) != 1 || !strings.Contains(res.RegionRows[0], "皖A") || !strings.Contains(res.RegionRows[0], "按地区分行新建") {
		t.Fatalf("应点名分行新建与同身份的另一地区行: %v", res.RegionRows)
	}
	joined := strings.Join(res.Renames, "；")
	if len(res.Renames) != 2 || !strings.Contains(joined, "L01 → L01-2") || !strings.Contains(joined, "M1 → M1-2") || !strings.Contains(joined, "id=") {
		t.Fatalf("Renames 应带两条「工作簿码 → 库内码 (id=N)」: %v", res.Renames)
	}

	rows := regionRowsOf(t, s)
	if len(rows) != 2 {
		t.Fatalf("普工应有两行（皖A 原行 + 皖B 新行），实得 %d", len(rows))
	}
	byRegion := map[string]Resource{}
	for _, r := range rows {
		byRegion[r.Region] = r
	}
	if a := byRegion["皖A"]; a.ID != orig.ID || a.Code != "L01" || a.CurrentPrice != 100 {
		t.Fatalf("原行应原样未动（id/编码/价/地区）: %+v", a)
	}
	if b := byRegion["皖B"]; b.Code != "L01-2" || b.CurrentPrice != 110 || b.ID == orig.ID {
		t.Fatalf("新行应是 皖B / L01-2 / 110: %+v", b)
	}
}

// 并入面三种：原行地区是旧写法（不是字典值）、本次导入没给地区、原行没标地区。
func TestApplyProjectRegionMergeCases(t *testing.T) {
	type tc struct {
		name         string
		dicts        []string
		seedRegion   string // 建原行时用的字典地区
		legacyRegion string // 非空=建完后直写库改成这个旧写法
		importRegion string
		wantRegion   string
		wantNote     string
	}
	cases := []tc{
		{"原行地区是旧写法：并入并规范成导入地区", []string{"皖B"}, "皖B", "安徽-铜陵", "皖B", "皖B", "安徽-铜陵→皖B"},
		{"本次导入没给地区：并入并继承原行地区", []string{"皖B"}, "皖B", "", "", "皖B", "皖B→皖B"},
		{"原行没标地区：并入并补上导入地区", []string{"皖B"}, "", "", "皖B", "皖B", "未标注→皖B"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := regionStore(t, c.dicts...)
			orig, err := s.SaveResource(Resource{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 100, Region: c.seedRegion})
			if err != nil {
				t.Fatalf("建原行失败: %v", err)
			}
			if c.legacyRegion != "" {
				// 旧写法的地区过不了今天的字典校验，只能像存量库那样直写。
				if _, err := s.db.Exec(`UPDATE gf_resources SET region=? WHERE id=?`, c.legacyRegion, orig.ID); err != nil {
					t.Fatalf("直写旧写法地区失败: %v", err)
				}
			}
			res := s.ApplyProjectBundle(&ProjectBundle{
				FileName: "重导入.xlsx", Project: "并入", Region: c.importRegion, Period: "2026-06",
				Resources: []ProjectResource{{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日", Price: 110}},
			})
			if len(res.Errors) > 0 {
				t.Fatalf("不应有错误: %v", res.Errors)
			}
			if res.ResourceNew != 0 || res.ResourceUpd != 1 || len(res.RegionRows) != 0 {
				t.Fatalf("应并入原行（新增 0 / 更新 1 / 分行 0），实得 %d/%d/%v", res.ResourceNew, res.ResourceUpd, res.RegionRows)
			}
			if len(res.RegionMerged) != 1 || !strings.Contains(res.RegionMerged[0], c.wantNote) {
				t.Fatalf("RegionMerged 应含 %q: %v", c.wantNote, res.RegionMerged)
			}
			rows := regionRowsOf(t, s)
			if len(rows) != 1 {
				t.Fatalf("普工应仍是 1 条（并入原行，不消歧），实得 %d", len(rows))
			}
			if r := rows[0]; r.ID != orig.ID || r.Code != "L01" || r.Region != c.wantRegion || r.CurrentPrice != 110 {
				t.Fatalf("并入后应是原行 id、原码、地区 %s、价 110: %+v", c.wantRegion, r)
			}
		})
	}
}

// 暂存提交走同一裁决，且暂存码 → 库内码的别名要落到含量行上（此前并入后暂存
// 定额仍引用提交码，落库即悬空）。
func TestImportCommitRegionSplitKeepsOriginalAndRemaps(t *testing.T) {
	s := regionStore(t, "四川省/遂宁市", "四川省/成都市")
	for _, code := range []string{"KWH-3", "WTR-3"} {
		if _, err := s.SaveResource(Resource{Code: code, Kind: KindMaterial, Title: "施工用电" + code[:3], Unit: "kWh", CurrentPrice: 0.8, Region: "四川省/遂宁市"}); err != nil {
			t.Fatalf("建遂宁行 %s 失败: %v", code, err)
		}
	}
	commit := func(t *testing.T, projectRegion, resCode, quotaCode, wantPreview string) *ImportWriteResult {
		t.Helper()
		imp, err := s.OpenImport("", ImportProject{Name: "百锦路", Region: projectRegion, PricePeriod: "2026-11"}, ImportControls{})
		if err != nil {
			t.Fatalf("开导入会话失败: %v", err)
		}
		defer imp.Close()
		price := 0.9
		added, err := imp.AddResources(s, []ImportResource{{Code: resCode, Kind: "材料", Title: "施工用电" + resCode[:3], Unit: "kWh", Price: &price}})
		if err != nil || added.Accepted != 1 {
			t.Fatalf("工料机应入暂存 1 条: %+v %v", added, err)
		}
		if _, err := imp.AddQuotas(s, []ImportQuota{{Code: quotaCode, Title: "某工序", Unit: "m³",
			Items: []ImportQuotaItem{{ResourceCode: resCode, Quantity: 10}}}}); err != nil {
			t.Fatalf("定额入暂存失败: %v", err)
		}
		// 提交前的落地预演（只读）：把将并入 / 将分行提前摆出来。
		preview, err := imp.LandingPreview(s)
		if err != nil {
			t.Fatalf("落地预演失败: %v", err)
		}
		if joined := strings.Join(preview, "；"); !strings.Contains(joined, wantPreview) {
			t.Fatalf("落地预演应含 %q，实得 %v", wantPreview, preview)
		}
		res, err := imp.Commit(s)
		if err != nil {
			t.Fatalf("提交失败: %v", err)
		}
		if len(res.Errors) > 0 {
			t.Fatalf("不应有写入错误: %v", res.Errors)
		}
		return res
	}
	quotaItemCode := func(t *testing.T, code string) string {
		t.Helper()
		q, err := s.GetQuota(code)
		if err != nil {
			t.Fatalf("取定额 %s 失败: %v", code, err)
		}
		if len(q.Items) != 1 {
			t.Fatalf("定额 %s 应有 1 条含量行，实得 %d", code, len(q.Items))
		}
		return q.Items[0].ResourceCode
	}

	// ① 带项目地区成都、同身份已有遂宁行：分行新建，遂宁行原样，含量行指向新行。
	res := commit(t, "四川省/成都市", "KWH", "Q1", "将按地区分行新建")
	if res.ResourcesNew != 1 || res.ResourcesUpd != 0 || len(res.RegionMerged) != 0 {
		t.Fatalf("应分行新建 1 条、不并入: %+v", res)
	}
	if len(res.RegionRows) != 1 || !strings.Contains(res.RegionRows[0], "四川省/遂宁市") {
		t.Fatalf("应点名同身份的遂宁行: %v", res.RegionRows)
	}
	if got := quotaItemCode(t, "Q1"); got != "KWH" {
		t.Fatalf("含量行应指向新行 KWH，实得 %q", got)
	}
	for code, wantRegion := range map[string]string{"KWH-3": "四川省/遂宁市", "KWH": "四川省/成都市"} {
		r, err := s.GetResourceByCode(code)
		if err != nil || r == nil || r.Region != wantRegion {
			t.Fatalf("%s 应是 %s 的行: %+v %v", code, wantRegion, r, err)
		}
	}

	// ② 没给地区：并入遂宁行并继承其地区；暂存定额里的 WTR 改指库内码 WTR-3。
	res = commit(t, "", "WTR", "Q2", "将并入既有行 WTR-3")
	if res.ResourcesNew != 0 || res.ResourcesUpd != 1 || len(res.RegionMerged) != 1 {
		t.Fatalf("应并入 1 条: %+v", res)
	}
	if len(res.Renames) != 1 || !strings.Contains(res.Renames[0], "WTR → WTR-3") {
		t.Fatalf("应报出暂存码 → 库内码映射: %v", res.Renames)
	}
	if got := quotaItemCode(t, "Q2"); got != "WTR-3" {
		t.Fatalf("并入后含量行应改指库内码 WTR-3（否则悬空），实得 %q", got)
	}
}

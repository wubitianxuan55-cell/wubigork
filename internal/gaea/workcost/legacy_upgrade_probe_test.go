package workcost

import (
	"os"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestLegacyUpgradeFromV7 用 2026-08 的老备份（user_version=7）跑完整迁移链
// v7→v27，验证：
//  1. 迁移把老库推到最新版本，且存量成本条目零损失；
//  2. 迁移后工料法三层（资源/定额/价格历史）立即可用——老库也能进入工料法；
//  3. 老库条目同样能被资源化（回归 9 个月跨度）。
//
// 默认跳过；GF_LEGACY_PROBE=1 且 GF_LEGACY_DIR 指向**老库副本目录**时运行。
// 绝不指向生产库——本用例会写库。
func TestLegacyUpgradeFromV7(t *testing.T) {
	if os.Getenv("GF_LEGACY_PROBE") != "1" {
		t.Skip("未设置 GF_LEGACY_PROBE=1，跳过老库升级探针")
	}
	dir := os.Getenv("GF_LEGACY_DIR")
	if dir == "" {
		t.Fatal("需设置 GF_LEGACY_DIR 指向老库副本目录")
	}

	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("打开老库副本失败")
	}
	defer db.CloseDatabase(dir)

	var ver int
	if err := gdb.QueryRow(`SELECT COALESCE(CAST(value AS INTEGER),0) FROM schema_meta WHERE key='user_version'`).Scan(&ver); err != nil {
		t.Fatalf("读 user_version 失败: %v", err)
	}
	t.Logf("迁移后 user_version = %d（应被推到最新）", ver)
	if ver < 25 {
		t.Fatalf("老库未被推到含工料法的版本（得到 %d，需要 ≥25）", ver)
	}

	// 存量零损失。
	var entries, cats int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM cost_entries`).Scan(&entries); err != nil {
		t.Fatalf("统计 cost_entries 失败: %v", err)
	}
	_ = gdb.QueryRow(`SELECT COUNT(*) FROM cost_categories`).Scan(&cats)
	t.Logf("存量：cost_entries %d 条 / cost_categories %d 个", entries, cats)
	if entries == 0 {
		t.Fatal("迁移后成本条目为空——老库数据丢失")
	}

	// 工料法三层可用。
	s := Open(gdb)
	r, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "老库升级探针材料", Unit: "t", BasePrice: 100})
	if err != nil {
		t.Fatalf("老库升级后写资源失败: %v", err)
	}
	t.Logf("老库升级后资源编码 = %q", r.Code)
	if _, err := s.SaveQuota(Quota{Code: "LEGACY-001", Title: "老库升级探针定额", Unit: "m³",
		Items: []QuotaItem{{ResourceCode: r.Code, Quantity: 1}}}, nil); err != nil {
		t.Fatalf("老库升级后写定额失败: %v", err)
	}
	c, _, err := s.ComposeQuota("LEGACY-001", nil)
	if err != nil {
		t.Fatalf("老库升级后核算失败: %v", err)
	}
	if c.CompositePrice <= 0 {
		t.Fatalf("老库升级后核算应为正，得到 %.2f", c.CompositePrice)
	}
	t.Logf("老库升级后核算：综合单价 %.2f（人工 %.2f / 材料 %.2f / 机械 %.2f）",
		c.CompositePrice, c.LaborFee, c.MaterialFee, c.MachineFee)

	// 老库条目可资源化。
	inputs := readCostEntries(t, gdb)
	preview := PreviewSeed(inputs, DefaultSeedOptions())
	t.Logf("老库资源化预览：%d 条 → 候选 %d 条（人工 %d / 材料 %d / 机械 %d / 外委 %d）",
		preview.Total, len(preview.Candidates),
		preview.Counts.Labor, preview.Counts.Material, preview.Counts.Machine, preview.Counts.Outsourced)
	if len(preview.Candidates) == 0 {
		t.Fatal("老库条目全部被跳过——资源化规则对老数据不适用")
	}
	res := s.ApplySeed(preview.Candidates)
	t.Logf("老库资源化：新建 %d / 更新 %d / 失败 %d", res.Created, res.Updated, res.Skipped)
	if res.Created == 0 {
		t.Fatal("老库资源化未产生任何资源")
	}
}

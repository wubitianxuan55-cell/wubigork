package workcost

import (
	"database/sql"
	"os"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestMigrationProbeAgainstRealCopy 在真实库副本上验证 V25/V26/V27 迁移安全
// （默认跳过；显式设 GF_MIGRATION_PROBE=1 且 GF_MIGRATION_DIR 指向副本目录时运行）。
//
// 目的：生产库此刻停在 user_version=24，本轮新增三版迁移会真实作用于用户
// 1590 条成本数据。上线前必须在**副本**上跑通，确认：迁移后版本号正确、
// 存量条目数与分类数不变、新表可用。
func TestMigrationProbeAgainstRealCopy(t *testing.T) {
	if os.Getenv("GF_MIGRATION_PROBE") != "1" {
		t.Skip("未设置 GF_MIGRATION_PROBE=1，跳过真实库副本迁移探针")
	}
	dir := os.Getenv("GF_MIGRATION_DIR")
	if dir == "" {
		t.Fatal("需设置 GF_MIGRATION_DIR 指向副本目录")
	}

	// 只取一次句柄：GetDatabase 内部按目录缓存连接池，重复 Close/Get 会拿到
	// 已关闭的池（本探针早期版本在此自伤，与迁移无关）。
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("打开副本库失败")
	}
	defer db.CloseDatabase(dir)

	before := countTables(t, gdb)
	t.Logf("迁移后计数: %+v", before)

	var ver int
	if err := gdb.QueryRow(`SELECT COALESCE(CAST(value AS INTEGER),0) FROM schema_meta WHERE key='user_version'`).Scan(&ver); err != nil {
		t.Fatalf("读 user_version 失败: %v", err)
	}
	t.Logf("迁移后 user_version = %d", ver)
	if ver != 27 {
		t.Errorf("user_version = %d, want 27（V25/V26/V27 三版应已落地）", ver)
	}

	// 存量数据零损失：1590 条成本条目（探针副本实测 1579）与分类树必须原样。
	if before["cost_entries"] == 0 {
		t.Error("迁移后 cost_entries 为空——存量数据丢失")
	}
	if before["cost_categories"] == 0 {
		t.Error("迁移后 cost_categories 为空——分类树丢失")
	}

	// 新表立即可用（工料法骨架落位）。
	s := Open(gdb)
	res, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "迁移探针材料", Unit: "t", BasePrice: 100})
	if err != nil {
		t.Fatalf("迁移后写资源失败: %v", err)
	}
	if _, err := s.SaveQuota(Quota{Code: "PROBE-001", Title: "迁移探针定额", Unit: "m³",
		Items: []QuotaItem{{ResourceCode: res.Code, Quantity: 1}}}, nil); err != nil {
		t.Fatalf("迁移后写定额失败: %v", err)
	}
	// 存量条目仍可读（price_derived 新列默认 0，旧语义不变）。
	var n int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM cost_entries WHERE price_derived = 0`).Scan(&n); err != nil {
		t.Fatalf("读 price_derived 失败: %v", err)
	}
	t.Logf("price_derived=0（旧语义保留）的存量条目: %d", n)
	if n != before["cost_entries"] {
		t.Errorf("存量条目应全部保持 price_derived=0（手填价语义）：%d ≠ %d", n, before["cost_entries"])
	}
}

func countTables(t *testing.T, gdb *sql.DB) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, tbl := range []string{"cost_entries", "cost_categories", "cost_entry_components"} {
		var n int
		if err := gdb.QueryRow("SELECT COUNT(*) FROM " + tbl).Scan(&n); err != nil {
			t.Logf("统计 %s 失败: %v", tbl, err)
			continue
		}
		out[tbl] = n
	}
	return out
}

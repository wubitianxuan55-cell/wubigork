package workcost

import (
	"database/sql"
	"os"
	"sort"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestSeedPreviewAgainstRealCopy 在真实库副本上跑资源化干跑，输出归类分布。
// 默认跳过；GF_MIGRATION_PROBE=1 且 GF_MIGRATION_DIR 指向副本时运行。
func TestSeedPreviewAgainstRealCopy(t *testing.T) {
	if os.Getenv("GF_MIGRATION_PROBE") != "1" {
		t.Skip("未设置 GF_MIGRATION_PROBE=1")
	}
	dir := os.Getenv("GF_MIGRATION_DIR")
	if dir == "" {
		t.Fatal("需设置 GF_MIGRATION_DIR")
	}
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("打开副本库失败")
	}
	defer db.CloseDatabase(dir)

	inputs := readCostEntries(t, gdb)
	t.Logf("库内成本条目: %d 条", len(inputs))

	p := PreviewSeed(inputs, DefaultSeedOptions())
	t.Logf("资源化候选: %d 条", len(p.Candidates))
	t.Logf("归类分布: 人工 %d / 材料 %d / 机械 %d", p.Counts.Labor, p.Counts.Material, p.Counts.Machine)
	t.Logf("跳过 %d 条，原因:", p.Total-len(p.Candidates))
	keys := make([]string, 0, len(p.Skipped))
	for k := range p.Skipped {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("   %s: %d", k, p.Skipped[k])
	}

	// 抽样打印，人工核对归类是否合理。
	shown := 0
	for _, c := range p.Candidates {
		if shown >= 25 {
			break
		}
		t.Logf("   [%s] %s | 规格=%q 单位=%s ¥%.2f | %s", c.Kind, c.Title, c.Spec, c.Unit, c.Price, c.CategoryPath)
		shown++
	}

	if len(p.Candidates) == 0 {
		t.Fatal("候选为空——资源化规则把全部条目都跳过了")
	}
}

func readCostEntries(t *testing.T, gdb *sql.DB) []SeedInput {
	t.Helper()
	rows, err := gdb.Query(`
SELECT name, title, spec, unit, price, category_path, source, region, price_date, price_type, body
FROM cost_entries`)
	if err != nil {
		t.Fatalf("读成本条目失败: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []SeedInput
	for rows.Next() {
		var in SeedInput
		if err := rows.Scan(&in.Name, &in.Title, &in.Spec, &in.Unit, &in.Price,
			&in.CategoryPath, &in.Source, &in.Region, &in.PriceDate, &in.PriceType, &in.Body); err != nil {
			continue
		}
		out = append(out, in)
	}
	return out
}

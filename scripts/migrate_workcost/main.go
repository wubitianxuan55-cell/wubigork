// Command migrate_workcost 一次性迁移：把存量成本条目按类别归入工料机资源库
// （工料法第①层）。幂等——同身份资源走更新，重复执行零新增。
//
// 用法：
//
//	go run ./scripts/migrate_workcost              # 干跑预览（默认，不写库）
//	go run ./scripts/migrate_workcost -apply       # 实际入库
//	go run ./scripts/migrate_workcost -apply -include-composite
//	                                               # 连同「综合单价/…」条目一并资源化
//
// 设计纪律：**默认干跑**。写库必须显式 -apply——存量资源化会改动用户的成本库，
// 默认安全比方便重要（与前端「先预览再入库」同一约定）。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

func main() {
	apply := flag.Bool("apply", false, "实际入库（缺省仅干跑预览）")
	includeComposite := flag.Bool("include-composite", false, "把「综合单价/…」条目也资源化（缺省保留在综合单价层）")
	flag.Parse()

	userDir := config.MemoryUserDir()
	if userDir == "" {
		fmt.Fprintln(os.Stderr, "无法确定 gaea 用户目录（config.MemoryUserDir 返回空）")
		os.Exit(1)
	}
	fmt.Printf("用户目录: %s\n", userDir)

	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		fmt.Fprintln(os.Stderr, "打开 Hephaestus.db 失败")
		os.Exit(1)
	}
	defer func() { _ = db.CloseDatabase(userDir) }()

	// 迁移是否已落地（工料法三表是否可用）。
	if err := checkWorkcostTables(gdb); err != nil {
		fmt.Fprintf(os.Stderr, "工料法表不可用（迁移未执行？）: %v\n", err)
		os.Exit(1)
	}

	inputs, err := readEntries(gdb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取成本条目失败: %v\n", err)
		os.Exit(1)
	}

	opt := workcost.SeedOptions{IncludeComposite: *includeComposite}
	preview := workcost.PreviewSeed(inputs, opt)

	fmt.Printf("\n成本条目 %d 条 → 资源候选 %d 条\n", preview.Total, len(preview.Candidates))
	fmt.Printf("归类分布：人工 %d / 材料 %d / 机械 %d / 外委 %d\n",
		preview.Counts.Labor, preview.Counts.Material, preview.Counts.Machine, preview.Counts.Outsourced)
	if n := preview.Total - len(preview.Candidates); n > 0 {
		fmt.Printf("跳过 %d 条：\n", n)
		for reason, cnt := range preview.Skipped {
			fmt.Printf("  · %s = %d\n", reason, cnt)
		}
	}

	if !*apply {
		fmt.Println("\n（干跑预览，未写库。确认无误后加 -apply 执行入库。）")
		return
	}

	store := workcost.Open(gdb)
	res := store.ApplySeed(preview.Candidates)
	fmt.Printf("\n入库结果：新建 %d / 更新 %d / 失败 %d\n", res.Created, res.Updated, res.Skipped)
	for i, e := range res.Errors {
		if i >= 10 {
			fmt.Printf("  … 其余 %d 条\n", len(res.Errors)-i)
			break
		}
		fmt.Printf("  失败：%s\n", e)
	}

	// 入库后复核：资源库规模与无价资源数。
	var total, zero int
	_ = gdb.QueryRow(`SELECT COUNT(*) FROM gf_resources`).Scan(&total)
	_ = gdb.QueryRow(`SELECT COUNT(*) FROM gf_resources WHERE current_price <= 0 AND base_price <= 0`).Scan(&zero)
	fmt.Printf("\n资源库现状：%d 个资源（其中无价 %d 个）\n", total, zero)
	fmt.Println("完成。前端「造价数据库 → 工料机 → 资源库」可查看。")
}

// checkWorkcostTables 确认工料法三表存在（迁移已落地）。
func checkWorkcostTables(gdb *sql.DB) error {
	for _, t := range []string{"gf_resources", "gf_quotas", "gf_quota_items"} {
		var n int
		if err := gdb.QueryRow(`SELECT COUNT(*) FROM ` + t).Scan(&n); err != nil {
			return fmt.Errorf("%s: %w", t, err)
		}
	}
	return nil
}

// readEntries 读取全部成本条目为资源化输入。
func readEntries(gdb *sql.DB) ([]workcost.SeedInput, error) {
	rows, err := gdb.Query(`
SELECT name, title, spec, unit, price, category_path, source, region, price_date, price_type, body
FROM cost_entries`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []workcost.SeedInput
	for rows.Next() {
		var in workcost.SeedInput
		if err := rows.Scan(&in.Name, &in.Title, &in.Spec, &in.Unit, &in.Price,
			&in.CategoryPath, &in.Source, &in.Region, &in.PriceDate, &in.PriceType, &in.Body); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

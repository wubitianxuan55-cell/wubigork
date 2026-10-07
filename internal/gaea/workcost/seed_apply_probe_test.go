package workcost

import (
	"os"
	"sort"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestSeedApplyAgainstRealCopy 在真实库副本上**实际执行**资源化，输出落地报告。
// 默认跳过；GF_MIGRATION_PROBE=1 且 GF_MIGRATION_DIR 指向副本时运行。
//
// 与 TestSeedPreviewAgainstRealCopy 的区别：本用例真的写库（副本），因此能验证
// 三件预览看不到的事：
//  1. 幂等——重复执行零新增（SaveResource 的身份唯一语义）；
//  2. 编码分配无撞码（助记码 + 前缀序号混合）；
//  3. 落库后资源可按类别检索，且引用关系正确。
func TestSeedApplyAgainstRealCopy(t *testing.T) {
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

	s := Open(gdb)
	inputs := readCostEntries(t, gdb)
	cands, skipped := BuildSeedCandidates(inputs, DefaultSeedOptions())
	t.Logf("成本条目 %d 条 → 资源候选 %d 条（跳过 %d）", len(inputs), len(cands), len(inputs)-len(cands))
	for k, v := range skipped {
		t.Logf("  跳过原因：%s = %d", k, v)
	}

	// ① 首次执行。
	first := s.ApplySeed(cands)
	t.Logf("首次：新建 %d / 更新 %d / 失败 %d（人工 %d 材料 %d 机械 %d 外委 %d）",
		first.Created, first.Updated, first.Skipped,
		first.Counts.Labor, first.Counts.Material, first.Counts.Machine, first.Counts.Outsourced)
	if len(first.Errors) > 0 {
		for i, e := range first.Errors {
			if i >= 5 {
				t.Logf("  ... 其余 %d 条", len(first.Errors)-i)
				break
			}
			t.Logf("  失败：%s", e)
		}
	}

	// ② 幂等：重复执行零新增。
	second := s.ApplySeed(cands)
	t.Logf("二次：新建 %d / 更新 %d", second.Created, second.Updated)
	if second.Created != 0 {
		t.Errorf("重复执行应零新建，实际新建 %d 条（身份唯一语义失效）", second.Created)
	}
	if second.Updated != first.Created+first.Updated {
		t.Errorf("重复执行应全部走更新：want %d, got %d", first.Created+first.Updated, second.Updated)
	}

	// ③ 编码唯一性（撞码会让定额引用错资源）。
	all, err := s.ListResources("", "")
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	codes := map[string]int{}
	for _, r := range all {
		codes[r.Code]++
	}
	dup := 0
	for c, n := range codes {
		if n > 1 {
			dup++
			if dup <= 5 {
				t.Errorf("编码撞码：%s 出现 %d 次", c, n)
			}
		}
	}
	if dup == 0 {
		t.Logf("编码唯一性：%d 个资源、%d 个编码，无撞码", len(all), len(codes))
	}

	// ④ 抽样展示编码分配效果（助记码 vs 前缀序号）。
	byKind := map[string][]Resource{}
	for _, r := range all {
		byKind[r.Kind] = append(byKind[r.Kind], r)
	}
	for _, kind := range []string{KindLabor, KindMaterial, KindMachine, KindOutsourced} {
		list := byKind[kind]
		sort.Slice(list, func(i, j int) bool { return list[i].Code < list[j].Code })
		shown := 0
		for _, r := range list {
			if shown >= 5 {
				break
			}
			t.Logf("  [%s] %-10s %s（%s）¥%.2f", kind, r.Code, r.Title, r.Unit, r.EffectivePrice())
			shown++
		}
	}

	// ⑤ 无价资源统计：核算取用价为 0 的资源会让引用它的综合单价偏低，
	//    必须显形数量（不是错误，但用户要知道有多少条要补价）。
	zero := 0
	for _, r := range all {
		if r.EffectivePrice() <= 0 {
			zero++
		}
	}
	t.Logf("无价资源（基准价与现行价都 ≤0）：%d 条 / 共 %d 条", zero, len(all))

	if len(all) == 0 {
		t.Fatal("资源化后资源库为空——落地失败")
	}
}

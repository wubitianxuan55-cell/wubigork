package skill

// cost_estimate_skill_test.go — 内置 cost-estimate 技能（2026-11-25 六表测算）：技能
// 已注册、可按需加载，正文覆盖测算口径与填法的关键条款。知识进技能不进内核：
// 这里的条款就是「填法知识已经从工具说明里挪出来」的凭据。

import (
	"strings"
	"testing"
)

func TestCostEstimateSkillRegistered(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("cost-estimate")
	if !ok {
		t.Fatal("内置 cost-estimate 技能未注册")
	}
	if sk.Scope != ScopeBuiltin || sk.RunAs != RunInline {
		t.Fatalf("应为内置 inline 技能: %+v", sk)
	}
}

func TestCostEstimateSkill_CoversKeyClauses(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("cost-estimate")
	if !ok {
		t.Fatal("技能未找到")
	}
	clauses := []string{
		"样表",             // 版式真源=样表（2026-11-25 删渲染器拍板）
		"导入契约",           // 什么不能破
		"三道门",            // recalc/audit/dry_run 验收
		"xlsx_recalc",    // 重算门
		"import_project", // 回导入口
		"import_open",    // 非标准资料走模型读表
		"dry_run",        // 先演练
		"三级行组",           // 行组结构
		"系数",             // 系数口径
		"综合单价只含人材机",      // 核心口径
		"外委并入材料桶",        // 三费归集
		"费用汇总",           // 取费只在合计层
		"YYYY-MM",        // 期数写法
		"1:N",            // 一个清单挂多条定额
		"编码」列的值，不是「自有码」列",   // 明细行填哪一列
		"没填的组、没填的定额块",       // 空组自动跳过
		"只取第一个",             // 重复挂接去重
		"人工 / 材料 / 机械 / 外委", // 类别四类
	}
	for _, c := range clauses {
		if !strings.Contains(sk.Body, c) {
			t.Fatalf("正文缺关键条款 %q", c)
		}
	}
	// 样表资产在位（版式真源随程序带一份种子；工作区可覆盖）。
	if len(CostEstimateSampleXLSX()) == 0 {
		t.Fatal("内嵌测算样表缺失")
	}
	if CostEstimateSampleOverrideRel == "" {
		t.Fatal("工作区样表覆盖路径未定义")
	}
}

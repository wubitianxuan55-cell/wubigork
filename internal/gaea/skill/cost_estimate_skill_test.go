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
		"两道门",            // recalc（回执自带审计）/dry_run 验收；xlsx_audit 不再是独立一道门（2026-11-25 B4）
		"达到交付门",          // 重算回执写明结论才算过
		"xlsx_recalc",    // 重算门
		"import_project", // 回导入口
		"import_open",    // 非标准资料走模型读表
		"dry_run",        // 先演练
		"三级树表",           // 行组结构（综合单价：清单 → 定额 → 人材机，行型看层级）
		"只有综合单价表是硬约束",    // 2026-11-27 用户拍板：其余表是参考，模板只是参考
		"换算系数",           // 定额行的换算系数必须进金额链
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
		"不得参考投资估算的价格",       // 用户规矩（2026-11-25 C2：自用户级 AGENTS.md 迁入技能，不再全板块常驻）
		"管理费、利润、规费、税金不进综合单价",
	}
	for _, c := range clauses {
		if !strings.Contains(sk.Body, c) {
			t.Fatalf("正文缺关键条款 %q", c)
		}
	}
	// 2026-11-27 用户令「优化造价技能」：新做测算优先在样表上编、样表槽位只是示例、
	// 测算表须经用户检查同意才入库、说明书随样表发放。
	for _, c := range []string{"在测算样表上编", "使用说明书", "不是数量限制", "不要只写 4 行就停", "入库红线", "用户明确说可以入库", "工作簿内对账", "疑似明细行溢出"} {
		if !strings.Contains(sk.Body, c) {
			t.Fatalf("正文缺工作流条款 %q", c)
		}
	}
	// 使用说明书随程序带一份（取代旧适配规范 v5.6）：要覆盖加行配方、结构规范、
	// 入库红线，并把 v5.6 里已被推翻的说法改正。
	manual := string(CostEstimateManual())
	for _, c := range []string{"入库红线", "用户明确说可以入库", "槽位不够", "不是数量限制", "公式模板", "三层 ROUND", "import_project",
		"疑似明细行溢出", "工作簿内对账", "附录 B", "地区只到省", "清单工程量"} {
		if !strings.Contains(manual, c) {
			t.Fatalf("使用说明书缺 %q", c)
		}
	}
	if strings.Contains(manual, "两层 ROUND（定额金额、清单小计）必须保留") {
		t.Fatal("使用说明书不应再保留 v5.6「两层 ROUND」的旧口径")
	}
	if CostEstimateManualOverrideRel == "" {
		t.Fatal("工作区说明书覆盖路径未定义")
	}
	// 样表资产在位（版式真源随程序带一份种子；工作区可覆盖）。
	if len(CostEstimateSampleXLSX()) == 0 {
		t.Fatal("内嵌测算样表缺失")
	}
	if CostEstimateSampleOverrideRel == "" {
		t.Fatal("工作区样表覆盖路径未定义")
	}
}

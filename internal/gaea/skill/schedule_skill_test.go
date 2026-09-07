package skill

import (
	"strings"
	"testing"
)

// v4.114.0 刀5：schedule-edit 内置 inline 技能——进度计划编制/调整纪律
// （AI 产建议、引擎裁决；对话即排程的工具使用规范）。

func TestScheduleEditBuiltinSkill(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("schedule-edit")
	if !ok {
		t.Fatal("built-in schedule-edit skill not found")
	}
	if sk.Scope != ScopeBuiltin {
		t.Errorf("schedule-edit Scope = %s, want builtin", sk.Scope)
	}
	if sk.RunAs != RunInline {
		t.Errorf("schedule-edit RunAs = %s, want inline", sk.RunAs)
	}
	if _, listed := find(st.List(), "schedule-edit"); !listed {
		t.Error("schedule-edit should appear in List()")
	}

	// 关键纪律条款锚点（防止后续改写丢内容）
	anchors := []string{
		"先读后写",
		"工具成功 ≠ 正确",
		"引擎裁决",
		"schedule_get",
		"schedule_apply",
		"schedule_analyze",
		"JGJ/T 121-2015", // 先分解后编网
		"工作日",            // 工期口径
		"总工期",            // 回执核对/汇报口径
		"关键",             // 关键线路汇报
		"里程碑",
		"isMilestone",
		"FS", // 搭接缺省
		// v4.115 刀6：推荐逻辑关系 + 报告模板
		"auto_chain",
		"推荐逻辑关系",
		"分析报告模板",
		"近关键",
		// v4.116 刀7：基线对比
		"set_baseline",
		"clear_baseline",
		"baselineDrift",
		// v4.117 刀8：倒排工期
		"倒排校核",
		"deadline",
		"deadlineCheck",
		// v4.122 刀2：资源与成本（ops 扩枚举 + 成本回执 + 未定价发现）
		"资源与成本",
		"元/工日",
		"upsert_resource",
		"patch_resource",
		"remove_resource",
		"set_assignments",
		"fixedCost",
		"已分配未定价",
		// v4.148 刀C：确认与回滚（diff 确认闭环）
		"确认与回滚",
		"禁止原样重发",
		"以回执为准",
		"回滚本次",
		// v4.151 双工期刀2：日历天口径（cd 用法纪律 + cdTasks 汇报清单）
		"日历天",
		"cdTasks",
	}
	// 锚点锁数：增删锚点须如实更新此期望（当前 23→31→35→37，v4.151 双工期刀2 +2）
	if len(anchors) != 37 {
		t.Fatalf("锚点锁数变化：len(anchors)=%d, want 37", len(anchors))
	}
	body := sk.Body
	for _, a := range anchors {
		if !strings.Contains(body, a) {
			t.Errorf("schedule-edit body 缺锚点 %q", a)
		}
	}
}

// v4.122.0 刀2：schedule-edit「资源与成本」分节——三类资源口径、费率单位纪律、
// 先建资源再挂分配、成本改动回执核对要求。
func TestScheduleEditResourceCostSection(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("schedule-edit")
	if !ok {
		t.Fatal("built-in schedule-edit skill not found")
	}
	// 分节位置：调整流程之后、基线对比之前
	iSection := strings.Index(sk.Body, "## 资源与成本")
	iAdjust := strings.Index(sk.Body, "## 调整流程")
	iBaseline := strings.Index(sk.Body, "## 基线对比")
	if iSection < 0 || iAdjust < 0 || iBaseline < 0 || !(iAdjust < iSection && iSection < iBaseline) {
		t.Fatalf("资源与成本分节缺失或位置不对：adjust=%d section=%d baseline=%d", iAdjust, iSection, iBaseline)
	}
	for _, a := range []string{
		"work 工时", "material 材料", "cost 成本", // 三类资源口径
		"元/工日", "元/单位", // 费率单位纪律
		"先建资源", "set_assignments", // 先建资源再挂分配
		"totalCost", "taskCosts", // 成本改动回执核对
		"已分配未定价", // 未定价点破
	} {
		if !strings.Contains(sk.Body, a) {
			t.Errorf("schedule-edit 资源成本分节缺锚点 %q", a)
		}
	}
	if !strings.Contains(sk.Description, "成本") {
		t.Errorf("Description 应含资源成本触发词：%s", sk.Description)
	}
}

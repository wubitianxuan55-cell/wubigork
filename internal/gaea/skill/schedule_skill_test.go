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
		"工作日",           // 工期口径
		"总工期",           // 回执核对/汇报口径
		"关键",             // 关键线路汇报
		"里程碑",
		"isMilestone",
		"FS", // 搭接缺省
		// v4.115 刀6：推荐逻辑关系 + 报告模板
		"auto_chain",
		"推荐逻辑关系",
		"分析报告模板",
		"近关键",
	}
	body := sk.Body
	for _, a := range anchors {
		if !strings.Contains(body, a) {
			t.Errorf("schedule-edit body 缺锚点 %q", a)
		}
	}
}

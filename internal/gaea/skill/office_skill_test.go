package skill

import (
	"strings"
	"testing"
)

// U1（docs/gaea-dsh-univer-office-distill-plan-2026-09.md §5）：office-edit
// 内置 inline 技能——蒸馏 dsh-univer-office 的编辑纪律（verify 循环「成功≠正确」、
// 状态机、错误码路由）+ 思维导图 / .gbase.json 多维表视图的 gaea 原生口径。

func TestOfficeEditBuiltinSkill(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("office-edit")
	if !ok {
		t.Fatal("built-in office-edit skill not found")
	}
	if sk.Scope != ScopeBuiltin {
		t.Errorf("office-edit Scope = %s, want builtin", sk.Scope)
	}
	if sk.RunAs != RunInline {
		t.Errorf("office-edit RunAs = %s, want inline (纪律由主代理执行，不派子代理)", sk.RunAs)
	}
	if _, listed := find(st.List(), "office-edit"); !listed {
		t.Error("office-edit should appear in List() so it reaches the slash menu")
	}

	// 关键纪律条款必须在场（蒸馏契约的锚点，防止后续改写丢内容）
	anchors := []string{
		"先读后写",
		"工具成功 ≠ 正确",
		"宁拒不误改",
		"soffice",           // 渲染取证通道
		"逐 run",             // docx/pptx 防摊平
		".gbase.json",       // 多维表视图口径
		`"version":1`,       // schema 锚点
		"Error [FORMAT_",    // 错误码路由
		"思维导图",             // markdown 大纲口径
		"openpyxl",
		"python-pptx",
	}
	for _, a := range anchors {
		if !strings.Contains(sk.Body, a) {
			t.Errorf("office-edit body missing anchor %q", a)
		}
	}
}

// TestSkillIndexWithinBudget：内置技能全量进索引后不得触发截断（③ 不超限校验）。
func TestSkillIndexWithinBudget(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	idx := ApplyIndex("", st.List())
	if strings.Contains(idx, "(truncated") {
		t.Error("builtin skills index exceeded IndexMaxChars and got truncated")
	}
}

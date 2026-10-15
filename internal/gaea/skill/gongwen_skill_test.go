package skill

// 阶段十一 11.6 GB/T 9704 办公规范包测试：技能可读、关键规范点在正文、
// 与 office-edit 纪律互补不重复。

import (
	"strings"
	"testing"
)

func TestGongwenSkillRegistered(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("office-gongwen")
	if !ok {
		t.Fatal("内置 office-gongwen 技能未注册")
	}
	if sk.Scope != ScopeBuiltin || sk.RunAs != RunInline {
		t.Fatalf("应为内置 inline 技能: %+v", sk)
	}
}

func TestGongwenSkill_CoversKeyClauses(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	sk, ok := st.Read("office-gongwen")
	if !ok {
		t.Fatal("技能未找到")
	}
	// GB/T 9704-2012 关键规范点逐项在正文（出口口径：模板技能化=规范可按需加载）
	clauses := []string{
		"GB/T 9704-2012",       // 依据版本标注
		"37", "35", "28", "26", // 页边距四值
		"仿宋", "3 号", // 正文字体字号
		"2 号",                    // 标题字号
		"一、", "（一）", "1.", "（1）", // 结构层次四级
		"〔",        // 发文字号六角括号
		"右空四字",     // 成文日期编排
		"eastAsia", // python-docx 中文字体落地要点
		"常见错误",     // 错误清单
	}
	for _, c := range clauses {
		if !strings.Contains(sk.Body, c) {
			t.Fatalf("正文缺关键规范点 %q", c)
		}
	}
}

func TestGongwenSkill_ComplementsOfficeEdit(t *testing.T) {
	st := New(Options{HomeDir: t.TempDir()})
	gw, ok := st.Read("office-gongwen")
	if !ok {
		t.Fatal("技能未找到")
	}
	oe, ok := st.Read("office-edit")
	if !ok {
		t.Fatal("office-edit 未找到")
	}
	// 分工：gongwen 管公文版式规范（领域知识），office-edit 管编辑纪律（通用机
	// 制）——gongwen 引用 office-edit 的验证闭环而不复制其铁律全文。
	if !strings.Contains(gw.Body, "office-edit") {
		t.Fatal("gongwen 应引用 office-edit 纪律（不复制铁律）")
	}
	if strings.Count(gw.Body, "先读后写") > 1 {
		t.Fatal("gongwen 不应复制 office-edit 铁律全文（纪律单源在 office-edit）")
	}
	if !strings.Contains(oe.Description, "纪律总纲") {
		t.Fatal("office-edit 应仍是纪律总纲")
	}
}

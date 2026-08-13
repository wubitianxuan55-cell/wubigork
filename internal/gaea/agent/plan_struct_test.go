package agent

import (
	"testing"
)

func TestParsePlanValidJSON(t *testing.T) {
	raw := `{"goal":"整理本周周报","steps":[{"title":"读取资料","detail":"读取工作区 docs 下的周报素材","resources":["docs/周报.md"],"tools":["read_file"],"deliverable":"周报初稿"},{"title":"校验数据","tools":["xlsx_edit"]}],"questions":["周报统计口径按本月还是本季？"]}`
	p := ParsePlan(raw)
	if p == nil {
		t.Fatal("ParsePlan = nil, want structured")
	}
	if p.Goal != "整理本周周报" || len(p.Steps) != 2 {
		t.Fatalf("ParsePlan = %+v", p)
	}
	if len(p.Steps[0].Resources) != 1 || p.Steps[0].Resources[0] != "docs/周报.md" {
		t.Fatalf("resources = %v", p.Steps[0].Resources)
	}
	if len(p.Questions) != 1 {
		t.Fatalf("questions = %v", p.Questions)
	}
}

func TestParsePlanFencedJSON(t *testing.T) {
	raw := "```json\n{\"goal\":\"生成成本表\",\"steps\":[{\"title\":\"读取成本数据\",\"tools\":[\"read_file\"]}]}\n```"
	p := ParsePlan(raw)
	if p == nil || p.Goal != "生成成本表" || len(p.Steps) != 1 {
		t.Fatalf("ParsePlan(fenced) = %+v", p)
	}
}

func TestParsePlanInvalidFallbackNil(t *testing.T) {
	for _, raw := range []string{"", "   ", "不是 JSON", "{}", `{"goal":"  ","steps":[]}`} {
		if p := ParsePlan(raw); p != nil {
			t.Fatalf("ParsePlan(%q) = %+v, want nil", raw, p)
		}
	}
}

func TestParsePlanCleansEmptyFields(t *testing.T) {
	raw := `{"goal":"测试","steps":[{"title":"","detail":"空标题应剔除"},{"title":" 有效步骤 ","resources":["  ","docs/a.md"],"tools":[""]}],"questions":[" ","真的问题"]}`
	p := ParsePlan(raw)
	if p == nil {
		t.Fatal("ParsePlan = nil")
	}
	if len(p.Steps) != 1 || p.Steps[0].Title != "有效步骤" {
		t.Fatalf("steps = %+v", p.Steps)
	}
	if len(p.Steps[0].Resources) != 1 || p.Steps[0].Resources[0] != "docs/a.md" {
		t.Fatalf("resources = %v", p.Steps[0].Resources)
	}
	if len(p.Questions) != 1 || p.Questions[0] != "真的问题" {
		t.Fatalf("questions = %v", p.Questions)
	}
}

func TestRenderPlanMarkdown(t *testing.T) {
	plan := ParsePlan(`{"goal":"整理成本测算","steps":[{"title":"读取数据","detail":"读取成本表","resources":["成本.xlsx"],"tools":["read_file"],"deliverable":"成本明细"}],"questions":["口径？"]}`)
	md := RenderPlanMarkdown(plan)
	if md == "" {
		t.Fatal("RenderPlanMarkdown empty")
	}
	for _, want := range []string{"整理成本测算", "读取数据", "成本.xlsx", "read_file", "成本明细", "口径？"} {
		if !containsStr(md, want) {
			t.Fatalf("RenderPlanMarkdown 缺少 %q:\n%s", want, md)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

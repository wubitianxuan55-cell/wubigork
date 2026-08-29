package standard

import "testing"

func TestLintTextComplete(t *testing.T) {
	head := "XX省人民政府文件\nX政发〔2026〕3号\n关于××工作的通知"
	body := "各市、州人民政府：\n……（正文）……\nXX省人民政府\n2026年8月29日\n（加盖印章）\n（此件公开发布）\n抄送：省委办公厅。"
	r := LintText("x.md", head, body)
	if !r.Passed {
		t.Fatalf("完整红头应通过，got %+v", r.Issues)
	}
}

func TestLintTextMissing(t *testing.T) {
	r := LintText("x.md", "随便一段文字", "正文内容")
	if r.Passed {
		t.Fatal("缺要素文档不应通过")
	}
	found := map[string]bool{}
	for _, it := range r.Issues {
		found[it.Element] = it.Found
	}
	if found["发文字号（如：×发〔2026〕3号）"] {
		t.Error("应检出缺发文字号")
	}
	if found["版记（抄送/印发机关/印发日期）"] {
		t.Error("应检出缺版记")
	}
}

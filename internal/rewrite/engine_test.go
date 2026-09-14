package rewrite

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// TestNormalizeRequest 归一与前置校验（MuMu schemas/regeneration.py + L4489）。
func TestNormalizeRequest(t *testing.T) {
	// custom 空指令 → 报错
	if _, err := NormalizeRequest(types.RewriteRequest{Source: types.RewriteSourceCustom}); err == nil {
		t.Fatalf("custom 空指令应报错")
	}
	// suggestions 无索引 → 报错
	if _, err := NormalizeRequest(types.RewriteRequest{Source: types.RewriteSourceSuggestions}); err == nil {
		t.Fatalf("suggestions 无索引应报错")
	}
	// mixed 两者皆空 → 报错
	if _, err := NormalizeRequest(types.RewriteRequest{Source: types.RewriteSourceMixed}); err == nil {
		t.Fatalf("mixed 全空应报错")
	}
	// 非法来源 → 报错
	if _, err := NormalizeRequest(types.RewriteRequest{Source: "bogus", CustomInstructions: "x"}); err == nil {
		t.Fatalf("非法来源应报错")
	}
	// 空来源 → custom；字数缺省 3000；未知 focus 丢弃；mode 缺省 whole
	req, err := NormalizeRequest(types.RewriteRequest{
		CustomInstructions: "收紧节奏",
		FocusAreas:         []string{"pacing", "bogus"},
	})
	if err != nil {
		t.Fatalf("合法请求不应报错: %v", err)
	}
	if req.Source != types.RewriteSourceCustom || req.Mode != types.RewriteModeWhole ||
		req.TargetWordCount != DefaultTargetWords || len(req.FocusAreas) != 1 {
		t.Fatalf("归一不对: %+v", req)
	}
	// 字数钳制 [500,10000]
	r2, _ := NormalizeRequest(types.RewriteRequest{CustomInstructions: "x", TargetWordCount: 99})
	if r2.TargetWordCount != MinTargetWords {
		t.Fatalf("下限钳制不对: %d", r2.TargetWordCount)
	}
	r3, _ := NormalizeRequest(types.RewriteRequest{CustomInstructions: "x", TargetWordCount: 99999})
	if r3.TargetWordCount != MaxTargetWords {
		t.Fatalf("上限钳制不对: %d", r3.TargetWordCount)
	}
}

// TestBuildModificationInstruction 四段固定顺序（MuMu L110-L179）：
// 越界索引静默跳过、未知 focus 丢弃、保留元素逐条展开。
func TestBuildModificationInstruction(t *testing.T) {
	req := types.RewriteRequest{
		Source:             types.RewriteSourceMixed,
		SuggestionIdxs:     []int{0, 2, 9}, // 9 越界静默跳过
		CustomInstructions: "结尾收得更快",
		FocusAreas:         []string{"pacing", "dialogue"},
		Preserve: &types.PreserveConfig{
			Structure:       true,
			Dialogues:       []string{"「我知道了。」"},
			PlotPoints:      []string{"铜匣开启"},
			CharacterTraits: true,
		},
	}
	suggestions := []string{"建议一", "建议二", "建议三"}
	out := BuildModificationInstruction(req, suggestions)

	for _, want := range []string{
		"# 章节修改指令",
		"## 需要改进的问题（来自AI分析）：", "1. 建议一", "2. 建议三",
		"## 用户自定义修改要求：", "结尾收得更快",
		"## 重点优化方向：", "节奏把控", "对话质量",
		"## 必须保留的元素：", "整体结构和情节框架", "* 「我知道了。」", "* 铜匣开启", "性格特征和行为模式",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("指令缺 %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "建议二") {
		t.Fatalf("未选中的建议不应出现:\n%s", out)
	}
	// 纯 custom：不含建议段
	custom := BuildModificationInstruction(types.RewriteRequest{
		Source: types.RewriteSourceCustom, CustomInstructions: "x",
	}, suggestions)
	if strings.Contains(custom, "需要改进的问题") {
		t.Fatalf("custom 来源不应有建议段:\n%s", custom)
	}
}

// TestCleanRewriteOutput 前缀剥离（8 前缀命中即 break）+ 成对引号剥离。
func TestCleanRewriteOutput(t *testing.T) {
	cases := []struct{ in, want string }{
		{"重写后：\n正文", "正文"},
		{"以下是重写后的内容:\n正文", "正文"},
		{"重写内容：正文", "正文"},
		{"“正文内容”", "正文内容"},
		{"「正文内容」", "正文内容"},
		{"『正文内容』", "正文内容"},
		{"  正文  ", "正文"},
		{"普通正文保持原样", "普通正文保持原样"},
	}
	for _, tc := range cases {
		if got := CleanRewriteOutput(tc.in); got != tc.want {
			t.Fatalf("Clean(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	// 非成对引号不剥
	if got := CleanRewriteOutput("「未闭合"); got != "「未闭合" {
		t.Fatalf("非成对引号不应剥离: %q", got)
	}
}

// TestComputeDiff 篇幅与相似度统计。
func TestComputeDiff(t *testing.T) {
	same := ComputeDiff("一模一样的正文", "一模一样的正文")
	if same.Similarity != 100 || same.Change != 0 {
		t.Fatalf("相同文本应 100 相似: %+v", same)
	}
	d := ComputeDiff(strings.Repeat("长", 100), strings.Repeat("长", 50)+"新结尾")
	if d.OriginalLen != 100 || d.NewLen != 53 || d.Change != -47 {
		t.Fatalf("篇幅统计不对: %+v", d)
	}
	if d.ChangePercent != -47 {
		t.Fatalf("变化百分比不对: %v", d.ChangePercent)
	}
	if d.Similarity < 0 || d.Similarity > 100 {
		t.Fatalf("相似度应在 0-100: %v", d.Similarity)
	}
}

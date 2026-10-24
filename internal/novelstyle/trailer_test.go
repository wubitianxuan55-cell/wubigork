package novelstyle

import (
	"os"
	"testing"
)

// ── 规则 12：章末升华/预告腔（oh-story T2 门禁 F 蒸馏，位置敏感）────────
//
// 判据是位置：同样的话出现在章中可能是正常叙述，章末窗口内才算收尾升华。

func TestRuleTrailerEnding_TailWindowHit(t *testing.T) {
	text := "他收拾好行囊，把灯吹灭。" // 前文填充，保证命中段落在窗口内
	for i := 0; i < 40; i++ {
		text += "这一夜风声不断，远山隐约传来更声。"
	}
	text += "他知道，命运的齿轮开始转动了，而这一切，才刚刚开始。"

	issues := ruleTrailerEnding(text, []rune(text))
	if len(issues) != 1 {
		t.Fatalf("章末窗口内升华应命中 1 条（advisory 封顶），got %d", len(issues))
	}
	if issues[0].Severity != "S3" {
		t.Fatalf("章末升华应为 S3 advisory, got %s", issues[0].Severity)
	}
	if issues[0].Start < len([]rune(text))-trailerEndingWindow {
		t.Fatalf("命中 span 应落在章末窗口内: start=%d", issues[0].Start)
	}
}

func TestRuleTrailerEnding_MidTextNotFlagged(t *testing.T) {
	// 「命运的齿轮」出现在章首叙事中——不是收尾，不判。
	text := "他忽然明白，命运的齿轮开始转动了。"
	for i := 0; i < 80; i++ {
		text += "两人一路无话，只有马蹄声敲着青石板，渐行渐远。"
	}
	text += "她把信折好塞回袖中，转身进了院子，轻轻带上了门。"

	issues := ruleTrailerEnding(text, []rune(text))
	if len(issues) != 0 {
		t.Fatalf("章中的同款措辞不应判章末升华, got %d: %+v", len(issues), issues)
	}
}

func TestRuleTrailerEnding_EmptyPatternTable(t *testing.T) {
	// 覆盖表只含其余两族（trailer 空）时规则应安静返回（不过滤不炸）。
	saved := patterns.Load()
	patterns.Store(&compiledPatterns{})
	defer patterns.Store(saved)

	if got := ruleTrailerEnding("命运的齿轮开始转动。", []rune("命运的齿轮开始转动。")); got != nil {
		t.Fatalf("空 trailer 表应返回 nil, got %d 条", len(got))
	}
}

func TestClassifyTasteLayer_TrailerIsNarrative(t *testing.T) {
	// 门禁 F 属叙述层（结尾收束方式），不归表达层。
	if got := classifyTasteLayer("章末升华/预告腔：结尾用抒情总结陈词收束（advisory）"); got != "narrative" {
		t.Fatalf("章末升华应归类 narrative, got %q", got)
	}
}

func TestLoadPatternsFile_TrailerOnlyOverrideAccepted(t *testing.T) {
	// 覆盖表允许只带 trailerEndings 一族（不再因「另两族为空」而整表拒绝）。
	dir := t.TempDir()
	path := dir + "/patterns.json"
	content := `{"trailerEndings":["命运的齿轮"]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写覆盖文件: %v", err)
	}
	saved := patterns.Load()
	defer patterns.Store(saved)

	if err := LoadPatternsFile(path); err != nil {
		t.Fatalf("trailer-only 覆盖表应被接受: %v", err)
	}
	if got := ruleTrailerEnding("尾声：命运的齿轮，缓缓咬合。", []rune("尾声：命运的齿轮，缓缓咬合。")); len(got) != 1 {
		t.Fatalf("覆盖后的内置表应只剩覆盖内容并命中, got %d", len(got))
	}
}

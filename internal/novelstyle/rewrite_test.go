package novelstyle

import (
	"strings"
	"testing"
)

// 一段充斥 AI 高频词的文本（眼帘/微微上扬/缓缓/仿佛/由此带出分数）。
const aiRewriteText = `他的眼帘微微上扬，内心充满了震撼，仿佛整个世界都静止了。

他缓缓抬起头，眸光流转，嘴角勾起一抹若有若无的笑意。

然而，他定睛看向远方，须臾之后，终于下定决心。`

func TestDeSlopRewrite_ReducesScoreAndReplacesAIWords(t *testing.T) {
	inScore, err := ScoreTextNoRef(aiRewriteText)
	if err != nil {
		t.Fatalf("ScoreTextNoRef(aiRewriteText) 失败: %v", err)
	}
	out, rep, err := DeSlopRewrite(aiRewriteText, inScore)
	if err != nil {
		t.Fatalf("DeSlopRewrite 失败: %v", err)
	}
	if rep.BeforeScore <= 0 {
		t.Fatalf("输入应是非零 AI 味分，got %d", rep.BeforeScore)
	}
	if rep.AfterScore >= rep.BeforeScore {
		t.Fatalf("去味后分数应下降: before=%d after=%d", rep.BeforeScore, rep.AfterScore)
	}
	// AI 高频词应被替换，不再以原形出现
	for _, ban := range []string{"眼帘", "微微上扬", "缓缓", "仿佛", "眸光流转", "嘴角勾起", "定睛", "须臾"} {
		if strings.Contains(out, ban) {
			t.Fatalf("去味后仍含 AI 词 %q：%s", ban, out)
		}
	}
	if len(rep.Changes) == 0 {
		t.Fatalf("应至少有一个改写条目")
	}
}

func TestDeSlopRewrite_PreservesCleanText(t *testing.T) {
	clean := `他攥紧了拳头，指节发白。她笑了笑，没说话。`
	before, err := ScoreTextNoRef(clean)
	if err != nil {
		t.Fatalf("打分失败: %v", err)
	}
	out, rep, err := DeSlopRewrite(clean, before)
	if err != nil {
		t.Fatalf("DeSlopRewrite 失败: %v", err)
	}
	if strings.TrimSpace(out) != strings.TrimSpace(clean) {
		t.Fatalf("干净文本不应被改写：\nbefore=%s\nafter=%s", clean, out)
	}
	if len(rep.Changes) != 0 {
		t.Fatalf("干净文本不应有改写条目，got %d", len(rep.Changes))
	}
}

func TestDeSlopRewrite_PunctCollapse(t *testing.T) {
	// 连串省略号/感叹号应归一，且不影响正文字词。
	in := "他说……然后沉默了……很震撼！！！"
	out, _, err := DeSlopRewrite(in, nil)
	if err != nil {
		t.Fatalf("DeSlopRewrite 失败: %v", err)
	}
	if strings.Contains(out, "！！！") {
		t.Fatalf("连串感叹号未归一：%s", out)
	}
	if !strings.Contains(out, "他说") {
		t.Fatalf("正文被误改：%s", out)
	}
}

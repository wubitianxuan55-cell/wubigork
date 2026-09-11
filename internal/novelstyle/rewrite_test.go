package novelstyle

import (
	"os"
	"path/filepath"
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

// ── v4.225 词表数据资产（规范知识出内核）────────────────────────────────

// TestWordsEmbeddedDefaultsGuard 漂移守卫：内置 words.json 与搬出前的 Go 表
// 同规模（27 组替换 + 19 个黑名单词），防误删误改。
func TestWordsEmbeddedDefaultsGuard(t *testing.T) {
	w := currentWords()
	if len(w.Replacements) != 27 {
		t.Fatalf("内置替换表应为 27 组, got %d", len(w.Replacements))
	}
	if len(w.Blacklist) != 19 {
		t.Fatalf("内置黑名单应为 19 词, got %d", len(w.Blacklist))
	}
	if w.Replacements["眸光流转"] != "目光流转" {
		t.Fatalf("内置表内容漂移: %+v", w.Replacements)
	}
}

// TestLoadWordsFileOverride 覆盖文件整体替换词表 + 分词词表重建：
// 替换表生效（新词被替换）、黑名单生效（新词参与打分）、文件不存在静默保默认。
func TestLoadWordsFileOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "words.json")
	override := `{
		"replacements": {"思维缜密": "心思周密"},
		"blacklist": ["思维缜密", "眼帘"]
	}`
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// 恢复内置词表（全局表会污染同包后续用例）。
		words.Store(mustLoadEmbeddedWords())
		rebuildVocab()
	})

	if err := LoadWordsFile(path); err != nil {
		t.Fatalf("LoadWordsFile: %v", err)
	}
	w := currentWords()
	if len(w.Replacements) != 1 || w.Replacements["思维缜密"] != "心思周密" {
		t.Fatalf("覆盖后替换表错误: %+v", w.Replacements)
	}

	// 替换引擎吃到覆盖表。
	out, _, err := DeSlopRewrite("他思维缜密地推演全局。", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "心思周密") || strings.Contains(out, "思维缜密") {
		t.Fatalf("覆盖替换未生效: %q", out)
	}

	// 打分吃到覆盖黑名单（新黑名单词计 AI 味），分词词表已随 LoadWordsFile 重建。
	scored, err := ScoreTextNoRef("他思维缜密地推演全局。眼帘低垂。")
	if err != nil {
		t.Fatal(err)
	}
	if scored == nil || scored.Score <= 0 {
		t.Fatalf("覆盖黑名单应参与打分: %+v", scored)
	}

	// 文件不存在 = 无覆盖,返回 nil 不报错（内置默认兜底语义）。
	if err := LoadWordsFile(filepath.Join(dir, "missing.json")); err != nil {
		t.Fatalf("缺失覆盖文件应静默: %v", err)
	}
}

package novelstyle

import (
	"strings"
	"testing"
)

// ── 测试样本 ──────────────────────────────────────────────

const fpSample1 = "夜色正浓，他忽然站在院子里，望着远处的灯火。风从廊下吹来，带着淡淡的花香。"
const fpSample2 = "「你怎么了？」她轻声问。他沉默了很久，才忽然抬起头。心里许多话，却不知从何说起。"
const fpSample3 = "一瞬间，他恍然大悟，千钧一发之际，他忽然猛地抓住机会，目瞪口呆地望着这一切。"

// TestComputeFingerprint 指纹各字段非零/合理 + ToJSON/LoadFingerprint 往返一致。
func TestComputeFingerprint(t *testing.T) {
	samples := []string{fpSample1, fpSample2, fpSample3}
	fp, err := ComputeFingerprint(samples)
	if err != nil {
		t.Fatalf("ComputeFingerprint err: %v", err)
	}
	if fp.SentenceLen.Mean <= 0 {
		t.Errorf("SentenceLen.Mean 应 >0, got %v", fp.SentenceLen.Mean)
	}
	if fp.ParaLen.Mean <= 0 {
		t.Errorf("ParaLen.Mean 应 >0, got %v", fp.ParaLen.Mean)
	}
	if fp.TTR1000 <= 0 || fp.TTR1000 > 1 {
		t.Errorf("TTR1000 应 (0,1], got %v", fp.TTR1000)
	}
	if fp.TTRSd < 0 {
		t.Errorf("TTRSd 应 >=0, got %v", fp.TTRSd)
	}
	if fp.HapaxRatio < 0 || fp.HapaxRatio > 1 {
		t.Errorf("HapaxRatio 应 [0,1], got %v", fp.HapaxRatio)
	}
	if fp.LexicalEntropy <= 0 {
		t.Errorf("LexicalEntropy 应 >0, got %v", fp.LexicalEntropy)
	}
	if fp.DialogRatio <= 0 {
		t.Errorf("DialogRatio 应 >0（样本含引号对话）, got %v", fp.DialogRatio)
	}
	if fp.FourCharRatio <= 0 {
		t.Errorf("FourCharRatio 应 >0（样本含四字格）, got %v", fp.FourCharRatio)
	}
	if len(fp.FunctionWordVec) < 30 {
		t.Errorf("FunctionWordVec 词数应 >=30, got %d", len(fp.FunctionWordVec))
	}
	if len(fp.TopBigrams) == 0 {
		t.Errorf("TopBigrams 应为非空")
	}
	if len(fp.AuthorSignWords) == 0 {
		t.Errorf("AuthorSignWords 应为非空（应有重复实词「忽然」）")
	}

	// ToJSON / LoadFingerprint 往返一致
	b, err := fp.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON err: %v", err)
	}
	fp2, err := LoadFingerprint(b)
	if err != nil {
		t.Fatalf("LoadFingerprint err: %v", err)
	}
	b2, err := fp2.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON2 err: %v", err)
	}
	if string(b) != string(b2) {
		t.Errorf("往返 JSON 不一致:\n%s\n%s", b, b2)
	}
}

// TestDelta 相同风格 Delta 小、不同风格 Delta 大（方向断言）。
func TestDelta(t *testing.T) {
	same1 := "夜色正浓，他站在庭院里，望着远处的灯火。风从廊下吹来，带着淡淡的花香。她推开门，一步一步走过去，心里许多话，却什么都说不出口。"
	same2 := "清晨的光落进房间，他醒了，坐起来，望向窗外的树影。鸟鸣一声接一声，院子安静极了。她端着一碗粥走进来，放在桌上，没有说话。时间慢下来，好像什么都不会再改变。"
	diff := "为什么呀？你为什么不说话呢？哦，这样啊，那我也没办法啦！好吧，那你说嘛，到底怎么回事呢？哎呀，真的急死人了呢。你倒是说呀！嗯？怎么这样子嘛？"

	fp1, err := ComputeFingerprint([]string{same1})
	if err != nil {
		t.Fatalf("fp1 err: %v", err)
	}
	fp2, err := ComputeFingerprint([]string{same2})
	if err != nil {
		t.Fatalf("fp2 err: %v", err)
	}
	fpD, err := ComputeFingerprint([]string{diff})
	if err != nil {
		t.Fatalf("fpD err: %v", err)
	}
	dSame := Delta(fp1, fp2)
	dDiff := Delta(fp1, fpD)
	t.Logf("Delta(same)=%.4f Delta(diff)=%.4f", dSame, dDiff)
	if dSame >= dDiff {
		t.Errorf("方向断言失败: 相同风格 Delta(%f) 应小于 不同风格 Delta(%f)", dSame, dDiff)
	}
	if dDiff <= 0 {
		t.Errorf("不同风格 Delta 应 >0, got %f", dDiff)
	}
}

const aiText = `他的眼帘微微上扬，内心充满了震撼，仿佛整个世界都静止了。
然而，他缓缓抬起头。然而，他看着远方。因此，他终于下定决心。
目瞪口呆惊心动魄，千钧一发之际，他忽然猛地想起了一切。`

// hasReason 判断 issues 里是否有 reason 含关键词。
func hasReason(ts *TasteScore, keyword string) bool {
	for _, iss := range ts.Issues {
		if strings.Contains(iss.Reason, keyword) {
			return true
		}
	}
	return false
}

// TestScoreText_DetectsAI AI 味文本命中多规则，且每个 span 落在文本真实位置。
func TestScoreText_DetectsAI(t *testing.T) {
	ts, err := ScoreText(aiText, nil)
	if err != nil {
		t.Fatalf("ScoreText err: %v", err)
	}
	if ts.Score <= 0 {
		t.Errorf("AI 味文本 Score 应 >0, got %d", ts.Score)
	}

	required := []string{"四字", "show-don't-tell", "连接词", "黑名单"}
	hit := 0
	for _, kw := range required {
		if hasReason(ts, kw) {
			hit++
		} else {
			t.Errorf("未命中规则: %s", kw)
		}
	}
	if hit < 3 {
		t.Errorf("应至少命中若干规则，实际命中 %d 个关键词", hit)
	}

	if len(ts.Issues) == 0 {
		t.Fatalf("应至少有一个 issue")
	}
	runes := []rune(aiText)
	for i, iss := range ts.Issues {
		if iss.Start < 0 || iss.End > len(runes) || iss.Start >= iss.End {
			t.Errorf("issue[%d] span 越界: [%d,%d) len=%d", i, iss.Start, iss.End, len(runes))
			continue
		}
		got := string(runes[iss.Start:iss.End])
		if got == "" || !strings.Contains(aiText, got) {
			t.Errorf("issue[%d] span 与原文不符: [%d,%d) => %q", i, iss.Start, iss.End, got)
		}
	}
}

const cleanText = "他推开门，走进院子。月光洒在地上，院子安静下来，只剩下风吹过树叶的沙沙声。她坐在石阶上，抬头看天。"

// TestScoreText_NoRef 无参考指纹也能打分，且干净文本得分更低。
func TestScoreText_NoRef(t *testing.T) {
	clean, err := ScoreTextNoRef(cleanText)
	if err != nil {
		t.Fatalf("ScoreTextNoRef(clean) err: %v", err)
	}
	if clean.Score < 0 || clean.Score > 100 {
		t.Errorf("clean Score 应在 [0,100], got %d", clean.Score)
	}
	ai, err := ScoreTextNoRef(aiText)
	if err != nil {
		t.Fatalf("ScoreTextNoRef(ai) err: %v", err)
	}
	if ai.Score < 0 || ai.Score > 100 {
		t.Errorf("ai Score 应在 [0,100], got %d", ai.Score)
	}
	t.Logf("clean=%d ai=%d", clean.Score, ai.Score)
	if ai.Score <= clean.Score {
		t.Errorf("AI 味文本得分应高于干净文本: ai=%d clean=%d", ai.Score, clean.Score)
	}
}

// TestScoreText_WithRef 提供参考指纹时仍能打分且不报错。
func TestScoreText_WithRef(t *testing.T) {
	fp, err := ComputeFingerprint([]string{fpSample1, fpSample2, fpSample3})
	if err != nil {
		t.Fatalf("ComputeFingerprint err: %v", err)
	}
	ts, err := ScoreText(aiText, fp)
	if err != nil {
		t.Fatalf("ScoreText(with ref) err: %v", err)
	}
	if ts.Score > 0 {
		t.Logf("Score(with ref)=%d", ts.Score)
	}
}

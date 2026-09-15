package routesuggest

import (
	"fmt"
	"strings"
	"testing"
)

func sr(v float64) *float64 { return &v }

// baseInput 基线场景：chat 绑定 xai/grok-4（成功率 0.80、样本足），候选池含
// 明显更优的 deepseek/deepseek-chat（成功率 0.96、成本更低）与样本不足的 herdsman。
func baseInput() Input {
	return Input{
		Features: []FeatureBinding{{Feature: "chat", EngineID: "xai", Model: "grok-4"}},
		Candidates: []Candidate{
			{EngineID: "xai", Model: "grok-4", SuccessRate: sr(0.80), Samples: 100, AvgMs: 4000, CostCNY: 0.02},
			{EngineID: "deepseek", Model: "deepseek-chat", SuccessRate: sr(0.96), Samples: 200, AvgMs: 2500, CostCNY: 0.002},
			{EngineID: "herdsman", Model: "qwen3:32b", IsLocal: true, SuccessRate: sr(0.99), Samples: 5, AvgMs: 8000, CostCNY: 0},
		},
	}
}

func TestSuggestBaselineEmitsOne(t *testing.T) {
	got := Suggest(baseInput(), nil)
	if len(got) != 1 {
		t.Fatalf("期望恰好 1 条建议，得到 %d", len(got))
	}
	s := got[0]
	if s.Feature != "chat" || s.From.EngineID != "xai" || s.To.EngineID != "deepseek" {
		t.Fatalf("建议指向错误： %+v", s)
	}
	if s.ScoreGap < ScoreGapThreshold {
		t.Fatalf("分差 %.3f 低于阈值 %.2f 不应出现", s.ScoreGap, ScoreGapThreshold)
	}
	wantID := BuildSuggestionID("chat", "xai", "grok-4", "deepseek", "deepseek-chat")
	if s.ID != wantID {
		t.Fatalf("ID 不符：got %s want %s", s.ID, wantID)
	}
	if s.Reason == "" || s.Evidence == "" {
		t.Fatal("理由/证据不得为空")
	}
}

func TestSuggestBelowThresholdSilent(t *testing.T) {
	in := baseInput()
	// 把候选压到与当前几乎同分（成功率略高但成本略贵），分差低于阈值。
	in.Candidates[1] = Candidate{EngineID: "deepseek", Model: "deepseek-chat", SuccessRate: sr(0.82), Samples: 200, AvgMs: 2500, CostCNY: 0.02}
	if got := Suggest(in, nil); len(got) != 0 {
		t.Fatalf("分差低于阈值不应出建议，得到 %d 条", len(got))
	}
}

func TestSuggestNoSampleSilent(t *testing.T) {
	in := baseInput()
	// 候选全部无样本：只剩当前绑定自身可评，无候选可比 → 静默。
	in.Candidates = []Candidate{
		{EngineID: "xai", Model: "grok-4", SuccessRate: sr(0.80), Samples: 100, CostCNY: 0.02},
		{EngineID: "glm", Model: "glm-5", SuccessRate: nil, Samples: 0, CostCNY: 0.01},
	}
	if got := Suggest(in, nil); len(got) != 0 {
		t.Fatalf("候选无样本不应出建议，得到 %d 条", len(got))
	}
}

func TestSuggestCurrentNoSampleSilent(t *testing.T) {
	in := baseInput()
	in.Candidates[0].SuccessRate = nil // 当前绑定无样本
	in.Candidates[0].Samples = 0
	if got := Suggest(in, nil); len(got) != 0 {
		t.Fatalf("当前绑定无样本（基线无从比起）不应出建议，得到 %d 条", len(got))
	}
}

func TestSuggestMinSamplesGate(t *testing.T) {
	in := baseInput()
	in.Candidates[1].Samples = MinSamples - 1 // 差一次都不行
	in.Candidates[1].SuccessRate = sr(1.0)
	in.Candidates[1].CostCNY = 0
	if got := Suggest(in, nil); len(got) != 0 {
		t.Fatalf("样本不足（%d<%d）不应出建议", MinSamples-1, MinSamples)
	}
	in.Candidates[1].Samples = MinSamples
	if got := Suggest(in, nil); len(got) != 1 {
		t.Fatalf("样本恰好达标应出建议，得到 %d 条", len(got))
	}
}

func TestSuggestIgnoredSilent(t *testing.T) {
	id := BuildSuggestionID("chat", "xai", "grok-4", "deepseek", "deepseek-chat")
	if got := Suggest(baseInput(), map[string]struct{}{id: {}}); len(got) != 0 {
		t.Fatalf("被忽略 ID 应静默，得到 %d 条", len(got))
	}
}

func TestSuggestLocalCandidateJoinsPool(t *testing.T) {
	// 判据④：本地引擎 0 成本=costFactor 1，同池参评。样本补足后本地应胜出。
	in := baseInput()
	in.Candidates[2].Samples = 50
	got := Suggest(in, nil)
	if len(got) != 1 || got[0].To.EngineID != "herdsman" {
		t.Fatalf("本地引擎（成功率 0.99×成本因子 1）应为最优候选： %+v", got)
	}
	if got[0].Evidence == "" || !strings.Contains(got[0].Evidence, "本地引擎") {
		t.Fatalf("证据串应标注本地引擎：%q", got[0].Evidence)
	}
}

func TestSuggestAllLocalCostDimensionOff(t *testing.T) {
	// 全池成本为 0：成本维度失效，纯成功率比较。
	in := Input{
		Features: []FeatureBinding{{Feature: "novel", EngineID: "herdsman", Model: "qwen3:32b"}},
		Candidates: []Candidate{
			{EngineID: "herdsman", Model: "qwen3:32b", IsLocal: true, SuccessRate: sr(0.70), Samples: 100, CostCNY: 0},
			{EngineID: "ollama", Model: "llama3:70b", IsLocal: true, SuccessRate: sr(0.97), Samples: 100, CostCNY: 0},
		},
	}
	got := Suggest(in, nil)
	if len(got) != 1 || got[0].To.Model != "llama3:70b" {
		t.Fatalf("全本地池应按纯成功率选优： %+v", got)
	}
}

func TestSuggestMultiFeatureIndependent(t *testing.T) {
	in := Input{
		Features: []FeatureBinding{
			{Feature: "chat", EngineID: "xai", Model: "grok-4"},
			{Feature: "novel", EngineID: "glm", Model: "glm-5"},
		},
		Candidates: []Candidate{
			{EngineID: "xai", Model: "grok-4", SuccessRate: sr(0.80), Samples: 100, CostCNY: 0.02},
			{EngineID: "glm", Model: "glm-5", SuccessRate: sr(0.85), Samples: 100, CostCNY: 0.01},
			{EngineID: "deepseek", Model: "deepseek-chat", SuccessRate: sr(0.99), Samples: 300, CostCNY: 0.002},
		},
	}
	got := Suggest(in, nil)
	if len(got) != 2 {
		t.Fatalf("两个功能域应各自出建议，得到 %d 条", len(got))
	}
	// 分差排序：两域分差相同（同一最优候选、不同当前），退回功能名字典序。
	if got[0].Feature != "chat" || got[1].Feature != "novel" {
		t.Fatalf("排序不稳定：%+v", got)
	}
}

func TestSuggestDeterministicIDs(t *testing.T) {
	a := Suggest(baseInput(), nil)
	b := Suggest(baseInput(), nil)
	if len(a) != len(b) {
		t.Fatal("两次重算条数不一致")
	}
	for i := range a {
		if a[i].ID != b[i].ID || fmt.Sprintf("%.6f", a[i].ScoreGap) != fmt.Sprintf("%.6f", b[i].ScoreGap) {
			t.Fatalf("重算不幂等：%+v vs %+v", a[i], b[i])
		}
	}
}

func TestSuggestionIDRoundTrip(t *testing.T) {
	id := BuildSuggestionID("office", "xai", "grok-4", "herdsman", "qwen3:32b")
	f, fe, fm, te, tm, ok := ParseSuggestionID(id)
	if !ok || f != "office" || fe != "xai" || fm != "grok-4" || te != "herdsman" || tm != "qwen3:32b" {
		t.Fatalf("ID 往返失败：%q → %q %q %q %q %q ok=%v", id, f, fe, fm, te, tm, ok)
	}
	for _, bad := range []string{"", "chat", "chat|xai>grok-4", "chat||grok-4>a>b>c", "chat|xai>>>b"} {
		if _, _, _, _, _, ok := ParseSuggestionID(bad); ok {
			t.Fatalf("非法 ID 应解析失败：%q", bad)
		}
	}
}

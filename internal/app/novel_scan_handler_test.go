package app

import (
	"fmt"
	"strings"
	"testing"
)

// ── 选题扫榜（novel-scan 蒸馏刀）测试：复用剧情分支的 httptest SSE 桩环境 ──

// scanReply 构造一段合规扫榜 JSON 回复：四条选题覆盖「维持高/样本封顶/中不动/趋势未知封顶」。
func scanReply() string {
	b := `{"market":{"genre_distribution":"都市脑洞 8 本居首","new_signals":"规则怪谈抬头","head_books":"老书靠稳定更新","classic_shifts":"玄幻转向轻繁衍","length_pattern":"200-300 万字","title_patterns":"长句梗概式书名","blurb_selling_points":"反差开局","new_elements":"直播流叠加"},"topics":[` +
		`{"direction":"都市规则怪谈","genre_mix":"都市+悬疑","audience":"男频 18-30","selling_point":"规则怪谈+直播流","why_hot":"假设：怪谈热延续","market_evidence":"粘贴文本中怪谈标签 20 本","differentiation":"主播视角差异化","feasibility":"高","feasibility_reason":"样本足、趋势涨","direction_sample":20,"trend":"涨","risks":"同质化","validation":"先写黄金三章看追读","benchmarks":"《怪谈电台》","scope":"200 万字·起点"},` +
		`{"direction":"女频种田","genre_mix":"古言+种田","audience":"女频 20-35","selling_point":"空间种田","why_hot":"假设","market_evidence":"粘贴文本仅 5 本","differentiation":"加美食线","feasibility":"高","feasibility_reason":"自评高","direction_sample":5,"trend":"涨","risks":"慢热","validation":"三章试水","benchmarks":"","scope":"100 万字"},` +
		`{"direction":"克苏鲁西幻","genre_mix":"奇幻+克苏鲁","audience":"男频","selling_point":"诡异氛围","why_hot":"假设","market_evidence":"内置知识·未实时校验","differentiation":"san 值面板","feasibility":"中","feasibility_reason":"样本不足","direction_sample":3,"trend":"平","risks":"小众","validation":"短篇试水","benchmarks":"","scope":"80 万字"},` +
		`{"direction":"御兽流","genre_mix":"玄幻+御兽","audience":"男频 15-25","selling_point":"宠兽养成","why_hot":"假设","market_evidence":"粘贴文本 20 本","differentiation":"御兽图鉴化","feasibility":"高","feasibility_reason":"样本足","direction_sample":20,"trend":"未知","risks":"红海","validation":"三章试水","benchmarks":"","scope":"250 万字"}]}`
	return b
}

// scanPaste 构造首行标题 + n 条条目的粘贴榜单。
func scanPaste(n int) string {
	var sb strings.Builder
	sb.WriteString("# 起点·风云榜\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "%d. 《测试书名%d》：都市脑洞，开局签到，简介一句话。\n", i, i)
	}
	return sb.String()
}

func TestNovelScanAnalyze_DeterministicCapsAndSource(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{scanReply()})

	res, err := env.a.NovelScanAnalyze("", scanPaste(20))
	if err != nil {
		t.Fatalf("NovelScanAnalyze: %v", err)
	}

	source, ok := res["source"].(map[string]interface{})
	if !ok {
		t.Fatalf("source 应为 map: %+v", res["source"])
	}
	if got := source["entry_count"].(int); got != 20 {
		t.Fatalf("entry_count 应为 20, got %v", got)
	}
	if got := source["platform"].(string); got != "起点" {
		t.Fatalf("平台应从标题推断为「起点」, got %q", got)
	}
	if verified, _ := source["verified"].(bool); verified {
		t.Fatalf("粘贴来源必须 verified=false（诚实纪律）")
	}

	topics, ok := res["topics"].([]novelScanTopic)
	if !ok || len(topics) != 4 {
		t.Fatalf("topics 应为 4 条: %T %v", res["topics"], res["topics"])
	}
	if topics[0].Feasibility != "高" {
		t.Fatalf("样本足+趋势涨的「高」不应被封顶, got %q (%s)", topics[0].Feasibility, topics[0].FeasibilityReason)
	}
	if topics[1].Feasibility != "中" || !strings.Contains(topics[1].FeasibilityReason, "同方向样本 5<15") {
		t.Fatalf("样本不足的「高」应封顶为中并写明原因, got %q (%s)", topics[1].Feasibility, topics[1].FeasibilityReason)
	}
	if topics[2].Feasibility != "中" {
		t.Fatalf("「中」不应被动, got %q", topics[2].Feasibility)
	}
	if topics[3].Feasibility != "中" || !strings.Contains(topics[3].FeasibilityReason, "趋势未知") {
		t.Fatalf("趋势未知的「高」应封顶为中, got %q (%s)", topics[3].Feasibility, topics[3].FeasibilityReason)
	}
}

func TestNovelScanAnalyze_ShortPasteRejected(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{scanReply()})
	if _, err := env.a.NovelScanAnalyze("", "太短"); err == nil || !strings.Contains(err.Error(), "至少 50 字") {
		t.Fatalf("过短粘贴应被拒: %v", err)
	}
}

func TestNovelScanAnalyze_BadReplyRejected(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{"抱歉，我无法以 JSON 形式回答，以下是分析：……"})
	if _, err := env.a.NovelScanAnalyze("", scanPaste(20)); err == nil || !strings.Contains(err.Error(), "扫榜") {
		t.Fatalf("坏回复应如实报错: %v", err)
	}
}

func TestParseRankPaste_HeaderAndEntries(t *testing.T) {
	platform, entries := parseRankPaste("# 番茄·新书榜\n\n1. 书A：简介\n2. 书B：简介\n---\n书C 简介")
	if platform != "番茄" {
		t.Fatalf("平台推断: got %q", platform)
	}
	if len(entries) != 3 {
		t.Fatalf("条目应 3 条（空行/标题/分隔线不计）, got %d: %v", len(entries), entries)
	}
	if p2, _ := parseRankPaste("随便粘的一段"); p2 != "" {
		t.Fatalf("无标题不应推断平台, got %q", p2)
	}
}

func TestCapScanFeasibility_CihuimaoThreshold(t *testing.T) {
	topics := []novelScanTopic{
		{Feasibility: "高", DirectionSample: 12, Trend: "涨"},
	}
	capScanFeasibility(topics, 12, "刺猬猫")
	if topics[0].Feasibility != "高" {
		t.Fatalf("刺猬猫门槛 10：样本 12 应维持「高」, got %q", topics[0].Feasibility)
	}
	capScanFeasibility(topics, 12, "起点")
	if topics[0].Feasibility != "中" {
		t.Fatalf("起点门槛 15：样本 12 应封顶「中」, got %q", topics[0].Feasibility)
	}
}

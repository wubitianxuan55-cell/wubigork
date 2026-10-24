package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/util"
)

// ── 选题扫榜（oh-story story-long-scan 机制重推导蒸馏）──────────────────
//
// 数据来源=作者粘贴的榜单文本（上游三级降级中的「作者粘贴」级；脚本爬榜通路
// 明确不做——ToS 灰色且站点脆弱，见 docs/gaea-novel-skill-distill-survey-2026-10.md
// D 档）。分析走 LLM（novel-scan 模板），可行性门槛做确定性封顶（模板约束对
// 模型是软的，这里是硬的，同 clampBranchPitch 范式）：
//   - 同方向样本数 < 平台阈值（刺猬猫 10，其余 15）→ 高封顶为中
//   - 环比趋势未知（含首次扫榜无上期可比）→ 高封顶为中
//   - 粘贴样本总量 < 平台阈值 → 全部高封顶为中
// 报告只返回不落盘（建议制红线：产物落盘须用户显式发起）。

// novelScanTopic 单条选题决策（12+2 字段：上游固定字段 + 门槛判定输入）。
type novelScanTopic struct {
	Direction         string `json:"direction"`
	GenreMix          string `json:"genre_mix"`
	Audience          string `json:"audience"`
	SellingPoint      string `json:"selling_point"`
	WhyHot            string `json:"why_hot"`
	MarketEvidence    string `json:"market_evidence"`
	Differentiation   string `json:"differentiation"`
	Feasibility       string `json:"feasibility"`
	FeasibilityReason string `json:"feasibility_reason"`
	DirectionSample   int    `json:"direction_sample"`
	Trend             string `json:"trend"`
	Risks             string `json:"risks"`
	Validation        string `json:"validation"`
	Benchmarks        string `json:"benchmarks"`
	Scope             string `json:"scope"`
}

// novelScanReport LLM 回复结构：八维市场观察 + 选题列表。
type novelScanReport struct {
	Market map[string]string `json:"market"`
	Topics []novelScanTopic  `json:"topics"`
}

// novelScanMaxInputRunes 粘贴文本进入 prompt 的预算（rune 口径）；超出截断
// 并在尾部标注，防超长粘贴灌爆上下文。
const novelScanMaxInputRunes = 8000

// NovelScanAnalyze 选题扫榜：解析粘贴榜单 → LLM 八维观察+选题决策 → 确定性
// 可行性封顶 → 返回报告（不落盘）。platform 为空时从粘贴文本首行标题推断
// （「# 起点·风云榜」→ 起点）。
func (a *writingState) NovelScanAnalyze(platform, rankText string) (map[string]interface{}, error) {
	rankText = strings.TrimSpace(rankText)
	if len([]rune(rankText)) < 50 {
		return nil, fmt.Errorf("扫榜文本太短：请粘贴榜单内容（书名/题材/简介等，至少 50 字）")
	}

	pastePlatform, entries := parseRankPaste(rankText)
	if strings.TrimSpace(platform) == "" {
		platform = pastePlatform
	}

	tmpl := a.eng.Get("novel-scan")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 novel-scan 模板文件")
	}
	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"platform":    platformOrUnknown(platform),
		"rank_text":   clampRunes(rankText, novelScanMaxInputRunes),
		"entry_count": strconv.Itoa(len(entries)),
	})

	eng, model, _ := a.routeModel("novel")
	// S1.5-B play 内容护栏：同剧情分支（max_output_tokens 钳制）。
	opts := ai.ChatSimpleOptions{EngineID: eng, Feature: "novel"}
	applyChatSimpleMaxTokens(&opts, playGuardrails().MaxOutputTokens)
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("扫榜分析失败: %w", err)
	}

	report, err := parseNovelScanReply(reply)
	if err != nil {
		return nil, err
	}

	capScanFeasibility(report.Topics, len(entries), platform)

	return map[string]interface{}{
		"market": report.Market,
		"topics": report.Topics,
		"source": map[string]interface{}{
			"kind":        "paste",
			"platform":    platformOrUnknown(platform),
			"entry_count": len(entries),
			"verified":    false,
			"disclaimer":  "榜单为作者粘贴文本，未实时校验；市场判断按扫描样本估算，引用内置知识处已标注",
		},
	}, nil
}

// parseNovelScanReply 从 AI 回复提取并归一扫榜 JSON。
func parseNovelScanReply(reply string) (*novelScanReport, error) {
	jsonStr := util.ExtractJSON(reply)
	var result novelScanReport
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析扫榜 JSON 失败: %w", err)
	}
	if len(result.Topics) == 0 {
		return nil, fmt.Errorf("AI 未给出任何选题决策")
	}
	for i := range result.Topics {
		t := &result.Topics[i]
		switch t.Feasibility {
		case "高", "中", "低":
		default:
			t.Feasibility = "中" // 模型口径漂移时落保守档
		}
		switch t.Trend {
		case "涨", "平", "跌":
		default:
			t.Trend = "未知"
		}
		if t.DirectionSample < 0 {
			t.DirectionSample = 0
		}
		t.Direction = clampRunes(t.Direction, 120)
		t.SellingPoint = clampRunes(t.SellingPoint, 200)
		t.MarketEvidence = clampRunes(t.MarketEvidence, 400)
		t.WhyHot = clampRunes(t.WhyHot, 200)
	}
	return &result, nil
}

// scanFeasibilityThreshold 可行性「高」的同方向样本门槛（上游：起点/番茄/晋江/
// 七猫 15 本，刺猬猫 10 本；未知平台按 15 从紧）。
func scanFeasibilityThreshold(platform string) int {
	if strings.Contains(platform, "刺猬猫") {
		return 10
	}
	return 15
}

// capScanFeasibility 可行性量化封顶（确定性）：只有「高」需要封顶；中/低保持
// 原判（封顶只降不升）。封顶原因追加进 feasibility_reason，作者能看到为什么。
func capScanFeasibility(topics []novelScanTopic, totalEntries int, platform string) {
	th := scanFeasibilityThreshold(platform)
	globalShort := totalEntries < th
	for i := range topics {
		t := &topics[i]
		if t.Feasibility != "高" {
			continue
		}
		reason := ""
		switch {
		case t.DirectionSample < th:
			reason = fmt.Sprintf("同方向样本 %d<%d", t.DirectionSample, th)
		case t.Trend == "未知":
			reason = "环比趋势未知（首次扫榜无上期可比）"
		case globalShort:
			reason = fmt.Sprintf("粘贴样本总量 %d<%d", totalEntries, th)
		default:
			continue
		}
		t.Feasibility = "中"
		t.FeasibilityReason = strings.TrimSpace(clampRunes(t.FeasibilityReason, 300) + "；按量化门槛封顶为中：" + reason)
	}
}

// parseRankPaste 解析粘贴榜单：首行「# 平台·榜单名」标题推断平台；其余非空
// 非标题行各算一条条目（上游约定「每书一段」，粘贴面按行计数容忍各种排版）。
func parseRankPaste(text string) (platform string, entries []string) {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "---" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if platform == "" {
				h := strings.TrimLeft(trimmed, "# \t")
				platform = rankHeaderPlatform(h)
			}
			continue
		}
		entries = append(entries, trimmed)
	}
	return platform, entries
}

// rankHeaderPlatform 从「起点·风云榜」「番茄 - 新书榜」式标题取平台名。
func rankHeaderPlatform(header string) string {
	for _, sep := range []string{"·", "•", "｜", "|", "-", "—", "：", ":"} {
		if j := strings.Index(header, sep); j > 0 {
			return strings.TrimSpace(header[:j])
		}
	}
	return strings.TrimSpace(header)
}

func platformOrUnknown(platform string) string {
	if strings.TrimSpace(platform) == "" {
		return "（未注明）"
	}
	return platform
}

// clampRunes 把字符串钳到 max rune 内（优先句界截断，无句读硬截加省略号）。
func clampRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	budget := runes[:max]
	for i := len(budget) - 1; i >= max/2; i-- {
		switch budget[i] {
		case '。', '！', '？', '；', '…':
			return string(budget[:i+1])
		}
	}
	return string(budget[:max-1]) + "…"
}

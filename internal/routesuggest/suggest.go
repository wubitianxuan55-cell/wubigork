// Package routesuggest 路由学习建议（阶段七 7.1-2）：按「成本 × 质量」评分差
// 生成功能域改绑建议的纯函数包。零外部依赖、零 IO——全部数据由调用方组装，
// 便于表驱动测试（先例：internal/memoryeval、internal/novelgate）。
//
// 评分与阈值（宁少勿扰）：
//   - score = quality × costFactor；quality=成功率（无样本/样本不足不出建议）
//   - costFactor 在该功能域候选集内归一：越便宜越接近 1，本地（成本 0）= 1
//   - 候选 score − 当前绑定 score ≥ ScoreGapThreshold 才提议；单功能至多 1 条
//   - ID 确定性（重算幂等），被忽略的 ID 静默跳过
package routesuggest

import (
	"fmt"
	"sort"
	"strings"
)

// ScoreGapThreshold 评分差阈值：候选优于当前的最小分差，低于它不出建议。
const ScoreGapThreshold = 0.25

// MinSamples 候选/当前绑定的最小样本量：低于它视为证据不足，不出建议。
const MinSamples = 20

// FeatureBinding 功能域当前生效绑定（routeModel 解析结果）。
type FeatureBinding struct {
	Feature  string
	EngineID string
	Model    string
}

// Candidate 候选模型（含当前绑定自身——评分需要同一口径）。
// SuccessRate 为 nil 表示无样本；CostCNY 为「单次调用」估算成本（窗口总额/调用数），
// 由调用方归一，本地引擎恒 0；TTFTMs/TPS 可选（herdsman benchmark），只进证据串不进公式。
type Candidate struct {
	EngineID    string   `json:"engine_id"`
	Model       string   `json:"model"`
	IsLocal     bool     `json:"is_local"`
	SuccessRate *float64 `json:"success_rate,omitempty"` // nil=无样本（序列化缺省）
	AvgMs       int64    `json:"avg_ms"`
	CostCNY     float64  `json:"cost_cny"`
	Samples     int64    `json:"samples"`
	TTFTMs      *float64 `json:"ttft_ms,omitempty"`
	TPS         *float64 `json:"tps,omitempty"`
}

// Endpoint 绑定端点。
type Endpoint struct {
	EngineID string `json:"engine_id"`
	Model    string `json:"model"`
}

// Suggestion 一条改绑建议。
type Suggestion struct {
	ID       string    `json:"id"`
	Feature  string    `json:"feature"`
	Reason   string    `json:"reason"`
	Evidence string    `json:"evidence"`
	From     Endpoint  `json:"from"`
	To       Candidate `json:"to"`
	ScoreGap float64   `json:"score_gap"`
}

// Input 建议生成输入：功能绑定集合 + 候选池（全部启用引擎的 llm 模型，本地云端同池）。
type Input struct {
	Features   []FeatureBinding
	Candidates []Candidate
}

const costEpsilon = 1e-6

// Suggest 生成建议。评分差的分母口径：候选池 + 当前绑定在同一功能域内做成本归一。
func Suggest(in Input, ignored map[string]struct{}) []Suggestion {
	// 候选索引（engine|model → 候选），供当前绑定取质量数据。
	idx := make(map[string]Candidate, len(in.Candidates))
	for _, c := range in.Candidates {
		idx[c.EngineID+"|"+c.Model] = c
	}
	var out []Suggestion
	for _, fb := range in.Features {
		cur, ok := idx[fb.EngineID+"|"+fb.Model]
		if !ok || !eligible(cur) {
			continue // 当前绑定无有效质量数据——证据不足不出声
		}
		// 本功能域成本归一分母：候选 + 当前
		maxCost := cur.CostCNY
		for _, c := range in.Candidates {
			if c.CostCNY > maxCost {
				maxCost = c.CostCNY
			}
		}
		curScore := score(cur, maxCost)
		var best *Candidate
		bestScore := curScore
		for i := range in.Candidates {
			c := in.Candidates[i]
			if c.EngineID == fb.EngineID && c.Model == fb.Model {
				continue
			}
			if !eligible(c) {
				continue
			}
			if s := score(c, maxCost); s > bestScore {
				best = &in.Candidates[i]
				bestScore = s
			}
		}
		if best == nil || bestScore-curScore < ScoreGapThreshold {
			continue
		}
		sg := buildSuggestion(fb, cur, *best, bestScore-curScore)
		if _, skip := ignored[sg.ID]; skip {
			continue
		}
		out = append(out, sg)
	}
	// 稳定排序：分差降序、同分差按功能名——输出确定性。
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScoreGap != out[j].ScoreGap {
			return out[i].ScoreGap > out[j].ScoreGap
		}
		return out[i].Feature < out[j].Feature
	})
	return out
}

// eligible 候选可参评：有成功率样本且样本量达标。
func eligible(c Candidate) bool {
	return c.SuccessRate != nil && c.Samples >= MinSamples
}

// score quality × costFactor（maxCost≤ε 时成本维度失效，全体 factor=1）。
func score(c Candidate, maxCost float64) float64 {
	q := *c.SuccessRate
	if maxCost <= costEpsilon {
		return q
	}
	factor := (maxCost - c.CostCNY + costEpsilon) / maxCost
	return q * factor
}

// BuildSuggestionID 确定性 ID（重算幂等）。约束：engine/model 不含 "|" 与 ">"（gaea 命名如此）。
func BuildSuggestionID(feature, fromEng, fromModel, toEng, toModel string) string {
	return feature + "|" + fromEng + ">" + fromModel + ">" + toEng + ">" + toModel
}

// ParseSuggestionID 解析 BuildSuggestionID 产物。
func ParseSuggestionID(id string) (feature, fromEng, fromModel, toEng, toModel string, ok bool) {
	head, rest, found := strings.Cut(id, "|")
	if !found || strings.ContainsAny(head, ">|") {
		return "", "", "", "", "", false
	}
	parts := strings.Split(rest, ">")
	if len(parts) != 4 {
		return "", "", "", "", "", false
	}
	for _, p := range parts {
		if p == "" || strings.Contains(p, "|") {
			return "", "", "", "", "", false
		}
	}
	return head, parts[0], parts[1], parts[2], parts[3], true
}

// buildSuggestion 组装修饰（中文理由/证据，供前端直显）。
func buildSuggestion(fb FeatureBinding, cur, to Candidate, gap float64) Suggestion {
	id := BuildSuggestionID(fb.Feature, fb.EngineID, fb.Model, to.EngineID, to.Model)
	reason := fmt.Sprintf("同功能下 %s/%s 评分领先当前 %.2f（成功率 %.2f vs %.2f，单次成本 %.4f vs %.4f CNY）",
		to.EngineID, to.Model, gap, *to.SuccessRate, *cur.SuccessRate, to.CostCNY, cur.CostCNY)
	var ev strings.Builder
	fmt.Fprintf(&ev, "候选：样本 %d 次、均时长 %dms", to.Samples, to.AvgMs)
	if to.IsLocal {
		ev.WriteString("、本地引擎（零成本）")
	}
	if to.TTFTMs != nil && to.TPS != nil {
		fmt.Fprintf(&ev, "、TTFT %.0fms/TPS %.1f", *to.TTFTMs, *to.TPS)
	}
	fmt.Fprintf(&ev, "；当前：样本 %d 次、均时长 %dms", cur.Samples, cur.AvgMs)
	return Suggestion{
		ID: id, Feature: fb.Feature, Reason: reason, Evidence: ev.String(),
		From: Endpoint{EngineID: fb.EngineID, Model: fb.Model},
		To:   to, ScoreGap: gap,
	}
}

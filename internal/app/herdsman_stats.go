package app

// Herdsman 调用统计（P4）：读取 model_stats/events.jsonl（Herdsman 逐请求性能
// 遥测：模型/类型/运行时/状态/耗时/TTFT/TPS/token），按模型聚合，供模型中心展示
// 本机真实性能数据。

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HerdsmanModelStat 单模型聚合统计。
type HerdsmanModelStat struct {
	Model           string  `json:"model"`
	Type            string  `json:"type"`
	Runtime         string  `json:"runtime"`
	Calls           int     `json:"calls"`
	Succeeded       int     `json:"succeeded"`
	Failed          int     `json:"failed"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	TotalDurationMs int64   `json:"total_duration_ms"`
	AvgDurationMs   int64   `json:"avg_duration_ms"`
	AvgTTFTMs       float64 `json:"avg_ttft_ms"`
	AvgPromptTPS    float64 `json:"avg_prompt_tps"`
	AvgPredictedTPS float64 `json:"avg_predicted_tps"`
	LastCalledAt    string  `json:"last_called_at"`
}

// HerdsmanModelStats 模型中心「Herdsman 本地用量」载荷。
type HerdsmanModelStats struct {
	Total    int                 `json:"total"`
	Since    string              `json:"since"`
	PerModel []HerdsmanModelStat `json:"per_model"`
	Source   string              `json:"source"`
	Error    string              `json:"error,omitempty"`
}

type herdsmanStatEvent struct {
	Model        string  `json:"model"`
	ModelType    string  `json:"model_type"`
	Runtime      string  `json:"runtime"`
	Status       string  `json:"status"`
	DurationMS   int64   `json:"duration_ms"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TTFTMS       float64 `json:"ttft_ms"`
	PromptTPS    float64 `json:"prompt_tps"`
	PredictedTPS float64 `json:"predicted_tps"`
	StartedAt    string  `json:"started_at"`
	EndedAt      string  `json:"ended_at"`
}

// parseHerdsmanModelStats 解析 events.jsonl 并按模型聚合（按调用次数降序）。
func parseHerdsmanModelStats(data []byte) (HerdsmanModelStats, error) {
	out := HerdsmanModelStats{Source: "herdsman-model-stats"}
	agg := map[string]*aggregatedStat{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		var ev herdsmanStatEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			continue
		}
		if strings.TrimSpace(ev.Model) == "" {
			continue
		}
		s := agg[ev.Model]
		if s == nil {
			s = &aggregatedStat{model: ev.Model, typ: ev.ModelType, runtime: ev.Runtime}
			agg[ev.Model] = s
		}
		s.calls++
		if ev.Status == "succeeded" {
			s.succeeded++
		} else {
			s.failed++
		}
		s.input += ev.InputTokens
		s.output += ev.OutputTokens
		s.duration += ev.DurationMS
		if ev.TTFTMS > 0 {
			s.ttftSum += ev.TTFTMS
			s.ttftN++
		}
		if ev.PromptTPS > 0 {
			s.promptSum += ev.PromptTPS
			s.promptN++
		}
		if ev.PredictedTPS > 0 {
			s.predSum += ev.PredictedTPS
			s.predN++
		}
		if s.lastCalled == "" || ev.EndedAt > s.lastCalled {
			s.lastCalled = ev.EndedAt
		}
		if out.Since == "" || (ev.StartedAt != "" && ev.StartedAt < out.Since) {
			out.Since = ev.StartedAt
		}
	}
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("读取 model_stats 失败: %w", err)
	}
	list := make([]HerdsmanModelStat, 0, len(agg))
	for _, s := range agg {
		item := HerdsmanModelStat{
			Model:           s.model,
			Type:            s.typ,
			Runtime:         s.runtime,
			Calls:           s.calls,
			Succeeded:       s.succeeded,
			Failed:          s.failed,
			InputTokens:     s.input,
			OutputTokens:    s.output,
			TotalDurationMs: s.duration,
			LastCalledAt:    s.lastCalled,
		}
		if s.calls > 0 {
			item.AvgDurationMs = s.duration / int64(s.calls)
		}
		if s.ttftN > 0 {
			item.AvgTTFTMs = s.ttftSum / float64(s.ttftN)
		}
		if s.promptN > 0 {
			item.AvgPromptTPS = s.promptSum / float64(s.promptN)
		}
		if s.predN > 0 {
			item.AvgPredictedTPS = s.predSum / float64(s.predN)
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Calls != list[j].Calls {
			return list[i].Calls > list[j].Calls
		}
		return list[i].Model < list[j].Model
	})
	out.PerModel = list
	out.Total = len(list)
	return out, nil
}

type aggregatedStat struct {
	model      string
	typ        string
	runtime    string
	calls      int
	succeeded  int
	failed     int
	input      int64
	output     int64
	duration   int64
	ttftSum    float64
	ttftN      int
	promptSum  float64
	promptN    int
	predSum    float64
	predN      int
	lastCalled string
}

// HerdsmanModelStats 返回 Herdsman 本地逐请求遥测的按模型聚合统计。
func (a *App) HerdsmanModelStats() (HerdsmanModelStats, error) {
	dataDir := herdsmanDataDir()
	if dataDir == "" {
		return HerdsmanModelStats{Source: "herdsman-model-stats", Error: "无法定位 Herdsman 数据目录"}, fmt.Errorf("无法定位 Herdsman 数据目录")
	}
	data, err := os.ReadFile(filepath.Join(dataDir, "model_stats", "events.jsonl"))
	if err != nil {
		return HerdsmanModelStats{Source: "herdsman-model-stats", Error: err.Error()}, err
	}
	return parseHerdsmanModelStats(data)
}
